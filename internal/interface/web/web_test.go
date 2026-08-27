package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/petherin/onto/internal/application/facade"
	"github.com/petherin/onto/internal/bootstrap"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer builds a *Server backed by an isolated, per-test data file so
// these tests never touch the repo's real data/locations.json, mirroring the
// helper used by the TUI tests.
func newTestServer(t *testing.T, opts ...facade.Option) *Server {
	t.Helper()
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))

	state, err := bootstrap.Bootstrap(bootstrap.DefaultConfig())
	require.NoError(t, err)
	app, err := facade.New(
		state.Universe,
		state.Repo,
		state.StartID,
		navigation.NewBFSPathfinder(),
		universe.NewSequentialLocationGenerator(),
		opts...,
	)
	require.NoError(t, err)
	return NewServer(app)
}

func getJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(out))
}

func postJSON(t *testing.T, url, body string, out any) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(out))
}

// TestBrowsableURL covers the linkification of listen addresses: host-less and
// all-interface addresses become localhost; explicit hosts are preserved.
func TestBrowsableURL(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{"host-less all-interfaces", ":8090", "http://localhost:8090"},
		{"explicit all-interfaces v4", "0.0.0.0:8090", "http://localhost:8090"},
		{"explicit all-interfaces v6", "[::]:8090", "http://localhost:8090"},
		{"explicit host preserved", "127.0.0.1:9000", "http://127.0.0.1:9000"},
		{"named host preserved", "localhost:3000", "http://localhost:3000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, browsableURL(tt.addr))
		})
	}
}

// TestExecute_HomeConfirmHandshake verifies the two-step home flow: from a
// routable location 'home' shows a plan and arms confirmation, then 'y'
// completes the journey back to the start location.
func TestExecute_HomeConfirmHandshake(t *testing.T) {
	s := newTestServer(t)
	start := s.app.Snapshot().Location

	s.execute("travel station")

	plan := s.execute("home")
	require.True(t, s.awaitingHomeConfirm, "home should arm confirmation from a routable location")
	assert.Contains(t, plan, "Route home")
	assert.True(t, facade.NeedsHomeConfirm(plan))

	s.execute("y")
	assert.False(t, s.awaitingHomeConfirm, "confirmation flag clears after y")
	assert.Equal(t, start, s.app.Snapshot().Location)
}

// TestExecute_HomeCancel confirms answering 'n' cancels the journey without
// moving the traveller.
func TestExecute_HomeCancel(t *testing.T) {
	s := newTestServer(t)
	s.execute("travel station")

	s.execute("home")
	require.True(t, s.awaitingHomeConfirm)

	resp := s.execute("n")
	assert.Equal(t, "Cancelled.", resp)
	assert.False(t, s.awaitingHomeConfirm)
	assert.Equal(t, "station", s.app.Snapshot().Location, "cancelling must not move the traveller")
}

// TestExecute_DeadEnd_NoConfirmPrompt covers the fixed bug: from a dead end
// with no path home, 'home' reports no route and never arms confirmation.
func TestExecute_DeadEnd_NoConfirmPrompt(t *testing.T) {
	s := newTestServer(t)
	s.execute("travel park")

	resp := s.execute("home")
	assert.Contains(t, resp, "No route home")
	assert.False(t, s.awaitingHomeConfirm)
}

// TestHandlers exercises the JSON API over httptest: state is served, execute
// requires POST, and a posted command is reflected in the returned session.
func TestHandlers(t *testing.T) {
	s := newTestServer(t)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	var initial stateDTO
	getJSON(t, srv.URL+"/api/state", &initial)
	require.NotEmpty(t, initial.Session.Location)

	resp, err := http.Get(srv.URL + "/api/execute")
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	var moved stateDTO
	postJSON(t, srv.URL+"/api/execute", `{"command":"travel station"}`, &moved)
	assert.Equal(t, "station", moved.Session.Location)
}

// TestReset_RestoresStartingMap covers the full server-side reset: after a
// reality transition has grown the graph and moved the session off base
// reality, POST /api/reset rebuilds the starting map (only the four starter
// nodes) and returns the session home in base reality.
func TestReset_RestoresStartingMap(t *testing.T) {
	s := newTestServer(t)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	starterIDs := s.app.GraphSnapshot().Nodes
	require.Len(t, starterIDs, 4, "the starter map has four nodes")

	// Grow the graph with a reality transition; the branch adds new locations.
	postJSON(t, srv.URL+"/api/execute", `{"command":"shift"}`, &stateDTO{})
	grown := s.app.GraphSnapshot()
	require.Greater(t, len(grown.Nodes), 4, "shift must branch new locations onto the map")

	var reset stateDTO
	postJSON(t, srv.URL+"/api/reset", `{}`, &reset)

	assert.Len(t, reset.Graph.Nodes, 4, "reset restores only the starter nodes")
	assert.Equal(t, "home", reset.Session.Location, "reset returns the session home")
	assert.Equal(t, "Q0", reset.Session.Quantum, "reset returns to base reality")
	assert.Equal(t, 0.0, reset.Session.CumulativeCost, "reset clears the journey cost")
	assert.True(t, reset.Dirty, "reset diverges from the saved map, so it is dirty")

	// A non-POST request is rejected, mirroring the other mutating endpoints.
	resp, err := http.Get(srv.URL + "/api/reset")
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// nodesByID indexes a graph snapshot's nodes for direct lookup in assertions.
func nodesByID(g facade.GraphSnapshot) map[string]facade.NodeSnapshot {
	m := make(map[string]facade.NodeSnapshot, len(g.Nodes))
	for _, n := range g.Nodes {
		m[n.ID] = n
	}
	return m
}

// TestGraphSnapshot_Reachability covers the reachability flag the web UI colours
// nodes by: from the start location every physically-connected node is reachable
// and the current node itself is not, edges expose their travel mode, and the
// flag recomputes for the new location after a move — from the 'park' dead end
// (no outgoing edges) nothing is reachable.
func TestGraphSnapshot_Reachability(t *testing.T) {
	s := newTestServer(t)
	start := s.app.Snapshot().Location

	g := s.app.GraphSnapshot()
	nodes := nodesByID(g)
	require.Contains(t, nodes, start, "start location must appear in the graph")
	assert.False(t, nodes[start].Reachable, "the current location is never marked reachable")
	assert.True(t, nodes["station"].Reachable, "station is one physical hop from home")
	assert.True(t, nodes["park"].Reachable, "park is one physical hop from home")
	assert.True(t, nodes["city-centre"].Reachable, "city-centre is reachable via the rail hop from station")

	modes := map[string]bool{}
	require.NotEmpty(t, g.Edges, "edges must be exposed for the map to draw")
	for _, e := range g.Edges {
		modes[e.Mode] = true
	}
	assert.True(t, modes["walk"], "walk edges are exposed with their mode string")
	assert.True(t, modes["rail"], "rail edges are exposed with their mode string")

	// Reachability is relative to the current location: after moving to park —
	// from which there is no route back — every node reachable from home flips to
	// unreachable, proving the flag recomputes per location rather than being
	// baked in once.
	s.execute("travel park")
	nodes = nodesByID(s.app.GraphSnapshot())
	assert.False(t, nodes["park"].Reachable, "the new current location is not reachable")
	for _, id := range []string{start, "station", "city-centre"} {
		assert.Falsef(t, nodes[id].Reachable, "%s has no route back from the park dead end", id)
	}
}

// TestState_JSONKeysMatchFrontend locks the JSON contract the browser relies on.
// The snapshot structs carry no json tags, so Go emits the exported field names
// verbatim; app.js reads exactly these keys, so if a tag is ever added this test
// fails before the UI silently breaks. Decoding into maps asserts the keys, not
// just that the typed structs happen to round-trip.
func TestState_JSONKeysMatchFrontend(t *testing.T) {
	s := newTestServer(t)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/state")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload struct {
		Session map[string]any `json:"session"`
		Graph   struct {
			Nodes []map[string]any `json:"Nodes"`
			Edges []map[string]any `json:"Edges"`
		} `json:"graph"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))

	for _, key := range []string{"Location", "Quantum", "ShortOntoAddress"} {
		assert.Containsf(t, payload.Session, key, "session JSON must expose %q for the HUD", key)
	}
	require.NotEmpty(t, payload.Graph.Nodes, "graph must expose nodes")
	for _, key := range []string{"ID", "Name", "Mathematics", "Universe", "Timeline", "Quantum", "Simulation", "Consensus", "Observer", "Location", "Depth", "Reachable"} {
		assert.Containsf(t, payload.Graph.Nodes[0], key, "node JSON must expose %q for colouring and deterministic layout", key)
	}
	require.NotEmpty(t, payload.Graph.Edges, "graph must expose edges")
	for _, key := range []string{"From", "To", "Mode"} {
		assert.Containsf(t, payload.Graph.Edges[0], key, "edge JSON must expose %q for mode styling", key)
	}
}

// TestState_GameFieldsSerialized covers the game HUD contract: with a budget and
// target in force, /api/state must expose every game field app.js reads
// (renderGame), and those fields must track win progression — the budget draws
// down as moves are spent, the objective flips to reached at the target, and to
// won once home again. A non-game server omits the game state, so only a
// game-enabled server proves the fields.
func TestState_GameFieldsSerialized(t *testing.T) {
	s := newTestServer(t,
		facade.WithBudget(facade.DefaultBudget),
		facade.WithTarget(facade.DefaultTarget(universe.DefaultCoordinateVO())),
	)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// Decode the raw JSON into a map so the assertions check the wire keys the
	// browser reads, not just that the typed struct round-trips.
	resp, err := http.Get(srv.URL + "/api/state")
	require.NoError(t, err)
	var keyed struct {
		Session map[string]any `json:"session"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&keyed))
	_ = resp.Body.Close()
	for _, key := range []string{"HasBudget", "Budget", "RemainingBudget", "HasTarget", "TargetAddress", "TargetShortAddress", "ReachedTarget", "Won"} {
		assert.Containsf(t, keyed.Session, key, "session JSON must expose %q for the game HUD", key)
	}

	var initial stateDTO
	getJSON(t, srv.URL+"/api/state", &initial)
	assert.True(t, initial.Session.HasBudget)
	assert.Equal(t, facade.DefaultBudget, initial.Session.Budget)
	assert.Equal(t, facade.DefaultBudget, initial.Session.RemainingBudget)
	assert.True(t, initial.Session.HasTarget)
	assert.False(t, initial.Session.ReachedTarget)
	assert.False(t, initial.Session.Won)

	// Two quantum shifts reach the Q2 objective: reached but not yet won, and the
	// budget has drawn down below its start.
	postJSON(t, srv.URL+"/api/execute", `{"command":"shift"}`, &stateDTO{})
	var reached stateDTO
	postJSON(t, srv.URL+"/api/execute", `{"command":"shift"}`, &reached)
	assert.True(t, reached.Session.ReachedTarget, "arriving at the target marks it reached")
	assert.False(t, reached.Session.Won, "reaching the target is not a win until home again")
	assert.Less(t, reached.Session.RemainingBudget, facade.DefaultBudget, "shifts spend from the budget")

	// Shifting back to home after reaching wins.
	postJSON(t, srv.URL+"/api/execute", `{"command":"shift back"}`, &stateDTO{})
	var won stateDTO
	postJSON(t, srv.URL+"/api/execute", `{"command":"shift back"}`, &won)
	assert.True(t, won.Session.Won, "reached target and returned home wins")
	assert.Equal(t, "home", won.Session.Location)
}

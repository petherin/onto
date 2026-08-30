package commands_test

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/require"
)

const (
	modelMaxDepth  = 5
	modelObserver  = "Cat"
	modelTimeStr   = "2020-01-02T03:04:05Z"
	modelMaxStates = 400000
)

// journeyMove is one state-changing action a user can take from a location:
// a forward reality shift on some axis, a physical walk to a neighbour, or the
// expansion of a new nearby location (dead-end exploration).
type journeyMove struct {
	kind string
	dest string
}

// modelExplorer performs a depth-bounded, deduplicated walk of every reachable
// world state and asserts the return-home invariants at each one. States are
// deduped on a full-world key (graph + session position + context stack) so
// convergent journeys collapse while genuinely distinct ones (e.g. every dead-
// end expansion) are each visited.
type modelExplorer struct {
	t           *testing.T
	pf          navigation.PathfinderService
	targetTime  time.Time
	defObserver string
	maxDepth    int
	visited     map[uint64]bool
	states      int
}

// TestReturnHomeModel exhaustively verifies, for every journey of up to
// modelMaxDepth moves, that auto-generated locations stay well-formed and that
// ReturnHome always plans and completes a clean trip back to home. This is the
// state-space backstop for the whole return-home / branching bug class: rather
// than enumerating the millions of ordered paths, it walks the finite set of
// distinct reachable states and checks the invariants once per state.
//
// Because dead-end expansion runs through the real ClusterLocationGenerator (see
// doExplore), the walk also spawns and explores every trap archetype in the
// nested realities it visits, so the no-hard-lock guarantee — you can always get
// home after landing in a trap and navigating around some more — is proven across
// the whole reachable trap state space, not just in isolated per-archetype unit
// tests.
func TestReturnHomeModel(t *testing.T) {
	targetTime, err := time.Parse(time.RFC3339, modelTimeStr)
	require.NoError(t, err)

	// The full depth-5 walk visits ~90k states in ~10s; -short trims the depth
	// so the suite stays fast while still covering the multi-axis interactions.
	maxDepth := modelMaxDepth
	if testing.Short() {
		maxDepth = 3
	}

	e := &modelExplorer{
		t:           t,
		pf:          navigation.NewBFSPathfinder(),
		targetTime:  targetTime.UTC(),
		defObserver: universe.DefaultCoordinateVO().Observer,
		maxDepth:    maxDepth,
		visited:     map[uint64]bool{},
	}

	u := mocks.NewTestUniverse()
	home, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", home.Coordinate)
	e.visited[e.stateKey(u, sess)] = true
	e.dfs(u, sess, 0, nil)

	t.Logf("model explored %d distinct states up to depth %d", e.states, maxDepth)
	require.Greater(t, e.states, 0)
}

// dfs checks the invariants at the current state, then branches into every
// applicable move, cloning the world per branch so siblings stay independent.
// path records the sequence of moves that reached this state, for diagnostics.
func (e *modelExplorer) dfs(u *universe.Aggregate, sess *exploration.Entity, depth int, path []string) {
	e.states++
	require.LessOrEqual(e.t, e.states, modelMaxStates, "state space exceeded cap — lower modelMaxDepth")
	e.checkInvariants(u, sess, path)
	if depth >= e.maxDepth {
		return
	}
	for _, m := range e.candidateMoves(u, sess) {
		cu := cloneUniverse(e.t, u)
		cs := sess.Clone()
		require.NoError(e.t, e.applyMove(cu, cs, m), "applying %s at %s", m.kind, sess.Location())
		key := e.stateKey(cu, cs)
		if e.visited[key] {
			continue
		}
		e.visited[key] = true
		label := m.kind
		if m.kind == "travel" {
			label = "travel:" + m.dest
		}
		e.dfs(cu, cs, depth+1, append(append([]string(nil), path...), label))
	}
}

// candidateMoves lists the state-changing moves available at the current
// position: the six numeric forward axes, observer/time forward (only while not
// already at the model's token), one dead-end expansion, and a walk to each
// physical neighbour. Reverse moves are omitted — they only undo a forward move
// and so reach no state that dedup would not already collapse.
func (e *modelExplorer) candidateMoves(u *universe.Aggregate, sess *exploration.Entity) []journeyMove {
	moves := []journeyMove{
		{kind: "shift"}, {kind: "jump"}, {kind: "universe"},
		{kind: "structure"}, {kind: "simulate"}, {kind: "drift"},
		{kind: "explore"},
	}
	if sess.Coordinate().Observer != modelObserver {
		moves = append(moves, journeyMove{kind: "observe"})
	}
	if !sess.Coordinate().Time.Equal(e.targetTime) {
		moves = append(moves, journeyMove{kind: "time"})
	}
	for _, edge := range u.EdgesFrom(sess.Location()) {
		if edge.Mode.IsPhysical() {
			moves = append(moves, journeyMove{kind: "travel", dest: edge.To})
		}
	}
	return moves
}

// applyMove executes the command that realises a journeyMove against the given
// world, mirroring exactly what the facade does for the user.
func (e *modelExplorer) applyMove(u *universe.Aggregate, sess *exploration.Entity, m journeyMove) error {
	switch m.kind {
	case "shift":
		_, err := (&commands.ShiftCommand{Universe: u, Session: sess}).Execute()
		return err
	case "jump":
		_, err := (&commands.JumpCommand{Universe: u, Session: sess}).Execute()
		return err
	case "universe":
		_, err := (&commands.UniverseCommand{Universe: u, Session: sess}).Execute()
		return err
	case "structure":
		_, err := (&commands.StructureCommand{Universe: u, Session: sess}).Execute()
		return err
	case "simulate":
		_, err := (&commands.SimulateCommand{Universe: u, Session: sess}).Execute()
		return err
	case "drift":
		_, err := (&commands.DriftCommand{Universe: u, Session: sess}).Execute()
		return err
	case "observe":
		_, err := (&commands.ObserveCommand{Universe: u, Session: sess, Observer: modelObserver}).Execute()
		return err
	case "time":
		_, err := (&commands.TimeCommand{Universe: u, Session: sess, Target: modelTimeStr}).Execute()
		return err
	case "travel":
		_, err := (&commands.TravelCommand{Universe: u, Session: sess, Pathfinder: e.pf}).Execute(m.dest)
		return err
	case "explore":
		return e.doExplore(u, sess)
	}
	return fmt.Errorf("unknown move kind %q", m.kind)
}

// doExplore expands the current dead end through the real production policy —
// ClusterLocationGenerator.Generate, which rolls SelectTrap and so may spawn a
// trap (sealed vault, tar pit, möbius maze, one-way sink) rather than an ordinary
// cluster in a nested reality — commits every node with its edges, and travels to
// the entry node. Routing through Generate (not NewNearbyCluster directly) is
// what pulls traps into the exhaustive walk: the DFS then explores onward from
// the trap and re-checks the return-home invariants at every state reachable
// through and beyond it, proving you can always get home after landing in a trap
// and navigating around some more.
func (e *modelExplorer) doExplore(u *universe.Aggregate, sess *exploration.Entity) error {
	origin, ok := u.GetLocation(sess.Location())
	if !ok {
		return fmt.Errorf("current location %q missing", sess.Location())
	}
	locs, edges, err := universe.NewClusterLocationGenerator().Generate(u, sess.Location(), origin.Coordinate)
	if err != nil {
		return err
	}
	for _, loc := range locs {
		if err := u.AddLocation(loc); err != nil {
			return err
		}
	}
	for _, edge := range edges {
		if err := u.AddEdge(edge); err != nil {
			return err
		}
	}
	// Every archetype's entry node (locs[0]: the cluster node, vault, tar-pit
	// node, maze hub, or trapdoor mouth) is reachable by the inbound walk edge
	// from the origin, so travelling to it always succeeds even when the node
	// itself is a physical sink (a sealed vault has no outgoing physical edge).
	_, err = (&commands.TravelCommand{Universe: u, Session: sess, Pathfinder: e.pf}).Execute(locs[0].ID)
	return err
}

// stateKey folds the whole world — sorted locations, sorted physical/contextual
// edges, and the session's position plus ordered context stack — into a 64-bit
// hash. Travel history and cumulative cost are deliberately excluded so a walk
// out and back collapses to the same state instead of spawning new ones.
func (e *modelExplorer) stateKey(u *universe.Aggregate, sess *exploration.Entity) uint64 {
	ids := u.AllLocationIDs()
	sort.Strings(ids)
	edges := make([]string, 0)
	for _, edge := range u.AllEdgesFlat() {
		edges = append(edges, fmt.Sprintf("%s>%s:%s", edge.From, edge.To, edge.Mode))
	}
	sort.Strings(edges)
	var b strings.Builder
	b.WriteString(strings.Join(ids, ","))
	b.WriteString("|")
	b.WriteString(strings.Join(edges, ","))
	b.WriteString("|@")
	b.WriteString(sess.Location())
	b.WriteString("|")
	b.WriteString(sess.Coordinate().OntoAddress())
	b.WriteString("|ctx:")
	for _, tr := range sess.ContextTransitions() {
		fmt.Fprintf(&b, "%s@%s;", tr.Mode, tr.OriginID)
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(b.String()))
	return h.Sum64()
}

// cloneUniverse rebuilds an independent copy of the aggregate via its public
// accessors so a branch of the search can mutate its own graph freely.
func cloneUniverse(t *testing.T, u *universe.Aggregate) *universe.Aggregate {
	t.Helper()
	n := universe.NewAggregate()
	for _, loc := range u.AllLocations() {
		require.NoError(t, n.AddLocation(loc))
	}
	for _, edge := range u.AllEdgesFlat() {
		require.NoError(t, n.AddEdge(edge))
	}
	return n
}

// checkInvariants asserts, at a single reachable state, everything that broke
// in this bug class: canonical IDs, bidirectional physical edges, well-formed
// auto-generated nearby nodes, and a return-home that both plans without stalls
// and executes cleanly back to home with an empty context stack. Trap nodes are
// deliberately excused from the ordinary-cluster structural rules (one-way sinks
// and maze decoys have directional edges; a sealed vault has no physical exit at
// all) — for them the no-hard-lock guarantee is enforced by the primary
// return-home invariant below and by checkTrapInvariants.
func (e *modelExplorer) checkInvariants(u *universe.Aggregate, sess *exploration.Entity, path []string) {
	t := e.t
	journey := strings.Join(path, " > ")
	for _, loc := range u.AllLocations() {
		require.Falsef(t, universe.LocationIDIsMalformed(loc.ID), "non-canonical id %q [journey: %s]", loc.ID, journey)
	}
	for _, edge := range u.AllEdgesFlat() {
		// Trap archetypes intentionally use one-way physical edges (the one-way
		// sink's trapdoor, the möbius decoys' silent loop-backs), so the
		// bidirectionality rule applies only to edges between non-trap nodes.
		if edge.Mode.IsPhysical() && !isTrapNode(u, edge.From) && !isTrapNode(u, edge.To) {
			require.Truef(t, hasPhysicalReverse(u, edge), "physical edge %s->%s has no reverse [journey: %s]", edge.From, edge.To, journey)
		}
	}
	for _, loc := range u.AllLocations() {
		if !loc.Generated {
			continue
		}
		if loc.Trap != universe.NoTrap {
			e.checkTrapInvariants(u, loc, journey)
			continue
		}
		var originID string
		for _, edge := range u.EdgesFrom(loc.ID) {
			if edge.Mode.IsPhysical() {
				originID = edge.To
				break
			}
		}
		require.NotEmptyf(t, originID, "nearby %q has no physical edge to an origin [journey: %s]", loc.ID, journey)
		origin, ok := u.GetLocation(originID)
		require.True(t, ok)
		require.Truef(t, loc.Coordinate.SamePhysicalReality(origin.Coordinate),
			"nearby %q is not in its origin's physical reality [journey: %s]", loc.ID, journey)
	}

	// Primary invariant: ReturnHome must execute cleanly back to home from every
	// reachable state. Run on its own clone so its mutations do not affect the
	// plan check below.
	xu := cloneUniverse(t, u)
	xs := sess.Clone()
	_, err := (&commands.ReturnHomeCommand{
		Universe: xu, Session: xs, Pathfinder: e.pf, HomeID: "home",
	}).Execute()
	require.NoErrorf(t, err, "return home from %s (loc %s) [journey: %s]", sess.Coordinate().ShortOntoAddress(), sess.Location(), journey)
	require.Equalf(t, "home", xs.Location(), "return-home did not land at home from %s [journey: %s]", sess.Location(), journey)
	require.Equal(t, 0, xs.UniverseLevel())
	require.Equal(t, 0, xs.QuantumLevel())
	require.Equal(t, 0, xs.TimelineLevel())
	require.Equal(t, 0, xs.MathematicsLevel())
	require.Equal(t, 0, xs.ConsensusLevel())
	require.Equal(t, 0, xs.SimulationLevel())
	require.Equal(t, e.defObserver, xs.Coordinate().Observer)
	require.True(t, xs.Coordinate().Time.IsZero())
	require.Empty(t, xs.ContextTransitions())

	// Secondary invariant: the plan preview shown to the user must agree — no
	// "unavailable" legs, no N → N stalls, and non-negative cost.
	pu := cloneUniverse(t, u)
	plan, cost := (&commands.ReturnHomeCommand{
		Universe: pu, Session: sess.Clone(), Pathfinder: e.pf, HomeID: "home",
	}).Plan()
	require.GreaterOrEqualf(t, cost, 0.0, "negative plan cost at %s [journey: %s]", sess.Location(), journey)
	for _, step := range plan {
		require.NotContainsf(t, step.Detail, "unavailable",
			"return-home plan step %q unavailable at %s (%s) [journey: %s]", step.Action, sess.Location(), sess.Coordinate().ShortOntoAddress(), journey)
		if sides := strings.SplitN(step.Detail, " → ", 2); len(sides) == 2 {
			require.NotEqualf(t, sides[0], sides[1], "plan step %q stalls: %q [journey: %s]", step.Action, step.Detail, journey)
		}
	}
}

// hasPhysicalReverse reports whether a physical edge has a matching physical
// edge in the opposite direction — the bidirectionality every walkable link
// (seeded, branch-mirrored, or nearby-generated) must maintain.
func hasPhysicalReverse(u *universe.Aggregate, edge universe.EdgeVO) bool {
	for _, back := range u.EdgesFrom(edge.To) {
		if back.To == edge.From && back.Mode.IsPhysical() {
			return true
		}
	}
	return false
}

// isTrapNode reports whether id is a generated trap node. Trap archetypes are
// deliberately exempt from the ordinary-cluster structural rules, so the model's
// bidirectionality and star-shape checks skip them.
func isTrapNode(u *universe.Aggregate, id string) bool {
	loc, ok := u.GetLocation(id)
	return ok && loc.Trap != universe.NoTrap
}

// checkTrapInvariants asserts a trap node is well-formed on its own terms: it
// carries a known archetype, and — the no-hard-lock guarantee at node level —
// FindRoute can always plot a way out toward home across some edge (a physical
// walk for tar pits and mazes, or the non-physical ConsensusShift escape edge for
// sealed vaults and one-way sinks). The whole-journey return-home invariant below
// then proves the traveller actually gets all the way back.
func (e *modelExplorer) checkTrapInvariants(u *universe.Aggregate, loc universe.LocationEntity, journey string) {
	t := e.t
	require.Truef(t, loc.Trap.IsKnown(), "trap %q carries an unknown archetype %q [journey: %s]", loc.ID, loc.Trap, journey)
	_, ok := e.pf.FindRoute(u, loc.ID, "home")
	require.Truef(t, ok, "trap %q (%s) has no route out toward home [journey: %s]", loc.ID, loc.Trap.Label(), journey)
}

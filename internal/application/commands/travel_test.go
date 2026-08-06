package commands_test

import (
	"testing"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTravelFixture(t *testing.T) (*universe.Aggregate, *exploration.Entity, *mocks.MockRepository, *mocks.MockPathfinderService) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	repo := mocks.NewMockRepository(t)
	pf := mocks.NewMockPathfinderService(t)
	return u, sess, repo, pf
}

func TestTravelCommand_Success(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	route := []universe.EdgeVO{{From: "home", To: "station", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "station").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	result, err := cmd.Execute("station")

	require.NoError(t, err)
	assert.Equal(t, "station", result.Location.ID)
	assert.Contains(t, result.History, "home -> station")
	assert.True(t, result.DeadEndHandled)
}

func TestTravelCommand_UnknownDestination(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	_, err := cmd.Execute("nowhere")

	require.ErrorIs(t, err, navigation.ErrUnknownDestination)
}

func TestTravelCommand_NoRoute(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	// island is reachable by name but has no graph path from home
	u.AddLocation(universe.LocationEntity{ID: "island", Name: "Island", Coordinate: universe.DefaultCoordinateVO()})
	pf.EXPECT().FindRoute(u, "home", "island").Return(nil, false)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	_, err := cmd.Execute("island")

	require.ErrorIs(t, err, navigation.ErrNoRoute)
}

func TestTravelCommand_QuantumEdge_Rejected(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	// home-q1 is connected only via a quantum edge — travel must not allow it
	q1Coord := universe.DefaultCoordinateVO()
	q1Coord.Quantum = "Q1"
	u.AddLocation(universe.LocationEntity{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "home-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost})

	quantumRoute := []universe.EdgeVO{{From: "home", To: "home-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost}}
	pf.EXPECT().FindRoute(u, "home", "home-q1").Return(quantumRoute, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	_, err := cmd.Execute("home-q1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no physical route")
}

func TestTravelCommand_ConsensusBoundaryRejected(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	divergent := universe.DefaultCoordinateVO()
	divergent.Consensus = 1
	u.AddLocation(universe.LocationEntity{ID: "divergent-station", Name: "Station", Coordinate: divergent})
	route := []universe.EdgeVO{{From: "home", To: "divergent-station", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "divergent-station").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	_, err := cmd.Execute("divergent-station")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "normal travel cannot cross reality boundaries")
	assert.Equal(t, "home", sess.Location())
}

func TestTravelCommand_DeadEnd_IsReportedWithoutMutation(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	// deadend has only a return edge back to home (no onward edges)
	deadCoord := universe.DefaultCoordinateVO()
	u.AddLocation(universe.LocationEntity{ID: "deadend", Name: "Dead End", Coordinate: deadCoord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1})
	u.AddEdge(universe.EdgeVO{From: "deadend", To: "home", Mode: universe.Walk, Cost: 1})

	route := []universe.EdgeVO{{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "deadend").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	result, err := cmd.Execute("deadend")

	require.NoError(t, err)
	assert.True(t, result.DeadEndHandled)
}

func TestTravelCommand_DeadEndWithContextualEdges_IsReported(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	deadCoord := universe.DefaultCoordinateVO()
	u.AddLocation(universe.LocationEntity{ID: "deadend", Name: "Dead End", Coordinate: deadCoord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1})
	u.AddEdge(universe.EdgeVO{From: "deadend", To: "home", Mode: universe.Walk, Cost: 1})
	u.AddEdge(universe.EdgeVO{From: "deadend", To: "deadend-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost})

	route := []universe.EdgeVO{{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "deadend").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	result, err := cmd.Execute("deadend")

	require.NoError(t, err)
	assert.True(t, result.DeadEndHandled)
}

func TestTravelCommand_NonDeadEnd_RepoNotCalled(t *testing.T) {
	u, sess, repo, pf := newTravelFixture(t)

	// station has home→station and station→home already; add station→park so
	// ensureOutgoing finds a non-home outgoing edge and skips the handler.
	u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: universe.DefaultCoordinateVO()})
	u.AddEdge(universe.EdgeVO{From: "station", To: "park", Mode: universe.Walk, Cost: 1})

	route := []universe.EdgeVO{{From: "home", To: "station", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "station").Return(route, true)

	// No gen or repo expectations set — any call to Handle/Save would fail the test
	cmd := &commands.TravelCommand{Universe: u, Session: sess, Repo: repo, Pathfinder: pf}
	result, err := cmd.Execute("station")

	require.NoError(t, err)
	assert.False(t, result.DeadEndHandled)
}

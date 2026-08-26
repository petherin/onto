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

func newTravelFixture(t *testing.T) (*universe.Aggregate, *exploration.Entity, *mocks.MockPathfinderService) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	pf := mocks.NewMockPathfinderService(t)
	return u, sess, pf
}

func TestTravelCommand_Success(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	route := []universe.EdgeVO{{From: "home", To: "station", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "station").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	result, err := cmd.Execute("station")

	require.NoError(t, err)
	assert.Equal(t, "station", result.Location.ID)
	assert.Contains(t, result.History, "home -> station")
	assert.True(t, result.DeadEndHandled)
}

func TestTravelCommand_UnknownDestination(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	_, err := cmd.Execute("nowhere")

	require.ErrorIs(t, err, navigation.ErrUnknownDestination)
}

func TestTravelCommand_NoRoute(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	// island is reachable by name but has no graph path from home
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "island", Name: "Island", Coordinate: universe.DefaultCoordinateVO()}))
	pf.EXPECT().FindRoute(u, "home", "island").Return(nil, false)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	_, err := cmd.Execute("island")

	require.ErrorIs(t, err, navigation.ErrNoRoute)
}

func TestTravelCommand_ResolvesPlainNameWithinCurrentReality(t *testing.T) {
	u, _, pf := newTravelFixture(t)

	// Build a Q1 copy of the world and start the session there.
	q1 := universe.DefaultCoordinateVO()
	q1.Quantum = "Q1"
	homeQ1 := q1
	homeQ1.Location = "Home"
	stationQ1 := q1
	stationQ1.Location = "Station"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-q1", Name: "Home (Q1)", Coordinate: homeQ1}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station-q1", Name: "Station (Q1)", Coordinate: stationQ1}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-q1", To: "station-q1", Mode: universe.Walk, Cost: 1}))
	sess := exploration.NewEntity("home-q1", homeQ1)

	// Typing the plain name "station" must resolve to the in-reality copy
	// (station-q1), not the base-reality "station".
	route := []universe.EdgeVO{{From: "home-q1", To: "station-q1", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home-q1", "station-q1").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	result, err := cmd.Execute("station")

	require.NoError(t, err)
	assert.Equal(t, "station-q1", result.Location.ID)
}

func TestTravelCommand_QuantumEdge_Rejected(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	// home-q1 is connected only via a quantum edge — travel must not allow it
	q1Coord := universe.DefaultCoordinateVO()
	q1Coord.Quantum = "Q1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "home-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost}))

	quantumRoute := []universe.EdgeVO{{From: "home", To: "home-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost}}
	pf.EXPECT().FindRoute(u, "home", "home-q1").Return(quantumRoute, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	_, err := cmd.Execute("home-q1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no physical route")
}

func TestTravelCommand_ConsensusBoundaryRejected(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	divergent := universe.DefaultCoordinateVO()
	divergent.Consensus = 1
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "divergent-station", Name: "Station", Coordinate: divergent}))
	route := []universe.EdgeVO{{From: "home", To: "divergent-station", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "divergent-station").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	_, err := cmd.Execute("divergent-station")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "normal travel cannot cross reality boundaries")
	assert.Equal(t, "home", sess.Location())
}

func TestTravelCommand_DeadEnd_IsReportedWithoutMutation(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	// deadend has only a return edge back to home (no onward edges)
	deadCoord := universe.DefaultCoordinateVO()
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "deadend", Name: "Dead End", Coordinate: deadCoord}))
	q1Coord := deadCoord
	q1Coord.Quantum = "Q1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "deadend-q1", Name: "Dead End (Q1)", Coordinate: q1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "deadend", To: "home", Mode: universe.Walk, Cost: 1}))

	route := []universe.EdgeVO{{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "deadend").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	result, err := cmd.Execute("deadend")

	require.NoError(t, err)
	assert.True(t, result.DeadEndHandled)
}

func TestTravelCommand_DeadEndWithContextualEdges_IsReported(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	deadCoord := universe.DefaultCoordinateVO()
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "deadend", Name: "Dead End", Coordinate: deadCoord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "deadend", To: "home", Mode: universe.Walk, Cost: 1}))
	branchCoord := deadCoord
	branchCoord.Quantum = "Q1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "deadend-q1", Coordinate: branchCoord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "deadend", To: "deadend-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost}))

	route := []universe.EdgeVO{{From: "home", To: "deadend", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "deadend").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	result, err := cmd.Execute("deadend")

	require.NoError(t, err)
	assert.True(t, result.DeadEndHandled)
}

func TestTravelCommand_NonDeadEnd_DoesNotReportDeadEnd(t *testing.T) {
	u, sess, pf := newTravelFixture(t)

	// station has home→station and station→home already; add station→park so
	// ensureOutgoing finds a non-home outgoing edge and therefore this is not
	// treated as a dead end.
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: universe.DefaultCoordinateVO()}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "station", To: "park", Mode: universe.Walk, Cost: 1}))

	route := []universe.EdgeVO{{From: "home", To: "station", Mode: universe.Walk, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "station").Return(route, true)

	cmd := &commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}
	result, err := cmd.Execute("station")

	require.NoError(t, err)
	assert.False(t, result.DeadEndHandled)
}

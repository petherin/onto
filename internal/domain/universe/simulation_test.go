package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchSimulation_CreatesLocationWithCorrectDepth(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchSimulation(u, "home", coord, "Home", "home-s1", 1))

	loc, ok := u.GetLocation("home-s1")
	require.True(t, ok)
	assert.Equal(t, 1, loc.Coordinate.Simulation)
	assert.Equal(t, "Home (sim:1)", loc.Name)
}

func TestBranchSimulation_PreservesOtherAxes(t *testing.T) {
	u, coord := newBaseUniverse(t)
	coord.Quantum = "Q2"
	coord.Timeline = "T1"

	require.NoError(t, universe.BranchSimulation(u, "home", coord, "Home", "home-s1", 1))

	loc, _ := u.GetLocation("home-s1")
	assert.Equal(t, "Q2", loc.Coordinate.Quantum)
	assert.Equal(t, "T1", loc.Coordinate.Timeline)
	assert.Equal(t, coord.Universe, loc.Coordinate.Universe)
}

func TestBranchSimulation_AddsBidirectionalSimulationEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchSimulation(u, "home", coord, "Home", "home-s1", 1))

	var forward, reverse universe.EdgeVO
	for _, e := range u.EdgesFrom("home") {
		if e.To == "home-s1" && e.Mode == universe.SimulationEntry {
			forward = e
		}
	}
	for _, e := range u.EdgesFrom("home-s1") {
		if e.To == "home" && e.Mode == universe.SimulationEntry {
			reverse = e
		}
	}
	assert.Equal(t, universe.SimulationEntryCost, forward.Cost)
	assert.Equal(t, universe.SimulationExitCost, reverse.Cost)
}

func TestBranchSimulation_MirrorsPhysicalEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.5, Cost: 1}))

	require.NoError(t, universe.BranchSimulation(u, "home", coord, "Home", "home-s1", 1))

	assert.True(t, edgeTo(u.EdgesFrom("home-s1"), "station-s1", universe.Walk))
	station, ok := u.GetLocation("station-s1")
	require.True(t, ok)
	assert.Equal(t, 1, station.Coordinate.Simulation)
}

func TestBranchSimulation_Idempotent(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchSimulation(u, "home", coord, "Home", "home-s1", 1))
	require.NoError(t, universe.BranchSimulation(u, "home", coord, "Home", "home-s1", 1))

	assert.Len(t, u.AllLocations(), 2)
}

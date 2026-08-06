package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchConsensusService_CreatesLocationWithCorrectLevel(t *testing.T) {
	u, coord := newBaseUniverse()

	universe.BranchConsensusService(u, "home", coord, "Home", "home-c1", 1)

	loc, ok := u.GetLocation("home-c1")
	require.True(t, ok)
	assert.Equal(t, 1, loc.Coordinate.Consensus)
	assert.Equal(t, "Home (consensus 1)", loc.Name)
}

func TestBranchConsensusService_AddsBidirectionalConsensusEdges(t *testing.T) {
	u, coord := newBaseUniverse()

	universe.BranchConsensusService(u, "home", coord, "Home", "home-c1", 1)

	assert.True(t, edgeTo(u.EdgesFrom("home"), "home-c1", universe.ConsensusShift))
	assert.True(t, edgeTo(u.EdgesFrom("home-c1"), "home", universe.ConsensusShift))
}

func TestBranchConsensusService_MirrorsOnlyPhysicalEdges(t *testing.T) {
	u, coord := newBaseUniverse()
	u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.5, Cost: 1})
	u.AddEdge(universe.EdgeVO{From: "home", To: "other", Mode: universe.ConsensusShift})

	universe.BranchConsensusService(u, "home", coord, "Home", "home-c1", 1)

	assert.True(t, edgeTo(u.EdgesFrom("home-c1"), "station-c1", universe.Walk))
	assert.False(t, edgeTo(u.EdgesFrom("home-c1"), "other", universe.ConsensusShift))

	station, ok := u.GetLocation("station-c1")
	require.True(t, ok)
	assert.Equal(t, 1, station.Coordinate.Consensus)
}

func TestBranchConsensusService_IsIdempotent(t *testing.T) {
	u, coord := newBaseUniverse()

	universe.BranchConsensusService(u, "home", coord, "Home", "home-c1", 1)
	universe.BranchConsensusService(u, "home", coord, "Home", "home-c1", 1)

	assert.Len(t, u.AllLocations(), 2)
}

func TestBranchConsensusService_AddsAlignmentAtCopiedLocations(t *testing.T) {
	u, coord := newBaseUniverse()
	u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk})

	universe.BranchConsensusService(u, "home", coord, "Home", "home-c1", 1)

	assert.True(t, edgeTo(u.EdgesFrom("station-c1"), "station", universe.ConsensusShift))
}

func TestBranchConsensusService_CopiedLocationsHavePhysicalAndContextualReturns(t *testing.T) {
	u, coord := newBaseUniverse()
	u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord})
	u.AddLocation(universe.LocationEntity{ID: "park", Coordinate: coord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Cost: 1})
	u.AddEdge(universe.EdgeVO{From: "home", To: "park", Mode: universe.Walk, Cost: 1})

	universe.BranchConsensusService(u, "home", coord, "Home", "home-c1", 1)

	branchHome, _ := u.GetLocation("home-c1")
	for _, id := range []string{"station-c1", "park-c1"} {
		loc, ok := u.GetLocation(id)
		require.True(t, ok)
		assert.True(t, branchHome.Coordinate.SamePhysicalReality(loc.Coordinate))
		assert.True(t, edgeTo(u.EdgesFrom(id), "home-c1", universe.Walk))
		baseID := id[:len(id)-len("-c1")]
		assert.True(t, edgeTo(u.EdgesFrom(id), baseID, universe.ConsensusShift))
	}
}

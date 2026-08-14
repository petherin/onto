package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchUniverse_CreatesLocationWithCorrectUniverse(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))

	loc, ok := u.GetLocation("home-u1")
	require.True(t, ok)
	assert.Equal(t, "U1", loc.Coordinate.Universe)
	assert.Equal(t, "Home (U1)", loc.Name)
}

func TestBranchUniverse_PreservesQuantumBranch(t *testing.T) {
	u, coord := newBaseUniverse(t)
	coord.Quantum = "Q2" // pretend we branched from a non-base quantum state

	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))

	loc, _ := u.GetLocation("home-u1")
	assert.Equal(t, "Q2", loc.Coordinate.Quantum,
		"entering a new universe preserves the current quantum branch")
}

func TestBranchUniverse_AddsBidirectionalUniverseEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))

	assert.True(t, edgeTo(u.EdgesFrom("home"), "home-u1", universe.UniverseShift),
		"source should have forward universe edge")
	assert.True(t, edgeTo(u.EdgesFrom("home-u1"), "home", universe.UniverseShift),
		"branch should have reverse universe edge")
}

func TestBranchUniverse_MirrorsPhysicalEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.5, Cost: 1}))

	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))

	assert.True(t, edgeTo(u.EdgesFrom("home-u1"), "station-u1", universe.Walk),
		"universe branch should inherit physical walk to station")

	station, ok := u.GetLocation("station-u1")
	require.True(t, ok)
	assert.Equal(t, "U1", station.Coordinate.Universe)
}

func TestBranchUniverse_DoesNotMirrorUniverseEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "other-universe", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "other-universe", Mode: universe.UniverseShift}))

	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))

	for _, e := range u.EdgesFrom("home-u1") {
		assert.NotEqual(t, "other-universe", e.To, "non-physical edges must not be mirrored")
	}
}

func TestBranchUniverse_Idempotent(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))
	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))

	assert.Len(t, u.AllLocations(), 2)
}

func TestBranchUniverse_PreservesOtherCoordinateFields(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchUniverse(u, "home", coord, "Home", "home-u1", "U1"))

	loc, _ := u.GetLocation("home-u1")
	assert.Equal(t, coord.Planet, loc.Coordinate.Planet)
	assert.Equal(t, coord.City, loc.Coordinate.City)
	assert.Equal(t, coord.Timeline, loc.Coordinate.Timeline)
}

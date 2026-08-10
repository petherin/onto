package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBaseUniverse(t *testing.T) (*universe.Aggregate, universe.CoordinateVO) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home", Coordinate: coord}))
	return u, coord
}

func TestBranchQuantum_CreatesLocationWithCorrectQuantum(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchQuantum(u, "home", coord, "Home", "home-q1", "Q1"))

	loc, ok := u.GetLocation("home-q1")
	require.True(t, ok)
	assert.Equal(t, "Q1", loc.Coordinate.Quantum)
	assert.Equal(t, "Home (Q1)", loc.Name)
}

func TestBranchQuantum_AddsBidirectionalQuantumEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchQuantum(u, "home", coord, "Home", "home-q1", "Q1"))

	assert.True(t, edgeTo(u.EdgesFrom("home"), "home-q1", universe.QuantumShift),
		"source should have forward quantum edge")
	assert.True(t, edgeTo(u.EdgesFrom("home-q1"), "home", universe.QuantumShift),
		"branch should have reverse quantum edge")
}

func TestBranchQuantum_MirrorsPhysicalEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.5, Cost: 1}))

	require.NoError(t, universe.BranchQuantum(u, "home", coord, "Home", "home-q1", "Q1"))

	assert.True(t, edgeTo(u.EdgesFrom("home-q1"), "station-q1", universe.Walk),
		"quantum branch should be able to walk to station")

	station, ok := u.GetLocation("station-q1")
	require.True(t, ok)
	assert.Equal(t, "Q1", station.Coordinate.Quantum)
}

func TestBranchQuantum_DoesNotMirrorQuantumEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "other-branch", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "other-branch", Mode: universe.QuantumShift}))

	require.NoError(t, universe.BranchQuantum(u, "home", coord, "Home", "home-q1", "Q1"))

	for _, e := range u.EdgesFrom("home-q1") {
		assert.NotEqual(t, "other-branch", e.To, "non-physical edges must not be mirrored")
	}
}

func TestBranchQuantum_Idempotent(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchQuantum(u, "home", coord, "Home", "home-q1", "Q1"))
	require.NoError(t, universe.BranchQuantum(u, "home", coord, "Home", "home-q1", "Q1"))

	// Second call is a no-op — only home and home-q1 should exist.
	assert.Len(t, u.AllLocations(), 2)
}

func TestBranchQuantum_PreservesOtherCoordinateFields(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchQuantum(u, "home", coord, "Home", "home-q1", "Q1"))

	loc, _ := u.GetLocation("home-q1")
	assert.Equal(t, coord.Planet, loc.Coordinate.Planet)
	assert.Equal(t, coord.City, loc.Coordinate.City)
	assert.Equal(t, coord.Timeline, loc.Coordinate.Timeline)
}

package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchMathematics_CreatesLocationWithCorrectMathematics(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))

	loc, ok := u.GetLocation("home-m1")
	require.True(t, ok)
	assert.Equal(t, "M1", loc.Coordinate.Mathematics)
	assert.Equal(t, "Home (M1)", loc.Name)
}

func TestBranchMathematics_PreservesUniverseBranch(t *testing.T) {
	u, coord := newBaseUniverse(t)
	coord.Universe = "U2"

	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))

	loc, _ := u.GetLocation("home-m1")
	assert.Equal(t, "U2", loc.Coordinate.Universe,
		"entering a new mathematical structure preserves the current bubble universe")
}

func TestBranchMathematics_AddsBidirectionalMathematicsEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))

	assert.True(t, edgeTo(u.EdgesFrom("home"), "home-m1", universe.MathematicalShift),
		"source should have forward mathematics edge")
	assert.True(t, edgeTo(u.EdgesFrom("home-m1"), "home", universe.MathematicalShift),
		"branch should have reverse mathematics edge")
}

func TestBranchMathematics_MirrorsPhysicalEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.5, Cost: 1}))

	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))

	assert.True(t, edgeTo(u.EdgesFrom("home-m1"), "station-m1", universe.Walk),
		"mathematics branch should inherit physical walk to station")

	station, ok := u.GetLocation("station-m1")
	require.True(t, ok)
	assert.Equal(t, "M1", station.Coordinate.Mathematics)
}

func TestBranchMathematics_DoesNotMirrorMathematicsEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "other-math", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "other-math", Mode: universe.MathematicalShift}))

	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))

	for _, e := range u.EdgesFrom("home-m1") {
		assert.NotEqual(t, "other-math", e.To, "non-physical edges must not be mirrored")
	}
}

func TestBranchMathematics_Idempotent(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))
	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))

	assert.Len(t, u.AllLocations(), 2)
}

func TestBranchMathematics_PreservesOtherCoordinateFields(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchMathematics(u, "home", coord, "Home", "home-m1", "M1"))

	loc, _ := u.GetLocation("home-m1")
	assert.Equal(t, coord.Planet, loc.Coordinate.Planet)
	assert.Equal(t, coord.City, loc.Coordinate.City)
	assert.Equal(t, coord.Timeline, loc.Coordinate.Timeline)
	assert.Equal(t, coord.Universe, loc.Coordinate.Universe)
}

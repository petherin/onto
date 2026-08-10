package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchTimeline_CreatesLocationWithCorrectTimeline(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))

	loc, ok := u.GetLocation("home-t1")
	require.True(t, ok)
	assert.Equal(t, "T1", loc.Coordinate.Timeline)
	assert.Equal(t, "Home (T1)", loc.Name)
}

func TestBranchTimeline_PreservesQuantumBranch(t *testing.T) {
	u, coord := newBaseUniverse(t)
	coord.Quantum = "Q2" // pretend we branched from a non-base quantum state

	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))

	loc, _ := u.GetLocation("home-t1")
	assert.Equal(t, "Q2", loc.Coordinate.Quantum,
		"entering a new timeline preserves the current quantum branch")
}

func TestBranchTimeline_AddsBidirectionalTimelineEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))

	assert.True(t, edgeTo(u.EdgesFrom("home"), "home-t1", universe.TimelineShift),
		"source should have forward timeline edge")
	assert.True(t, edgeTo(u.EdgesFrom("home-t1"), "home", universe.TimelineShift),
		"branch should have reverse timeline edge")
}

func TestBranchTimeline_MirrorsPhysicalEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.5, Cost: 1}))

	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))

	assert.True(t, edgeTo(u.EdgesFrom("home-t1"), "station-t1", universe.Walk),
		"timeline branch should inherit physical walk to station")

	station, ok := u.GetLocation("station-t1")
	require.True(t, ok)
	assert.Equal(t, "T1", station.Coordinate.Timeline)
}

func TestBranchTimeline_DoesNotMirrorTimelineEdges(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "other-tl", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "other-tl", Mode: universe.TimelineShift}))

	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))

	for _, e := range u.EdgesFrom("home-t1") {
		assert.NotEqual(t, "other-tl", e.To, "non-physical edges must not be mirrored")
	}
}

func TestBranchTimeline_Idempotent(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))
	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))

	assert.Len(t, u.AllLocations(), 2)
}

func TestBranchTimeline_PreservesOtherCoordinateFields(t *testing.T) {
	u, coord := newBaseUniverse(t)

	require.NoError(t, universe.BranchTimeline(u, "home", coord, "Home", "home-t1", "T1"))

	loc, _ := u.GetLocation("home-t1")
	assert.Equal(t, coord.Planet, loc.Coordinate.Planet)
	assert.Equal(t, coord.City, loc.Coordinate.City)
}

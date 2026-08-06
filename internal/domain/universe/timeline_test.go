package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchTimelineService_CreatesLocationWithCorrectTimeline(t *testing.T) {
	u, coord := newBaseUniverse()

	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")

	loc, ok := u.GetLocation("home-t1")
	require.True(t, ok)
	assert.Equal(t, "T1", loc.Coordinate.Timeline)
	assert.Equal(t, "Home (T1)", loc.Name)
}

func TestBranchTimelineService_PreservesQuantumBranch(t *testing.T) {
	u, coord := newBaseUniverse()
	coord.Quantum = "Q2" // pretend we branched from a non-base quantum state

	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")

	loc, _ := u.GetLocation("home-t1")
	assert.Equal(t, "Q2", loc.Coordinate.Quantum,
		"entering a new timeline preserves the current quantum branch")
}

func TestBranchTimelineService_AddsBidirectionalTimelineEdges(t *testing.T) {
	u, coord := newBaseUniverse()

	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")

	assert.True(t, edgeTo(u.EdgesFrom("home"), "home-t1", universe.TimelineShift),
		"source should have forward timeline edge")
	assert.True(t, edgeTo(u.EdgesFrom("home-t1"), "home", universe.TimelineShift),
		"branch should have reverse timeline edge")
}

func TestBranchTimelineService_MirrorsPhysicalEdges(t *testing.T) {
	u, coord := newBaseUniverse()
	u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.5, Cost: 1})

	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")

	assert.True(t, edgeTo(u.EdgesFrom("home-t1"), "station-t1", universe.Walk),
		"timeline branch should inherit physical walk to station")

	station, ok := u.GetLocation("station-t1")
	require.True(t, ok)
	assert.Equal(t, "T1", station.Coordinate.Timeline)
}

func TestBranchTimelineService_DoesNotMirrorTimelineEdges(t *testing.T) {
	u, coord := newBaseUniverse()
	u.AddEdge(universe.EdgeVO{From: "home", To: "other-tl", Mode: universe.TimelineShift})

	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")

	for _, e := range u.EdgesFrom("home-t1") {
		assert.NotEqual(t, "other-tl", e.To, "non-physical edges must not be mirrored")
	}
}

func TestBranchTimelineService_Idempotent(t *testing.T) {
	u, coord := newBaseUniverse()

	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")
	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")

	assert.Len(t, u.AllLocations(), 2)
}

func TestBranchTimelineService_PreservesOtherCoordinateFields(t *testing.T) {
	u, coord := newBaseUniverse()

	universe.BranchTimelineService(u, "home", coord, "Home", "home-t1", "T1")

	loc, _ := u.GetLocation("home-t1")
	assert.Equal(t, coord.Planet, loc.Coordinate.Planet)
	assert.Equal(t, coord.City, loc.Coordinate.City)
}

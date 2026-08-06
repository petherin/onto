package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// edgeTo returns true when edges contains an edge to `to` with mode `mode`.
func edgeTo(edges []universe.EdgeVO, to string, mode universe.TravelModeVO) bool {
	for _, e := range edges {
		if e.To == to && e.Mode == mode {
			return true
		}
	}
	return false
}

func TestAggregate_AddAndGetLocation(t *testing.T) {
	u := universe.NewAggregate()
	u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home"})

	got, ok := u.GetLocation("home")

	require.True(t, ok)
	assert.Equal(t, "home", got.ID)
	assert.Equal(t, "Home", got.Name)
}

func TestAggregate_GetLocation_NotFound(t *testing.T) {
	u := universe.NewAggregate()
	_, ok := u.GetLocation("missing")
	assert.False(t, ok)
}

func TestAggregate_AddLocation_RejectsDuplicateIdentity(t *testing.T) {
	u := universe.NewAggregate()
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home", Name: "Old"}))
	err := u.AddLocation(universe.LocationEntity{ID: "home", Name: "New"})

	got, _ := u.GetLocation("home")
	assert.ErrorIs(t, err, universe.ErrLocationAlreadyExists)
	assert.Equal(t, "Old", got.Name)
}

func TestAggregate_EdgesFrom_ReturnsOutgoingEdges(t *testing.T) {
	u := universe.NewAggregate()
	for _, id := range []string{"a", "b", "c"} {
		require.NoError(t, u.AddLocation(universe.LocationEntity{ID: id}))
	}
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "a", To: "b", Mode: universe.Walk}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "a", To: "c", Mode: universe.Rail}))

	edges := u.EdgesFrom("a")

	assert.Len(t, edges, 2)
	assert.True(t, edgeTo(edges, "b", universe.Walk))
	assert.True(t, edgeTo(edges, "c", universe.Rail))
}

func TestAggregate_EdgesFrom_NoEdgesReturnsEmpty(t *testing.T) {
	u := universe.NewAggregate()
	assert.Empty(t, u.EdgesFrom("nowhere"))
}

func TestAggregate_AllLocations(t *testing.T) {
	u := universe.NewAggregate()
	u.AddLocation(universe.LocationEntity{ID: "a"})
	u.AddLocation(universe.LocationEntity{ID: "b"})

	locs := u.AllLocations()
	assert.Len(t, locs, 2)
}

func TestAggregate_AllLocationIDs(t *testing.T) {
	u := universe.NewAggregate()
	u.AddLocation(universe.LocationEntity{ID: "x"})
	u.AddLocation(universe.LocationEntity{ID: "y"})

	ids := u.AllLocationIDs()
	assert.ElementsMatch(t, []string{"x", "y"}, ids)
}

func TestAggregate_AllEdgesFlat(t *testing.T) {
	u := universe.NewAggregate()
	for _, id := range []string{"a", "b", "c"} {
		require.NoError(t, u.AddLocation(universe.LocationEntity{ID: id}))
	}
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "a", To: "b", Mode: universe.Walk}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "b", To: "c", Mode: universe.Walk}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "a", To: "c", Mode: universe.Walk}))

	assert.Len(t, u.AllEdgesFlat(), 3)
}

// --- IsPhysical ---

func TestIsPhysical(t *testing.T) {
	tests := []struct {
		mode     universe.TravelModeVO
		physical bool
	}{
		{universe.Walk, true},
		{universe.Cycle, true},
		{universe.Drive, true},
		{universe.Rail, true},
		{universe.Flight, true},
		{universe.Orbit, true},
		{universe.Warp, true},
		{universe.QuantumShift, false},
		{universe.TimelineShift, false},
		{universe.UniverseShift, false},
		{universe.SimulationEntry, false},
		{universe.ObserverShift, false},
		{universe.MathematicalShift, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.mode), func(t *testing.T) {
			assert.Equal(t, tc.physical, tc.mode.IsPhysical())
		})
	}
}

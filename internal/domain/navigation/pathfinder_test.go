package navigation_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newLinearGraph builds a → b → c → d with known distances and costs.
func newLinearGraph() *universe.Aggregate {
	u := universe.NewAggregate()
	for _, id := range []string{"a", "b", "c", "d"} {
		u.AddLocation(universe.LocationEntity{ID: id})
	}
	u.AddEdge(universe.EdgeVO{From: "a", To: "b", Mode: universe.Walk, Distance: 1.0, Cost: 2.0})
	u.AddEdge(universe.EdgeVO{From: "b", To: "c", Mode: universe.Walk, Distance: 2.0, Cost: 3.0})
	u.AddEdge(universe.EdgeVO{From: "c", To: "d", Mode: universe.Walk, Distance: 1.5, Cost: 1.0})
	return u
}

func TestFindRoute(t *testing.T) {
	tests := []struct {
		name          string
		from, to      string
		wantFound     bool
		wantLen       int    // ignored when wantFound=false
		wantFirstFrom string // ignored when wantLen==0
		wantLastTo    string // ignored when wantLen==0
	}{
		{"same start and end", "a", "a", true, 0, "", ""},
		{"direct edge", "a", "b", true, 1, "a", "b"},
		{"two hops", "a", "c", true, 2, "a", "c"},
		{"three hops", "a", "d", true, 3, "a", "d"},
		{"no route — reverse direction", "d", "a", false, 0, "", ""},
		{"unknown destination", "a", "z", false, 0, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := newLinearGraph()
			path, ok := navigation.FindRoute(u, tc.from, tc.to)

			assert.Equal(t, tc.wantFound, ok)
			if !tc.wantFound {
				return
			}
			require.Len(t, path, tc.wantLen)
			if tc.wantLen > 0 {
				assert.Equal(t, tc.wantFirstFrom, path[0].From)
				assert.Equal(t, tc.wantLastTo, path[len(path)-1].To)
			}
		})
	}
}

func TestFindRoute_SkipsPhysicalRealityBoundary(t *testing.T) {
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()
	divergent := base
	divergent.Consensus = 1
	u.AddLocation(universe.LocationEntity{ID: "home", Coordinate: base})
	u.AddLocation(universe.LocationEntity{ID: "divergent-home", Coordinate: divergent})
	u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: base})
	u.AddEdge(universe.EdgeVO{From: "divergent-home", To: "station", Mode: universe.Walk})
	u.AddEdge(universe.EdgeVO{From: "divergent-home", To: "home", Mode: universe.ConsensusShift})
	u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk})

	path, ok := navigation.FindRoute(u, "divergent-home", "station")

	require.True(t, ok)
	require.Len(t, path, 2)
	assert.Equal(t, universe.ConsensusShift, path[0].Mode)
	assert.Equal(t, universe.Walk, path[1].Mode)
}

func TestPathDistance(t *testing.T) {
	tests := []struct {
		name string
		path []universe.EdgeVO
		want float64
	}{
		{"empty path", nil, 0.0},
		{"single edge", []universe.EdgeVO{{Distance: 3.0}}, 3.0},
		{"multiple edges", []universe.EdgeVO{{Distance: 1.5}, {Distance: 2.0}, {Distance: 0.5}}, 4.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, navigation.PathDistance(tc.path))
		})
	}
}

func TestPathCost(t *testing.T) {
	tests := []struct {
		name string
		path []universe.EdgeVO
		want float64
	}{
		{"empty path", nil, 0.0},
		{"single edge", []universe.EdgeVO{{Cost: 5.0}}, 5.0},
		{"multiple edges", []universe.EdgeVO{{Cost: 10.0}, {Cost: 5.0}, {Cost: 2.5}}, 17.5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, navigation.PathCost(tc.path))
		})
	}
}

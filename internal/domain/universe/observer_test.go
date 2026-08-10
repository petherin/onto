package universe_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchObserver_CreatesContextualPhysicalMap(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Cost: 1}))

	require.NoError(t, universe.BranchObserver(u, "home", coord, "Home", "home-o-machine", "Machine"))

	home, ok := u.GetLocation("home-o-machine")
	require.True(t, ok)
	assert.Equal(t, "Machine", home.Coordinate.Observer)
	assert.True(t, edgeTo(u.EdgesFrom("home-o-machine"), "station-o-machine", universe.Walk))
	assert.True(t, edgeTo(u.EdgesFrom("station-o-machine"), "station", universe.ObserverShift))
}

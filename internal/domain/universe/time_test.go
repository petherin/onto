package universe_test

import (
	"testing"
	"time"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchTimeService_CreatesContextualPhysicalMap(t *testing.T) {
	u, coord := newBaseUniverse(t)
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Cost: 1}))
	target := time.Date(2042, 1, 2, 3, 4, 5, 0, time.UTC)

	require.NoError(t, universe.BranchTimeService(u, "home", coord, "Home", "home-at-20420102t030405z", target))

	home, ok := u.GetLocation("home-at-20420102t030405z")
	require.True(t, ok)
	assert.Equal(t, target, home.Coordinate.Time)
	assert.True(t, edgeTo(u.EdgesFrom("home-at-20420102t030405z"), "station-at-20420102t030405z", universe.Walk))
	assert.True(t, edgeTo(u.EdgesFrom("station-at-20420102t030405z"), "station", universe.TimeShift))
}

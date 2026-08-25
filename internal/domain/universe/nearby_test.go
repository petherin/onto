package universe_test

import (
	"errors"
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addNearby generates the next nearby location off originID and commits it (and
// its bidirectional edges) to u, mirroring what the travel command does when it
// expands a dead end. It returns the new location so callers can chain from it.
func addNearby(t *testing.T, u *universe.Aggregate, originID string, coord universe.CoordinateVO) universe.LocationEntity {
	t.Helper()
	loc, out, back, err := universe.NewNearbyLocation(u, originID, coord)
	require.NoError(t, err)
	require.NoError(t, u.AddLocation(loc))
	require.NoError(t, u.AddEdge(out))
	require.NoError(t, u.AddEdge(back))
	return loc
}

// TestNewNearbyLocation_ChainedDeadEnds_UniqueNames reproduces the reported bug
// where chaining through dead ends produced several distinct nodes all named
// "Nearby 1". Each expansion must now get a universe-wide-unique display name
// while the ID keeps its origin-suffix scheme.
func TestNewNearbyLocation_ChainedDeadEnds_UniqueNames(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	first := addNearby(t, u, "park", coord)
	second := addNearby(t, u, first.ID, first.Coordinate)
	third := addNearby(t, u, second.ID, second.Coordinate)

	assert.Equal(t, "park-1", first.ID)
	assert.Equal(t, "park-1-1", second.ID)
	assert.Equal(t, "park-1-1-1", third.ID)

	assert.Equal(t, "Nearby 1", first.Name)
	assert.Equal(t, "Nearby 2", second.Name)
	assert.Equal(t, "Nearby 3", third.Name)

	distinct := map[string]struct{}{first.Name: {}, second.Name: {}, third.Name: {}}
	assert.Len(t, distinct, 3, "chained dead-end names must all be distinct")
}

// TestNewNearbyLocation_NumbersFromUniverseWideMax checks the display name is
// numbered one past the highest existing "Nearby N" anywhere in the universe,
// skipping gaps and ignoring non-nearby names.
func TestNewNearbyLocation_NumbersFromUniverseWideMax(t *testing.T) {
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()

	park := base
	park.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: park}))

	// Pre-existing nearby nodes with a gap (1 and 5), plus a non-nearby node
	// the scan must ignore.
	n1 := base
	n1.Location = "Nearby 1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "a", Name: "Nearby 1", Coordinate: n1}))
	n5 := base
	n5.Location = "Nearby 5"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "b", Name: "Nearby 5", Coordinate: n5}))
	cottage := base
	cottage.Location = "Cottage"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "cottage", Name: "Cottage", Coordinate: cottage}))

	loc, _, _, err := universe.NewNearbyLocation(u, "park", park)
	require.NoError(t, err)
	assert.Equal(t, "Nearby 6", loc.Name, "name is one past the highest existing Nearby N")
}

// TestNewNearbyLocation_UnknownOrigin_Errors confirms expanding from a missing
// origin is rejected rather than silently generating an orphan node.
func TestNewNearbyLocation_UnknownOrigin_Errors(t *testing.T) {
	u := universe.NewAggregate()

	_, _, _, err := universe.NewNearbyLocation(u, "ghost", universe.DefaultCoordinateVO())

	require.Error(t, err)
	assert.True(t, errors.Is(err, universe.ErrUnknownEdgeEndpoint))
}

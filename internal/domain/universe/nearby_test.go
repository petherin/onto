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

// TestNewNearbyLocation_InsideBranch_CanonicalID reproduces the reported bug
// where returning home from a parallel universe failed with "no universe path
// back from here". A nearby location spawned inside a branch was named by
// appending "-i" to the branched origin ID (e.g. "park-u1-1"), placing a bare
// index after the "-u1" axis suffix. That made the ID's encoded axes disagree
// with its coordinate, so LowerContextID/EnsureLowerContext saw universe level
// 0 and refused to step back. The ID must instead stay canonical.
func TestNewNearbyLocation_InsideBranch_CanonicalID(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	// Shift into universe U1, then generate a nearby location while inside it.
	require.NoError(t, universe.BranchUniverse(u, "park", coord, "Park", "park-u1", "U1"))
	u1, ok := u.GetLocation("park-u1")
	require.True(t, ok)
	nearby := addNearby(t, u, "park-u1", u1.Coordinate)

	// The index belongs on the base, not after the -u1 axis suffix, so the ID
	// stays canonical and its encoded universe axis matches the coordinate.
	assert.Equal(t, "park-1-u1", nearby.ID)
	assert.Equal(t, "U1", nearby.Coordinate.Universe)

	// universe back must now resolve (creating the Origin counterpart) rather
	// than reporting no path back.
	destID, err := universe.EnsureLowerContext(u, nearby.ID, universe.UniverseShift)
	require.NoError(t, err)
	assert.Equal(t, "park-1", destID)
}

// physicallyReachable reports whether targetID is reachable from fromID by
// following only physical (Walk/Rail) edges — the same connectivity the final
// "travel home" leg of return-home depends on.
func physicallyReachable(u *universe.Aggregate, fromID, targetID string) bool {
	seen := map[string]bool{fromID: true}
	queue := []string{fromID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id == targetID {
			return true
		}
		for _, e := range u.EdgesFrom(id) {
			if e.Mode.IsPhysical() && !seen[e.To] {
				seen[e.To] = true
				queue = append(queue, e.To)
			}
		}
	}
	return false
}

// TestEnsureLowerContext_NearbyInsideBranch_ConnectsHome reproduces the reported
// "no route: home" failure. Nearby locations spawned *inside* a universe branch
// have no counterpart in the parent reality, so stepping one back down a
// universe used to manufacture an isolated Origin node connected only by the
// universe edge — stranding the final walk home. EnsureLowerContext must instead
// mirror the branch node's physical edges onto the lower counterpart so it stays
// connected back to home.
func TestEnsureLowerContext_NearbyInsideBranch_ConnectsHome(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	// Enter U1 (clones park -> park-u1 with a universe-back edge), then expand
	// two dead ends while inside U1. These nearby nodes have no Origin twin.
	require.NoError(t, universe.BranchUniverse(u, "park", coord, "Park", "park-u1", "U1"))
	root, ok := u.GetLocation("park-u1")
	require.True(t, ok)
	first := addNearby(t, u, "park-u1", root.Coordinate)
	second := addNearby(t, u, first.ID, first.Coordinate)
	require.Equal(t, "park-1-u1", first.ID)
	require.Equal(t, "park-1-1-u1", second.ID)

	// Stepping the deepest branch node down a universe must land on a node that
	// is physically connected back to park (home), not an isolated counterpart.
	destID, err := universe.EnsureLowerContext(u, second.ID, universe.UniverseShift)
	require.NoError(t, err)
	assert.Equal(t, "park-1-1", destID)
	assert.True(t, physicallyReachable(u, destID, "park"),
		"lower-context counterpart of a nearby-in-branch node must reach home via physical edges")
}

// TestNewNearbyLocation_UnknownOrigin_Errors confirms expanding from a missing
// origin is rejected rather than silently generating an orphan node.
func TestNewNearbyLocation_UnknownOrigin_Errors(t *testing.T) {
	u := universe.NewAggregate()

	_, _, _, err := universe.NewNearbyLocation(u, "ghost", universe.DefaultCoordinateVO())

	require.Error(t, err)
	assert.True(t, errors.Is(err, universe.ErrUnknownEdgeEndpoint))
}

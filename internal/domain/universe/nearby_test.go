package universe_test

import (
	"errors"
	"strings"
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

// TestNewNearbyLocation_GeneratesRichDescription confirms auto-generated nearby
// nodes no longer share a flat placeholder: the description is non-empty, drops
// the old "Auto-generated nearby location" string, and anchors on the spatial
// setting so it reads as a real place.
func TestNewNearbyLocation_GeneratesRichDescription(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	loc := addNearby(t, u, "park", coord)

	assert.NotEmpty(t, loc.Description)
	assert.NotEqual(t, "Auto-generated nearby location", loc.Description)
	assert.Contains(t, loc.Description, "Leeds", "base-reality description anchors on the city")
}

// TestGenerateDescription_Deterministic confirms the generator is a pure
// function of the coordinate: the same coordinate always yields identical text,
// so descriptions are stable across reloads and reproducible in tests.
func TestGenerateDescription_Deterministic(t *testing.T) {
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Nearby 7"

	assert.Equal(t, universe.GenerateDescription(coord), universe.GenerateDescription(coord))
}

// TestGenerateDescription_ReflectsActiveAxes confirms each active non-default
// axis contributes an atmospheric clause naming its token, so a node deep in a
// branch reads differently from a plain base-reality one.
func TestGenerateDescription_ReflectsActiveAxes(t *testing.T) {
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Nearby 3"
	coord.Quantum = "Q2"
	coord.Observer = "Bat"

	desc := universe.GenerateDescription(coord)

	assert.Contains(t, desc, "Q2", "quantum branch token appears in the description")
	assert.Contains(t, desc, "Bat", "observer frame appears in the description")

	// A plain base-reality coordinate produces none of those axis clauses.
	base := universe.DefaultCoordinateVO()
	base.Location = "Nearby 3"
	plain := universe.GenerateDescription(base)
	assert.False(t, strings.Contains(plain, "Q2") || strings.Contains(plain, "Bat"),
		"base-reality description carries no exotic-axis clauses")
}

// TestIsPhysicalDeadEnd covers the shared dead-end predicate: a node is a dead
// end when its only outgoing physical edges point back the way it was arrived on
// (or it has no physical edges at all), while a non-physical exit does not count
// as an onward physical route.
func TestIsPhysicalDeadEnd(t *testing.T) {
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home", Coordinate: base}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "well", Name: "Well", Coordinate: base}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: base}))
	// well has a one-way physical drop in and a non-physical exit only.
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "well", Mode: universe.Walk, Cost: 1}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "well", To: "park", Mode: universe.ConsensusShift, Cost: universe.ConsensusShiftCost}))

	// Non-physical exit does not rescue it from being a physical dead end.
	assert.True(t, universe.IsPhysicalDeadEnd(u, "well", "home"))
	// A physical onward edge to somewhere other than cameFrom makes it not a dead end.
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "well", To: "home", Mode: universe.Walk, Cost: 1}))
	assert.False(t, universe.IsPhysicalDeadEnd(u, "well", "park"))
	// ...but if that only physical edge points back to cameFrom, it is still a dead end.
	assert.True(t, universe.IsPhysicalDeadEnd(u, "well", "home"))
}

// TestHasPhysicalEscape_DeterministicPerReality confirms the escape verdict is
// stable for a given coordinate (reproducible across reloads and tests) while
// varying across realities, and that base reality is never gated.
func TestHasPhysicalEscape_DeterministicPerReality(t *testing.T) {
	base := universe.DefaultCoordinateVO()
	base.Location = "Well"
	assert.True(t, universe.HasPhysicalEscape(base, universe.ConsensusShiftCost), "base reality is never gated")

	// The same non-base coordinate and cost always yield the same verdict.
	c := base
	c.Consensus = 3
	assert.Equal(t,
		universe.HasPhysicalEscape(c, universe.ConsensusShiftCost),
		universe.HasPhysicalEscape(c, universe.ConsensusShiftCost))

	// Across a run of realities both outcomes occur at a mid-range cost, so
	// escapability genuinely varies rather than being constant.
	sawEscape, sawBlocked := false, false
	for level := 1; level <= 40; level++ {
		cc := base
		cc.Consensus = level
		if universe.HasPhysicalEscape(cc, universe.QuantumShiftCost) {
			sawEscape = true
		} else {
			sawBlocked = true
		}
	}
	assert.True(t, sawEscape, "some realities offer a physical escape")
	assert.True(t, sawBlocked, "some realities offer no physical escape")
}

// TestEscapeProbability_ScalesWithCost confirms the gamble: cheap transitions
// give low escape odds, expensive ones high odds, monotonically between the
// configured bounds, with clamping outside the cost range.
func TestEscapeProbability_ScalesWithCost(t *testing.T) {
	// Cheapest transition sits at the floor, dearest at the ceiling.
	assert.InDelta(t, universe.EscapeProbMin, universe.EscapeProbability(universe.ObserverShiftCost), 1e-9)
	assert.InDelta(t, universe.EscapeProbMax, universe.EscapeProbability(universe.MathematicalShiftCost), 1e-9)

	// Below/above the range clamps to the bounds rather than extrapolating.
	assert.InDelta(t, universe.EscapeProbMin, universe.EscapeProbability(0), 1e-9)
	assert.InDelta(t, universe.EscapeProbMax, universe.EscapeProbability(universe.MathematicalShiftCost*10), 1e-9)

	// Probability rises monotonically across the transition cost ladder.
	ladder := []float64{
		universe.ObserverShiftCost, universe.ConsensusShiftCost, universe.SimulationEntryCost,
		universe.QuantumShiftCost, universe.TimeShiftCost, universe.TimelineShiftCost,
		universe.UniverseShiftCost, universe.MathematicalShiftCost,
	}
	for i := 1; i < len(ladder); i++ {
		assert.Greater(t, universe.EscapeProbability(ladder[i]), universe.EscapeProbability(ladder[i-1]),
			"a dearer transition must give better escape odds")
	}

	// A cheap transition is genuinely unlikely to escape; a dear one likely.
	cheapEscapes, dearEscapes := 0, 0
	base := universe.DefaultCoordinateVO()
	for level := 1; level <= 200; level++ {
		c := base
		c.Consensus = level
		if universe.HasPhysicalEscape(c, universe.ObserverShiftCost) {
			cheapEscapes++
		}
		if universe.HasPhysicalEscape(c, universe.MathematicalShiftCost) {
			dearEscapes++
		}
	}
	assert.Less(t, cheapEscapes, dearEscapes,
		"cheap transitions escape far less often than expensive ones over many realities")
}

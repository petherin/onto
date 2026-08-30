package universe_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addCluster generates the nearby cluster off originID and commits it (every
// location, then every edge) to u, mirroring what the travel command does when
// it expands a dead end. It returns the created locations so callers can chain.
func addCluster(t *testing.T, u *universe.Aggregate, originID string, coord universe.CoordinateVO) []universe.LocationEntity {
	t.Helper()
	locs, edges, err := universe.NewNearbyCluster(u, originID, coord)
	require.NoError(t, err)
	for _, loc := range locs {
		require.NoError(t, u.AddLocation(loc))
	}
	for _, e := range edges {
		require.NoError(t, u.AddEdge(e))
	}
	return locs
}

// TestNewNearbyCluster_ChainedDeadEnds_UniqueNames reproduces the reported bug
// where chaining through dead ends produced several distinct nodes all named
// "Nearby 1". Each expansion now yields a 1–3 node cluster; every node carries
// the Generated flag, and IDs and display names stay unique as the traveller
// chains from one cluster into the next.
func TestNewNearbyCluster_ChainedDeadEnds_UniqueNames(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	ids := map[string]struct{}{}
	names := map[string]struct{}{}
	originID, originCoord := "park", coord
	for step := 0; step < 3; step++ {
		locs := addCluster(t, u, originID, originCoord)
		require.GreaterOrEqual(t, len(locs), 1)
		require.LessOrEqual(t, len(locs), 3)
		for _, l := range locs {
			assert.True(t, l.Generated, "auto-generated node must carry the Generated flag")
			_, dupID := ids[l.ID]
			assert.False(t, dupID, "IDs must be unique across chained clusters")
			ids[l.ID] = struct{}{}
			_, dupName := names[l.Name]
			assert.False(t, dupName, "display names must be distinct across chained clusters")
			names[l.Name] = struct{}{}
		}
		originID, originCoord = locs[0].ID, locs[0].Coordinate
	}
}

// TestNewNearbyCluster_InsideBranch_CanonicalID reproduces the reported bug
// where returning home from a parallel universe failed with "no universe path
// back from here". A nearby location spawned inside a branch was named by
// appending "-i" to the branched origin ID (e.g. "park-u1-1"), placing a bare
// index after the "-u1" axis suffix. That made the ID's encoded axes disagree
// with its coordinate, so LowerContextID/EnsureLowerContext saw universe level
// 0 and refused to step back. Every node's ID must instead stay canonical.
func TestNewNearbyCluster_InsideBranch_CanonicalID(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	// Shift into universe U1, then generate a cluster while inside it.
	require.NoError(t, universe.BranchUniverse(u, "park", coord, "Park", "park-u1", "U1"))
	u1, ok := u.GetLocation("park-u1")
	require.True(t, ok)
	locs := addCluster(t, u, "park-u1", u1.Coordinate)

	for _, l := range locs {
		assert.False(t, universe.LocationIDIsMalformed(l.ID), "each node ID stays canonical")
		assert.Equal(t, "U1", l.Coordinate.Universe)

		// universe back must resolve (creating the Origin counterpart) rather
		// than reporting no path back, and land on the branch-free counterpart.
		destID, err := universe.EnsureLowerContext(u, l.ID, universe.UniverseShift)
		require.NoError(t, err)
		assert.NotEqual(t, l.ID, destID)
		assert.NotContains(t, destID, "-u1", "the lower counterpart drops the universe suffix")
	}
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
// mirror the branch node's physical edges (the star edges back to its origin)
// onto the lower counterpart so it stays connected back to home.
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
	first := addCluster(t, u, "park-u1", root.Coordinate)[0]
	second := addCluster(t, u, first.ID, first.Coordinate)[0]

	// Stepping the deepest branch node down a universe must land on a node that
	// is physically connected back to park (home), not an isolated counterpart.
	destID, err := universe.EnsureLowerContext(u, second.ID, universe.UniverseShift)
	require.NoError(t, err)
	assert.True(t, physicallyReachable(u, destID, "park"),
		"lower-context counterpart of a nearby-in-branch node must reach home via physical edges")
}

// TestNewNearbyCluster_UnknownOrigin_Errors confirms expanding from a missing
// origin is rejected rather than silently generating orphan nodes.
func TestNewNearbyCluster_UnknownOrigin_Errors(t *testing.T) {
	u := universe.NewAggregate()

	_, _, err := universe.NewNearbyCluster(u, "ghost", universe.DefaultCoordinateVO())

	require.Error(t, err)
	assert.True(t, errors.Is(err, universe.ErrUnknownEdgeEndpoint))
}

// TestNewNearbyCluster_GeneratesRichDescription confirms auto-generated nodes no
// longer share a flat placeholder or a bare "Nearby N" name: every node's
// description is non-empty, drops the old "Auto-generated nearby location"
// string, and anchors on the spatial setting so it reads as a real place.
func TestNewNearbyCluster_GeneratesRichDescription(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	locs := addCluster(t, u, "park", coord)

	for _, loc := range locs {
		assert.NotEmpty(t, loc.Description)
		assert.NotEqual(t, "Auto-generated nearby location", loc.Description)
		assert.Contains(t, loc.Description, "Leeds", "base-reality description anchors on the city")
		assert.False(t, strings.HasPrefix(loc.Name, "Nearby "), "names are varied, not a Nearby N counter")
	}
}

// TestNewNearbyCluster_Deterministic confirms the same origin coordinate expands
// identically on two fresh aggregates: same count, IDs, and names.
func TestNewNearbyCluster_Deterministic(t *testing.T) {
	build := func() []universe.LocationEntity {
		u := universe.NewAggregate()
		coord := universe.DefaultCoordinateVO()
		coord.Location = "Park"
		require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))
		locs, _, err := universe.NewNearbyCluster(u, "park", coord)
		require.NoError(t, err)
		return locs
	}

	a, b := build(), build()
	require.Equal(t, len(a), len(b))
	for i := range a {
		assert.Equal(t, a[i].ID, b[i].ID)
		assert.Equal(t, a[i].Name, b[i].Name)
	}
}

// TestNewNearbyCluster_WiresOriginAsStar confirms each generated node is wired
// to the origin with a bidirectional Walk edge and to nothing else: the cluster
// is a star, not a clique. Leaving siblings unconnected keeps each node a
// physical leaf (its only physical edge is back to the origin), so travelling to
// any generated node is itself a dead end that expands again — the property that
// lets the map chain outward indefinitely.
func TestNewNearbyCluster_WiresOriginAsStar(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))

	locs, edges, err := universe.NewNearbyCluster(u, "park", coord)
	require.NoError(t, err)

	hasWalk := func(from, to string) bool {
		for _, e := range edges {
			if e.From == from && e.To == to && e.Mode == universe.Walk {
				return true
			}
		}
		return false
	}

	for _, l := range locs {
		assert.True(t, hasWalk("park", l.ID), "origin -> node walk edge present")
		assert.True(t, hasWalk(l.ID, "park"), "node -> origin walk edge present")
	}
	// No edge may connect one cluster node to another: each must remain a leaf so
	// it expands again when reached.
	for i := 0; i < len(locs); i++ {
		for j := 0; j < len(locs); j++ {
			if i == j {
				continue
			}
			assert.False(t, hasWalk(locs[i].ID, locs[j].ID),
				"cluster nodes must not be interconnected, so each stays a leaf")
		}
	}
	// Every physical edge out of a generated node points back to the origin, so
	// each node has exactly one onward physical route: it is a dead end.
	for _, l := range locs {
		for _, e := range edges {
			if e.From == l.ID && e.Mode.IsPhysical() {
				assert.Equal(t, "park", e.To,
					"a generated node's only physical edge is back to the origin")
			}
		}
	}
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

package universe_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/petherin/onto/internal/domain/navigation"
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

// nestedOrigin builds an aggregate with a base "park" and a universe-branch
// counterpart "park-u1" (nesting depth > 0), returning the branch origin ID and
// coordinate. Traps only spawn in nested realities, so this is the setup the
// trap tests expand from.
func nestedOrigin(t *testing.T) (*universe.Aggregate, string, universe.CoordinateVO) {
	t.Helper()
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Location = "Park"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: coord}))
	require.NoError(t, universe.BranchUniverse(u, "park", coord, "Park", "park-u1", "U1"))
	root, ok := u.GetLocation("park-u1")
	require.True(t, ok)
	return u, "park-u1", root.Coordinate
}

// addTrap generates the given trap off originID and commits it (every location,
// then every edge), mirroring what the generator does when travel expands a dead
// end. It returns the created locations so tests can assert on them.
func addTrap(t *testing.T, u *universe.Aggregate, originID string, coord universe.CoordinateVO, trap universe.TrapType) []universe.LocationEntity {
	t.Helper()
	locs, edges, err := universe.GenerateTrap(u, originID, coord, trap)
	require.NoError(t, err)
	for _, l := range locs {
		require.NoError(t, u.AddLocation(l))
	}
	for _, e := range edges {
		require.NoError(t, u.AddEdge(e))
	}
	return locs
}

// TestSelectTrap_BaseRealityFree confirms base reality (nesting depth 0) is never
// trapped, however its spatial position varies — the starter world stays gentle.
func TestSelectTrap_BaseRealityFree(t *testing.T) {
	base := universe.DefaultCoordinateVO()
	for i := 0; i < 500; i++ {
		c := base
		c.Location = fmt.Sprintf("Place %d", i)
		trap, ok := universe.SelectTrap(c)
		assert.False(t, ok, "base reality must never spawn a trap")
		assert.Equal(t, universe.NoTrap, trap)
	}
}

// TestSelectTrap_DeterministicPerReality confirms the trap verdict is stable for
// a given coordinate (reproducible across reloads/tests) yet varies between
// realities, exactly like the cost-scaled escape gamble.
func TestSelectTrap_DeterministicPerReality(t *testing.T) {
	base := universe.DefaultCoordinateVO()
	base.Quantum = "Q1"
	distinctVerdicts := map[string]bool{}
	for i := 0; i < 50; i++ {
		c := base
		c.Location = fmt.Sprintf("Spot %d", i)
		trap1, ok1 := universe.SelectTrap(c)
		trap2, ok2 := universe.SelectTrap(c)
		assert.Equal(t, ok1, ok2, "same coordinate yields the same trap verdict")
		assert.Equal(t, trap1, trap2, "same coordinate yields the same trap type")
		distinctVerdicts[fmt.Sprintf("%v-%v", ok1, trap1)] = true
	}
	assert.Greater(t, len(distinctVerdicts), 1, "the verdict must vary across realities")
}

// TestSelectTrap_ProbabilityInBounds confirms traps stay occasional — roughly the
// tuned probability over many nested realities, and only ever an edge-wiring
// archetype.
func TestSelectTrap_ProbabilityInBounds(t *testing.T) {
	base := universe.DefaultCoordinateVO()
	base.Quantum = "Q1"
	const n = 4000
	traps := 0
	for i := 0; i < n; i++ {
		c := base
		c.Location = fmt.Sprintf("Spot %d", i)
		if trap, ok := universe.SelectTrap(c); ok {
			traps++
			assert.True(t, trap.IsKnown())
			assert.NotEqual(t, universe.NoTrap, trap)
		}
	}
	rate := float64(traps) / float64(n)
	assert.Greater(t, rate, 0.04, "traps should occur")
	assert.Less(t, rate, 0.12, "traps should stay occasional")
}

// TestGenerateTrap_SealedVault_NoPhysicalExitButRoutableHome confirms the harshest
// trap has no walkable way out (a physical sink, so travel never auto-expands it)
// yet home can always route out across its non-physical escape edge.
func TestGenerateTrap_SealedVault_NoPhysicalExitButRoutableHome(t *testing.T) {
	u, origin, coord := nestedOrigin(t)
	locs := addTrap(t, u, origin, coord, universe.TrapSealedVault)
	require.Len(t, locs, 1)
	vault := locs[0]
	assert.Equal(t, universe.TrapSealedVault, vault.Trap)
	assert.True(t, vault.Generated)
	assert.False(t, universe.HasPhysicalExit(u, vault.ID), "a sealed vault is a physical sink")
	_, ok := navigation.FindRoute(u, vault.ID, origin)
	assert.True(t, ok, "home must route out of a sealed vault via its escape edge")
}

// TestGenerateTrap_TarPit_WalkableButCostly confirms the tar pit keeps physical
// exits (so it still auto-expands and home can walk out) but each walk costs far
// more than an ordinary path.
func TestGenerateTrap_TarPit_WalkableButCostly(t *testing.T) {
	u, origin, coord := nestedOrigin(t)
	locs := addTrap(t, u, origin, coord, universe.TrapTarPit)
	require.GreaterOrEqual(t, len(locs), 1)
	require.LessOrEqual(t, len(locs), 3)
	for _, l := range locs {
		assert.Equal(t, universe.TrapTarPit, l.Trap)
		assert.True(t, universe.HasPhysicalExit(u, l.ID), "tar-pit nodes keep a physical exit")
		assert.True(t, physicallyReachable(u, l.ID, origin), "you can still walk out of the tar, at a price")
	}
	for _, e := range u.EdgesFrom(origin) {
		if e.Mode == universe.Walk {
			assert.Greater(t, e.Cost, 1.0, "wading into the tar costs more than an ordinary step")
		}
	}
}

// TestGenerateTrap_MobiusMaze_DecoysLoopWithoutStranding confirms the maze's
// decoys do not auto-expand (they are never plain dead ends) and the hub keeps a
// true physical way back to the origin.
func TestGenerateTrap_MobiusMaze_DecoysLoopWithoutStranding(t *testing.T) {
	u, origin, coord := nestedOrigin(t)
	locs := addTrap(t, u, origin, coord, universe.TrapMobiusMaze)
	require.Len(t, locs, 3)
	hub := locs[0]
	assert.True(t, physicallyReachable(u, hub.ID, origin), "the hub's true exit walks back to the origin")
	for _, decoy := range locs[1:] {
		assert.False(t, universe.IsPhysicalDeadEnd(u, decoy.ID, hub.ID),
			"maze decoys must not read as dead ends, or they would auto-expand")
		_, ok := navigation.FindRoute(u, decoy.ID, origin)
		assert.True(t, ok, "home must route out from any maze node")
	}
}

// TestGenerateTrap_OneWaySink_NoWalkBackButRoutableHome confirms you cannot walk
// back to the origin from the pocket (it is one-way) yet home can always route
// out via the entry node's escape edge.
func TestGenerateTrap_OneWaySink_NoWalkBackButRoutableHome(t *testing.T) {
	u, origin, coord := nestedOrigin(t)
	locs := addTrap(t, u, origin, coord, universe.TrapOneWaySink)
	require.Len(t, locs, 2)
	mouth, pit := locs[0], locs[1]
	assert.False(t, physicallyReachable(u, mouth.ID, origin), "the trapdoor is one-way — no walk back")
	assert.False(t, physicallyReachable(u, pit.ID, origin), "the pocket cannot walk back to the origin")
	for _, l := range locs {
		_, ok := navigation.FindRoute(u, l.ID, origin)
		assert.True(t, ok, "home must route out of a one-way sink via its escape edge")
	}
}

// TestGenerateTrap_UnknownOrigin_Errors confirms a trap cannot be wired off a
// missing origin.
func TestGenerateTrap_UnknownOrigin_Errors(t *testing.T) {
	u := universe.NewAggregate()
	_, _, err := universe.GenerateTrap(u, "ghost", universe.DefaultCoordinateVO(), universe.TrapSealedVault)
	require.Error(t, err)
	assert.True(t, errors.Is(err, universe.ErrUnknownEdgeEndpoint))
}

// TestClusterGenerator_TrapsOnlyInNestedRealities confirms the generator spawns
// traps only away from base reality: an ordinary base expansion carries no trap,
// while a nested reality that rolls a trap tags its nodes with the archetype.
func TestClusterGenerator_TrapsOnlyInNestedRealities(t *testing.T) {
	gen := universe.NewClusterLocationGenerator()

	base := universe.NewAggregate()
	baseCoord := universe.DefaultCoordinateVO()
	baseCoord.Location = "Park"
	require.NoError(t, base.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: baseCoord}))
	baseLocs, _, err := gen.Generate(base, "park", baseCoord)
	require.NoError(t, err)
	for _, l := range baseLocs {
		assert.Equal(t, universe.NoTrap, l.Trap, "base reality never traps")
	}

	// Find a nested coordinate that rolls a trap, then confirm Generate tags it.
	nested := universe.DefaultCoordinateVO()
	nested.Quantum = "Q1"
	var trapCoord universe.CoordinateVO
	var want universe.TrapType
	for i := 0; i < 100000; i++ {
		c := nested
		c.Location = fmt.Sprintf("Spot %d", i)
		if trap, ok := universe.SelectTrap(c); ok {
			trapCoord, want = c, trap
			break
		}
	}
	require.NotEqual(t, universe.NoTrap, want, "expected to find a trapping coordinate")

	u := universe.NewAggregate()
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "park", Name: "Park", Coordinate: trapCoord}))
	locs, _, err := gen.Generate(u, "park", trapCoord)
	require.NoError(t, err)
	require.NotEmpty(t, locs)
	for _, l := range locs {
		assert.Equal(t, want, l.Trap, "a trapping coordinate tags its generated nodes")
	}
}

package universe

import (
	"fmt"
	"math"
	"math/rand"
)

// LocationGeneratorService is a domain service interface for the policy that
// expands a dead end into a small cluster of new nearby locations and their
// bidirectional physical connections. Depending on this abstraction — rather
// than the concrete ClusterLocationGenerator — lets callers substitute a
// different nearby-location policy without changing the command that uses it.
type LocationGeneratorService interface {
	Generate(u *Aggregate, originID string, coordinate CoordinateVO) ([]LocationEntity, []EdgeVO, error)
}

// ClusterLocationGenerator is the domain's standard nearby-location policy: it
// expands a dead end into a deterministic 1–3 node cluster, wiring each node to
// the origin only (a star, not a clique). Leaving the cluster nodes
// unconnected to one another keeps each of them a physical leaf, so walking to
// any generated node is itself a dead end that expands again — letting the map
// grow outward indefinitely as the traveller explores.
type ClusterLocationGenerator struct{}

// NewClusterLocationGenerator returns the standard cluster nearby-location
// generator.
func NewClusterLocationGenerator() *ClusterLocationGenerator {
	return &ClusterLocationGenerator{}
}

// Generate implements LocationGeneratorService using the cluster policy.
func (ClusterLocationGenerator) Generate(u *Aggregate, originID string, coordinate CoordinateVO) ([]LocationEntity, []EdgeVO, error) {
	return NewNearbyCluster(u, originID, coordinate)
}

// NewNearbyCluster expands a dead end into a deterministic cluster of 1–3
// distinctly-named nearby locations, each wired to the origin by a bidirectional
// physical connection (a star, not a clique). It contains the domain policy for
// expanding a dead end without performing any persistence or user interaction.
// The cluster size, IDs, and names are seeded from the origin coordinate, so the
// same reality always expands the same way. Crucially, the cluster nodes are not
// linked to one another: each therefore has exactly one physical edge (back to
// the origin), so it is itself a dead end that expands again on arrival, letting
// the traveller chain deeper and grow the map without bound.
func NewNearbyCluster(u *Aggregate, originID string, coordinate CoordinateVO) ([]LocationEntity, []EdgeVO, error) {
	if _, exists := u.GetLocation(originID); !exists {
		return nil, nil, fmt.Errorf("%w: %s", ErrUnknownEdgeEndpoint, originID)
	}
	rng := rand.New(rand.NewSource(coordinateSeed(coordinate)))
	count := 1 + rng.Intn(3)

	// Number each nearby index onto the origin's stable base and reassemble
	// canonically, so a location spawned inside a reality branch keeps its
	// axis suffixes in canonical order (e.g. "park-1-u1", not "park-u1-1").
	// A bare "-i" appended after an axis suffix would make the ID's encoded
	// axes disagree with its coordinate, breaking LowerContextID and hence
	// every *back / return-home step for that axis.
	base, ax := parseLocationID(originID)
	ids := make([]string, 0, count)
	for i := 1; i < 1000 && len(ids) < count; i++ {
		id := buildLocationID(fmt.Sprintf("%s-%d", base, i), ax)
		if _, exists := u.GetLocation(id); exists {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) < count {
		return nil, nil, fmt.Errorf("%w: no nearby location ID available", ErrInvalidLocation)
	}

	// Seed the name-uniqueness set with every existing display name so cluster
	// names are unique across the whole universe, and grow it per node so two
	// siblings in the same batch cannot collide either.
	used := make(map[string]bool)
	for _, loc := range u.AllLocations() {
		used[loc.Name] = true
	}

	locations := make([]LocationEntity, 0, count)
	var edges []EdgeVO
	for _, id := range ids {
		name := generateNearbyName(rng, used)
		used[name] = true
		nodeCoord := coordinate
		nodeCoord.Location = name
		locations = append(locations, LocationEntity{
			ID:          id,
			Name:        name,
			Description: GenerateDescription(nodeCoord),
			Coordinate:  nodeCoord,
			Generated:   true,
		})
		edges = append(edges,
			EdgeVO{From: originID, To: id, Mode: Walk, Distance: 1, Cost: 1, Description: "Auto-generated path"},
			EdgeVO{From: id, To: originID, Mode: Walk, Distance: 1, Cost: 1, Description: "Auto-generated return path"},
		)
	}
	// Deliberately leave the cluster nodes unconnected to one another. Each node
	// then has a single physical edge — back to the origin — so it is itself a
	// dead end that auto-expands on arrival, letting the traveller chain deeper
	// and grow the map without bound. Interconnecting them would make each node a
	// non-leaf and stop the chain.
	return locations, edges, nil
}

// IsPhysicalDeadEnd reports whether a location has no outgoing physical edge
// other than one leading back to cameFrom (the edge just arrived on). It mirrors
// the travel command's dead-end policy so callers outside that package can ask
// the same question — e.g. after a non-physical move lands the traveller in a
// new reality.
func IsPhysicalDeadEnd(u *Aggregate, id, cameFrom string) bool {
	for _, e := range u.EdgesFrom(id) {
		if e.Mode.IsPhysical() && e.To != cameFrom {
			return false
		}
	}
	return true
}

// HasPhysicalExit reports whether a location has at least one outgoing physical
// edge. A location with none is a genuine physical sink — it can only be left by
// a non-physical move (e.g. the well, whose sole exit is a consensus drift). Such
// a sink must not be auto-expanded into a nearby "ladder" on arrival, otherwise
// it becomes trivially escapable on foot and stops being a real dead end. Leaves
// that are still physically connected (an ordinary node whose only walkable edge
// is back the way you came, or an auto-generated nearby node) do have a physical
// exit, so they keep expanding as before.
func HasPhysicalExit(u *Aggregate, id string) bool {
	for _, e := range u.EdgesFrom(id) {
		if e.Mode.IsPhysical() {
			return true
		}
	}
	return false
}

// Escape-probability bounds. The chance that a dead end reached by a
// non-physical move offers a physical way out scales with the σ cost of that
// move: cheap transitions (an observer shift at 2 σ) rarely pay off, while
// expensive ones (a mathematical-structure jump at 50000 σ) almost always do —
// so the traveller gambles more σ for better odds of escaping. Costs span a
// wide range, so the cost→probability mapping is logarithmic (see edgeWeight in
// the web layer for the same reasoning), clamped to [EscapeCostMin, EscapeCostMax].
const (
	EscapeCostMin = ObserverShiftCost    // cheapest transition (2 σ) → EscapeProbMin
	EscapeCostMax = MathematicalShiftCost // dearest transition (50000 σ) → EscapeProbMax
	EscapeProbMin = 0.10
	EscapeProbMax = 0.90
)

// EscapeProbability maps a transition's σ cost to the probability that the dead
// end it lands on offers a physical escape, on a log curve between EscapeProbMin
// and EscapeProbMax.
func EscapeProbability(transitionCost float64) float64 {
	c := math.Min(EscapeCostMax, math.Max(EscapeCostMin, transitionCost))
	t := math.Log(c/EscapeCostMin) / math.Log(EscapeCostMax/EscapeCostMin)
	return EscapeProbMin + t*(EscapeProbMax-EscapeProbMin)
}

// HasPhysicalEscape decides whether a dead end in the reality identified by
// coord offers a physical way out (a "ladder"). The answer is derived
// deterministically from the coordinate's seed — the same reality always gives
// the same verdict, so it is reproducible across reloads and in tests — yet it
// varies from one reality to the next, so escapability feels random as the
// traveller moves between realities. The odds are set by transitionCost: the
// more σ the move that arrived here cost, the more likely the escape (see
// EscapeProbability). Base reality (nesting depth 0) is never gated: ordinary
// dead ends there always expand, preserving existing behaviour.
func HasPhysicalEscape(coord CoordinateVO, transitionCost float64) bool {
	if coord.NestingDepth() == 0 {
		return true
	}
	rng := rand.New(rand.NewSource(coordinateSeed(coord)))
	return rng.Float64() < EscapeProbability(transitionCost)
}

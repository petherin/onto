package universe

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// nearbyNamePrefix is the display-name prefix shared by every auto-generated
// nearby location. It is also the signal the location validator uses to skip
// these nodes, so it must not change casually.
const nearbyNamePrefix = "Nearby "

// LocationGeneratorService is a domain service interface for the policy that
// expands a dead end into a new nearby location and its bidirectional
// physical connections. Depending on this abstraction — rather than the
// concrete SequentialLocationGenerator — lets callers substitute a different
// nearby-location policy without changing the command that uses it.
type LocationGeneratorService interface {
	Generate(u *Aggregate, originID string, coordinate CoordinateVO) (LocationEntity, EdgeVO, EdgeVO, error)
}

// SequentialLocationGenerator is the domain's standard nearby-location policy:
// it numbers new locations sequentially off the origin ID (e.g. "home-1",
// "home-2", ...).
type SequentialLocationGenerator struct{}

// NewSequentialLocationGenerator returns the standard, sequential-numbering
// nearby-location generator.
func NewSequentialLocationGenerator() *SequentialLocationGenerator {
	return &SequentialLocationGenerator{}
}

// Generate implements LocationGeneratorService using the sequential-numbering policy.
func (SequentialLocationGenerator) Generate(u *Aggregate, originID string, coordinate CoordinateVO) (LocationEntity, EdgeVO, EdgeVO, error) {
	return NewNearbyLocation(u, originID, coordinate)
}

// NewNearbyLocation returns the next available nearby location and its
// bidirectional physical connections. It contains the domain policy for
// expanding a dead end without performing any persistence or user interaction.
func NewNearbyLocation(u *Aggregate, originID string, coordinate CoordinateVO) (LocationEntity, EdgeVO, EdgeVO, error) {
	if _, exists := u.GetLocation(originID); !exists {
		return LocationEntity{}, EdgeVO{}, EdgeVO{}, fmt.Errorf("%w: %s", ErrUnknownEdgeEndpoint, originID)
	}
	// Number the nearby index onto the origin's stable base and reassemble
	// canonically, so a location spawned inside a reality branch keeps its
	// axis suffixes in canonical order (e.g. "park-1-u1", not "park-u1-1").
	// A bare "-i" appended after an axis suffix would make the ID's encoded
	// axes disagree with its coordinate, breaking LowerContextID and hence
	// every *back / return-home step for that axis.
	base, ax := parseLocationID(originID)
	for i := 1; i < 1000; i++ {
		id := buildLocationID(fmt.Sprintf("%s-%d", base, i), ax)
		if _, exists := u.GetLocation(id); exists {
			continue
		}
		// The display name is numbered from a universe-wide count of existing
		// nearby locations, not the per-origin index i. A dead end spawns its
		// first nearby node with i == 1, so numbering by i would name every
		// dead end's child "Nearby 1" — producing many distinct nodes with
		// identical names as the user chains through them.
		coordinate.Location = fmt.Sprintf("%s%d", nearbyNamePrefix, nextNearbyNumber(u))
		location := LocationEntity{
			ID:          id,
			Name:        coordinate.Location,
			Description: GenerateDescription(coordinate),
			Coordinate:  coordinate,
		}
		outbound := EdgeVO{From: originID, To: id, Mode: Walk, Distance: 1, Cost: 1, Description: "Auto-generated path"}
		returning := EdgeVO{From: id, To: originID, Mode: Walk, Distance: 1, Cost: 1, Description: "Auto-generated return path"}
		return location, outbound, returning, nil
	}
	return LocationEntity{}, EdgeVO{}, EdgeVO{}, fmt.Errorf("%w: no nearby location ID available", ErrInvalidLocation)
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

// nextNearbyNumber returns one past the highest "Nearby N" sequence number
// currently present in the universe, so each auto-generated nearby location
// gets a name unique across all dead ends rather than one reset per origin.
func nextNearbyNumber(u *Aggregate) int {
	highest := 0
	for _, loc := range u.AllLocations() {
		if !strings.HasPrefix(loc.Coordinate.Location, nearbyNamePrefix) {
			continue
		}
		suffix := strings.TrimSpace(strings.TrimPrefix(loc.Coordinate.Location, nearbyNamePrefix))
		if n, err := strconv.Atoi(suffix); err == nil && n > highest {
			highest = n
		}
	}
	return highest + 1
}

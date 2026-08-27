package universe

import (
	"fmt"
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

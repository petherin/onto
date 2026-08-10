package universe

import "fmt"

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
	for i := 1; i < 1000; i++ {
		id := fmt.Sprintf("%s-%d", originID, i)
		if _, exists := u.GetLocation(id); exists {
			continue
		}
		coordinate.Location = fmt.Sprintf("Nearby %d", i)
		location := LocationEntity{
			ID:          id,
			Name:        coordinate.Location,
			Description: "Auto-generated nearby location",
			Coordinate:  coordinate,
		}
		outbound := EdgeVO{From: originID, To: id, Mode: Walk, Distance: 1, Cost: 1, Description: "Auto-generated path"}
		returning := EdgeVO{From: id, To: originID, Mode: Walk, Distance: 1, Cost: 1, Description: "Auto-generated return path"}
		return location, outbound, returning, nil
	}
	return LocationEntity{}, EdgeVO{}, EdgeVO{}, fmt.Errorf("%w: no nearby location ID available", ErrInvalidLocation)
}

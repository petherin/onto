package universe

import "fmt"

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

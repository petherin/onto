package commands

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/universe"
)

// GenerateNearbyLocationCommand expands a dead end according to the domain's
// nearby-location policy.
type GenerateNearbyLocationCommand struct {
	Universe  *universe.Aggregate
	Generator universe.LocationGeneratorService
	OriginID  string
}

// Execute creates the nearby-location cluster for the dead end. It commits every
// generated location first, then every edge, so both endpoints of each edge
// already exist when it is added. It returns the created locations.
func (c *GenerateNearbyLocationCommand) Execute() ([]universe.LocationEntity, error) {
	origin, ok := c.Universe.GetLocation(c.OriginID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", universe.ErrUnknownEdgeEndpoint, c.OriginID)
	}
	locations, edges, err := c.Generator.Generate(c.Universe, c.OriginID, origin.Coordinate)
	if err != nil {
		return nil, err
	}
	for _, location := range locations {
		if err := c.Universe.AddLocation(location); err != nil {
			return nil, err
		}
	}
	for _, edge := range edges {
		if err := c.Universe.AddEdge(edge); err != nil {
			return nil, err
		}
	}
	return locations, nil
}

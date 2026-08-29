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

// Execute creates the next nearby location.
func (c *GenerateNearbyLocationCommand) Execute() (universe.LocationEntity, error) {
	origin, ok := c.Universe.GetLocation(c.OriginID)
	if !ok {
		return universe.LocationEntity{}, fmt.Errorf("%w: %s", universe.ErrUnknownEdgeEndpoint, c.OriginID)
	}
	location, outbound, returning, err := c.Generator.Generate(c.Universe, c.OriginID, origin.Coordinate)
	if err != nil {
		return universe.LocationEntity{}, err
	}
	if err := c.Universe.AddLocation(location); err != nil {
		return universe.LocationEntity{}, err
	}
	if err := c.Universe.AddEdge(outbound); err != nil {
		return universe.LocationEntity{}, err
	}
	if err := c.Universe.AddEdge(returning); err != nil {
		return universe.LocationEntity{}, err
	}
	return location, nil
}

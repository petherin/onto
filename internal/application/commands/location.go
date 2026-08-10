package commands

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/universe"
)

// CreateLocationCommand adds a location and its bidirectional walking
// connection to an existing location.
type CreateLocationCommand struct {
	Universe *universe.Aggregate
	Repo     universe.Repository
	OriginID string
	Location universe.LocationEntity
	Distance float64
	Cost     float64
}

// Execute applies the graph change atomically in the aggregate and persists it.
func (c *CreateLocationCommand) Execute() error {
	if _, exists := c.Universe.GetLocation(c.OriginID); !exists {
		return fmt.Errorf("%w: %s", universe.ErrUnknownEdgeEndpoint, c.OriginID)
	}
	if err := c.Universe.AddLocation(c.Location); err != nil {
		return err
	}
	outbound := universe.EdgeVO{From: c.OriginID, To: c.Location.ID, Mode: universe.Walk, Distance: c.Distance, Cost: c.Cost, Description: "User-created path"}
	if err := c.Universe.AddEdge(outbound); err != nil {
		return err
	}
	returning := universe.EdgeVO{From: c.Location.ID, To: c.OriginID, Mode: universe.Walk, Distance: c.Distance, Cost: c.Cost, Description: "User-created return path"}
	if err := c.Universe.AddEdge(returning); err != nil {
		return err
	}
	return c.Repo.Save(c.Universe)
}

// GenerateNearbyLocationCommand expands a dead end according to the domain's
// nearby-location policy.
type GenerateNearbyLocationCommand struct {
	Universe  *universe.Aggregate
	Repo      universe.Repository
	Generator universe.LocationGeneratorService
	OriginID  string
}

// Execute creates and persists the next nearby location.
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
	if err := c.Repo.Save(c.Universe); err != nil {
		return universe.LocationEntity{}, err
	}
	return location, nil
}

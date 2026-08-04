// Package commands contains the write-side use cases (CQRS commands) that
// mutate session and universe state. Each command validates its inputs, applies
// domain logic, persists the result through the Repository interface, and
// returns a result struct for the interface layer to render.
package commands

import (
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// TravelResult is the value returned by a successful TravelCommand execution.
type TravelResult struct {
	Location       universe.LocationEntity
	Path           []universe.EdgeVO // edges traversed to reach the destination
	Edges          []universe.EdgeVO
	History        []string
	DeadEndHandled bool
}

// TravelCommand moves the session to a physical destination. It rejects paths
// that cross non-physical boundaries (quantum, timeline) and optionally
// invokes a LocationGeneratorService when the destination is a dead end.
type TravelCommand struct {
	Universe       *universe.Aggregate
	Session        *exploration.Entity
	Repo           universe.Repository
	Pathfinder     navigation.PathfinderService
	DeadEndHandler universe.LocationGeneratorService
}

// Execute validates the target, finds a physical-only route, moves the session,
// handles dead ends, and persists any newly generated locations.
func (c *TravelCommand) Execute(target string) (*TravelResult, error) {
	norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
	if _, ok := c.Universe.GetLocation(norm); !ok {
		return nil, fmt.Errorf("%w: %s", navigation.ErrUnknownDestination, target)
	}

	path, ok := c.Pathfinder.FindRoute(c.Universe, c.Session.Location(), norm)
	if !ok {
		return nil, fmt.Errorf("%w: %s", navigation.ErrNoRoute, target)
	}
	for _, e := range path {
		if !e.Mode.IsPhysical() {
			return nil, fmt.Errorf("no physical route to %s — quantum boundaries cannot be crossed on foot", target)
		}
	}

	loc, _ := c.Universe.GetLocation(norm)
	previous := c.Session.Location()
	var pathCost float64
	for _, e := range path {
		pathCost += e.Cost
	}
	c.Session.MoveTo(loc, pathCost)

	deadEndHandled := false
	if c.DeadEndHandler != nil {
		deadEndHandled = ensureOutgoing(c.Universe, norm, previous, c.DeadEndHandler)
	}

	result := &TravelResult{
		Location:       loc,
		Path:           path,
		Edges:          c.Universe.EdgesFrom(norm),
		History:        c.Session.History(),
		DeadEndHandled: deadEndHandled,
	}

	if deadEndHandled {
		if err := c.Repo.Save(c.Universe); err != nil {
			// Return the result (travel succeeded) alongside the save error so
			// callers can display output and warn about the persistence failure.
			return result, err
		}
	}

	return result, nil
}

// ensureOutgoing returns true if the location is a dead end and the handler created new edges.
func ensureOutgoing(u *universe.Aggregate, id, cameFrom string, handler universe.LocationGeneratorService) bool {
	for _, e := range u.EdgesFrom(id) {
		if e.To != cameFrom {
			return false
		}
	}
	loc, _ := u.GetLocation(id)
	return handler.Handle(u, id, loc.Coordinate)
}

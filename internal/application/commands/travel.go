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
// that cross non-physical or reality boundaries.
type TravelCommand struct {
	Universe   *universe.Aggregate
	Session    *exploration.Entity
	Repo       universe.Repository
	Pathfinder navigation.PathfinderService
}

// Execute validates the target, finds a physical-only route, moves the session,
// and reports whether the destination is a dead end.
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
			return nil, fmt.Errorf("no physical route to %s — reality-transition boundaries cannot be crossed on foot", target)
		}
		from, fromOK := c.Universe.GetLocation(e.From)
		to, toOK := c.Universe.GetLocation(e.To)
		if !fromOK || !toOK || !from.Coordinate.SamePhysicalReality(to.Coordinate) {
			return nil, fmt.Errorf("no physical route to %s — normal travel cannot cross reality boundaries", target)
		}
	}

	loc, _ := c.Universe.GetLocation(norm)
	previous := c.Session.Location()
	var pathCost float64
	for _, e := range path {
		pathCost += e.Cost
	}
	c.Session.MoveTo(loc, pathCost)

	result := &TravelResult{
		Location:       loc,
		Path:           path,
		Edges:          c.Universe.EdgesFrom(norm),
		History:        c.Session.History(),
		DeadEndHandled: isDeadEnd(c.Universe, norm, previous),
	}
	return result, nil
}

// isDeadEnd reports whether a location has no physical onward route.
func isDeadEnd(u *universe.Aggregate, id, cameFrom string) bool {
	for _, e := range u.EdgesFrom(id) {
		if e.Mode.IsPhysical() && e.To != cameFrom {
			return false
		}
	}
	return true
}

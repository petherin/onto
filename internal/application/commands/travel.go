// Package commands contains the write-side use cases (CQRS commands) that
// mutate session and universe state. Each command validates its inputs,
// applies domain logic, and returns a result struct for the interface layer
// to render.
package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// ErrInsufficientBudget is returned by a move command when the session's
// remaining budget cannot cover the move's cost. The move is not applied.
var ErrInsufficientBudget = errors.New("not enough budget")

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
	Pathfinder navigation.PathfinderService
	// IgnoreBudget bypasses the affordability gate. Returning home must always
	// succeed even when the remaining budget cannot cover the walk home, so the
	// return-home workflow sets this; ordinary travel leaves it false.
	IgnoreBudget bool
}

// Execute validates the target, finds a physical-only route, moves the session,
// and reports whether the destination is a dead end.
func (c *TravelCommand) Execute(target string) (*TravelResult, error) {
	norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
	// Resolve the destination within the session's current reality first, so a
	// plain name typed while nested ("park") means the copy of that place in
	// this reality, not the base-reality original. Fall back to a literal ID
	// that exists in another reality so the pathfinder can still reject it with
	// the informative "no physical route" message rather than "unknown".
	if loc, ok := c.Universe.FindInReality(c.Session.Coordinate(), target); ok {
		norm = loc.ID
	} else if _, ok := c.Universe.GetLocation(norm); !ok {
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
	if !c.IgnoreBudget && !c.Session.CanAfford(pathCost) {
		return nil, fmt.Errorf("%w to travel to %s — it costs %.0f but only %.0f remains",
			ErrInsufficientBudget, target, pathCost, c.Session.RemainingBudget())
	}
	c.Session.MoveTo(loc, pathCost)

	// The dead-end test must ignore the edge we just arrived on, so it needs the
	// *immediate* predecessor — the source of the last edge — not the journey's
	// origin. For a multi-hop walk (e.g. clicking a far node on the map) previous
	// is several nodes back, so using it would let the destination's edge home to
	// its true neighbour masquerade as an onward route and suppress expansion.
	cameFrom := previous
	if len(path) > 0 {
		cameFrom = path[len(path)-1].From
	}
	result := &TravelResult{
		Location:       loc,
		Path:           path,
		Edges:          c.Universe.EdgesFrom(norm),
		History:        c.Session.History(),
		DeadEndHandled: isDeadEnd(c.Universe, norm, cameFrom),
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

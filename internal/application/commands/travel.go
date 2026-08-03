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

const (
	errFmtUnknownDest = "unknown destination: %s"
	errFmtNoRoute     = "no route: %s"
)

// TravelResult is the value returned by a successful TravelCommand execution.
type TravelResult struct {
	Location       universe.Location
	Path           []universe.Edge // edges traversed to reach the destination
	Edges          []universe.Edge
	History        []string
	DeadEndHandled bool
	Persisted      bool
	SaveErr        error
}

// TravelCommand moves the session to a physical destination. It rejects paths
// that cross non-physical boundaries (quantum, timeline) and optionally
// invokes a LocationGenerator when the destination is a dead end.
type TravelCommand struct {
	Universe       *universe.Universe
	Session        *exploration.Session
	Repo           universe.Repository
	Pathfinder     navigation.Pathfinder
	DeadEndHandler universe.LocationGenerator
}

// Execute validates the target, finds a physical-only route, moves the session,
// handles dead ends, and persists any newly generated locations.
func (c *TravelCommand) Execute(target string) (*TravelResult, error) {
	norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
	if _, ok := c.Universe.GetLocation(norm); !ok {
		return nil, fmt.Errorf(errFmtUnknownDest, target)
	}

	path, ok := c.Pathfinder.FindRoute(c.Universe, c.Session.CurrentLocation, norm)
	if !ok {
		return nil, fmt.Errorf(errFmtNoRoute, target)
	}
	for _, e := range path {
		if !e.Mode.IsPhysical() {
			return nil, fmt.Errorf("no physical route to %s — quantum boundaries cannot be crossed on foot", target)
		}
	}

	loc, _ := c.Universe.GetLocation(norm)
	previous := c.Session.CurrentLocation
	c.Session.MoveTo(loc)

	deadEndHandled := false
	if c.DeadEndHandler != nil {
		deadEndHandled = ensureOutgoing(c.Universe, norm, previous, c.DeadEndHandler)
	}

	result := &TravelResult{
		Location:       loc,
		Path:           path,
		Edges:          c.Universe.EdgesFrom(norm),
		History:        c.Session.TravelHistory,
		DeadEndHandled: deadEndHandled,
	}

	if deadEndHandled {
		if err := c.Repo.Save(c.Universe); err != nil {
			result.SaveErr = err
		} else {
			result.Persisted = true
		}
	}

	return result, nil
}

// ensureOutgoing returns true if the location is a dead end and the handler created new edges.
func ensureOutgoing(u *universe.Universe, id, cameFrom string, handler universe.LocationGenerator) bool {
	for _, e := range u.EdgesFrom(id) {
		if e.To != cameFrom {
			return false
		}
	}
	loc, _ := u.GetLocation(id)
	return handler.Handle(u, id, loc.Coordinate)
}

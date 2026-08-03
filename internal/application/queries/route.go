package queries

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

// RouteResult holds the data returned by a Route query.
type RouteResult struct {
	Steps    []universe.Edge
	Distance float64
	Cost     float64
}

// RouteQuery plans a route from the current session position to a target.
type RouteQuery struct {
	Universe   *universe.Universe
	Session    *exploration.Session
	Pathfinder navigation.Pathfinder
}

// Execute plans a route from the current session position to target, returning
// the ordered edge list, total distance, and total cost.
func (q *RouteQuery) Execute(target string) (*RouteResult, error) {
	norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
	if _, ok := q.Universe.GetLocation(norm); !ok {
		return nil, fmt.Errorf(errFmtUnknownDest, target)
	}

	path, ok := q.Pathfinder.FindRoute(q.Universe, q.Session.CurrentLocation, norm)
	if !ok {
		return nil, fmt.Errorf(errFmtNoRoute, target)
	}

	return &RouteResult{
		Steps:    path,
		Distance: navigation.PathDistance(path),
		Cost:     navigation.PathCost(path),
	}, nil
}

// Package queries contains the read-side use cases (CQRS queries) that return
// information about the current session and universe without mutating any
// state. Queries are safe to call at any time and have no side effects.
package queries

import (
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// WhereResult holds the data returned by a Where query.
type WhereResult struct {
	Coordinate  universe.CoordinateVO
	Edges       []universe.EdgeVO
	NextQuantum string
	History     []string
}

// LookResult holds the data returned by a Look query.
type LookResult struct {
	Name        string
	Description string
}

// ListResult holds the data returned by a List query.
type ListResult struct {
	Edges       []universe.EdgeVO
	NextQuantum string
}

// LookupQuery handles Where, Look, and List reads against the universe and session.
type LookupQuery struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
}

// Where returns the current reality coordinate, outgoing edges, and travel history.
func (q *LookupQuery) Where() *WhereResult {
	return &WhereResult{
		Coordinate:  q.Session.CurrentCoordinate,
		Edges:       q.Universe.EdgesFrom(q.Session.CurrentLocation),
		NextQuantum: q.Session.NextQuantumID(),
		History:     q.Session.TravelHistory,
	}
}

// Look returns the name and description of the current location, or false if
// the location cannot be found in the universe.
func (q *LookupQuery) Look() (*LookResult, bool) {
	loc, ok := q.Universe.GetLocation(q.Session.CurrentLocation)
	if !ok {
		return nil, false
	}
	return &LookResult{Name: loc.Name, Description: loc.Description}, true
}

// List returns the outgoing edges from the current location.
func (q *LookupQuery) List() *ListResult {
	return &ListResult{
		Edges:       q.Universe.EdgesFrom(q.Session.CurrentLocation),
		NextQuantum: q.Session.NextQuantumID(),
	}
}

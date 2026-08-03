package queries

import (
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// WhereResult holds the data returned by a Where query.
type WhereResult struct {
	Coordinate  universe.Coordinate
	Edges       []universe.Edge
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
	Edges       []universe.Edge
	NextQuantum string
}

// LookupQuery handles Where, Look, and List reads against the universe and session.
type LookupQuery struct {
	Universe *universe.Universe
	Session  *exploration.Session
}

func (q *LookupQuery) Where() *WhereResult {
	return &WhereResult{
		Coordinate:  q.Session.CurrentCoordinate,
		Edges:       q.Universe.EdgesFrom(q.Session.CurrentLocation),
		NextQuantum: q.Session.NextQuantumID(),
		History:     q.Session.TravelHistory,
	}
}

func (q *LookupQuery) Look() (*LookResult, bool) {
	loc, ok := q.Universe.GetLocation(q.Session.CurrentLocation)
	if !ok {
		return nil, false
	}
	return &LookResult{Name: loc.Name, Description: loc.Description}, true
}

func (q *LookupQuery) List() *ListResult {
	return &ListResult{
		Edges:       q.Universe.EdgesFrom(q.Session.CurrentLocation),
		NextQuantum: q.Session.NextQuantumID(),
	}
}

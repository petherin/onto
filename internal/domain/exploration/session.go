// Package exploration tracks the user's position within a universe. The
// Entity records the current location, coordinate, and travel history,
// and exposes the movement methods (MoveTo, ShiftTo, JumpTo) that
// commands call after a successful route or branch transition.
package exploration

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/universe"
)

// Entity is a domain entity that tracks the user's position within
// the universe across physical travel, quantum shifts, and timeline jumps.
type Entity struct {
	CurrentLocation   string
	CurrentCoordinate universe.CoordinateVO
	TravelHistory     []string
	CumulativeCost    float64
}

// NewEntity creates an Entity positioned at the given location and coordinate.
func NewEntity(location string, coord universe.CoordinateVO) *Entity {
	return &Entity{
		CurrentLocation:   location,
		CurrentCoordinate: coord,
		TravelHistory:     []string{},
	}
}

// MoveTo updates position, adds cost to the cumulative total, and records the move in travel history.
func (s *Entity) MoveTo(loc universe.LocationEntity, cost float64) {
	prev := s.CurrentLocation
	s.CurrentLocation = loc.ID
	s.CurrentCoordinate = loc.Coordinate
	s.CumulativeCost += cost
	s.TravelHistory = append(s.TravelHistory, fmt.Sprintf("%s -> %s", prev, loc.ID))
}

// ShiftTo updates position for a quantum shift, adds cost, and records it in travel history.
func (s *Entity) ShiftTo(loc universe.LocationEntity, cost float64) {
	prev := s.CurrentLocation
	s.CurrentLocation = loc.ID
	s.CurrentCoordinate = loc.Coordinate
	s.CumulativeCost += cost
	s.TravelHistory = append(s.TravelHistory, fmt.Sprintf("%s -> %s (quantum shift)", prev, loc.ID))
}

// QuantumLevel returns the numeric quantum level of the current position (Q0 → 0, Q1 → 1, …).
func (s *Entity) QuantumLevel() int {
	return s.CurrentCoordinate.QuantumLevel()
}

// NextQuantumID returns the location ID that 'shift' would move to from the current position.
func (s *Entity) NextQuantumID() string {
	return fmt.Sprintf("%s-q%d", s.CurrentLocation, s.QuantumLevel()+1)
}

// JumpTo updates position for a timeline shift, adds cost, and records it in travel history.
func (s *Entity) JumpTo(loc universe.LocationEntity, cost float64) {
	prev := s.CurrentLocation
	s.CurrentLocation = loc.ID
	s.CurrentCoordinate = loc.Coordinate
	s.CumulativeCost += cost
	s.TravelHistory = append(s.TravelHistory, fmt.Sprintf("%s -> %s (timeline shift)", prev, loc.ID))
}

// TimelineLevel returns the numeric timeline level of the current position ("Prime" → 0, "T1" → 1, …).
func (s *Entity) TimelineLevel() int {
	return s.CurrentCoordinate.TimelineLevel()
}

// NextTimelineID returns the location ID that 'jump' would move to from the current position.
func (s *Entity) NextTimelineID() string {
	return fmt.Sprintf("%s-t%d", s.CurrentLocation, s.TimelineLevel()+1)
}

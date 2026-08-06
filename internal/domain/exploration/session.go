// Package exploration tracks the user's position within a universe. The
// Entity records the current location, coordinate, and travel history,
// and exposes the movement methods (MoveTo, ShiftTo, JumpTo) that
// commands call after a successful route or branch transition.
package exploration

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/universe"
)

// Entity is a domain entity that tracks the user's position within the universe
// across physical travel and reality transitions.
// Fields are unexported to ensure all mutations go through the movement
// methods (MoveTo, ShiftTo, JumpTo), keeping history and cost consistent.
type Entity struct {
	currentLocation   string
	currentCoordinate universe.CoordinateVO
	travelHistory     []string
	cumulativeCost    float64
}

// NewEntity creates an Entity positioned at the given location and coordinate.
func NewEntity(location string, coord universe.CoordinateVO) *Entity {
	return &Entity{
		currentLocation:   location,
		currentCoordinate: coord,
		travelHistory:     []string{},
	}
}

// Location returns the ID of the current location.
func (s *Entity) Location() string { return s.currentLocation }

// Coordinate returns the coordinate of the current position.
func (s *Entity) Coordinate() universe.CoordinateVO { return s.currentCoordinate }

// History returns a snapshot of the travel history slice.
func (s *Entity) History() []string {
	out := make([]string, len(s.travelHistory))
	copy(out, s.travelHistory)
	return out
}

// CumulativeCost returns the total cost accumulated across all movements.
func (s *Entity) CumulativeCost() float64 { return s.cumulativeCost }

// MoveTo updates position, adds cost to the cumulative total, and records the move in travel history.
func (s *Entity) MoveTo(loc universe.LocationEntity, cost float64) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	s.travelHistory = append(s.travelHistory, fmt.Sprintf("%s -> %s", prev, loc.ID))
}

// ShiftTo updates position for a quantum shift, adds cost, and records it in travel history.
func (s *Entity) ShiftTo(loc universe.LocationEntity, cost float64) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	s.travelHistory = append(s.travelHistory, fmt.Sprintf("%s -> %s (quantum shift)", prev, loc.ID))
}

// JumpTo updates position for a timeline shift, adds cost, and records it in travel history.
func (s *Entity) JumpTo(loc universe.LocationEntity, cost float64) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	s.travelHistory = append(s.travelHistory, fmt.Sprintf("%s -> %s (timeline shift)", prev, loc.ID))
}

// DriftTo updates position for a consensus divergence transition, adds cost,
// and records it in travel history.
func (s *Entity) DriftTo(loc universe.LocationEntity, cost float64) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	s.travelHistory = append(s.travelHistory, fmt.Sprintf("%s -> %s (consensus drift)", prev, loc.ID))
}

// ObserveTo updates position for an observer shift, adds cost, and records it
// in travel history.
func (s *Entity) ObserveTo(loc universe.LocationEntity, cost float64) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	s.travelHistory = append(s.travelHistory, fmt.Sprintf("%s -> %s (observer shift)", prev, loc.ID))
}

// QuantumLevel returns the numeric quantum level of the current position (Q0 → 0, Q1 → 1, …).
func (s *Entity) QuantumLevel() int {
	return s.currentCoordinate.QuantumLevel()
}

// NextQuantumID returns the location ID that 'shift' would move to from the current position.
func (s *Entity) NextQuantumID() string {
	return fmt.Sprintf("%s-q%d", s.currentLocation, s.QuantumLevel()+1)
}

// TimelineLevel returns the numeric timeline level of the current position ("Prime" → 0, "T1" → 1, …).
func (s *Entity) TimelineLevel() int {
	return s.currentCoordinate.TimelineLevel()
}

// NextTimelineID returns the location ID that 'jump' would move to from the current position.
func (s *Entity) NextTimelineID() string {
	return fmt.Sprintf("%s-t%d", s.currentLocation, s.TimelineLevel()+1)
}

// ConsensusLevel returns the current depth of divergence from shared consensus.
func (s *Entity) ConsensusLevel() int {
	return s.currentCoordinate.Consensus
}

// NextConsensusID returns the location ID that 'drift' would move to from the
// current position.
func (s *Entity) NextConsensusID() string {
	return fmt.Sprintf("%s-c%d", s.currentLocation, s.ConsensusLevel()+1)
}

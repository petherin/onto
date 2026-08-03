package exploration

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/universe"
)

// Session tracks the user's position within the universe.
type Session struct {
	CurrentLocation   string
	CurrentCoordinate universe.Coordinate
	TravelHistory     []string
}

func NewSession(location string, coord universe.Coordinate) *Session {
	return &Session{
		CurrentLocation:   location,
		CurrentCoordinate: coord,
		TravelHistory:     []string{},
	}
}

// MoveTo updates position and records the move in travel history.
func (s *Session) MoveTo(loc universe.Location) {
	prev := s.CurrentLocation
	s.CurrentLocation = loc.ID
	s.CurrentCoordinate = loc.Coordinate
	s.TravelHistory = append(s.TravelHistory, fmt.Sprintf("%s -> %s", prev, loc.ID))
}

// ShiftTo updates position for a quantum shift and records it in travel history.
func (s *Session) ShiftTo(loc universe.Location) {
	prev := s.CurrentLocation
	s.CurrentLocation = loc.ID
	s.CurrentCoordinate = loc.Coordinate
	s.TravelHistory = append(s.TravelHistory, fmt.Sprintf("%s -> %s (quantum shift)", prev, loc.ID))
}

// QuantumLevel returns the numeric quantum level of the current position (Q0 → 0, Q1 → 1, …).
func (s *Session) QuantumLevel() int {
	return s.CurrentCoordinate.QuantumLevel()
}

// NextQuantumID returns the location ID that 'shift' would jump to from the current position.
func (s *Session) NextQuantumID() string {
	return fmt.Sprintf("%s-q%d", s.CurrentLocation, s.QuantumLevel()+1)
}

// JumpTo updates position for a timeline shift and records it in travel history.
func (s *Session) JumpTo(loc universe.Location) {
	prev := s.CurrentLocation
	s.CurrentLocation = loc.ID
	s.CurrentCoordinate = loc.Coordinate
	s.TravelHistory = append(s.TravelHistory, fmt.Sprintf("%s -> %s (timeline shift)", prev, loc.ID))
}

// TimelineLevel returns the numeric timeline level of the current position ("Prime" → 0, "T1" → 1, …).
func (s *Session) TimelineLevel() int {
	return s.CurrentCoordinate.TimelineLevel()
}

// NextTimelineID returns the location ID that 'jump' would shift to from the current position.
func (s *Session) NextTimelineID() string {
	return fmt.Sprintf("%s-t%d", s.CurrentLocation, s.TimelineLevel()+1)
}

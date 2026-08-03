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

// NextQuantumID returns the location ID that 'shift' would jump to from the current position.
func (s *Session) NextQuantumID() string {
	n := 0
	q := s.CurrentCoordinate.Quantum
	if len(q) > 1 && q[0] == 'Q' {
		fmt.Sscanf(q[1:], "%d", &n)
	}
	return fmt.Sprintf("%s-q%d", s.CurrentLocation, n+1)
}

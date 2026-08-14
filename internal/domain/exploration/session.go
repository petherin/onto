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
	contextStack      []ContextTransition
}

// ContextTransition records one entered contextual branch so it can be
// unwound in the exact reverse order.
type ContextTransition struct {
	Mode     universe.TravelModeVO
	OriginID string
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

// ContextTransitions returns a snapshot of entered contextual branches.
func (s *Entity) ContextTransitions() []ContextTransition {
	out := make([]ContextTransition, len(s.contextStack))
	copy(out, s.contextStack)
	return out
}

// MoveTo updates position, adds cost to the cumulative total, and records the move in travel history.
func (s *Entity) MoveTo(loc universe.LocationEntity, cost float64) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	s.travelHistory = append(s.travelHistory, fmt.Sprintf("%s -> %s", prev, loc.ID))
}

// TransitionTo applies a contextual movement and records or removes its
// ancestry entry. Reversed transitions only pop the matching latest entry.
func (s *Entity) TransitionTo(loc universe.LocationEntity, cost float64, mode universe.TravelModeVO, reversed bool) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	label := string(mode) + " shift"
	switch mode {
	case universe.ConsensusShift:
		label = "consensus drift"
	case universe.MathematicalShift:
		label = "mathematical structure shift"
	case universe.SimulationEntry:
		if reversed {
			label = "simulation exit"
		} else {
			label = "simulation entry"
		}
	}
	s.travelHistory = append(s.travelHistory, fmt.Sprintf("%s -> %s (%s)", prev, loc.ID, label))
	if reversed {
		last := len(s.contextStack) - 1
		if last >= 0 && s.contextStack[last].Mode == mode {
			s.contextStack = s.contextStack[:last]
		}
		return
	}
	s.contextStack = append(s.contextStack, ContextTransition{Mode: mode, OriginID: prev})
}

// QuantumLevel returns the numeric quantum level of the current position (Q0 → 0, Q1 → 1, …).
func (s *Entity) QuantumLevel() int {
	return s.currentCoordinate.QuantumLevel()
}

// UniverseLevel returns the numeric bubble-universe level of the current
// position (Origin → 0, U1 → 1, U2 → 2, …).
func (s *Entity) UniverseLevel() int {
	return s.currentCoordinate.UniverseLevel()
}

// NextUniverseID returns the location ID that 'universe' would move to from
// the current position, canonicalized (see NextQuantumID).
func (s *Entity) NextUniverseID() string {
	return universe.CanonicalIDWithUniverse(s.currentLocation, s.UniverseLevel()+1)
}

// MathematicsLevel returns the numeric mathematical-structure level of the
// current position (Classical → 0, M1 → 1, M2 → 2, …).
func (s *Entity) MathematicsLevel() int {
	return s.currentCoordinate.MathematicsLevel()
}

// NextMathematicsID returns the location ID that 'structure' would move to
// from the current position, canonicalized (see NextQuantumID).
func (s *Entity) NextMathematicsID() string {
	return universe.CanonicalIDWithMathematics(s.currentLocation, s.MathematicsLevel()+1)
}

// NextQuantumID returns the location ID that 'shift' would move to from the
// current position. IDs are canonicalized so reaching the same coordinate via
// a different order of shifts (e.g. shift-then-drift vs. drift-then-shift)
// always produces the same location ID.
func (s *Entity) NextQuantumID() string {
	return universe.CanonicalIDWithQuantum(s.currentLocation, s.QuantumLevel()+1)
}

// TimelineLevel returns the numeric timeline level of the current position ("Prime" → 0, "T1" → 1, …).
func (s *Entity) TimelineLevel() int {
	return s.currentCoordinate.TimelineLevel()
}

// NextTimelineID returns the location ID that 'jump' would move to from the
// current position, canonicalized (see NextQuantumID).
func (s *Entity) NextTimelineID() string {
	return universe.CanonicalIDWithTimeline(s.currentLocation, s.TimelineLevel()+1)
}

// ConsensusLevel returns the current depth of divergence from shared consensus.
func (s *Entity) ConsensusLevel() int {
	return s.currentCoordinate.Consensus
}

// NextConsensusID returns the location ID that 'drift' would move to from the
// current position, canonicalized (see NextQuantumID).
func (s *Entity) NextConsensusID() string {
	return universe.CanonicalIDWithConsensus(s.currentLocation, s.ConsensusLevel()+1)
}

// SimulationLevel returns the current nested-simulation depth (0 = base reality).
func (s *Entity) SimulationLevel() int {
	return s.currentCoordinate.Simulation
}

// NextSimulationID returns the location ID that 'simulate' would move to from
// the current position, canonicalized (see NextQuantumID).
func (s *Entity) NextSimulationID() string {
	return universe.CanonicalIDWithSimulation(s.currentLocation, s.SimulationLevel()+1)
}

// NextObserverID returns the location ID that 'observe' would move to from
// the current position for the given observer, canonicalized (see
// NextQuantumID). Unlike the numeric axes, observer suffixes are always
// appended outermost/sequentially since perspective-nesting order is
// semantically meaningful.
func (s *Entity) NextObserverID(observerToken string) string {
	return universe.CanonicalIDWithObserver(s.currentLocation, observerToken)
}

// NextTimeID returns the location ID that 'time' would move to from the
// current position for the given timestamp token, canonicalized (see
// NextQuantumID).
func (s *Entity) NextTimeID(timeToken string) string {
	return universe.CanonicalIDWithTime(s.currentLocation, timeToken)
}

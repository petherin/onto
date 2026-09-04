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
	startLocation     string
	travelHistory     []string
	cumulativeCost    float64
	contextStack      []ContextTransition

	// Game state. budget is the spending pool: a value of 0 means unlimited
	// (no budget in force). targets is the ordered quest chain of objective
	// coordinates; an empty chain means no objective. Each objective is a round
	// trip: it is only complete once its waypoint has been reached AND the
	// traveller has since returned to the start location. objectiveIndex counts
	// how many objectives have been completed that way, in order, and also indexes
	// the current objective still to do. reachedCurrent records that the current
	// objective's waypoint has been reached this trip and only the return home
	// remains. won records that every objective has been completed.
	budget         float64
	targets        []universe.CoordinateVO
	objectiveIndex int
	reachedCurrent bool
	won            bool
}

// ContextTransition records one entered contextual branch so it can be
// unwound later. There is exactly one entry per outstanding forward
// transition on a given axis, so the number of entries for a mode always
// equals that axis's current level.
type ContextTransition struct {
	Mode     universe.TravelModeVO
	OriginID string
}

// NewEntity creates an Entity positioned at the given location and coordinate.
// The start location is recorded so a win condition can detect a return home.
// No budget or objective is set by default (unlimited spending, no objective);
// use SetBudget and SetTarget/SetTargets to enable game rules.
func NewEntity(location string, coord universe.CoordinateVO) *Entity {
	return &Entity{
		currentLocation:   location,
		currentCoordinate: coord,
		startLocation:     location,
		travelHistory:     []string{},
	}
}

// SetBudget installs a spending pool. A budget of 0 (the default) means
// spending is unlimited and CanAfford always succeeds.
func (s *Entity) SetBudget(budget float64) { s.budget = budget }

// SetTarget installs a single-objective quest (a chain of length one) and
// re-evaluates progress against the current position. It is preserved for
// callers with one objective; SetTargets installs a multi-objective chain.
func (s *Entity) SetTarget(target universe.CoordinateVO) {
	s.SetTargets([]universe.CoordinateVO{target})
}

// SetTargets installs an ordered quest chain of objective coordinates and
// re-evaluates progress against the current position (so an objective that
// coincides with the start location is not counted as already reached unless the
// traveller is genuinely there). Objectives are done in order and each is a
// round trip: reach its waypoint, then return to the start location before the
// next objective begins. The game is won once every objective is completed.
func (s *Entity) SetTargets(targets []universe.CoordinateVO) {
	s.targets = append([]universe.CoordinateVO(nil), targets...)
	s.objectiveIndex = 0
	s.reachedCurrent = false
	s.won = false
	s.evaluateGoal()
}

// Clone returns a deep copy of the entity so callers can branch exploration
// from a given position without mutating the original session's location,
// history, or context stack. The travel history and context stack slices are
// copied rather than shared so later moves on either copy stay independent.
func (s *Entity) Clone() *Entity {
	cp := *s
	cp.travelHistory = append([]string(nil), s.travelHistory...)
	cp.contextStack = append([]ContextTransition(nil), s.contextStack...)
	return &cp
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

// StartLocation returns the ID of the location the session began at (home).
func (s *Entity) StartLocation() string { return s.startLocation }

// Budget returns the spending pool (0 means unlimited).
func (s *Entity) Budget() float64 { return s.budget }

// HasBudget reports whether a finite budget is in force.
func (s *Entity) HasBudget() bool { return s.budget > 0 }

// RemainingBudget returns how much of the budget is left after cumulative cost.
// With no budget in force it returns 0; callers should gate on HasBudget.
// Returning home is always permitted even when it costs more than the budget
// covers (see ReturnHomeCommand), so cumulative cost can exceed the budget; a
// depleted budget is reported as empty (0) rather than a negative value.
func (s *Entity) RemainingBudget() float64 {
	if !s.HasBudget() {
		return 0
	}
	if remaining := s.budget - s.cumulativeCost; remaining > 0 {
		return remaining
	}
	return 0
}

// CanAfford reports whether a move of the given cost is within budget. With no
// budget in force every move is affordable.
func (s *Entity) CanAfford(cost float64) bool {
	if !s.HasBudget() {
		return true
	}
	return s.cumulativeCost+cost <= s.budget
}

// Target returns the current objective coordinate: the next waypoint still to
// reach, or the final waypoint once the whole chain has been reached. It returns
// the zero coordinate when no chain is set.
func (s *Entity) Target() universe.CoordinateVO {
	if len(s.targets) == 0 {
		return universe.CoordinateVO{}
	}
	if s.objectiveIndex >= len(s.targets) {
		return s.targets[len(s.targets)-1]
	}
	return s.targets[s.objectiveIndex]
}

// Targets returns a snapshot of the ordered objective chain.
func (s *Entity) Targets() []universe.CoordinateVO {
	out := make([]universe.CoordinateVO, len(s.targets))
	copy(out, s.targets)
	return out
}

// ObjectiveCount returns the number of objectives in the quest chain.
func (s *Entity) ObjectiveCount() int { return len(s.targets) }

// ObjectiveIndex returns how many objectives have been completed so far (each
// reached and then returned home from), which also indexes the current objective
// still to do.
func (s *Entity) ObjectiveIndex() int { return s.objectiveIndex }

// HasTarget reports whether an objective (a quest chain of one or more
// waypoints) is set.
func (s *Entity) HasTarget() bool { return len(s.targets) > 0 }

// ReachedTarget reports whether the current objective's waypoint has been
// reached this trip, so the only remaining step to complete it is returning to
// the start location. It is false once the objective is banked (on return home)
// and false again while heading out to the next one.
func (s *Entity) ReachedTarget() bool {
	return s.reachedCurrent
}

// Won reports whether every objective is complete: each waypoint reached and
// returned home from, in order.
func (s *Entity) Won() bool { return s.won }

// evaluateGoal advances the quest against the current position after every
// movement. Each objective is a round trip completed in order: the current
// objective's waypoint must be reached first (marking reachedCurrent), and only
// once the traveller is back at the start location does that objective count as
// done — advancing objectiveIndex to the next one, or winning after the last.
// Only the current objective's waypoint is checked, so reaching a later one
// early does not count. Coordinates are compared by canonical Onto Address so a
// position reached via any route counts as the same waypoint.
func (s *Entity) evaluateGoal() {
	if len(s.targets) == 0 || s.objectiveIndex >= len(s.targets) {
		return
	}
	if !s.reachedCurrent &&
		s.currentCoordinate.OntoAddress() == s.targets[s.objectiveIndex].OntoAddress() {
		s.reachedCurrent = true
	}
	if s.reachedCurrent && s.currentLocation == s.startLocation {
		s.objectiveIndex++
		s.reachedCurrent = false
		if s.objectiveIndex >= len(s.targets) {
			s.won = true
		}
	}
}

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
	s.evaluateGoal()
}

// TransitionTo applies a contextual movement and records or removes its
// ancestry entry. A reversed transition removes the most recent entry for the
// same mode, regardless of its position in the stack, so unwinding axes in a
// different order than they were entered (e.g. 'universe back' before
// 'shift back') keeps the stack consistent with the coordinate.
func (s *Entity) TransitionTo(loc universe.LocationEntity, cost float64, mode universe.TravelModeVO, reversed bool) {
	prev := s.currentLocation
	s.currentLocation = loc.ID
	s.currentCoordinate = loc.Coordinate
	s.cumulativeCost += cost
	s.evaluateGoal()
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
		for i := len(s.contextStack) - 1; i >= 0; i-- {
			if s.contextStack[i].Mode == mode {
				s.contextStack = append(s.contextStack[:i], s.contextStack[i+1:]...)
				break
			}
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

// NextMathematicsID returns the location ID that 'mathematical' would move to
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

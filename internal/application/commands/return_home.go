package commands

import (
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// ReturnHomeCommand coordinates the contextual returns and final physical trip
// required to return a session to its start location.
type ReturnHomeCommand struct {
	Universe   *universe.Aggregate
	Session    *exploration.Entity
	Pathfinder navigation.PathfinderService
	HomeID     string
}

// ReturnHomeStep is one planned or completed leg of a return-home journey.
type ReturnHomeStep struct {
	Action string
	Detail string
	Cost   float64
}

// Plan returns the ordered return-home steps without mutating the session.
// The session's recorded context stack is the single source of truth for what
// must be unwound (empty stack ⇔ already at base reality, so the plan is just
// the physical walk home).
func (c *ReturnHomeCommand) Plan() ([]ReturnHomeStep, float64) {
	return c.planRecordedTransitions(c.Session.ContextTransitions())
}

// Execute applies the return-home workflow, unwinding the recorded context
// stack (if any) and then walking the remaining physical route home.
func (c *ReturnHomeCommand) Execute() ([]ReturnHomeStep, error) {
	return c.unwindRecordedTransitions()
}

func (c *ReturnHomeCommand) planRecordedTransitions(transitions []exploration.ContextTransition) ([]ReturnHomeStep, float64) {
	steps := make([]ReturnHomeStep, 0, len(transitions))
	var total float64
	current, ok := c.Universe.GetLocation(c.Session.Location())
	if !ok {
		return nil, 0
	}
	// Track coordinates independently of graph lookup success so plan labels
	// always show the true N → N-1 change even when a reverse edge was never
	// materialised (common after generating nearby places inside a branch).
	plannedCoord := current.Coordinate
	plannedID := current.ID
	lastExistingID := current.ID

	for i := len(transitions) - 1; i >= 0; i-- {
		mode := transitions[i].Mode
		fromCoord := plannedCoord
		toCoord, ok := universe.LowerContextCoordinate(fromCoord, mode)
		if !ok {
			// Observer/time and unknown modes: the return is edge-defined. Prefer
			// an existing return edge; otherwise self-heal the enclosing
			// counterpart from the recorded origin (the same repair Execute does)
			// so the plan matches a trip that will actually succeed.
			id, found := c.returnDestination(plannedID, mode)
			if !found {
				if healed, err := universe.EnsureContextualReturn(c.Universe, plannedID, transitions[i].OriginID, mode); err == nil {
					id, found = healed, true
				}
			}
			if found {
				origin, _ := c.Universe.GetLocation(id)
				steps = append(steps, ReturnHomeStep{
					Action: returnAction(mode),
					Detail: planDetail(mode, locationWithCoord(plannedID, fromCoord), origin),
					Cost:   returnCost(mode),
				})
				total += returnCost(mode)
				plannedID = origin.ID
				plannedCoord = origin.Coordinate
				lastExistingID = origin.ID
				continue
			}
			steps = append(steps, ReturnHomeStep{Action: returnAction(mode), Detail: "return path unavailable"})
			continue
		}

		detail := planDetailCoords(mode, fromCoord, toCoord)
		if id, found := c.returnDestination(plannedID, mode); found {
			origin, _ := c.Universe.GetLocation(id)
			toCoord = origin.Coordinate
			detail = planDetailCoords(mode, fromCoord, toCoord)
			plannedID = origin.ID
			plannedCoord = origin.Coordinate
			lastExistingID = origin.ID
		} else if id, found := universe.LowerContextID(plannedID, mode); found {
			// Advance the planned identity even if the node is absent so later
			// steps keep decreasing the right axis instead of repeating N → N.
			plannedID = id
			plannedCoord = toCoord
			if _, exists := c.Universe.GetLocation(id); exists {
				lastExistingID = id
			}
		} else {
			plannedCoord = toCoord
		}

		steps = append(steps, ReturnHomeStep{Action: returnAction(mode), Detail: detail, Cost: returnCost(mode)})
		total += returnCost(mode)
	}

	if lastExistingID != c.HomeID {
		if path, ok := c.Pathfinder.FindRoute(c.Universe, lastExistingID, c.HomeID); ok {
			for _, edge := range path {
				if !edge.Mode.IsPhysical() {
					// Context should already be unwound; a non-physical hop means
					// the residual route is not a pure walk home — stop rather than
					// pretend later physical edges are reachable without it.
					break
				}
				steps = append(steps, ReturnHomeStep{Action: "travel", Detail: fmt.Sprintf("%s -> %s", edge.From, edge.To), Cost: edge.Cost})
				total += edge.Cost
			}
		}
	}
	return steps, total
}

func locationWithCoord(id string, coord universe.CoordinateVO) universe.LocationEntity {
	return universe.LocationEntity{ID: id, Coordinate: coord}
}

// planDetailCoords formats a from → to label using coordinates directly so the
// plan never depends on a successfully resolved destination entity.
func planDetailCoords(mode universe.TravelModeVO, from, to universe.CoordinateVO) string {
	switch mode {
	case universe.ObserverShift:
		return fmt.Sprintf("%s → %s", from.Observer, to.Observer)
	case universe.ConsensusShift:
		return fmt.Sprintf("consensus %d → %d", from.Consensus, to.Consensus)
	case universe.SimulationEntry:
		return fmt.Sprintf("simulation %d → %d", from.Simulation, to.Simulation)
	case universe.TimelineShift:
		return fmt.Sprintf("timeline %s → %s", from.Timeline, to.Timeline)
	case universe.QuantumShift:
		return fmt.Sprintf("quantum %s → %s", from.Quantum, to.Quantum)
	case universe.UniverseShift:
		return fmt.Sprintf("universe %s → %s", from.Universe, to.Universe)
	case universe.MathematicalShift:
		return fmt.Sprintf("mathematics %s → %s", from.Mathematics, to.Mathematics)
	case universe.TimeShift:
		return fmt.Sprintf("%s → %s", from.Time.Format("2006-01-02T15:04:05Z07:00"), to.Time.Format("2006-01-02T15:04:05Z07:00"))
	}
	return ""
}

func (c *ReturnHomeCommand) returnDestination(from string, mode universe.TravelModeVO) (string, bool) {
	switch mode {
	case universe.ObserverShift:
		return c.observerReturn(from)
	case universe.TimeShift:
		return c.timeReturn(from)
	case universe.ConsensusShift, universe.SimulationEntry, universe.TimelineShift, universe.QuantumShift, universe.UniverseShift, universe.MathematicalShift:
		return c.lowerContext(from, mode)
	}
	return "", false
}

func (c *ReturnHomeCommand) unwindRecordedTransitions() ([]ReturnHomeStep, error) {
	var steps []ReturnHomeStep
	for len(c.Session.ContextTransitions()) > 0 {
		transitions := c.Session.ContextTransitions()
		transition := transitions[len(transitions)-1]
		mode := transition.Mode
		current, _ := c.Universe.GetLocation(c.Session.Location())
		origin, _ := c.Universe.GetLocation(transition.OriginID)
		// Observer/time returns are edge-defined and cannot self-heal by ID
		// arithmetic the way the numeric axes do inside EnsureLowerContext. A
		// node spawned inside such a branch (e.g. a nearby dead-end) has no
		// return edge, so reconstruct the enclosing counterpart from the
		// recorded origin before unwinding.
		if mode == universe.ObserverShift || mode == universe.TimeShift {
			if _, found := c.returnDestination(c.Session.Location(), mode); !found {
				if _, err := universe.EnsureContextualReturn(c.Universe, c.Session.Location(), transition.OriginID, mode); err != nil {
					return steps, err
				}
			}
		}
		if err := c.unwind(mode); err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: returnAction(mode), Detail: returnDetail(mode, current, origin), Cost: returnCost(mode)})
	}

	if c.Session.Location() != c.HomeID {
		result, err := (&TravelCommand{Universe: c.Universe, Session: c.Session, Pathfinder: c.Pathfinder}).Execute(c.HomeID)
		if err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: "travel", Detail: result.Location.Name})
	}
	return steps, nil
}

func returnDetail(mode universe.TravelModeVO, current, origin universe.LocationEntity) string {
	switch mode {
	case universe.ObserverShift:
		return origin.Coordinate.Observer
	case universe.ConsensusShift:
		return fmt.Sprintf("%d", origin.Coordinate.Consensus)
	case universe.SimulationEntry:
		return fmt.Sprintf("%d", origin.Coordinate.Simulation)
	case universe.TimelineShift:
		return origin.Coordinate.Timeline
	case universe.QuantumShift:
		return origin.Coordinate.Quantum
	case universe.UniverseShift:
		return origin.Coordinate.Universe
	case universe.MathematicalShift:
		return origin.Coordinate.Mathematics
	case universe.TimeShift:
		return origin.Coordinate.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return ""
}

// planDetail formats a from → to label for a step from current down to origin.
// It is the entity-typed convenience wrapper over planDetailCoords.
func planDetail(mode universe.TravelModeVO, current, origin universe.LocationEntity) string {
	return planDetailCoords(mode, current.Coordinate, origin.Coordinate)
}

func (c *ReturnHomeCommand) unwind(mode universe.TravelModeVO) error {
	switch mode {
	case universe.ObserverShift:
		_, err := (&ObserveCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	case universe.ConsensusShift:
		_, err := (&DriftCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	case universe.SimulationEntry:
		_, err := (&SimulateCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	case universe.TimelineShift:
		_, err := (&JumpCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	case universe.QuantumShift:
		_, err := (&ShiftCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	case universe.UniverseShift:
		_, err := (&UniverseCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	case universe.MathematicalShift:
		_, err := (&StructureCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	case universe.TimeShift:
		_, err := (&TimeCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
		return err
	}
	return fmt.Errorf("cannot unwind contextual transition %q", mode)
}

func returnAction(mode universe.TravelModeVO) string {
	switch mode {
	case universe.ObserverShift:
		return "observe back"
	case universe.ConsensusShift:
		return "align"
	case universe.SimulationEntry:
		return "simulate back"
	case universe.TimelineShift:
		return "jump back"
	case universe.QuantumShift:
		return "shift back"
	case universe.UniverseShift:
		return "universe back"
	case universe.MathematicalShift:
		return "structure back"
	case universe.TimeShift:
		return "time back"
	}
	return "return"
}

func returnCost(mode universe.TravelModeVO) float64 {
	switch mode {
	case universe.ObserverShift:
		return universe.ObserverShiftCost
	case universe.ConsensusShift:
		return universe.ConsensusShiftCost
	case universe.SimulationEntry:
		return universe.SimulationExitCost
	case universe.TimelineShift:
		return universe.TimelineShiftCost
	case universe.QuantumShift:
		return universe.QuantumShiftCost
	case universe.UniverseShift:
		return universe.UniverseShiftCost
	case universe.MathematicalShift:
		return universe.MathematicalShiftCost
	case universe.TimeShift:
		return universe.TimeShiftCost
	}
	return 0
}

func (c *ReturnHomeCommand) lowerContext(from string, mode universe.TravelModeVO) (string, bool) {
	// Ensure (not just Find) so planning a return home also backfills missing
	// reverse edges — the same repair Execute needs when nearby places were
	// spawned inside a branch without reverse contextual links.
	id, err := universe.EnsureLowerContext(c.Universe, from, mode)
	return id, err == nil
}

func (c *ReturnHomeCommand) observerReturn(from string) (string, bool) {
	for _, edge := range c.Universe.EdgesFrom(from) {
		if edge.IsObserverReturn() {
			return edge.To, true
		}
	}
	return "", false
}

func (c *ReturnHomeCommand) timeReturn(from string) (string, bool) {
	for _, edge := range c.Universe.EdgesFrom(from) {
		if edge.Mode == universe.TimeShift && strings.HasPrefix(edge.Description, "Time shift back to ") {
			return edge.To, true
		}
	}
	return "", false
}

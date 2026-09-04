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

// returnPlanState carries the running plan as each recorded transition is
// unwound: the accumulated steps and cost, plus the planned identity/coordinate
// (advanced even when a node is absent so labels keep showing the true N → N-1
// change) and the last identity that actually exists in the graph, which anchors
// the residual physical walk home.
type returnPlanState struct {
	steps          []ReturnHomeStep
	total          float64
	plannedCoord   universe.CoordinateVO
	plannedID      string
	lastExistingID string
}

func (c *ReturnHomeCommand) planRecordedTransitions(transitions []exploration.ContextTransition) ([]ReturnHomeStep, float64) {
	current, ok := c.Universe.GetLocation(c.Session.Location())
	if !ok {
		return nil, 0
	}
	st := &returnPlanState{
		steps:          make([]ReturnHomeStep, 0, len(transitions)),
		plannedCoord:   current.Coordinate,
		plannedID:      current.ID,
		lastExistingID: current.ID,
	}
	for i := len(transitions) - 1; i >= 0; i-- {
		c.planOneTransition(st, transitions[i])
	}
	return c.appendResidualRouteHome(st.steps, st.total, st.lastExistingID)
}

// planOneTransition plans the return leg for a single recorded transition,
// dispatching to the edge-defined path (observer/time and unknown modes, whose
// return is defined by a graph edge) or the lowered-axis path.
func (c *ReturnHomeCommand) planOneTransition(st *returnPlanState, t exploration.ContextTransition) {
	fromCoord := st.plannedCoord
	toCoord, ok := universe.LowerContextCoordinate(fromCoord, t.Mode)
	if !ok {
		c.planEdgeDefinedReturn(st, t, fromCoord)
		return
	}
	c.planLoweredReturn(st, t.Mode, fromCoord, toCoord)
}

// planEdgeDefinedReturn plans a return whose destination is defined by a graph
// edge. It prefers an existing return edge; otherwise it self-heals the
// enclosing counterpart from the recorded origin (the same repair Execute does)
// so the plan matches a trip that will actually succeed.
func (c *ReturnHomeCommand) planEdgeDefinedReturn(st *returnPlanState, t exploration.ContextTransition, fromCoord universe.CoordinateVO) {
	mode := t.Mode
	id, found := c.returnDestination(st.plannedID, mode)
	if !found {
		if healed, err := universe.EnsureContextualReturn(c.Universe, st.plannedID, t.OriginID, mode); err == nil {
			id, found = healed, true
		}
	}
	if !found {
		st.steps = append(st.steps, ReturnHomeStep{Action: returnAction(mode), Detail: "return path unavailable"})
		return
	}
	origin, _ := c.Universe.GetLocation(id)
	st.steps = append(st.steps, ReturnHomeStep{
		Action: returnAction(mode),
		Detail: planDetail(mode, locationWithCoord(st.plannedID, fromCoord), origin),
		Cost:   returnCost(mode),
	})
	st.total += returnCost(mode)
	st.plannedID = origin.ID
	st.plannedCoord = origin.Coordinate
	st.lastExistingID = origin.ID
}

// planLoweredReturn plans a return along an axis that has a well-defined lower
// context coordinate, advancing the planned identity even when the target node
// is absent so later steps keep decreasing the right axis.
func (c *ReturnHomeCommand) planLoweredReturn(st *returnPlanState, mode universe.TravelModeVO, fromCoord, toCoord universe.CoordinateVO) {
	detail := planDetailCoords(mode, fromCoord, toCoord)
	if id, found := c.returnDestination(st.plannedID, mode); found {
		origin, _ := c.Universe.GetLocation(id)
		detail = planDetailCoords(mode, fromCoord, origin.Coordinate)
		st.plannedID = origin.ID
		st.plannedCoord = origin.Coordinate
		st.lastExistingID = origin.ID
	} else if id, found := universe.LowerContextID(st.plannedID, mode); found {
		st.plannedID = id
		st.plannedCoord = toCoord
		if _, exists := c.Universe.GetLocation(id); exists {
			st.lastExistingID = id
		}
	} else {
		st.plannedCoord = toCoord
	}
	st.steps = append(st.steps, ReturnHomeStep{Action: returnAction(mode), Detail: detail, Cost: returnCost(mode)})
	st.total += returnCost(mode)
}

// appendResidualRouteHome appends the final physical walk from lastExistingID to
// HomeID (if any) onto an in-progress plan, returning the extended steps and
// running total. return home is the safety hatch: it must always get the
// traveller back. Normally the residual route home is a pure physical walk, but
// a genuine sink (the well) can only be left by a non-physical exit edge (its
// drift back to the surface). If the route home crosses such an edge, it is
// included as an "escape" step rather than stopping short — otherwise the plan
// would advertise a journey that cannot complete.
func (c *ReturnHomeCommand) appendResidualRouteHome(steps []ReturnHomeStep, total float64, lastExistingID string) ([]ReturnHomeStep, float64) {
	if lastExistingID == c.HomeID {
		return steps, total
	}
	path, ok := c.Pathfinder.FindRoute(c.Universe, lastExistingID, c.HomeID)
	if !ok {
		return steps, total
	}
	for _, edge := range path {
		steps = append(steps, ReturnHomeStep{Action: routeStepAction(edge.Mode), Detail: fmt.Sprintf("%s -> %s", edge.From, edge.To), Cost: edge.Cost})
		total += edge.Cost
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
		result, err := (&TravelCommand{Universe: c.Universe, Session: c.Session, Pathfinder: c.Pathfinder, IgnoreBudget: true}).Execute(c.HomeID)
		if err == nil {
			steps = append(steps, ReturnHomeStep{Action: "travel", Detail: result.Location.Name})
			return steps, nil
		}
		// No pure-physical walk home: the traveller is in a genuine sink (the
		// well) whose only exit is a non-physical edge. return home is the safety
		// hatch, so follow the full route home — including that escape edge —
		// rather than stranding them here.
		escapeSteps, escapeErr := c.walkHomeAcrossEscapes()
		if escapeErr != nil {
			return steps, err
		}
		steps = append(steps, escapeSteps...)
	}
	return steps, nil
}

// walkHomeAcrossEscapes applies the full route from the current location to home,
// following non-physical exit edges (e.g. the well's drift back to the surface)
// as well as physical ones. It is the fallback used by return home when no pure
// physical walk home exists, so a genuine sink is never a soft-lock. Non-physical
// escape edges here are seed edges, not recorded context transitions, so they are
// applied with MoveTo (no context-stack change); physical edges reuse MoveTo too
// since the route was already validated by the pathfinder.
func (c *ReturnHomeCommand) walkHomeAcrossEscapes() ([]ReturnHomeStep, error) {
	path, ok := c.Pathfinder.FindRoute(c.Universe, c.Session.Location(), c.HomeID)
	if !ok {
		return nil, fmt.Errorf("no route home from %s", c.Session.Location())
	}
	var steps []ReturnHomeStep
	for _, edge := range path {
		dest, ok := c.Universe.GetLocation(edge.To)
		if !ok {
			return steps, fmt.Errorf("%w: %s", universe.ErrUnknownEdgeEndpoint, edge.To)
		}
		c.Session.MoveTo(dest, edge.Cost)
		steps = append(steps, ReturnHomeStep{Action: routeStepAction(edge.Mode), Detail: dest.Name})
	}
	return steps, nil
}

// routeStepAction labels a residual route-home edge for display and unwinding: a
// physical leg is an ordinary "travel", while a non-physical exit edge (e.g. the
// well's drift back to the surface) is an "escape" — the safety-hatch hop that
// leaves a sink. Both Plan and Execute label the same route this way, so the rule
// lives in one place.
func routeStepAction(mode universe.TravelModeVO) string {
	if mode.IsPhysical() {
		return "travel"
	}
	return "escape"
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
		_, err := (&MathematicalCommand{Universe: c.Universe, Session: c.Session, Back: true}).Execute()
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
		return "mathematical back"
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

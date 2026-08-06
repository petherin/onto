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
	Universe        *universe.Aggregate
	Session         *exploration.Entity
	Repo            universe.Repository
	Pathfinder      navigation.PathfinderService
	HomeID          string
	DefaultObserver string
}

// ReturnHomeStep is one planned or completed leg of a return-home journey.
type ReturnHomeStep struct {
	Action string
	Detail string
	Cost   float64
}

// Plan returns the ordered return-home steps without mutating the session.
func (c *ReturnHomeCommand) Plan() ([]ReturnHomeStep, float64) {
	if transitions := c.Session.ContextTransitions(); len(transitions) > 0 {
		return c.planRecordedTransitions(transitions)
	}
	var steps []ReturnHomeStep
	planned := c.Session.Location()
	for {
		current, ok := c.Universe.GetLocation(planned)
		if !ok || current.Coordinate.Observer == c.DefaultObserver {
			break
		}
		next, ok := c.observerReturn(planned)
		if !ok {
			return append(steps, ReturnHomeStep{Action: "observe back", Detail: "return path unavailable"}), 0
		}
		dest, _ := c.Universe.GetLocation(next)
		steps = append(steps, ReturnHomeStep{Action: "observe back", Detail: fmt.Sprintf("%s → %s", current.Coordinate.Observer, dest.Coordinate.Observer), Cost: universe.ObserverShiftCost})
		planned = next
	}
	for _, transition := range []struct {
		mode   universe.TravelModeVO
		action string
		cost   float64
		count  int
	}{
		{universe.ConsensusShift, "align", universe.ConsensusShiftCost, c.Session.ConsensusLevel()},
		{universe.TimelineShift, "jump back", universe.TimelineShiftCost, c.Session.TimelineLevel()},
		{universe.QuantumShift, "shift back", universe.QuantumShiftCost, c.Session.QuantumLevel()},
	} {
		for range transition.count {
			current, _ := c.Universe.GetLocation(planned)
			detail := ""
			switch transition.mode {
			case universe.ConsensusShift:
				detail = fmt.Sprintf("consensus %d → %d", current.Coordinate.Consensus, current.Coordinate.Consensus-1)
			case universe.TimelineShift:
				detail = fmt.Sprintf("timeline %s → T%d", current.Coordinate.Timeline, current.Coordinate.TimelineLevel()-1)
			case universe.QuantumShift:
				detail = fmt.Sprintf("quantum %s → Q%d", current.Coordinate.Quantum, current.Coordinate.QuantumLevel()-1)
			}
			steps = append(steps, ReturnHomeStep{Action: transition.action, Detail: detail, Cost: transition.cost})
			if next, ok := c.lowerContext(planned, transition.mode); ok {
				planned = next
			}
		}
	}
	for {
		current, ok := c.Universe.GetLocation(planned)
		if !ok || current.Coordinate.Time.IsZero() {
			break
		}
		next, ok := c.timeReturn(planned)
		if !ok {
			return append(steps, ReturnHomeStep{Action: "time back", Detail: "return path unavailable"}), 0
		}
		dest, _ := c.Universe.GetLocation(next)
		steps = append(steps, ReturnHomeStep{Action: "time back", Detail: fmt.Sprintf("%s → %s", current.Coordinate.Time.Format("2006-01-02T15:04:05Z07:00"), dest.Coordinate.Time.Format("2006-01-02T15:04:05Z07:00")), Cost: universe.TimeShiftCost})
		planned = next
	}
	if planned != c.HomeID {
		if path, ok := c.Pathfinder.FindRoute(c.Universe, planned, c.HomeID); ok {
			for _, edge := range path {
				if !edge.Mode.IsPhysical() {
					break
				}
				steps = append(steps, ReturnHomeStep{Action: "travel", Detail: fmt.Sprintf("%s -> %s", edge.From, edge.To), Cost: edge.Cost})
			}
		}
	}
	var cost float64
	for _, step := range steps {
		cost += step.Cost
	}
	return steps, cost
}

// Execute applies the return-home workflow.
func (c *ReturnHomeCommand) Execute() ([]ReturnHomeStep, error) {
	if len(c.Session.ContextTransitions()) > 0 {
		return c.unwindRecordedTransitions()
	}
	var steps []ReturnHomeStep
	for c.Session.Coordinate().Observer != c.DefaultObserver {
		result, err := (&ObserveCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		if err != nil {
			return steps, err
		}

		steps = append(steps, ReturnHomeStep{Action: "observe back", Detail: result.Observer, Cost: universe.ObserverShiftCost})
	}
	for c.Session.ConsensusLevel() > 0 {
		result, err := (&DriftCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		if err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: "align", Detail: fmt.Sprintf("%d", result.Consensus), Cost: universe.ConsensusShiftCost})
	}
	for c.Session.TimelineLevel() > 0 {
		result, err := (&JumpCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		if err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: "jump back", Detail: result.NextTimeline, Cost: universe.TimelineShiftCost})
	}
	for c.Session.QuantumLevel() > 0 {
		result, err := (&ShiftCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		if err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: "shift back", Detail: result.NextQuantum, Cost: universe.QuantumShiftCost})
	}
	for !c.Session.Coordinate().Time.IsZero() {
		result, err := (&TimeCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		if err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: "time back", Detail: result.Time.Format("2006-01-02T15:04:05Z07:00"), Cost: universe.TimeShiftCost})
	}
	if c.Session.Location() != c.HomeID {
		result, err := (&TravelCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Pathfinder: c.Pathfinder}).Execute(c.HomeID)
		if err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: "travel", Detail: result.Location.Name})
	}
	return steps, nil
}

func (c *ReturnHomeCommand) planRecordedTransitions(transitions []exploration.ContextTransition) ([]ReturnHomeStep, float64) {
	steps := make([]ReturnHomeStep, 0, len(transitions))
	var total float64
	current, _ := c.Universe.GetLocation(c.Session.Location())
	for i := len(transitions) - 1; i >= 0; i-- {
		mode := transitions[i].Mode
		origin := current
		if id, ok := c.returnDestination(current.ID, mode); ok {
			origin, _ = c.Universe.GetLocation(id)
		}
		steps = append(steps, ReturnHomeStep{Action: returnAction(mode), Detail: planDetail(mode, current, origin), Cost: returnCost(mode)})
		total += returnCost(mode)
		current = origin
	}

	if current.ID != c.HomeID {
		if path, ok := c.Pathfinder.FindRoute(c.Universe, current.ID, c.HomeID); ok {
			for _, edge := range path {
				if !edge.Mode.IsPhysical() {
					break
				}
				steps = append(steps, ReturnHomeStep{Action: "travel", Detail: fmt.Sprintf("%s -> %s", edge.From, edge.To), Cost: edge.Cost})
				total += edge.Cost
			}
		}
	}
	return steps, total
}

func (c *ReturnHomeCommand) returnDestination(from string, mode universe.TravelModeVO) (string, bool) {
	switch mode {
	case universe.ObserverShift:
		return c.observerReturn(from)
	case universe.TimeShift:
		return c.timeReturn(from)
	case universe.ConsensusShift, universe.TimelineShift, universe.QuantumShift:
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
		if err := c.unwind(mode); err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: returnAction(mode), Detail: returnDetail(mode, current, origin), Cost: returnCost(mode)})
	}

	if c.Session.Location() != c.HomeID {
		result, err := (&TravelCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Pathfinder: c.Pathfinder}).Execute(c.HomeID)
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
	case universe.TimelineShift:
		return origin.Coordinate.Timeline
	case universe.QuantumShift:
		return origin.Coordinate.Quantum
	case universe.TimeShift:
		return origin.Coordinate.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return ""
}

func planDetail(mode universe.TravelModeVO, current, origin universe.LocationEntity) string {
	switch mode {
	case universe.ObserverShift:
		return fmt.Sprintf("%s → %s", current.Coordinate.Observer, origin.Coordinate.Observer)
	case universe.ConsensusShift:
		return fmt.Sprintf("consensus %d → %d", current.Coordinate.Consensus, origin.Coordinate.Consensus)
	case universe.TimelineShift:
		return fmt.Sprintf("timeline %s → %s", current.Coordinate.Timeline, origin.Coordinate.Timeline)
	case universe.QuantumShift:
		return fmt.Sprintf("quantum %s → %s", current.Coordinate.Quantum, origin.Coordinate.Quantum)
	case universe.TimeShift:
		return fmt.Sprintf("%s → %s", current.Coordinate.Time.Format("2006-01-02T15:04:05Z07:00"), origin.Coordinate.Time.Format("2006-01-02T15:04:05Z07:00"))
	}
	return ""
}

func (c *ReturnHomeCommand) unwind(mode universe.TravelModeVO) error {
	switch mode {
	case universe.ObserverShift:
		_, err := (&ObserveCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		return err
	case universe.ConsensusShift:
		_, err := (&DriftCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		return err
	case universe.TimelineShift:
		_, err := (&JumpCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		return err
	case universe.QuantumShift:
		_, err := (&ShiftCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
		return err
	case universe.TimeShift:
		_, err := (&TimeCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Back: true}).Execute()
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
	case universe.TimelineShift:
		return "jump back"
	case universe.QuantumShift:
		return "shift back"
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
	case universe.TimelineShift:
		return universe.TimelineShiftCost
	case universe.QuantumShift:
		return universe.QuantumShiftCost
	case universe.TimeShift:
		return universe.TimeShiftCost
	}
	return 0
}

func (c *ReturnHomeCommand) lowerContext(from string, mode universe.TravelModeVO) (string, bool) {
	current, ok := c.Universe.GetLocation(from)
	if !ok {
		return "", false
	}
	for _, edge := range c.Universe.EdgesFrom(from) {
		dest, ok := c.Universe.GetLocation(edge.To)
		if !ok || edge.Mode != mode {
			continue
		}
		switch mode {
		case universe.ConsensusShift:
			if dest.Coordinate.Consensus < current.Coordinate.Consensus {
				return dest.ID, true
			}
		case universe.TimelineShift:
			if dest.Coordinate.TimelineLevel() < current.Coordinate.TimelineLevel() {
				return dest.ID, true
			}
		case universe.QuantumShift:
			if dest.Coordinate.QuantumLevel() < current.Coordinate.QuantumLevel() {
				return dest.ID, true
			}
		}
	}
	return "", false
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

package commands

import (
	"fmt"

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
	if c.Session.Location() != c.HomeID {
		result, err := (&TravelCommand{Universe: c.Universe, Session: c.Session, Repo: c.Repo, Pathfinder: c.Pathfinder}).Execute(c.HomeID)
		if err != nil {
			return steps, err
		}
		steps = append(steps, ReturnHomeStep{Action: "travel", Detail: result.Location.Name})
	}
	return steps, nil
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

package commands

import (
	"errors"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// Sentinel errors for SimulateCommand. Callers may use errors.Is for precise handling.
var (
	ErrAlreadyAtBaseReality = errors.New("already in base reality (simulation depth 0) — cannot exit further")
	ErrNoSimulationPathBack = errors.New("no simulation exit path from here")
)

// SimulateResult is the value returned by a successful SimulateCommand execution.
type SimulateResult struct {
	Simulation int
	Location   universe.LocationEntity
	Edges      []universe.EdgeVO
	History    []string
	Reversed   bool // true when exiting to a shallower simulation depth
}

// SimulateCommand moves the session one simulation layer deeper, or back out
// when Back is true, creating the branch if it does not yet exist.
type SimulateCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Back     bool
}

// Execute runs the command.
func (c *SimulateCommand) Execute() (*SimulateResult, error) {
	if c.Back {
		return c.simulateBack()
	}
	return c.simulateForward()
}

func (c *SimulateCommand) simulateForward() (*SimulateResult, error) {
	nextDepth := c.Session.SimulationLevel() + 1
	destID := c.Session.NextSimulationID()
	currentName := locationName(c.Universe, c.Session.Location())
	if err := universe.BranchSimulation(c.Universe, c.Session.Location(), c.Session.Coordinate(), currentName, destID, nextDepth); err != nil {
		return nil, err
	}
	return c.completeSimulate(destID, nextDepth, false)
}

func (c *SimulateCommand) simulateBack() (*SimulateResult, error) {
	currentLevel := c.Session.SimulationLevel()
	if currentLevel == 0 {
		return nil, ErrAlreadyAtBaseReality
	}

	for _, e := range c.Universe.EdgesFrom(c.Session.Location()) {
		if e.Mode != universe.SimulationEntry {
			continue
		}
		dest, ok := c.Universe.GetLocation(e.To)
		if !ok {
			continue
		}
		if dest.Coordinate.Simulation < currentLevel {
			return c.completeSimulate(dest.ID, dest.Coordinate.Simulation, true)
		}
	}

	return nil, ErrNoSimulationPathBack
}

func (c *SimulateCommand) completeSimulate(destID string, depth int, reversed bool) (*SimulateResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	cost := universe.SimulationEntryCost
	if reversed {
		cost = universe.SimulationExitCost
	}
	c.Session.TransitionTo(loc, cost, universe.SimulationEntry, reversed)

	result := &SimulateResult{
		Simulation: depth,
		Location:   loc,
		Edges:      c.Universe.EdgesFrom(destID),
		History:    c.Session.History(),
		Reversed:   reversed,
	}
	return result, nil
}

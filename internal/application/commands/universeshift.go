package commands

import (
	"errors"
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// Sentinel errors for UniverseCommand. Callers may use errors.Is for precise handling.
var (
	ErrAlreadyAtBaseUniverse = errors.New("already at base universe (Origin) — cannot go back further")
	ErrNoUniversePathBack    = errors.New("no universe path back from here")
)

// UniverseResult is the value returned by a successful UniverseCommand execution.
type UniverseResult struct {
	NextUniverse string
	Location     universe.LocationEntity
	Edges        []universe.EdgeVO
	History      []string
	Reversed     bool // true when shifting back to a lower universe level
}

// UniverseCommand moves the session to the next (or previous) bubble-universe
// branch of the current location, creating the branch if it does not yet exist.
type UniverseCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Back     bool // if true, traverse the reverse universe edge instead of creating a new branch
}

// Execute runs the command. It delegates to universeForward or universeBack
// depending on the Back flag.
func (c *UniverseCommand) Execute() (*UniverseResult, error) {
	if c.Back {
		return c.universeBack()
	}
	return c.universeForward()
}

func (c *UniverseCommand) universeForward() (*UniverseResult, error) {
	nextU := fmt.Sprintf("U%d", c.Session.UniverseLevel()+1)
	destID := c.Session.NextUniverseID()
	currentName := locationName(c.Universe, c.Session.Location())
	if err := universe.BranchUniverse(c.Universe, c.Session.Location(), c.Session.Coordinate(), currentName, destID, nextU); err != nil {
		return nil, err
	}
	return c.completeUniverse(destID, nextU, false)
}

func (c *UniverseCommand) universeBack() (*UniverseResult, error) {
	currentLevel := c.Session.UniverseLevel()
	if currentLevel == 0 {
		return nil, ErrAlreadyAtBaseUniverse
	}

	// Find the universe edge that leads to a lower universe level.
	for _, e := range c.Universe.EdgesFrom(c.Session.Location()) {
		if e.Mode != universe.UniverseShift {
			continue
		}
		dest, ok := c.Universe.GetLocation(e.To)
		if !ok {
			continue
		}
		if dest.Coordinate.UniverseLevel() < currentLevel {
			return c.completeUniverse(dest.ID, dest.Coordinate.Universe, true)
		}
	}

	return nil, ErrNoUniversePathBack
}

func (c *UniverseCommand) completeUniverse(destID, nextUniverse string, reversed bool) (*UniverseResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.TransitionTo(loc, universe.UniverseShiftCost, universe.UniverseShift, reversed)

	result := &UniverseResult{
		NextUniverse: nextUniverse,
		Location:     loc,
		Edges:        c.Universe.EdgesFrom(destID),
		History:      c.Session.History(),
		Reversed:     reversed,
	}
	return result, nil
}

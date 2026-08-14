package commands

import (
	"errors"
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// Sentinel errors for StructureCommand. Callers may use errors.Is for precise handling.
var (
	ErrAlreadyAtBaseMathematics = errors.New("already at base mathematical structure (Classical) — cannot go back further")
	ErrNoMathematicsPathBack    = errors.New("no mathematical-structure path back from here")
)

// StructureResult is the value returned by a successful StructureCommand execution.
type StructureResult struct {
	NextMathematics string
	Location        universe.LocationEntity
	Edges           []universe.EdgeVO
	History         []string
	Reversed        bool // true when shifting back to a lower mathematical level
}

// StructureCommand moves the session to the next (or previous) mathematical
// structure of the current location, creating the branch if it does not yet exist.
type StructureCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Back     bool // if true, traverse the reverse math edge instead of creating a new branch
}

// Execute runs the command. It delegates to structureForward or structureBack
// depending on the Back flag.
func (c *StructureCommand) Execute() (*StructureResult, error) {
	if c.Back {
		return c.structureBack()
	}
	return c.structureForward()
}

func (c *StructureCommand) structureForward() (*StructureResult, error) {
	nextM := fmt.Sprintf("M%d", c.Session.MathematicsLevel()+1)
	destID := c.Session.NextMathematicsID()
	currentName := locationName(c.Universe, c.Session.Location())
	if err := universe.BranchMathematics(c.Universe, c.Session.Location(), c.Session.Coordinate(), currentName, destID, nextM); err != nil {
		return nil, err
	}
	return c.completeStructure(destID, nextM, false)
}

func (c *StructureCommand) structureBack() (*StructureResult, error) {
	if c.Session.MathematicsLevel() == 0 {
		return nil, ErrAlreadyAtBaseMathematics
	}

	destID, err := universe.EnsureLowerContext(c.Universe, c.Session.Location(), universe.MathematicalShift)
	if err != nil {
		return nil, ErrNoMathematicsPathBack
	}
	dest, ok := c.Universe.GetLocation(destID)
	if !ok {
		return nil, ErrNoMathematicsPathBack
	}
	return c.completeStructure(dest.ID, dest.Coordinate.Mathematics, true)
}

func (c *StructureCommand) completeStructure(destID, nextMathematics string, reversed bool) (*StructureResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.TransitionTo(loc, universe.MathematicalShiftCost, universe.MathematicalShift, reversed)

	result := &StructureResult{
		NextMathematics: nextMathematics,
		Location:        loc,
		Edges:           c.Universe.EdgesFrom(destID),
		History:         c.Session.History(),
		Reversed:        reversed,
	}
	return result, nil
}

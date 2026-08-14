package commands

import (
	"errors"
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// Sentinel errors for ShiftCommand. Callers may use errors.Is for precise handling.
var (
	ErrAlreadyAtBaseQuantum = errors.New("already at base quantum level (Q0) — cannot shift back further")
	ErrNoQuantumPathBack    = errors.New("no quantum path back from here")
)

// ShiftResult is the value returned by a successful ShiftCommand execution.
type ShiftResult struct {
	NextQuantum string
	Location    universe.LocationEntity
	Edges       []universe.EdgeVO
	History     []string
	Reversed    bool // true when shifting back to a lower quantum level
}

// ShiftCommand moves the session to the next (or previous) quantum branch of
// the current location, creating the branch if it does not yet exist.
type ShiftCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Back     bool // if true, traverse the reverse quantum edge instead of creating a new branch
}

// Execute runs the command. It delegates to shiftForward or shiftBack depending
// on the Back flag.
func (c *ShiftCommand) Execute() (*ShiftResult, error) {
	if c.Back {
		return c.shiftBack()
	}
	return c.shiftForward()
}

func (c *ShiftCommand) shiftForward() (*ShiftResult, error) {
	nextQ := fmt.Sprintf("Q%d", c.Session.QuantumLevel()+1)
	destID := c.Session.NextQuantumID()
	currentName := locationName(c.Universe, c.Session.Location())
	if err := universe.BranchQuantum(c.Universe, c.Session.Location(), c.Session.Coordinate(), currentName, destID, nextQ); err != nil {
		return nil, err
	}
	return c.completeShift(destID, nextQ, false)
}

func (c *ShiftCommand) shiftBack() (*ShiftResult, error) {
	if c.Session.QuantumLevel() == 0 {
		return nil, ErrAlreadyAtBaseQuantum
	}

	destID, err := universe.EnsureLowerContext(c.Universe, c.Session.Location(), universe.QuantumShift)
	if err != nil {
		return nil, ErrNoQuantumPathBack
	}
	dest, ok := c.Universe.GetLocation(destID)
	if !ok {
		return nil, ErrNoQuantumPathBack
	}
	return c.completeShift(dest.ID, dest.Coordinate.Quantum, true)
}

func (c *ShiftCommand) completeShift(destID, quantum string, reversed bool) (*ShiftResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.TransitionTo(loc, universe.QuantumShiftCost, universe.QuantumShift, reversed)

	result := &ShiftResult{
		NextQuantum: quantum,
		Location:    loc,
		Edges:       c.Universe.EdgesFrom(destID),
		History:     c.Session.History(),
		Reversed:    reversed,
	}
	return result, nil
}

func locationName(u *universe.Aggregate, id string) string {
	if loc, ok := u.GetLocation(id); ok && loc.Name != "" {
		return loc.Name
	}
	return id
}

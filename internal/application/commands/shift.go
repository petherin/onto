package commands

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// ShiftResult is the value returned by a successful ShiftCommand execution.
type ShiftResult struct {
	NextQuantum string
	Location    universe.LocationEntity
	Edges       []universe.EdgeVO
	History     []string
	Persisted   bool
	Reversed    bool // true when shifting back to a lower quantum level
	SaveErr     error
}

// ShiftCommand moves the session to the next (or previous) quantum branch of
// the current location, creating the branch if it does not yet exist.
type ShiftCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Repo     universe.Repository
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
	currentName := locationName(c.Universe, c.Session.CurrentLocation)
	universe.BranchQuantumService(c.Universe, c.Session.CurrentLocation, c.Session.CurrentCoordinate, currentName, destID, nextQ)
	return c.completeShift(destID, nextQ, false)
}

func (c *ShiftCommand) shiftBack() (*ShiftResult, error) {
	currentLevel := c.Session.QuantumLevel()
	if currentLevel == 0 {
		return nil, fmt.Errorf("already at base quantum level (Q0) — cannot shift back further")
	}

	// Find the quantum edge that leads to a lower quantum level.
	for _, e := range c.Universe.EdgesFrom(c.Session.CurrentLocation) {
		if e.Mode != universe.QuantumShift {
			continue
		}
		dest, ok := c.Universe.GetLocation(e.To)
		if !ok {
			continue
		}
		if dest.Coordinate.QuantumLevel() < currentLevel {
			return c.completeShift(dest.ID, dest.Coordinate.Quantum, true)
		}
	}

	return nil, fmt.Errorf("no quantum path back from here")
}

func (c *ShiftCommand) completeShift(destID, quantum string, reversed bool) (*ShiftResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.ShiftTo(loc, universe.QuantumShiftCost)

	result := &ShiftResult{
		NextQuantum: quantum,
		Location:    loc,
		Edges:       c.Universe.EdgesFrom(destID),
		History:     c.Session.TravelHistory,
		Reversed:    reversed,
	}

	if err := c.Repo.Save(c.Universe); err != nil {
		result.SaveErr = err
	} else {
		result.Persisted = true
	}
	return result, nil
}

func locationName(u *universe.Aggregate, id string) string {
	if loc, ok := u.GetLocation(id); ok && loc.Name != "" {
		return loc.Name
	}
	return id
}

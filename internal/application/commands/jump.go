package commands

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// JumpResult is the value returned by a successful JumpCommand execution.
type JumpResult struct {
	NextTimeline string
	Location     universe.LocationEntity
	Edges        []universe.EdgeVO
	History      []string
	Persisted    bool
	Reversed     bool // true when jumping back to a lower timeline level
	SaveErr      error
}

// JumpCommand moves the session to the next (or previous) timeline branch of
// the current location, creating the branch if it does not yet exist.
type JumpCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Repo     universe.Repository
	Back     bool // if true, traverse the reverse timeline edge instead of creating a new branch
}

// Execute runs the command. It delegates to jumpForward or jumpBack depending
// on the Back flag.
func (c *JumpCommand) Execute() (*JumpResult, error) {
	if c.Back {
		return c.jumpBack()
	}
	return c.jumpForward()
}

func (c *JumpCommand) jumpForward() (*JumpResult, error) {
	nextT := fmt.Sprintf("T%d", c.Session.TimelineLevel()+1)
	destID := c.Session.NextTimelineID()
	currentName := locationName(c.Universe, c.Session.CurrentLocation)
	universe.BranchTimelineService(c.Universe, c.Session.CurrentLocation, c.Session.CurrentCoordinate, currentName, destID, nextT)
	return c.completeJump(destID, nextT, false)
}

func (c *JumpCommand) jumpBack() (*JumpResult, error) {
	currentLevel := c.Session.TimelineLevel()
	if currentLevel == 0 {
		return nil, fmt.Errorf("already at base timeline (Prime) — cannot jump back further")
	}

	// Find the timeline edge that leads to a lower timeline level.
	for _, e := range c.Universe.EdgesFrom(c.Session.CurrentLocation) {
		if e.Mode != universe.TimelineShift {
			continue
		}
		dest, ok := c.Universe.GetLocation(e.To)
		if !ok {
			continue
		}
		if dest.Coordinate.TimelineLevel() < currentLevel {
			return c.completeJump(dest.ID, dest.Coordinate.Timeline, true)
		}
	}

	return nil, fmt.Errorf("no timeline path back from here")
}

func (c *JumpCommand) completeJump(destID, timeline string, reversed bool) (*JumpResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.JumpTo(loc, universe.TimelineShiftCost)

	result := &JumpResult{
		NextTimeline: timeline,
		Location:     loc,
		Edges:        c.Universe.EdgesFrom(destID),
		History:      c.Session.TravelHistory,
		Reversed:     reversed,
	}

	if err := c.Repo.Save(c.Universe); err != nil {
		result.SaveErr = err
	} else {
		result.Persisted = true
	}
	return result, nil
}

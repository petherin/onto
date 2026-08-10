package commands

import (
	"errors"
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// Sentinel errors for JumpCommand. Callers may use errors.Is for precise handling.
var (
	ErrAlreadyAtBaseTimeline = errors.New("already at base timeline (Prime) — cannot jump back further")
	ErrNoTimelinePathBack    = errors.New("no timeline path back from here")
)

// JumpResult is the value returned by a successful JumpCommand execution.
type JumpResult struct {
	NextTimeline string
	Location     universe.LocationEntity
	Edges        []universe.EdgeVO
	History      []string
	Reversed     bool // true when jumping back to a lower timeline level
}

// JumpCommand moves the session to the next (or previous) timeline branch of
// the current location, creating the branch if it does not yet exist.
type JumpCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
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
	currentName := locationName(c.Universe, c.Session.Location())
	if err := universe.BranchTimeline(c.Universe, c.Session.Location(), c.Session.Coordinate(), currentName, destID, nextT); err != nil {
		return nil, err
	}
	return c.completeJump(destID, nextT, false)
}

func (c *JumpCommand) jumpBack() (*JumpResult, error) {
	currentLevel := c.Session.TimelineLevel()
	if currentLevel == 0 {
		return nil, ErrAlreadyAtBaseTimeline
	}

	// Find the timeline edge that leads to a lower timeline level.
	for _, e := range c.Universe.EdgesFrom(c.Session.Location()) {
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

	return nil, ErrNoTimelinePathBack
}

func (c *JumpCommand) completeJump(destID, timeline string, reversed bool) (*JumpResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.TransitionTo(loc, universe.TimelineShiftCost, universe.TimelineShift, reversed)

	result := &JumpResult{
		NextTimeline: timeline,
		Location:     loc,
		Edges:        c.Universe.EdgesFrom(destID),
		History:      c.Session.History(),
		Reversed:     reversed,
	}
	return result, nil
}

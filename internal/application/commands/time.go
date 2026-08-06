package commands

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

var (
	// ErrInvalidTimeTarget reports an unparsable temporal destination.
	ErrInvalidTimeTarget = errors.New("time must be an RFC3339 timestamp")
	// ErrTimeUnchanged reports a transition to the current temporal coordinate.
	ErrTimeUnchanged = errors.New("already at that time")
	// ErrNoTimePathBack reports that no temporal return edge is available.
	ErrNoTimePathBack = errors.New("no time path back from here")
)

// TimeResult is returned after entering or leaving a temporal branch.
type TimeResult struct {
	Time     time.Time
	Location universe.LocationEntity
	Edges    []universe.EdgeVO
	History  []string
	Reversed bool
}

// TimeCommand changes the session's temporal coordinate or follows a temporal
// return edge when Back is true.
type TimeCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Repo     universe.Repository
	Target   string
	Back     bool
}

// Execute applies the requested temporal transition.
func (c *TimeCommand) Execute() (*TimeResult, error) {
	if c.Back {
		for _, edge := range c.Universe.EdgesFrom(c.Session.Location()) {
			if edge.Mode != universe.TimeShift || !strings.HasPrefix(edge.Description, "Time shift back to ") {
				continue
			}
			if dest, ok := c.Universe.GetLocation(edge.To); ok {
				return c.complete(dest, true)
			}
		}
		return nil, ErrNoTimePathBack
	}

	target, err := time.Parse(time.RFC3339, strings.TrimSpace(c.Target))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTimeTarget, c.Target)
	}
	target = target.UTC()
	if target.Equal(c.Session.Coordinate().Time) {
		return nil, ErrTimeUnchanged
	}
	destID := fmt.Sprintf("%s-at-%s", c.Session.Location(), target.Format("20060102t150405z"))
	if err := universe.BranchTimeService(c.Universe, c.Session.Location(), c.Session.Coordinate(), locationName(c.Universe, c.Session.Location()), destID, target); err != nil {
		return nil, err
	}
	loc, _ := c.Universe.GetLocation(destID)
	return c.complete(loc, false)
}

func (c *TimeCommand) complete(loc universe.LocationEntity, reversed bool) (*TimeResult, error) {
	c.Session.TransitionTo(loc, universe.TimeShiftCost, universe.TimeShift, reversed)
	result := &TimeResult{Time: loc.Coordinate.Time, Location: loc, Edges: c.Universe.EdgesFrom(loc.ID), History: c.Session.History(), Reversed: reversed}
	if err := c.Repo.Save(c.Universe); err != nil {
		return result, err
	}
	return result, nil
}

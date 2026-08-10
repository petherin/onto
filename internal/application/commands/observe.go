package commands

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

var (
	// ErrObserverUnchanged reports an observer shift to the current perspective.
	ErrObserverUnchanged = errors.New("already using that observer perspective")
	// ErrNoObserverPathBack reports that no reverse observer edge is available.
	ErrNoObserverPathBack = errors.New("no observer path back from here")
)

// ObserveResult is the value returned by a successful ObserveCommand execution.
type ObserveResult struct {
	Observer string
	Location universe.LocationEntity
	Edges    []universe.EdgeVO
	History  []string
	Reversed bool
}

// ObserveCommand changes the session's observer perspective, or returns to
// the preceding perspective when Back is true.
type ObserveCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Repo     universe.Repository
	Observer string
	Back     bool
}

// Execute runs the command.
func (c *ObserveCommand) Execute() (*ObserveResult, error) {
	if c.Back {
		return c.observeBack()
	}
	if strings.EqualFold(strings.TrimSpace(c.Observer), c.Session.Coordinate().Observer) {
		return nil, ErrObserverUnchanged
	}

	observer := strings.TrimSpace(c.Observer)
	destID := fmt.Sprintf("%s-o-%s", c.Session.Location(), observerID(observer))
	if err := universe.BranchObserver(c.Universe, c.Session.Location(), c.Session.Coordinate(), locationName(c.Universe, c.Session.Location()), destID, observer); err != nil {
		return nil, err
	}
	return c.completeObserve(destID, observer, false)
}

func (c *ObserveCommand) observeBack() (*ObserveResult, error) {
	for _, edge := range c.Universe.EdgesFrom(c.Session.Location()) {
		if !edge.IsObserverReturn() {
			continue
		}
		dest, ok := c.Universe.GetLocation(edge.To)
		if ok {
			return c.completeObserve(dest.ID, dest.Coordinate.Observer, true)
		}
	}
	return nil, ErrNoObserverPathBack
}

func (c *ObserveCommand) completeObserve(destID, observer string, reversed bool) (*ObserveResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.TransitionTo(loc, universe.ObserverShiftCost, universe.ObserverShift, reversed)
	result := &ObserveResult{
		Observer: observer,
		Location: loc,
		Edges:    c.Universe.EdgesFrom(destID),
		History:  c.Session.History(),
		Reversed: reversed,
	}
	if err := c.Repo.Save(c.Universe); err != nil {
		return result, err
	}
	return result, nil
}

func observerID(observer string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(observer) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

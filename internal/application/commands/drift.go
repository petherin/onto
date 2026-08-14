package commands

import (
	"errors"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

// Sentinel errors for DriftCommand. Callers may use errors.Is for precise handling.
var (
	ErrAlreadyAtConsensus  = errors.New("already aligned with shared consensus — cannot align further")
	ErrNoConsensusPathBack = errors.New("no consensus path back from here")
)

// DriftResult is the value returned by a successful DriftCommand execution.
type DriftResult struct {
	Consensus int
	Location  universe.LocationEntity
	Edges     []universe.EdgeVO
	History   []string
	Reversed  bool
}

// DriftCommand moves the session into the next consensus divergence, or back
// one level when Back is true.
type DriftCommand struct {
	Universe *universe.Aggregate
	Session  *exploration.Entity
	Back     bool
}

// Execute runs the command.
func (c *DriftCommand) Execute() (*DriftResult, error) {
	if c.Back {
		return c.align()
	}

	nextLevel := c.Session.ConsensusLevel() + 1
	destID := c.Session.NextConsensusID()
	if err := universe.BranchConsensus(c.Universe, c.Session.Location(), c.Session.Coordinate(), locationName(c.Universe, c.Session.Location()), destID, nextLevel); err != nil {
		return nil, err
	}
	return c.completeDrift(destID, nextLevel, false)
}

func (c *DriftCommand) align() (*DriftResult, error) {
	if c.Session.ConsensusLevel() == 0 {
		return nil, ErrAlreadyAtConsensus
	}

	destID, err := universe.EnsureLowerContext(c.Universe, c.Session.Location(), universe.ConsensusShift)
	if err != nil {
		return nil, ErrNoConsensusPathBack
	}
	dest, ok := c.Universe.GetLocation(destID)
	if !ok {
		return nil, ErrNoConsensusPathBack
	}
	return c.completeDrift(dest.ID, dest.Coordinate.Consensus, true)
}

func (c *DriftCommand) completeDrift(destID string, consensus int, reversed bool) (*DriftResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.TransitionTo(loc, universe.ConsensusShiftCost, universe.ConsensusShift, reversed)
	result := &DriftResult{
		Consensus: consensus,
		Location:  loc,
		Edges:     c.Universe.EdgesFrom(destID),
		History:   c.Session.History(),
		Reversed:  reversed,
	}
	return result, nil
}

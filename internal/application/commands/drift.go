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
	Repo     universe.Repository
	Back     bool
}

// Execute runs the command.
func (c *DriftCommand) Execute() (*DriftResult, error) {
	if c.Back {
		return c.align()
	}

	nextLevel := c.Session.ConsensusLevel() + 1
	destID := c.Session.NextConsensusID()
	universe.BranchConsensusService(c.Universe, c.Session.Location(), c.Session.Coordinate(), locationName(c.Universe, c.Session.Location()), destID, nextLevel)
	return c.completeDrift(destID, nextLevel, false)
}

func (c *DriftCommand) align() (*DriftResult, error) {
	currentLevel := c.Session.ConsensusLevel()
	if currentLevel == 0 {
		return nil, ErrAlreadyAtConsensus
	}

	for _, e := range c.Universe.EdgesFrom(c.Session.Location()) {
		if e.Mode != universe.ConsensusShift {
			continue
		}
		dest, ok := c.Universe.GetLocation(e.To)
		if ok && dest.Coordinate.Consensus < currentLevel {
			return c.completeDrift(dest.ID, dest.Coordinate.Consensus, true)
		}
	}

	return nil, ErrNoConsensusPathBack
}

func (c *DriftCommand) completeDrift(destID string, consensus int, reversed bool) (*DriftResult, error) {
	loc, _ := c.Universe.GetLocation(destID)
	c.Session.DriftTo(loc, universe.ConsensusShiftCost)
	result := &DriftResult{
		Consensus: consensus,
		Location:  loc,
		Edges:     c.Universe.EdgesFrom(destID),
		History:   c.Session.History(),
		Reversed:  reversed,
	}
	if err := c.Repo.Save(c.Universe); err != nil {
		return result, err
	}
	return result, nil
}

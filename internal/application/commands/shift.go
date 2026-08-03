package commands

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

type ShiftResult struct {
	NextQuantum string
	Location    universe.Location
	Edges       []universe.Edge
	History     []string
	Persisted   bool
	SaveErr     error
}

type ShiftCommand struct {
	Universe *universe.Universe
	Session  *exploration.Session
	Repo     universe.Repository
}

func (c *ShiftCommand) Execute() (*ShiftResult, error) {
	currentQ := c.Session.CurrentCoordinate.Quantum
	n := 0
	if len(currentQ) > 1 && currentQ[0] == 'Q' {
		fmt.Sscanf(currentQ[1:], "%d", &n)
	}
	nextN := n + 1
	nextQ := fmt.Sprintf("Q%d", nextN)

	destID := c.Session.NextQuantumID()

	if _, exists := c.Universe.GetLocation(destID); !exists {
		coord := c.Session.CurrentCoordinate
		coord.Quantum = nextQ
		currentName := locationName(c.Universe, c.Session.CurrentLocation)
		loc := universe.Location{
			ID:          destID,
			Name:        fmt.Sprintf("%s (%s)", currentName, nextQ),
			Description: fmt.Sprintf("A neighbouring quantum branch of %s. The surroundings are almost identical, but something is subtly different.", currentName),
			Coordinate:  coord,
		}
		c.Universe.AddLocation(loc)
		c.Universe.AddEdge(universe.Edge{
			From:        c.Session.CurrentLocation,
			To:          destID,
			Mode:        universe.QuantumShift,
			Cost:        universe.QuantumShiftCost,
			Description: fmt.Sprintf("Quantum shift to %s", nextQ),
		})
		c.Universe.AddEdge(universe.Edge{
			From:        destID,
			To:          c.Session.CurrentLocation,
			Mode:        universe.QuantumShift,
			Cost:        universe.QuantumShiftCost,
			Description: fmt.Sprintf("Quantum shift back to %s", currentQ),
		})
	}

	loc, _ := c.Universe.GetLocation(destID)
	c.Session.ShiftTo(loc)

	result := &ShiftResult{
		NextQuantum: nextQ,
		Location:    loc,
		Edges:       c.Universe.Edges[destID],
		History:     c.Session.TravelHistory,
	}

	if err := c.Repo.Save(c.Universe); err != nil {
		result.SaveErr = err
	} else {
		result.Persisted = true
	}

	return result, nil
}

func locationName(u *universe.Universe, id string) string {
	if loc, ok := u.GetLocation(id); ok && loc.Name != "" {
		return loc.Name
	}
	return id
}

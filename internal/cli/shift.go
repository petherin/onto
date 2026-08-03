package cli

import (
	"fmt"

	"github.com/petherin/onto/internal/reality"
)

func (a *App) Shift() string {
	currentQ := a.currentCoordinate.Quantum
	n := 0
	if len(currentQ) > 1 && currentQ[0] == 'Q' {
		fmt.Sscanf(currentQ[1:], "%d", &n)
	}
	nextN := n + 1
	nextQ := fmt.Sprintf("Q%d", nextN)

	destID := a.nextQuantumID()

	if _, exists := a.universe.GetLocation(destID); !exists {
		coord := a.currentCoordinate
		coord.Quantum = nextQ
		loc := reality.Location{
			ID:          destID,
			Name:        fmt.Sprintf("%s (%s)", a.displayName(a.currentLocation), nextQ),
			Description: fmt.Sprintf("A neighbouring quantum branch of %s. The surroundings are almost identical, but something is subtly different.", a.displayName(a.currentLocation)),
			Coordinate:  coord,
		}
		a.universe.AddLocation(loc)
		a.universe.AddEdge(reality.Edge{
			From:        a.currentLocation,
			To:          destID,
			Mode:        reality.QuantumShift,
			Cost:        QuantumShiftCost,
			Description: fmt.Sprintf("Quantum shift to %s", nextQ),
		})
		a.universe.AddEdge(reality.Edge{
			From:        destID,
			To:          a.currentLocation,
			Mode:        reality.QuantumShift,
			Cost:        QuantumShiftCost,
			Description: fmt.Sprintf("Quantum shift back to %s", currentQ),
		})
	}

	loc, _ := a.universe.GetLocation(destID)
	previous := a.currentLocation
	a.currentLocation = destID
	a.currentCoordinate = loc.Coordinate
	a.travelHistory = append(a.travelHistory, fmt.Sprintf("%s -> %s (quantum shift)", previous, destID))

	if err := a.saveConfig(); err != nil {
		return fmt.Sprintf("Shifting...\n\nQuantum branch entered: %s\n\nCurrent Location\n%s\n\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s\n\nWarning: failed to save config: %v",
			nextQ, loc.Name, loc.Description, a.formatPossibleJourneys(), a.formatTravelHistory(), err)
	}
	return fmt.Sprintf("Shifting...\n\nQuantum branch entered: %s\n\nCurrent Location\n%s\n\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s\n\nPersisted to %s",
		nextQ, loc.Name, loc.Description, a.formatPossibleJourneys(), a.formatTravelHistory(), dataFile())
}

// nextQuantumID returns the location ID that 'shift' would jump to from the current position.
func (a *App) nextQuantumID() string {
	n := 0
	q := a.currentCoordinate.Quantum
	if len(q) > 1 && q[0] == 'Q' {
		fmt.Sscanf(q[1:], "%d", &n)
	}
	return fmt.Sprintf("%s-q%d", a.currentLocation, n+1)
}

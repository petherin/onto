package universe

import "fmt"

// BranchQuantum creates a new quantum branch from the given location into the universe,
// adding bidirectional QuantumShift edges. It is idempotent — if destID already exists
// in the universe, it does nothing.
func BranchQuantum(u *Universe, fromID string, fromCoord Coordinate, fromName, destID, nextQ string) {
	if _, exists := u.GetLocation(destID); exists {
		return
	}

	coord := fromCoord
	coord.Quantum = nextQ

	u.AddLocation(Location{
		ID:          destID,
		Name:        fmt.Sprintf("%s (%s)", fromName, nextQ),
		Description: fmt.Sprintf("A neighbouring quantum branch of %s. The surroundings are almost identical, but something is subtly different.", fromName),
		Coordinate:  coord,
	})
	u.AddEdge(Edge{
		From:        fromID,
		To:          destID,
		Mode:        QuantumShift,
		Cost:        QuantumShiftCost,
		Description: fmt.Sprintf("Quantum shift to %s", nextQ),
	})
	u.AddEdge(Edge{
		From:        destID,
		To:          fromID,
		Mode:        QuantumShift,
		Cost:        QuantumShiftCost,
		Description: fmt.Sprintf("Quantum shift back to %s", fromCoord.Quantum),
	})
}

package universe

import "fmt"

// BranchQuantumService creates a new quantum branch from the given location
// into the aggregate, adding bidirectional QuantumShift edges. It is
// idempotent — if destID already exists in the aggregate, it does nothing.
func BranchQuantumService(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, nextQ string) {
	if _, exists := u.GetLocation(destID); exists {
		return
	}

	coord := fromCoord
	coord.Quantum = nextQ

	u.AddLocation(LocationEntity{
		ID:          destID,
		Name:        fmt.Sprintf("%s (%s)", fromName, nextQ),
		Description: fmt.Sprintf("A neighbouring quantum branch of %s. The surroundings are almost identical, but something is subtly different.", fromName),
		Coordinate:  coord,
	})

	// Mirror physical outgoing edges from the source so the branch is
	// immediately explorable on foot without needing further shifts.
	for _, e := range u.EdgesFrom(fromID) {
		if e.Mode.IsPhysical() {
			u.AddEdge(EdgeVO{From: destID, To: e.To, Mode: e.Mode, Distance: e.Distance, Cost: e.Cost, Description: e.Description})
		}
	}

	u.AddEdge(EdgeVO{
		From:        fromID,
		To:          destID,
		Mode:        QuantumShift,
		Cost:        QuantumShiftCost,
		Description: fmt.Sprintf("Quantum shift to %s", nextQ),
	})
	u.AddEdge(EdgeVO{
		From:        destID,
		To:          fromID,
		Mode:        QuantumShift,
		Cost:        QuantumShiftCost,
		Description: fmt.Sprintf("Quantum shift back to %s", fromCoord.Quantum),
	})
}

package universe

import "fmt"

// BranchQuantumService creates a new quantum branch from the given location
// into the aggregate, including a coordinate-matched physical subgraph.
func BranchQuantumService(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, nextQ string) {
	destCoord := fromCoord
	destCoord.Quantum = nextQ
	BranchContextualService(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("A neighbouring quantum branch of %s. The surroundings are almost identical, but something is subtly different.", fromName),
		ContextualTransitionSpec{
			Mode:               QuantumShift,
			Cost:               QuantumShiftCost,
			Label:              nextQ,
			ForwardDescription: fmt.Sprintf("Quantum shift to %s", nextQ),
			ReverseDescription: fmt.Sprintf("Quantum shift back to %s", fromCoord.Quantum),
		},
	)
}

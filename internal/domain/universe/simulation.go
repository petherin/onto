package universe

import "fmt"

// BranchSimulation creates a new simulation-depth branch from the given
// location into the aggregate, including a coordinate-matched physical
// subgraph. Entering is cheaper than exiting (see SimulationEntryCost and
// SimulationExitCost).
func BranchSimulation(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID string, nextDepth int) error {
	destCoord := fromCoord
	destCoord.Simulation = nextDepth
	label := fmt.Sprintf("sim:%d", nextDepth)
	return BranchContextual(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("A nested simulation of %s. The world here is computed — its rules can be rewritten, and the base layer is no longer directly reachable without finding an exit.", fromName),
		ContextualTransitionSpec{
			Mode:               SimulationEntry,
			Cost:               SimulationEntryCost,
			ReverseCost:        SimulationExitCost,
			Label:              label,
			ForwardDescription: fmt.Sprintf("Simulation entry to depth %d", nextDepth),
			ReverseDescription: fmt.Sprintf("Simulation exit to depth %d", fromCoord.Simulation),
		},
	)
}

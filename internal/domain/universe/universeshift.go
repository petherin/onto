package universe

import "fmt"

// BranchUniverse creates a new bubble-universe branch from the given location
// into the aggregate, including a coordinate-matched physical subgraph
// (Tegmark Level II: a parallel universe with different physical constants).
func BranchUniverse(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, nextU string) error {
	destCoord := fromCoord
	destCoord.Universe = nextU
	return BranchContextual(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("A parallel bubble universe of %s. The physical constants here are subtly or radically different.", fromName),
		ContextualTransitionSpec{
			Mode:               UniverseShift,
			Cost:               UniverseShiftCost,
			Label:              nextU,
			ForwardDescription: fmt.Sprintf("Universe shift to %s", nextU),
			ReverseDescription: fmt.Sprintf("Universe shift back to %s", fromCoord.Universe),
		},
	)
}

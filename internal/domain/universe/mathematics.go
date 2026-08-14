package universe

import "fmt"

// BranchMathematics creates a new mathematical-structure branch from the given
// location into the aggregate, including a coordinate-matched physical subgraph
// (Tegmark Level IV: a self-consistent formal system with different rules of
// mathematics / physics).
func BranchMathematics(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, nextM string) error {
	destCoord := fromCoord
	destCoord.Mathematics = nextM
	return BranchContextual(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("A neighbouring mathematical structure of %s. The formal rules of existence here are different — dimensions, logic, or physical law may not match the Classical frame.", fromName),
		ContextualTransitionSpec{
			Mode:               MathematicalShift,
			Cost:               MathematicalShiftCost,
			Label:              nextM,
			ForwardDescription: fmt.Sprintf("Mathematical structure shift to %s", nextM),
			ReverseDescription: fmt.Sprintf("Mathematical structure shift back to %s", fromCoord.Mathematics),
		},
	)
}

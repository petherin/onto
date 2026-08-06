package universe

import "fmt"

// BranchConsensusService creates a new consensus divergence from the given
// location, including a coordinate-matched physical subgraph.
func BranchConsensusService(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID string, nextLevel int) {
	destCoord := fromCoord
	destCoord.Consensus = nextLevel
	BranchContextualService(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("A reality diverged from shared consensus at depth %d. Its rules may no longer match the world you left behind.", nextLevel),
		ContextualTransitionSpec{
			Mode:               ConsensusShift,
			Cost:               ConsensusShiftCost,
			Label:              fmt.Sprintf("consensus %d", nextLevel),
			ForwardDescription: fmt.Sprintf("Drift to consensus divergence %d", nextLevel),
			ReverseDescription: fmt.Sprintf("Align with consensus divergence %d", fromCoord.Consensus),
		},
	)
}

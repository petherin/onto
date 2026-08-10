package universe

import (
	"fmt"
	"time"
)

// BranchTime creates a temporal branch at target, including a
// coordinate-matched physical subgraph.
func BranchTime(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID string, target time.Time) error {
	destCoord := fromCoord
	destCoord.Time = target.UTC()
	return BranchContextual(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("The same reality at %s.", target.UTC().Format(time.RFC3339)),
		ContextualTransitionSpec{
			Mode:               TimeShift,
			Cost:               TimeShiftCost,
			Label:              target.UTC().Format(time.RFC3339),
			ForwardDescription: fmt.Sprintf("Time shift to %s", target.UTC().Format(time.RFC3339)),
			ReverseDescription: fmt.Sprintf("Time shift back to %s", fromCoord.Time.UTC().Format(time.RFC3339)),
		},
	)
}

package universe

import "fmt"

// BranchTimelineService creates a new timeline branch from the given location
// into the aggregate, including a coordinate-matched physical subgraph.
func BranchTimelineService(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, nextT string) error {
	destCoord := fromCoord
	destCoord.Timeline = nextT
	return BranchContextualService(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("An alternate timeline branch of %s. History diverged at some point — the differences may be subtle or catastrophic.", fromName),
		ContextualTransitionSpec{
			Mode:               TimelineShift,
			Cost:               TimelineShiftCost,
			Label:              nextT,
			ForwardDescription: fmt.Sprintf("Timeline shift to %s", nextT),
			ReverseDescription: fmt.Sprintf("Timeline shift back to %s", fromCoord.Timeline),
		},
	)
}

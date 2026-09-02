package universe

import "fmt"

// BranchTimeline creates a new timeline branch from the given location
// into the aggregate, including a coordinate-matched physical subgraph.
//
// Unlike BranchQuantum, BranchUniverse, and BranchMathematics, this is not a
// different kind of reality (Tegmark Level I: same physics, same universe).
// It is ordinary travel to a real, unfathomably distant Hubble volume that
// happens to share this location's local geography but not its history —
// no amount of speed closes that gap, since you would still be crossing every
// metre of intervening space. It takes a jump drive that threads a wormhole
// straight to the target Hubble volume, which is why it is modeled as a
// dedicated jump rather than an extension of travel.
func BranchTimeline(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, nextT string) error {
	destCoord := fromCoord
	destCoord.Timeline = nextT
	return BranchContextual(u, fromID, destCoord, fromName, destID,
		fmt.Sprintf("A distant Hubble volume sharing %s's geography, reached only because something here can thread a wormhole to it. Its history diverged at some point — the differences may be subtle or catastrophic.", fromName),
		ContextualTransitionSpec{
			Mode:               TimelineShift,
			Cost:               TimelineShiftCost,
			Label:              nextT,
			ForwardDescription: fmt.Sprintf("Timeline shift to %s", nextT),
			ReverseDescription: fmt.Sprintf("Timeline shift back to %s", fromCoord.Timeline),
		},
	)
}

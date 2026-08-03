package universe

import "fmt"

// BranchTimeline creates a new timeline branch from the given location into the universe,
// adding bidirectional TimelineShift edges. It is idempotent — if destID already exists
// in the universe, it does nothing.
func BranchTimeline(u *Universe, fromID string, fromCoord Coordinate, fromName, destID, nextT string) {
	if _, exists := u.GetLocation(destID); exists {
		return
	}

	coord := fromCoord
	coord.Timeline = nextT
	coord.Quantum = "Q0" // reset quantum depth when entering a new timeline

	u.AddLocation(Location{
		ID:          destID,
		Name:        fmt.Sprintf("%s (%s)", fromName, nextT),
		Description: fmt.Sprintf("An alternate timeline branch of %s. History diverged at some point — the differences may be subtle or catastrophic.", fromName),
		Coordinate:  coord,
	})
	u.AddEdge(Edge{
		From:        fromID,
		To:          destID,
		Mode:        TimelineShift,
		Cost:        TimelineShiftCost,
		Description: fmt.Sprintf("Timeline shift to %s", nextT),
	})
	u.AddEdge(Edge{
		From:        destID,
		To:          fromID,
		Mode:        TimelineShift,
		Cost:        TimelineShiftCost,
		Description: fmt.Sprintf("Timeline shift back to %s", fromCoord.Timeline),
	})
}

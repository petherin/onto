package universe

import "fmt"

// BranchTimelineService creates a new timeline branch from the given location
// into the aggregate, adding bidirectional TimelineShift edges. It is
// idempotent — if destID already exists in the aggregate, it does nothing.
func BranchTimelineService(u *Aggregate, fromID string, fromCoord CoordinateVO, fromName, destID, nextT string) {
	if _, exists := u.GetLocation(destID); exists {
		return
	}

	coord := fromCoord
	coord.Timeline = nextT
	coord.Quantum = "Q0" // reset quantum depth when entering a new timeline

	u.AddLocation(LocationEntity{
		ID:          destID,
		Name:        fmt.Sprintf("%s (%s)", fromName, nextT),
		Description: fmt.Sprintf("An alternate timeline branch of %s. History diverged at some point — the differences may be subtle or catastrophic.", fromName),
		Coordinate:  coord,
	})

	// Mirror physical outgoing edges from the source so the branch is
	// immediately explorable on foot without needing further jumps.
	for _, e := range u.EdgesFrom(fromID) {
		if e.Mode.IsPhysical() {
			u.AddEdge(EdgeVO{From: destID, To: e.To, Mode: e.Mode, Distance: e.Distance, Cost: e.Cost, Description: e.Description})
		}
	}

	u.AddEdge(EdgeVO{
		From:        fromID,
		To:          destID,
		Mode:        TimelineShift,
		Cost:        TimelineShiftCost,
		Description: fmt.Sprintf("Timeline shift to %s", nextT),
	})
	u.AddEdge(EdgeVO{
		From:        destID,
		To:          fromID,
		Mode:        TimelineShift,
		Cost:        TimelineShiftCost,
		Description: fmt.Sprintf("Timeline shift back to %s", fromCoord.Timeline),
	})
}

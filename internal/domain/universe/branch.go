package universe

import (
	"fmt"
	"strings"
)

// ContextualTransitionSpec defines how one non-spatial transition creates a
// coordinate-matched branch of the physical graph.
type ContextualTransitionSpec struct {
	Mode               TravelModeVO
	Cost               float64
	Label              string
	ForwardDescription string
	ReverseDescription string
}

// BranchContextualService creates the branch root and materializes every
// reachable physical location in the destination context. Existing branches
// are backfilled, so saved graphs from older versions gain contextual routes.
func BranchContextualService(
	u *Aggregate,
	fromID string,
	destCoord CoordinateVO,
	fromName, destID, destinationDescription string,
	spec ContextualTransitionSpec,
) {
	if _, exists := u.GetLocation(destID); !exists {
		u.AddLocation(LocationEntity{
			ID:          destID,
			Name:        fmt.Sprintf("%s (%s)", fromName, spec.Label),
			Description: destinationDescription,
			Coordinate:  destCoord,
		})
	}
	materializePhysicalBranch(u, fromID, destID, destCoord, spec)
}

// materializePhysicalBranch copies the physical graph reachable from fromID
// into the destination reality context. Each copied location receives the
// branch suffix, keeping physical travel within the same reality.
func materializePhysicalBranch(
	u *Aggregate,
	fromID, destID string,
	destCoord CoordinateVO,
	spec ContextualTransitionSpec,
) {
	suffix := strings.TrimPrefix(destID, fromID)
	if suffix == "" {
		return
	}

	ids := map[string]string{fromID: destID}
	queue := []string{fromID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		currentBranchID := ids[currentID]

		addEdgeOnce(u, EdgeVO{
			From:        currentID,
			To:          currentBranchID,
			Mode:        spec.Mode,
			Cost:        spec.Cost,
			Description: spec.ForwardDescription,
		})
		addEdgeOnce(u, EdgeVO{
			From:        currentBranchID,
			To:          currentID,
			Mode:        spec.Mode,
			Cost:        spec.Cost,
			Description: spec.ReverseDescription,
		})

		for _, edge := range u.EdgesFrom(currentID) {
			if !edge.Mode.IsPhysical() {
				continue
			}

			targetID, exists := ids[edge.To]
			if !exists {
				targetID = edge.To + suffix
				ids[edge.To] = targetID
				queue = append(queue, edge.To)

				if target, ok := u.GetLocation(edge.To); ok {
					if _, alreadyExists := u.GetLocation(targetID); !alreadyExists {
						target.ID = targetID
						target.Name = fmt.Sprintf("%s (%s)", target.Name, spec.Label)
						target.Coordinate = withRealityContext(target.Coordinate, destCoord)
						u.AddLocation(target)
					}
				}
			}

			addEdgeOnce(u, EdgeVO{
				From:        currentBranchID,
				To:          targetID,
				Mode:        edge.Mode,
				Distance:    edge.Distance,
				Cost:        edge.Cost,
				Description: edge.Description,
			})
			addEdgeOnce(u, EdgeVO{
				From:        targetID,
				To:          currentBranchID,
				Mode:        edge.Mode,
				Distance:    edge.Distance,
				Cost:        edge.Cost,
				Description: fmt.Sprintf("Return via %s", edge.Description),
			})
		}
	}
}

func withRealityContext(coord, context CoordinateVO) CoordinateVO {
	coord.Meta = context.Meta
	coord.Mathematics = context.Mathematics
	coord.Universe = context.Universe
	coord.Timeline = context.Timeline
	coord.Quantum = context.Quantum
	coord.Simulation = context.Simulation
	coord.Consensus = context.Consensus
	coord.Observer = context.Observer
	coord.Time = context.Time
	return coord
}

func addEdgeOnce(u *Aggregate, edge EdgeVO) {
	for _, existing := range u.EdgesFrom(edge.From) {
		if existing.To == edge.To && existing.Mode == edge.Mode {
			return
		}
	}
	u.AddEdge(edge)
}

package universe

import (
	"fmt"
)

// ContextualTransitionSpec defines how one non-spatial transition creates a
// coordinate-matched branch of the physical graph.
type ContextualTransitionSpec struct {
	Mode               TravelModeVO
	Cost               float64
	ReverseCost        float64 // if zero, reverse edges use Cost
	Label              string
	ForwardDescription string
	ReverseDescription string
}

func (s ContextualTransitionSpec) reverseCost() float64 {
	if s.ReverseCost != 0 {
		return s.ReverseCost
	}
	return s.Cost
}

// BranchContextual creates the branch root and materializes every
// reachable physical location in the destination context. Existing branches
// are backfilled, so saved graphs from older versions gain contextual routes.
func BranchContextual(
	u *Aggregate,
	fromID string,
	destCoord CoordinateVO,
	fromName, destID, destinationDescription string,
	spec ContextualTransitionSpec,
) error {
	if _, exists := u.GetLocation(destID); !exists {
		if err := u.AddLocation(LocationEntity{
			ID:          destID,
			Name:        fmt.Sprintf("%s (%s)", fromName, spec.Label),
			Description: destinationDescription,
			Coordinate:  destCoord,
		}); err != nil {
			return err
		}
	}
	return materializePhysicalBranch(u, fromID, destID, destCoord, spec)
}

// materializePhysicalBranch copies the physical graph reachable from fromID
// into the destination reality context. Each copied location's ID is
// canonicalized onto the SAME axis that fromID→destID just changed (rather
// than string-concatenated), so neighbor IDs stay consistent and correctly
// ordered even if fromID or a neighbor already carries other reality-branch
// suffixes.
func materializePhysicalBranch(
	u *Aggregate,
	fromID, destID string,
	destCoord CoordinateVO,
	spec ContextualTransitionSpec,
) error {
	if destID == fromID {
		return nil
	}
	_, destAxes := parseLocationID(destID)

	ids := map[string]string{fromID: destID}
	queue := []string{fromID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		currentBranchID := ids[currentID]

		if err := addEdgeOnce(u, EdgeVO{
			From:        currentID,
			To:          currentBranchID,
			Mode:        spec.Mode,
			Cost:        spec.Cost,
			Description: spec.ForwardDescription,
		}); err != nil {
			return err
		}
		if err := addEdgeOnce(u, EdgeVO{
			From:        currentBranchID,
			To:          currentID,
			Mode:        spec.Mode,
			Cost:        spec.reverseCost(),
			Description: spec.ReverseDescription,
		}); err != nil {
			return err
		}

		var err error
		queue, err = mirrorPhysicalNeighbors(u, currentID, currentBranchID, queue, ids, spec, destAxes, destCoord)
		if err != nil {
			return err
		}
	}
	return nil
}

// mirrorPhysicalNeighbors mirrors every physical edge leaving currentID into the
// branched reality: each not-yet-seen neighbor is cloned into the destination
// context (new ID, labelled name, rewritten coordinate) and enqueued, then the
// edge is reproduced in both directions between the branched endpoints. It
// returns the queue with any newly discovered neighbors appended.
func mirrorPhysicalNeighbors(
	u *Aggregate,
	currentID, currentBranchID string,
	queue []string,
	ids map[string]string,
	spec ContextualTransitionSpec,
	destAxes axisSuffixes,
	destCoord CoordinateVO,
) ([]string, error) {
	for _, edge := range u.EdgesFrom(currentID) {
		if !edge.Mode.IsPhysical() {
			continue
		}

		targetID, exists := ids[edge.To]
		if !exists {
			targetID = neighborBranchID(edge.To, spec.Mode, destAxes)
			ids[edge.To] = targetID
			queue = append(queue, edge.To)

			if target, ok := u.GetLocation(edge.To); ok {
				if _, alreadyExists := u.GetLocation(targetID); !alreadyExists {
					target.ID = targetID
					target.Name = fmt.Sprintf("%s (%s)", target.Name, spec.Label)
					target.Coordinate = withRealityContext(target.Coordinate, destCoord)
					if err := u.AddLocation(target); err != nil {
						return queue, err
					}
				}
			}
		}

		if err := addEdgeOnce(u, EdgeVO{
			From:        currentBranchID,
			To:          targetID,
			Mode:        edge.Mode,
			Distance:    edge.Distance,
			Cost:        edge.Cost,
			Description: edge.Description,
		}); err != nil {
			return queue, err
		}
		if err := addEdgeOnce(u, EdgeVO{
			From:        targetID,
			To:          currentBranchID,
			Mode:        edge.Mode,
			Distance:    edge.Distance,
			Cost:        edge.Cost,
			Description: fmt.Sprintf("Return via %s", edge.Description),
		}); err != nil {
			return queue, err
		}
	}
	return queue, nil
}

// neighborBranchID computes a physically-reachable neighbor's ID in the
// destination reality context: it parses the neighbor's own base/axes and
// overwrites just the axis this branch operation changed (spec.Mode) with
// the destination's value for that axis, then reassembles canonically. This
// mirrors withRealityContext, which overwrites the same single axis on the
// neighbor's Coordinate.
func neighborBranchID(neighborID string, mode TravelModeVO, destAxes axisSuffixes) string {
	base, ax := parseLocationID(neighborID)
	switch mode {
	case QuantumShift:
		ax.quantum = destAxes.quantum
	case TimelineShift:
		ax.timeline = destAxes.timeline
	case ConsensusShift:
		ax.consensus = destAxes.consensus
	case TimeShift:
		ax.time = destAxes.time
	case ObserverShift:
		ax.observer = destAxes.observer
	case UniverseShift:
		ax.universe = destAxes.universe
	case MathematicalShift:
		ax.mathematics = destAxes.mathematics
	case SimulationEntry:
		ax.simulation = destAxes.simulation
	}
	return buildLocationID(base, ax)
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

func addEdgeOnce(u *Aggregate, edge EdgeVO) error {
	for _, existing := range u.EdgesFrom(edge.From) {
		if existing.To == edge.To && existing.Mode == edge.Mode {
			return nil
		}
	}
	return u.AddEdge(edge)
}

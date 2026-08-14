package universe

import "fmt"

// LowerContextID returns the canonical location ID one step down the given
// contextual axis. It fails for axes that are already at their base value, and
// for observer/time axes (those returns are edge-defined, not level-encoded).
func LowerContextID(id string, mode TravelModeVO) (string, bool) {
	base, ax := parseLocationID(id)
	switch mode {
	case QuantumShift:
		if ax.quantum <= 0 {
			return "", false
		}
		ax.quantum--
	case TimelineShift:
		if ax.timeline <= 0 {
			return "", false
		}
		ax.timeline--
	case UniverseShift:
		if ax.universe <= 0 {
			return "", false
		}
		ax.universe--
	case MathematicalShift:
		if ax.mathematics <= 0 {
			return "", false
		}
		ax.mathematics--
	case ConsensusShift:
		if ax.consensus <= 0 {
			return "", false
		}
		ax.consensus--
	case SimulationEntry:
		if ax.simulation <= 0 {
			return "", false
		}
		ax.simulation--
	default:
		return "", false
	}
	return buildLocationID(base, ax), true
}

// LowerContextCoordinate returns a copy of coord with the given contextual
// axis decreased by one level.
func LowerContextCoordinate(coord CoordinateVO, mode TravelModeVO) (CoordinateVO, bool) {
	switch mode {
	case QuantumShift:
		level := coord.QuantumLevel()
		if level <= 0 {
			return coord, false
		}
		coord.Quantum = fmt.Sprintf("Q%d", level-1)
	case TimelineShift:
		level := coord.TimelineLevel()
		if level <= 0 {
			return coord, false
		}
		if level == 1 {
			coord.Timeline = "Prime"
		} else {
			coord.Timeline = fmt.Sprintf("T%d", level-1)
		}
	case UniverseShift:
		level := coord.UniverseLevel()
		if level <= 0 {
			return coord, false
		}
		if level == 1 {
			coord.Universe = "Origin"
		} else {
			coord.Universe = fmt.Sprintf("U%d", level-1)
		}
	case MathematicalShift:
		level := coord.MathematicsLevel()
		if level <= 0 {
			return coord, false
		}
		if level == 1 {
			coord.Mathematics = "Classical"
		} else {
			coord.Mathematics = fmt.Sprintf("M%d", level-1)
		}
	case ConsensusShift:
		if coord.Consensus <= 0 {
			return coord, false
		}
		coord.Consensus--
	case SimulationEntry:
		if coord.Simulation <= 0 {
			return coord, false
		}
		coord.Simulation--
	default:
		return coord, false
	}
	return coord, true
}

// FindLowerContext locates the destination one contextual step down from
// fromID: prefer a reverse edge on mode, otherwise a pre-existing canonical
// lower-axis location ID.
func FindLowerContext(u *Aggregate, fromID string, mode TravelModeVO) (string, bool) {
	current, ok := u.GetLocation(fromID)
	if !ok {
		return "", false
	}
	for _, edge := range u.EdgesFrom(fromID) {
		if edge.Mode != mode {
			continue
		}
		dest, ok := u.GetLocation(edge.To)
		if !ok {
			continue
		}
		if isLowerContextCoord(current.Coordinate, dest.Coordinate, mode) {
			return dest.ID, true
		}
	}
	if lowerID, ok := LowerContextID(fromID, mode); ok {
		if _, exists := u.GetLocation(lowerID); exists {
			return lowerID, true
		}
	}
	return "", false
}

// EnsureLowerContext finds or creates the location one contextual step down
// from fromID and guarantees bidirectional edges on mode. This backfills
// reverse paths for locations generated inside a branch (e.g. nearby places
// spawned after a simulation entry) so return-home and *back commands work.
func EnsureLowerContext(u *Aggregate, fromID string, mode TravelModeVO) (string, error) {
	if destID, ok := FindLowerContext(u, fromID, mode); ok {
		if err := ensureContextPair(u, destID, fromID, mode); err != nil {
			return "", err
		}
		return destID, nil
	}

	current, ok := u.GetLocation(fromID)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownEdgeEndpoint, fromID)
	}
	lowerID, ok := LowerContextID(fromID, mode)
	if !ok {
		return "", fmt.Errorf("already at base context for %s", mode)
	}
	lowerCoord, ok := LowerContextCoordinate(current.Coordinate, mode)
	if !ok {
		return "", fmt.Errorf("already at base context for %s", mode)
	}

	if _, exists := u.GetLocation(lowerID); !exists {
		loc := LocationEntity{
			ID:          lowerID,
			Name:        current.Name,
			Description: current.Description,
			Coordinate:  lowerCoord,
		}
		if err := u.AddLocation(loc); err != nil {
			return "", err
		}
	}
	if err := ensureContextPair(u, lowerID, fromID, mode); err != nil {
		return "", err
	}
	return lowerID, nil
}

func isLowerContextCoord(from, to CoordinateVO, mode TravelModeVO) bool {
	switch mode {
	case QuantumShift:
		return to.QuantumLevel() < from.QuantumLevel()
	case TimelineShift:
		return to.TimelineLevel() < from.TimelineLevel()
	case UniverseShift:
		return to.UniverseLevel() < from.UniverseLevel()
	case MathematicalShift:
		return to.MathematicsLevel() < from.MathematicsLevel()
	case ConsensusShift:
		return to.Consensus < from.Consensus
	case SimulationEntry:
		return to.Simulation < from.Simulation
	case ObserverShift:
		return to.Observer != from.Observer
	case TimeShift:
		return !to.Time.Equal(from.Time)
	default:
		return false
	}
}

func ensureContextPair(u *Aggregate, lowerID, higherID string, mode TravelModeVO) error {
	forwardCost, reverseCost := contextPairCosts(mode)
	if err := addEdgeOnce(u, EdgeVO{
		From:        lowerID,
		To:          higherID,
		Mode:        mode,
		Cost:        forwardCost,
		Description: fmt.Sprintf("%s forward", mode),
	}); err != nil {
		return err
	}
	return addEdgeOnce(u, EdgeVO{
		From:        higherID,
		To:          lowerID,
		Mode:        mode,
		Cost:        reverseCost,
		Description: fmt.Sprintf("%s back", mode),
	})
}

func contextPairCosts(mode TravelModeVO) (forward, reverse float64) {
	switch mode {
	case QuantumShift:
		return QuantumShiftCost, QuantumShiftCost
	case TimelineShift:
		return TimelineShiftCost, TimelineShiftCost
	case UniverseShift:
		return UniverseShiftCost, UniverseShiftCost
	case MathematicalShift:
		return MathematicalShiftCost, MathematicalShiftCost
	case ConsensusShift:
		return ConsensusShiftCost, ConsensusShiftCost
	case SimulationEntry:
		return SimulationEntryCost, SimulationExitCost
	case ObserverShift:
		return ObserverShiftCost, ObserverShiftCost
	case TimeShift:
		return TimeShiftCost, TimeShiftCost
	default:
		return 0, 0
	}
}

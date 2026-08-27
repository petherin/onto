package universe

import (
	"fmt"
	"time"
)

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

	newlyCreated := false
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
		newlyCreated = true
	}
	if err := ensureContextPair(u, lowerID, fromID, mode); err != nil {
		return "", err
	}
	// A node generated inside a branch (e.g. a nearby place spawned after a
	// universe shift) has no counterpart in the parent reality, so the node just
	// created above would be isolated — stranding return-home with "no route:
	// home". Mirror fromID's physical edges onto the lower counterparts,
	// recursing so each neighbour's lower context is created and linked too. The
	// ensureContextPair call above adds the reverse edge before recursion, so a
	// bidirectional back-reference resolves via FindLowerContext and the
	// recursion terminates at already-connected parent-reality nodes.
	if newlyCreated {
		if err := mirrorPhysicalEdgesToLower(u, fromID, lowerID, func(neighborID string) (string, error) {
			return EnsureLowerContext(u, neighborID, mode)
		}); err != nil {
			return "", err
		}
	}
	return lowerID, nil
}

// mirrorPhysicalEdgesToLower reconstructs the lower context's physical
// connectivity by copying each of fromID's physical edges onto the lower
// counterparts, using ensureNeighbor to find (creating if absent) each
// neighbour's own lower counterpart first so the new node is not stranded. The
// fromID edge list is snapshotted so additions during recursion cannot disturb
// iteration. Numeric axes pass EnsureLowerContext; observer/time pass
// EnsureContextualReturn (which also threads the enclosing origin).
func mirrorPhysicalEdgesToLower(u *Aggregate, fromID, lowerID string, ensureNeighbor func(neighborID string) (string, error)) error {
	for _, edge := range append([]EdgeVO(nil), u.EdgesFrom(fromID)...) {
		if !edge.Mode.IsPhysical() {
			continue
		}
		neighborLowerID, err := ensureNeighbor(edge.To)
		if err != nil {
			return err
		}
		if err := mirrorEdgeBoth(u, lowerID, neighborLowerID, edge); err != nil {
			return err
		}
	}
	return nil
}

// mirrorEdgeBoth adds edge between aID and bID in both directions, preserving
// the original edge's mode, distance, cost, and description.
func mirrorEdgeBoth(u *Aggregate, aID, bID string, edge EdgeVO) error {
	if err := addEdgeOnce(u, EdgeVO{
		From:        aID,
		To:          bID,
		Mode:        edge.Mode,
		Distance:    edge.Distance,
		Cost:        edge.Cost,
		Description: edge.Description,
	}); err != nil {
		return err
	}
	return addEdgeOnce(u, EdgeVO{
		From:        bID,
		To:          aID,
		Mode:        edge.Mode,
		Distance:    edge.Distance,
		Cost:        edge.Cost,
		Description: edge.Description,
	})
}

// EnsureContextualReturn is the observer/time analogue of EnsureLowerContext.
// Observer and time returns are edge-defined — the enclosing value is not
// encoded in the ID or coordinate — so the enclosing value is taken from
// originID, the node the forward transition was made from (recorded on the
// session's context stack). It finds or creates the counterpart one step down at
// fromID's current physical position, wires the bidirectional <mode> edges with
// the same descriptions the Branch* functions use (so observeBack/timeReturn
// recognise them), and mirrors fromID's physical edges down so the counterpart
// is not stranded. This lets return-home unwind an observer/time branch even
// from a nearby dead-end spawned inside it.
func EnsureContextualReturn(u *Aggregate, fromID, originID string, mode TravelModeVO) (string, error) {
	from, ok := u.GetLocation(fromID)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownEdgeEndpoint, fromID)
	}
	if destID, found := FindLowerContext(u, fromID, mode); found {
		if err := ensureContextualReturnPair(u, destID, from, mode); err != nil {
			return "", err
		}
		return destID, nil
	}

	origin, ok := u.GetLocation(originID)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownEdgeEndpoint, originID)
	}
	base, ax := parseLocationID(fromID)
	_, originAx := parseLocationID(originID)
	lowerCoord := from.Coordinate
	switch mode {
	case ObserverShift:
		ax.observer = originAx.observer
		lowerCoord.Observer = origin.Coordinate.Observer
	case TimeShift:
		ax.time = originAx.time
		lowerCoord.Time = origin.Coordinate.Time
	default:
		return "", fmt.Errorf("EnsureContextualReturn only handles observer/time, got %s", mode)
	}
	lowerID := buildLocationID(base, ax)

	newlyCreated := false
	if _, exists := u.GetLocation(lowerID); !exists {
		if err := u.AddLocation(LocationEntity{
			ID:          lowerID,
			Name:        from.Name,
			Description: from.Description,
			Coordinate:  lowerCoord,
		}); err != nil {
			return "", err
		}
		newlyCreated = true
	}
	if err := ensureContextualReturnPair(u, lowerID, from, mode); err != nil {
		return "", err
	}
	if newlyCreated {
		if err := mirrorPhysicalEdgesToLower(u, fromID, lowerID, func(neighborID string) (string, error) {
			return EnsureContextualReturn(u, neighborID, originID, mode)
		}); err != nil {
			return "", err
		}
	}
	return lowerID, nil
}

// ensureContextualReturnPair adds the bidirectional observer/time edges between
// a lower (enclosing) node and the higher (current) node, using the exact
// descriptions BranchObserver/BranchTime emit so observeBack, timeReturn, and
// their plan lookups recognise the reverse edge.
func ensureContextualReturnPair(u *Aggregate, lowerID string, higher LocationEntity, mode TravelModeVO) error {
	lower, ok := u.GetLocation(lowerID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownEdgeEndpoint, lowerID)
	}
	var forwardDesc, reverseDesc string
	switch mode {
	case ObserverShift:
		forwardDesc = fmt.Sprintf("Observer shift to %s", higher.Coordinate.Observer)
		reverseDesc = fmt.Sprintf("Observer shift back to %s", lower.Coordinate.Observer)
	case TimeShift:
		forwardDesc = fmt.Sprintf("Time shift to %s", higher.Coordinate.Time.UTC().Format(time.RFC3339))
		reverseDesc = fmt.Sprintf("Time shift back to %s", lower.Coordinate.Time.UTC().Format(time.RFC3339))
	default:
		return fmt.Errorf("ensureContextualReturnPair only handles observer/time, got %s", mode)
	}
	forwardCost, reverseCost := contextPairCosts(mode)
	if err := addEdgeOnce(u, EdgeVO{From: lowerID, To: higher.ID, Mode: mode, Cost: forwardCost, Description: forwardDesc}); err != nil {
		return err
	}
	return addEdgeOnce(u, EdgeVO{From: higher.ID, To: lowerID, Mode: mode, Cost: reverseCost, Description: reverseDesc})
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

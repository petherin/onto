package universe

import "time"

// TransitionCost returns the minimal reality-axis transition cost to move from
// one coordinate to another. For each enumerable reality axis that differs it
// sums the number of steps times that axis's per-step cost, using the entry
// cost when descending deeper into a simulation and the exit cost when leaving.
// The observer and time axes are not numeric ladders, so a change of either
// costs a single flat shift. It covers every reality axis a quest objective can
// vary and be reached (quantum, timeline, universe, mathematics, simulation,
// consensus, observer, time), so par is exact for objectives that differ only on
// those axes — the default quest, the ONTO_OBJECTIVES pool, and any custom
// reality objective. It does not itself account for physical travel between
// locations (that cost depends on the location graph and is added separately by
// the facade's par calculation), and the Meta axis has no transition command so
// an objective differing on it cannot be reached at all.
func TransitionCost(from, to CoordinateVO) float64 {
	return axisStepCost(from.QuantumLevel(), to.QuantumLevel(), QuantumShiftCost, QuantumShiftCost) +
		axisStepCost(from.TimelineLevel(), to.TimelineLevel(), TimelineShiftCost, TimelineShiftCost) +
		axisStepCost(from.UniverseLevel(), to.UniverseLevel(), UniverseShiftCost, UniverseShiftCost) +
		axisStepCost(from.MathematicsLevel(), to.MathematicsLevel(), MathematicalShiftCost, MathematicalShiftCost) +
		axisStepCost(from.Simulation, to.Simulation, SimulationEntryCost, SimulationExitCost) +
		axisStepCost(from.Consensus, to.Consensus, ConsensusShiftCost, ConsensusShiftCost) +
		observerStepCost(from.Observer, to.Observer) +
		timeStepCost(from.Time, to.Time)
}

// observerStepCost costs a change of observer perspective: a single
// ObserverShiftCost when the two observers differ, zero when they match. The
// observer axis is a flat set of perspectives rather than a numeric ladder, so
// reaching any different observer is one shift.
func observerStepCost(from, to string) float64 {
	if from == to {
		return 0
	}
	return ObserverShiftCost
}

// timeStepCost costs a change of temporal coordinate: a single TimeShiftCost
// when the two times differ, zero when they match. Like the observer axis a time
// branch is a flat move (the time command costs the same regardless of how far
// it jumps), rather than a numeric ladder.
func timeStepCost(from, to time.Time) float64 {
	if from.Equal(to) {
		return 0
	}
	return TimeShiftCost
}

// axisStepCost costs moving from one integer level to another along a single
// axis: upCost per step when the level increases (deeper), downCost per step
// when it decreases.
func axisStepCost(from, to int, upCost, downCost float64) float64 {
	if to >= from {
		return float64(to-from) * upCost
	}
	return float64(from-to) * downCost
}

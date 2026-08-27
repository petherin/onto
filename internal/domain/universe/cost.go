package universe

// TransitionCost returns the minimal reality-axis transition cost to move from
// one coordinate to another. For each enumerable reality axis that differs it
// sums the number of steps times that axis's per-step cost, using the entry
// cost when descending deeper into a simulation and the exit cost when leaving.
// It covers the axes that objectives vary (quantum, timeline, universe,
// mathematics, simulation, consensus); it does not account for physical travel
// between locations, so it is exact for objectives that differ only on reality
// axes — as DefaultTarget does.
func TransitionCost(from, to CoordinateVO) float64 {
	return axisStepCost(from.QuantumLevel(), to.QuantumLevel(), QuantumShiftCost, QuantumShiftCost) +
		axisStepCost(from.TimelineLevel(), to.TimelineLevel(), TimelineShiftCost, TimelineShiftCost) +
		axisStepCost(from.UniverseLevel(), to.UniverseLevel(), UniverseShiftCost, UniverseShiftCost) +
		axisStepCost(from.MathematicsLevel(), to.MathematicsLevel(), MathematicalShiftCost, MathematicalShiftCost) +
		axisStepCost(from.Simulation, to.Simulation, SimulationEntryCost, SimulationExitCost) +
		axisStepCost(from.Consensus, to.Consensus, ConsensusShiftCost, ConsensusShiftCost)
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

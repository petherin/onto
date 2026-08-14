package universe

import "strings"

// TravelModeVO is a value object identifying how a traveller moves along an
// edge. Physical modes (Walk through Warp) are usable with the travel command;
// non-physical modes require dedicated commands (shift, jump, etc.).
type TravelModeVO string

// Travel mode constants ordered from most mundane to most exotic.
// Walk through Warp are physical modes usable with the travel command.
// The remaining modes require dedicated commands (shift, jump, etc.).
const (
	Walk              TravelModeVO = "walk"       // on foot
	Cycle             TravelModeVO = "cycle"      // by bicycle
	Drive             TravelModeVO = "drive"      // by road vehicle
	Rail              TravelModeVO = "rail"       // by train or metro
	Flight            TravelModeVO = "flight"     // by aircraft
	Orbit             TravelModeVO = "orbit"      // sub-orbital / space shuttle
	Warp              TravelModeVO = "warp"       // faster-than-light
	QuantumShift      TravelModeVO = "quantum"    // cross to an adjacent quantum branch
	TimelineShift     TravelModeVO = "timeline"   // cross to an adjacent timeline
	UniverseShift     TravelModeVO = "universe"   // cross to a parallel universe
	SimulationEntry   TravelModeVO = "simulation" // enter or exit a simulation layer
	ObserverShift     TravelModeVO = "observer"   // change observer perspective
	ConsensusShift    TravelModeVO = "consensus"  // enter or exit a consensus divergence
	TimeShift         TravelModeVO = "time"       // move to another point in time
	MathematicalShift TravelModeVO = "math"       // traverse a mathematical abstraction
)

// QuantumShiftCost is the cost of a single quantum branch jump.
const QuantumShiftCost = 20.0

// TimelineShiftCost is the cost of a single timeline jump.
const TimelineShiftCost = 800.0

// UniverseShiftCost is the cost of a single bubble-universe jump (Tegmark
// Level II). It is deliberately higher than TimelineShiftCost — crossing into
// a parallel universe with different physical constants is more fundamental
// than a timeline fork, but still below the mathematical layer.
const UniverseShiftCost = 5000.0

// MathematicalShiftCost is the cost of a single mathematical-structure jump
// (Tegmark Level IV). It is the most expensive implemented transition —
// leaving one formal system for another is more extreme than changing
// physical constants within the same mathematics.
const MathematicalShiftCost = 50000.0

// ConsensusShiftCost is the cost of entering or exiting a consensus divergence.
const ConsensusShiftCost = 5.0

// ObserverShiftCost is the cost of changing observer perspective.
const ObserverShiftCost = 2.0

// TimeShiftCost is the cost of changing the temporal coordinate.
const TimeShiftCost = 100.0

// SimulationEntryCost is the cost of entering one simulation layer deeper.
// The boundary is intentionally cheap to cross inward.
const SimulationEntryCost = 10.0

// SimulationExitCost is the cost of leaving one simulation layer.
// Exiting is harder than entering — you must find or construct a way out.
const SimulationExitCost = 50.0

// IsPhysical reports whether a TravelModeVO can be used with the travel command.
// Non-physical modes (quantum, timeline, etc.) require dedicated commands.
func (m TravelModeVO) IsPhysical() bool {
	switch m {
	case Walk, Cycle, Drive, Rail, Flight, Orbit, Warp:
		return true
	}
	return false
}

// IsKnown reports whether m is a supported travel mode.
func (m TravelModeVO) IsKnown() bool {
	switch m {
	case Walk, Cycle, Drive, Rail, Flight, Orbit, Warp,
		QuantumShift, TimelineShift, UniverseShift, SimulationEntry,
		ObserverShift, ConsensusShift, TimeShift, MathematicalShift:
		return true
	}
	return false
}

// EdgeVO is a value object representing a directional connection between two
// locations. Cost is used by the pathfinder; Distance is informational only.
type EdgeVO struct {
	From        string
	To          string
	Mode        TravelModeVO
	Distance    float64
	Cost        float64
	Description string
}

// IsObserverReturn reports whether edge returns to the previous observer
// perspective rather than entering a new one.
func (e EdgeVO) IsObserverReturn() bool {
	return e.Mode == ObserverShift && strings.HasPrefix(e.Description, "Observer shift back to ")
}

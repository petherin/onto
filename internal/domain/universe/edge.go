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
	MathematicalShift TravelModeVO = "math"       // traverse a mathematical abstraction
)

// QuantumShiftCost is the cost of a single quantum branch jump.
const QuantumShiftCost = 20.0

// TimelineShiftCost is the cost of a single timeline jump.
const TimelineShiftCost = 800.0

// ConsensusShiftCost is the cost of entering or exiting a consensus divergence.
const ConsensusShiftCost = 5.0

// ObserverShiftCost is the cost of changing observer perspective.
const ObserverShiftCost = 2.0

// IsPhysical reports whether a TravelModeVO can be used with the travel command.
// Non-physical modes (quantum, timeline, etc.) require dedicated commands.
func (m TravelModeVO) IsPhysical() bool {
	switch m {
	case Walk, Cycle, Drive, Rail, Flight, Orbit, Warp:
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

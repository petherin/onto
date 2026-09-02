package universe

import "strings"

// TravelModeVO is a value object identifying how a traveller moves along an
// edge. Physical modes (Walk through Warp) are usable with the travel command;
// non-physical modes require dedicated commands (shift, jump, etc.).
//
// TimelineShift is a special case: unlike Quantum/Universe/Mathematical shifts
// — which are not journeys through space at all (a branching outcome, a
// different set of physical constants, a different formal system) —
// a timeline is just an ordinary, extremely distant region of the *same*
// Level I universe (see BranchTimeline). The destination is real and the
// physics is unchanged, but the distance (many Hubble volumes) is far beyond
// anything even Warp could cross in finite time: no amount of speed helps,
// because you would still be traversing the intervening space. Reaching it
// takes a jump drive that threads a wormhole straight to the target Hubble
// volume — a shortcut through spacetime, not a faster crawl through it. It is
// grouped with the non-physical, dedicated-command modes here purely because
// that shortcut is required — not because the destination is a different kind
// of reality.
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
	QuantumShift      TravelModeVO = "quantum"    // cross to an adjacent quantum branch (not travel — a different outcome, same place)
	TimelineShift     TravelModeVO = "timeline"   // travel to a distant Hubble volume with a different history (needs a wormhole-threading jump drive)
	UniverseShift     TravelModeVO = "universe"   // cross to a parallel universe (not travel — different physical constants)
	SimulationEntry   TravelModeVO = "simulation" // enter or exit a simulation layer
	ObserverShift     TravelModeVO = "observer"   // change observer perspective
	ConsensusShift    TravelModeVO = "consensus"  // enter or exit a consensus divergence
	TimeShift         TravelModeVO = "time"       // move to another point in time
	MathematicalShift TravelModeVO = "math"       // traverse a mathematical abstraction (not travel — a different formal system)
)

// QuantumShiftCost is the cost of a single quantum branch jump.
const QuantumShiftCost = 20.0

// TimelineShiftCost is the cost of a single timeline jump — modeled as the
// energy/risk of threading a wormhole to a distant Hubble volume of the same
// Level I universe. It is ordinary travel in kind, just travel far beyond the
// reach of any implemented physical mode, made possible only by a shortcut
// through spacetime rather than a faster crossing of it.
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
// TimelineShift is conceptually still physical travel (see the TravelModeVO
// doc comment); it is excluded here only because the trip needs a
// wormhole-threading jump drive rather than any modeled physical mode, not
// because it changes anything other than location.
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

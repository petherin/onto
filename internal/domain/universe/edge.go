package universe

// TravelMode identifies how a traveller moves along an edge. Physical modes
// (Walk through Warp) are usable with the travel command; non-physical modes
// require dedicated commands (shift, jump, etc.).
type TravelMode string

const (
	Walk              TravelMode = "walk"        // on foot
	Cycle             TravelMode = "cycle"       // by bicycle
	Drive             TravelMode = "drive"       // by road vehicle
	Rail              TravelMode = "rail"        // by train or metro
	Flight            TravelMode = "flight"      // by aircraft
	Orbit             TravelMode = "orbit"       // sub-orbital / space shuttle
	Warp              TravelMode = "warp"        // faster-than-light
	QuantumShift      TravelMode = "quantum"     // cross to an adjacent quantum branch
	TimelineShift     TravelMode = "timeline"    // cross to an adjacent timeline
	UniverseShift     TravelMode = "universe"    // cross to a parallel universe
	SimulationEntry   TravelMode = "simulation"  // enter or exit a simulation layer
	ObserverShift     TravelMode = "observer"    // change observer perspective
	MathematicalShift TravelMode = "math"        // traverse a mathematical abstraction
)

// QuantumShiftCost is the cost of a single quantum branch jump.
const QuantumShiftCost = 20.0

// TimelineShiftCost is the cost of a single timeline jump.
const TimelineShiftCost = 800.0

// IsPhysical reports whether a TravelMode can be used with the travel command.
// Non-physical modes (quantum, timeline, etc.) require dedicated commands.
func (m TravelMode) IsPhysical() bool {
	switch m {
	case Walk, Cycle, Drive, Rail, Flight, Orbit, Warp:
		return true
	}
	return false
}

// Edge represents a directional connection between two locations. Cost is used
// by the pathfinder; Distance is informational only.
type Edge struct {
	From        string
	To          string
	Mode        TravelMode
	Distance    float64
	Cost        float64
	Description string
}

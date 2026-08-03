package universe

type TravelMode string

const (
	Walk              TravelMode = "walk"
	Cycle             TravelMode = "cycle"
	Drive             TravelMode = "drive"
	Rail              TravelMode = "rail"
	Flight            TravelMode = "flight"
	Orbit             TravelMode = "orbit"
	Warp              TravelMode = "warp"
	QuantumShift      TravelMode = "quantum"
	TimelineShift     TravelMode = "timeline"
	UniverseShift     TravelMode = "universe"
	SimulationEntry   TravelMode = "simulation"
	ObserverShift     TravelMode = "observer"
	MathematicalShift TravelMode = "math"
)

// QuantumShiftCost is the cost of a single quantum branch jump.
const QuantumShiftCost = 20.0

type Edge struct {
	From        string
	To          string
	Mode        TravelMode
	Distance    float64
	Cost        float64
	Description string
}

package reality

type TravelMode string

const (
	Walk              TravelMode = "walk"
	Cycle                        = "cycle"
	Drive                        = "drive"
	Rail                         = "rail"
	Flight                       = "flight"
	Orbit                        = "orbit"
	Warp                         = "warp"
	QuantumShift                 = "quantum"
	TimelineShift                = "timeline"
	UniverseShift                = "universe"
	SimulationEntry              = "simulation"
	ObserverShift                = "observer"
	MathematicalShift            = "math"
)

type Edge struct {
	From        string
	To          string
	Mode        TravelMode
	Distance    float64
	Cost        float64
	Description string
}

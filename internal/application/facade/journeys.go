package facade

import (
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/domain/universe"
)

// JourneyKind identifies what kind of journey a JourneyOption represents.
type JourneyKind int

// Journey kind constants classify each numbered option shown to the player.
const (
	JourneyTravel        JourneyKind = iota // physical travel to an adjacent location
	JourneyShift                            // advance to the next quantum branch
	JourneyShiftBack                        // return to the previous quantum branch
	JourneyJump                             // jump drive: thread a wormhole to a distant Hubble volume (timeline)
	JourneyJumpBack                         // return to the previous Hubble volume
	JourneyUniverse                         // shift to the next bubble universe
	JourneyUniverseBack                     // return to the previous bubble universe
	JourneyStructure                        // shift to the next mathematical structure
	JourneyStructureBack                    // return to the previous mathematical structure
	JourneySimulate                         // enter the next nested simulation layer
	JourneySimulateBack                     // exit one simulation layer toward base reality
	JourneyDrift                            // enter the next consensus divergence
	JourneyAlign                            // return one level toward shared consensus
	JourneyObserveBack                      // restore the previous observer perspective
	JourneyTimeBack                         // return through the temporal branch
)

// JourneyOption is a single numbered journey the player can take from the
// current location.
type JourneyOption struct {
	Kind        JourneyKind
	Target      string
	Description string
}

// JourneyOptions builds the ordered list of possible journeys from a set of
// outgoing edges. Exported so the CLI completer and REPL can reuse it. It is two
// flat switches over the fixed set of transition axes — detect which reverse
// affordances exist, then emit options cheapest-first — so funlen is silenced;
// collapsing the per-axis cases would need level accessors that do not exist and
// would obscure the deliberate cost ordering.
//
//nolint:funlen // flat per-axis switches over the fixed set of transition modes
func (a *App) JourneyOptions(edges []universe.EdgeVO) ([]JourneyOption, bool) {
	var options []JourneyOption
	hasReverseQuantum := false
	hasReverseTimeline := false
	hasReverseUniverse := false
	hasReverseMathematics := false
	hasReverseSimulation := false
	hasReverseConsensus := false
	hasReverseObserver := false
	hasReverseTime := false

	for _, edge := range edges {
		switch edge.Mode {
		case universe.Walk, universe.Cycle, universe.Drive, universe.Rail, universe.Flight, universe.Orbit, universe.Warp:
			dest, ok := a.univ.GetLocation(edge.To)
			if !ok || !a.session.Coordinate().SamePhysicalReality(dest.Coordinate) {
				continue
			}
			options = append(options, JourneyOption{
				Kind:        JourneyTravel,
				Target:      edge.To,
				Description: fmt.Sprintf("%s (%s, %.0f — travel %s)", a.locationName(edge.To), string(edge.Mode), edge.Cost, edge.To),
			})
		case universe.QuantumShift:
			if dest, ok := a.univ.GetLocation(edge.To); ok {
				if dest.Coordinate.QuantumLevel() < a.session.QuantumLevel() {
					hasReverseQuantum = true
				}
			}
		case universe.TimelineShift:
			if dest, ok := a.univ.GetLocation(edge.To); ok {
				if dest.Coordinate.TimelineLevel() < a.session.TimelineLevel() {
					hasReverseTimeline = true
				}
			}
		case universe.UniverseShift:
			if dest, ok := a.univ.GetLocation(edge.To); ok {
				if dest.Coordinate.UniverseLevel() < a.session.UniverseLevel() {
					hasReverseUniverse = true
				}
			}
		case universe.MathematicalShift:
			if dest, ok := a.univ.GetLocation(edge.To); ok {
				if dest.Coordinate.MathematicsLevel() < a.session.MathematicsLevel() {
					hasReverseMathematics = true
				}
			}
		case universe.SimulationEntry:
			if dest, ok := a.univ.GetLocation(edge.To); ok {
				if dest.Coordinate.Simulation < a.session.SimulationLevel() {
					hasReverseSimulation = true
				}
			}
		case universe.ConsensusShift:
			if dest, ok := a.univ.GetLocation(edge.To); ok {
				if dest.Coordinate.Consensus < a.session.ConsensusLevel() {
					hasReverseConsensus = true
				}
			}
		case universe.ObserverShift:
			if edge.IsObserverReturn() {
				hasReverseObserver = true
			}
		case universe.TimeShift:
			if strings.HasPrefix(edge.Description, "Time shift back to ") {
				hasReverseTime = true
			}
		}
	}

	// Contextual transitions are listed cheapest-first by forward cost, each
	// forward option immediately followed by its return: drift (5), simulate
	// (10), shift (20), jump (800), universe (5,000), structure (50,000). The
	// observer (2) and time (100) affordances are back-only here and trail the
	// branch group.
	options = append(options, JourneyOption{Kind: JourneyDrift, Description: fmt.Sprintf("%s (consensus divergence, %.0f — drift)", a.session.NextConsensusID(), universe.ConsensusShiftCost)})
	if hasReverseConsensus {
		options = append(options, JourneyOption{Kind: JourneyAlign, Description: "Return toward shared consensus (align)"})
	}
	options = append(options, JourneyOption{Kind: JourneySimulate, Description: fmt.Sprintf("%s (simulation, %.0f — simulate)", a.session.NextSimulationID(), universe.SimulationEntryCost)})
	if hasReverseSimulation {
		options = append(options, JourneyOption{Kind: JourneySimulateBack, Description: fmt.Sprintf("Exit one simulation layer (simulate back, %.0f)", universe.SimulationExitCost)})
	}
	options = append(options, JourneyOption{Kind: JourneyShift, Description: fmt.Sprintf("%s (quantum, %.0f — shift)", a.session.NextQuantumID(), universe.QuantumShiftCost)})
	if hasReverseQuantum {
		options = append(options, JourneyOption{Kind: JourneyShiftBack, Description: "Return to the previous quantum branch (shift back)"})
	}
	options = append(options, JourneyOption{Kind: JourneyJump, Description: fmt.Sprintf("%s (Hubble volume, %.0f — jump)", a.session.NextTimelineID(), universe.TimelineShiftCost)})
	if hasReverseTimeline {
		options = append(options, JourneyOption{Kind: JourneyJumpBack, Description: "Return to the previous Hubble volume (jump back)"})
	}
	options = append(options, JourneyOption{Kind: JourneyUniverse, Description: fmt.Sprintf("%s (universe, %.0f — universe)", a.session.NextUniverseID(), universe.UniverseShiftCost)})
	if hasReverseUniverse {
		options = append(options, JourneyOption{Kind: JourneyUniverseBack, Description: "Return to the previous bubble universe (universe back)"})
	}
	options = append(options, JourneyOption{Kind: JourneyStructure, Description: fmt.Sprintf("%s (mathematics, %.0f — structure)", a.session.NextMathematicsID(), universe.MathematicalShiftCost)})
	if hasReverseMathematics {
		options = append(options, JourneyOption{Kind: JourneyStructureBack, Description: "Return to the previous mathematical structure (structure back)"})
	}
	if hasReverseObserver {
		options = append(options, JourneyOption{Kind: JourneyObserveBack, Description: "Return to the previous observer perspective (observe back)"})
	}
	if hasReverseTime {
		options = append(options, JourneyOption{Kind: JourneyTimeBack, Description: "Return to the previous temporal branch (time back)"})
	}
	return options, true
}

// ExecuteJourney executes the one-based journey option currently shown by ls.
func (a *App) ExecuteJourney(number int) string {
	options, _ := a.JourneyOptions(a.univ.EdgesFrom(a.session.Location()))
	if number < 1 || number > len(options) {
		return fmt.Sprintf("No possible journey numbered %d. Use 'ls' to view available journeys.", number)
	}
	option := options[number-1]
	switch option.Kind {
	case JourneyTravel:
		return a.Travel(option.Target)
	case JourneyShift:
		return a.Shift()
	case JourneyShiftBack:
		return a.ShiftBack()
	case JourneyJump:
		return a.Jump()
	case JourneyJumpBack:
		return a.JumpBack()
	case JourneyUniverse:
		return a.Universe()
	case JourneyUniverseBack:
		return a.UniverseBack()
	case JourneyStructure:
		return a.Structure()
	case JourneyStructureBack:
		return a.StructureBack()
	case JourneySimulate:
		return a.Simulate()
	case JourneySimulateBack:
		return a.SimulateBack()
	case JourneyDrift:
		return a.Drift()
	case JourneyAlign:
		return a.Align()
	case JourneyObserveBack:
		return a.ObserveBack()
	case JourneyTimeBack:
		return a.TimeBack()
	default:
		return "Selected journey is unavailable."
	}
}

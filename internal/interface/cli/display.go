package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/universe"
)

// Where formats the full reality coordinate, possible journeys, and recent history.
func (a *App) Where() string {
	q := &queries.LookupQuery{Universe: a.universe, Session: a.session}
	r := q.Where()
	coord := r.Coordinate
	return fmt.Sprintf(
		"Reality Coordinate\n%s\n\nUniverse : %s\nTimeline : %s\nQuantum  : %s\nConsensus: %d\nPlanet   : %s\nCountry  : %s\nRegion   : %s\nCity     : %s\nLocation : %s\nObserver : %s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		coord.OntoAddress(),
		coord.Universe, coord.Timeline, coord.Quantum, coord.Consensus,
		coord.Planet, coord.Country, coord.Region, coord.City, coord.Location, coord.Observer,
		a.session.CumulativeCost(),
		a.formatEdges(r.Edges),
	)
}

func (a *App) formatDriftResult(r *commands.DriftResult) string {
	verb := "Consensus divergence entered"
	if r.Reversed {
		verb = "Shared consensus approached"
	}
	base := fmt.Sprintf("Drifting...\n\n%s: level %d\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.Consensus, r.Location.Description,
		a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
	return base
}

func (a *App) formatObserveResult(r *commands.ObserveResult) string {
	verb := "Observer perspective entered"
	if r.Reversed {
		verb = "Observer perspective restored"
	}

	base := fmt.Sprintf("Observing...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.Observer, r.Location.Description,
		a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
	return base
}

func (a *App) formatTimeResult(r *commands.TimeResult) string {
	verb := "Temporal branch entered"
	if r.Reversed {
		verb = "Temporal branch exited"
	}
	base := fmt.Sprintf("Time shifting...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.Time.Format(time.RFC3339), r.Location.Description,
		a.session.CumulativeCost(), a.formatEdges(r.Edges),
	)
	return base
}

// Look formats the name and description of the current location.
func (a *App) Look() string {
	q := &queries.LookupQuery{Universe: a.universe, Session: a.session}
	result, ok := q.Look()
	if !ok {
		return "Current location is unknown."
	}
	return fmt.Sprintf("%s\n\n%s", result.Name, result.Description)
}

// List formats the outgoing edges from the current location.
func (a *App) List() string {
	q := &queries.LookupQuery{Universe: a.universe, Session: a.session}
	r := q.List()
	return a.formatEdges(r.Edges)
}

func (a *App) formatTravelResult(r *commands.TravelResult) string {
	return fmt.Sprintf("%s\n\nArrived.\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		travelVerb(r.Path),
		a.session.CumulativeCost(),
		a.formatEdges(r.Edges),
	)
}

func (a *App) formatShiftResult(r *commands.ShiftResult) string {
	verb := "Quantum branch entered"
	if r.Reversed {
		verb = "Quantum branch exited"
	}
	base := fmt.Sprintf("Shifting...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.NextQuantum, r.Location.Description,
		a.session.CumulativeCost(),
		a.formatEdges(r.Edges),
	)
	return base
}

func (a *App) formatJumpResult(r *commands.JumpResult) string {
	verb := "Timeline branch entered"
	if r.Reversed {
		verb = "Timeline branch exited"
	}
	base := fmt.Sprintf("Jumping...\n\n%s: %s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s",
		verb, r.NextTimeline, r.Location.Description,
		a.session.CumulativeCost(),
		a.formatEdges(r.Edges),
	)
	return base
}

func (a *App) formatRouteResult(r *queries.RouteResult) string {
	var steps []string
	for _, edge := range r.Steps {
		steps = append(steps, fmt.Sprintf("%s (%s)", a.locationName(edge.To), string(edge.Mode)))
	}
	return fmt.Sprintf("Route\n%s\n\nDistance\n%.1f km\n\nTravel Cost\n%.0f",
		strings.Join(steps, "\n"), r.Distance, r.Cost)
}

func (a *App) formatEdges(edges []universe.EdgeVO) string {
	options, observerShiftAvailable := a.journeyOptions(edges)
	lines := make([]string, 0, len(options)+1)
	for i, option := range options {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, option.description))
	}
	if observerShiftAvailable {
		lines = append(lines, "- observer perspective (2 cost — observe <observer>)")
		lines = append(lines, "- temporal branch (100 cost — time <RFC3339>)")
	}
	return strings.Join(lines, "\n")
}

type journeyKind int

const (
	journeyTravel journeyKind = iota
	journeyShift
	journeyShiftBack
	journeyJump
	journeyJumpBack
	journeyDrift
	journeyAlign
	journeyObserveBack
	journeyTimeBack
)

type journeyOption struct {
	kind        journeyKind
	target      string
	description string
}

func (a *App) journeyOptions(edges []universe.EdgeVO) ([]journeyOption, bool) {
	var options []journeyOption
	hasReverseQuantum := false
	hasReverseTimeline := false
	hasReverseConsensus := false
	hasReverseObserver := false
	hasReverseTime := false

	for _, edge := range edges {
		switch edge.Mode {
		case universe.Walk, universe.Cycle, universe.Drive, universe.Rail, universe.Flight, universe.Orbit, universe.Warp:
			dest, ok := a.universe.GetLocation(edge.To)
			if !ok || !a.session.Coordinate().SamePhysicalReality(dest.Coordinate) {
				continue
			}
			options = append(options, journeyOption{
				kind:        journeyTravel,
				target:      edge.To,
				description: fmt.Sprintf("%s (%s, %.0f — travel %s)", a.locationName(edge.To), string(edge.Mode), edge.Cost, edge.To),
			})
		case universe.QuantumShift:
			// Don't list quantum edges as regular journeys — they need 'shift' or 'shift back'.
			if dest, ok := a.universe.GetLocation(edge.To); ok {
				if dest.Coordinate.QuantumLevel() < a.session.QuantumLevel() {
					hasReverseQuantum = true
				}
			}
		case universe.TimelineShift:
			// Don't list timeline edges as regular journeys — they need 'jump' or 'jump back'.
			if dest, ok := a.universe.GetLocation(edge.To); ok {
				if dest.Coordinate.TimelineLevel() < a.session.TimelineLevel() {
					hasReverseTimeline = true
				}
			}
		case universe.ConsensusShift:
			if dest, ok := a.universe.GetLocation(edge.To); ok {
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

	options = append(options, journeyOption{
		kind:        journeyShift,
		description: fmt.Sprintf("%s (quantum, %.0f — shift)", a.session.NextQuantumID(), universe.QuantumShiftCost),
	})
	if hasReverseQuantum {
		options = append(options, journeyOption{
			kind:        journeyShiftBack,
			description: "Return to the previous quantum branch (shift back)",
		})
	}
	options = append(options, journeyOption{
		kind:        journeyJump,
		description: fmt.Sprintf("%s (timeline, %.0f — jump)", a.session.NextTimelineID(), universe.TimelineShiftCost),
	})
	if hasReverseTimeline {
		options = append(options, journeyOption{
			kind:        journeyJumpBack,
			description: "Return to the previous timeline branch (jump back)",
		})
	}
	options = append(options, journeyOption{
		kind:        journeyDrift,
		description: fmt.Sprintf("%s (consensus divergence, %.0f — drift)", a.session.NextConsensusID(), universe.ConsensusShiftCost),
	})
	if hasReverseConsensus {
		options = append(options, journeyOption{
			kind:        journeyAlign,
			description: "Return toward shared consensus (align)",
		})
	}
	if hasReverseObserver {
		options = append(options, journeyOption{
			kind:        journeyObserveBack,
			description: "Return to the previous observer perspective (observe back)",
		})
	}
	if hasReverseTime {
		options = append(options, journeyOption{
			kind:        journeyTimeBack,
			description: "Return to the previous temporal branch (time back)",
		})
	}
	return options, true
}

// travelVerb picks a human-readable progress line based on the dominant mode in the path.
// It uses the most "exotic" mode, defined by the priority order in the order slice
// (Warp > Orbit > Flight > Rail > Drive > Cycle > Walk).
func travelVerb(path []universe.EdgeVO) string {
	order := []universe.TravelModeVO{
		universe.Warp, universe.Orbit, universe.Flight,
		universe.Rail, universe.Drive, universe.Cycle, universe.Walk,
	}
	modes := make(map[universe.TravelModeVO]bool, len(path))
	for _, e := range path {
		modes[e.Mode] = true
	}
	for _, m := range order {
		if modes[m] {
			switch m {
			case universe.Walk:
				return "Walking..."
			case universe.Cycle:
				return "Cycling..."
			case universe.Drive:
				return "Driving..."
			case universe.Rail:
				return "Taking the train..."
			case universe.Flight:
				return "Flying..."
			case universe.Orbit:
				return "Entering orbit..."
			case universe.Warp:
				return "Warping..."
			}
		}
	}
	return "Travelling..."
}

func (a *App) locationName(id string) string {
	if loc, ok := a.universe.GetLocation(id); ok && loc.Name != "" {
		return loc.Name
	}
	return id
}

func (a *App) routeUnavailableDiagnostics(target string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Route unavailable to %s.\n", target)
	fmt.Fprintf(&b, "Current location id: %s\n", a.session.Location())
	sl := startLocation()
	if _, ok := a.universe.GetLocation(sl); ok {
		fmt.Fprintf(&b, "%s is present in universe\n", sl)
	} else {
		fmt.Fprintf(&b, "%s is NOT present in universe\n", sl)
	}
	b.WriteString("Outgoing from current location:\n")
	for _, e := range a.universe.EdgesFrom(a.session.Location()) {
		fmt.Fprintf(&b, "- %s (%s)\n", e.To, e.Mode)
	}
	b.WriteString("\nKnown location IDs:\n")
	for _, id := range a.universe.AllLocationIDs() {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	return b.String()
}

package cli

import (
	"fmt"
	"strings"

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
		"Reality Coordinate\nUniverse : %s\nTimeline : %s\nQuantum  : %s\nPlanet   : %s\nCountry  : %s\nRegion   : %s\nCity     : %s\nLocation : %s\nObserver : %s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s\n\nRecent travel history\n%s",
		coord.Universe, coord.Timeline, coord.Quantum,
		coord.Planet, coord.Country, coord.Region, coord.City, coord.Location, coord.Observer,
		a.session.CumulativeCost,
		a.formatEdges(r.Edges),
		a.formatHistory(r.History),
	)
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
	base := fmt.Sprintf("%s\n\nArrived.\n\nCurrent Location\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s\n\nTravel history\n%s",
		travelVerb(r.Path),
		r.Location.Name,
		a.session.CumulativeCost,
		a.formatEdges(r.Edges),
		a.formatHistory(r.History),
	)
	if r.DeadEndHandled {
		if r.SaveErr != nil {
			return base + fmt.Sprintf(fmtSaveWarning, r.SaveErr)
		}
		return base + fmt.Sprintf("\n\nPersisted auto-generated route(s) to %s", dataFile())
	}
	return base
}

func (a *App) formatShiftResult(r *commands.ShiftResult) string {
	verb := "Quantum branch entered"
	if r.Reversed {
		verb = "Quantum branch exited"
	}
	base := fmt.Sprintf("Shifting...\n\n%s: %s\n\nCurrent Location\n%s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s\n\nTravel history\n%s",
		verb, r.NextQuantum, r.Location.Name, r.Location.Description,
		a.session.CumulativeCost,
		a.formatEdges(r.Edges),
		a.formatHistory(r.History),
	)
	if r.SaveErr != nil {
		return base + fmt.Sprintf(fmtSaveWarning, r.SaveErr)
	}
	return base + fmt.Sprintf("\n\nPersisted to %s", dataFile())
}

func (a *App) formatJumpResult(r *commands.JumpResult) string {
	verb := "Timeline branch entered"
	if r.Reversed {
		verb = "Timeline branch exited"
	}
	base := fmt.Sprintf("Jumping...\n\n%s: %s\n\nCurrent Location\n%s\n\n%s\n\nCumulative journey cost\n%.0f\n\nPossible journeys\n%s\n\nTravel history\n%s",
		verb, r.NextTimeline, r.Location.Name, r.Location.Description,
		a.session.CumulativeCost,
		a.formatEdges(r.Edges),
		a.formatHistory(r.History),
	)
	if r.SaveErr != nil {
		return base + fmt.Sprintf(fmtSaveWarning, r.SaveErr)
	}
	return base + fmt.Sprintf("\n\nPersisted to %s", dataFile())
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
	var lines []string
	hasReverseQuantum := false
	hasReverseTimeline := false

	for _, edge := range edges {
		switch edge.Mode {
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
		default:
			lines = append(lines, fmt.Sprintf("- %s (%s, %.0f cost)", a.locationName(edge.To), string(edge.Mode), edge.Cost))
		}
	}

	lines = append(lines, fmt.Sprintf("- %s (quantum, %.0f cost — use 'shift')", a.session.NextQuantumID(), universe.QuantumShiftCost))
	if hasReverseQuantum {
		lines = append(lines, "  (use 'shift back' to return to the previous quantum branch)")
	}
	lines = append(lines, fmt.Sprintf("- %s (timeline, %.0f cost — use 'jump')", a.session.NextTimelineID(), universe.TimelineShiftCost))
	if hasReverseTimeline {
		lines = append(lines, "  (use 'jump back' to return to the previous timeline branch)")
	}
	return strings.Join(lines, "\n")
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

func (a *App) formatHistory(history []string) string {
	if len(history) == 0 {
		return "None yet"
	}
	return strings.Join(history, "\n")
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
	fmt.Fprintf(&b, "Current location id: %s\n", a.session.CurrentLocation)
	sl := startLocation()
	if _, ok := a.universe.GetLocation(sl); ok {
		fmt.Fprintf(&b, "%s is present in universe\n", sl)
	} else {
		fmt.Fprintf(&b, "%s is NOT present in universe\n", sl)
	}
	b.WriteString("Outgoing from current location:\n")
	for _, e := range a.universe.EdgesFrom(a.session.CurrentLocation) {
		fmt.Fprintf(&b, "- %s (%s)\n", e.To, e.Mode)
	}
	b.WriteString("\nKnown location IDs:\n")
	for _, id := range a.universe.AllLocationIDs() {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	return b.String()
}

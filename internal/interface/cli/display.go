package cli

import (
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/universe"
)

func (a *App) Where() string {
	q := &queries.LookupQuery{Universe: a.universe, Session: a.session}
	r := q.Where()
	coord := r.Coordinate
	return fmt.Sprintf(
		"Reality Coordinate\nUniverse : %s\nTimeline : %s\nQuantum  : %s\nPlanet   : %s\nCountry  : %s\nRegion   : %s\nCity     : %s\nLocation : %s\nObserver : %s\n\nPossible journeys\n%s\n\nRecent travel history\n%s",
		coord.Universe, coord.Timeline, coord.Quantum,
		coord.Planet, coord.Country, coord.Region, coord.City, coord.Location, coord.Observer,
		a.formatEdges(r.Edges, r.NextQuantum),
		a.formatHistory(r.History),
	)
}

func (a *App) Look() string {
	q := &queries.LookupQuery{Universe: a.universe, Session: a.session}
	result, ok := q.Look()
	if !ok {
		return "Current location is unknown."
	}
	return fmt.Sprintf("%s\n\n%s", result.Name, result.Description)
}

func (a *App) List() string {
	q := &queries.LookupQuery{Universe: a.universe, Session: a.session}
	r := q.List()
	return a.formatEdges(r.Edges, r.NextQuantum)
}

func (a *App) formatTravelResult(r *commands.TravelResult, target string) string {
	base := fmt.Sprintf("Walking...\n\nArrived.\n\nCurrent Location\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s",
		r.Location.Name,
		a.formatEdges(r.Edges, a.session.NextQuantumID()),
		a.formatHistory(r.History),
	)
	if r.DeadEndHandled {
		if r.SaveErr != nil {
			return base + fmt.Sprintf("\n\nWarning: failed to save config: %v", r.SaveErr)
		}
		return base + fmt.Sprintf("\n\nPersisted auto-generated route(s) to %s", dataFile())
	}
	return base
}

func (a *App) formatShiftResult(r *commands.ShiftResult) string {
	base := fmt.Sprintf("Shifting...\n\nQuantum branch entered: %s\n\nCurrent Location\n%s\n\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s",
		r.NextQuantum, r.Location.Name, r.Location.Description,
		a.formatEdges(r.Edges, a.session.NextQuantumID()),
		a.formatHistory(r.History),
	)
	if r.SaveErr != nil {
		return base + fmt.Sprintf("\n\nWarning: failed to save config: %v", r.SaveErr)
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

func (a *App) formatEdges(edges []universe.Edge, nextQuantum string) string {
	var lines []string
	for _, edge := range edges {
		lines = append(lines, fmt.Sprintf("- %s (%s, %.0f cost)", a.locationName(edge.To), string(edge.Mode), edge.Cost))
	}
	lines = append(lines, fmt.Sprintf("- %s (quantum, %.0f cost — use 'shift')", nextQuantum, universe.QuantumShiftCost))
	return strings.Join(lines, "\n")
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
	b.WriteString(fmt.Sprintf("Route unavailable to %s.\n", target))
	b.WriteString(fmt.Sprintf("Current location id: %s\n", a.session.CurrentLocation))
	sl := startLocation()
	if _, ok := a.universe.GetLocation(sl); ok {
		b.WriteString(fmt.Sprintf("%s is present in universe\n", sl))
	} else {
		b.WriteString(fmt.Sprintf("%s is NOT present in universe\n", sl))
	}
	b.WriteString("Outgoing from current location:\n")
	for _, e := range a.universe.Edges[a.session.CurrentLocation] {
		b.WriteString(fmt.Sprintf("- %s (%s)\n", e.To, e.Mode))
	}
	b.WriteString("\nKnown location IDs:\n")
	for id := range a.universe.Locations {
		b.WriteString(fmt.Sprintf("- %s\n", id))
	}
	return b.String()
}

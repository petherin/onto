package cli

import (
	"fmt"
	"strings"
)

func (a *App) Where() string {
	coord := a.currentCoordinate
	return fmt.Sprintf(
		"Reality Coordinate\nUniverse : %s\nTimeline : %s\nQuantum  : %s\nPlanet   : %s\nCountry  : %s\nRegion   : %s\nCity     : %s\nLocation : %s\nObserver : %s\n\nPossible journeys\n%s\n\nRecent travel history\n%s",
		coord.Universe, coord.Timeline, coord.Quantum,
		coord.Planet, coord.Country, coord.Region, coord.City, coord.Location, coord.Observer,
		a.formatPossibleJourneys(), a.formatTravelHistory(),
	)
}

func (a *App) Look() string {
	location, ok := a.universe.GetLocation(a.currentLocation)
	if !ok {
		return "Current location is unknown."
	}
	return fmt.Sprintf("%s\n\n%s", location.Name, location.Description)
}

func (a *App) List() string {
	var lines []string
	for _, edge := range a.universe.Edges[a.currentLocation] {
		lines = append(lines, fmt.Sprintf("- %s", a.displayName(edge.To)))
	}
	lines = append(lines, fmt.Sprintf("- %s (quantum shift — use 'shift')", a.nextQuantumID()))
	return strings.Join(lines, "\n")
}

// displayName returns the human-readable name for a location ID by looking it
// up in the universe. Falls back to the raw ID for unknown locations.
func (a *App) displayName(id string) string {
	if loc, ok := a.universe.GetLocation(id); ok && loc.Name != "" {
		return loc.Name
	}
	return id
}

func (a *App) formatPossibleJourneys() string {
	var lines []string
	for _, edge := range a.universe.Edges[a.currentLocation] {
		lines = append(lines, fmt.Sprintf("- %s (%s, %.0f cost)", a.displayName(edge.To), string(edge.Mode), edge.Cost))
	}
	lines = append(lines, fmt.Sprintf("- %s (quantum, %.0f cost — use 'shift')", a.nextQuantumID(), QuantumShiftCost))
	return strings.Join(lines, "\n")
}

func (a *App) formatTravelHistory() string {
	if len(a.travelHistory) == 0 {
		return "None yet"
	}
	return strings.Join(a.travelHistory, "\n")
}

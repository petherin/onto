package cli

import (
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/reality"
)

func (a *App) Route(target string) string {
	norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
	if _, ok := a.universe.GetLocation(norm); !ok {
		if suggestion := a.suggestDestination(target); suggestion != "" {
			return fmt.Sprintf("Unknown destination: %s\n\nDid you mean '%s'?", target, suggestion)
		}
		return fmt.Sprintf("Route unavailable to %s.", target)
	}

	path, ok := a.universe.FindRoute(a.currentLocation, norm)
	if !ok {
		return a.routeUnavailableDiagnostics(target)
	}

	var steps []string
	for _, edge := range path {
		steps = append(steps, fmt.Sprintf("%s (%s)", a.displayName(edge.To), string(edge.Mode)))
	}

	return fmt.Sprintf("Route\n%s\n\nDistance\n%.1f km\n\nTravel Cost\n%.0f", strings.Join(steps, "\n"), pathDistance(path), pathCost(path))
}

func (a *App) Travel(target string) string {
	norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
	if _, ok := a.universe.GetLocation(norm); !ok {
		if suggestion := a.suggestDestination(target); suggestion != "" {
			return fmt.Sprintf("Unknown destination: %s\n\nDid you mean '%s'?", target, suggestion)
		}
		return fmt.Sprintf("Destination %s is unknown.", target)
	}

	_, ok := a.universe.FindRoute(a.currentLocation, norm)
	if !ok {
		return a.routeUnavailableDiagnostics(target)
	}

	location, ok := a.universe.GetLocation(norm)
	if !ok {
		return fmt.Sprintf("Destination %s is unknown.", target)
	}

	previous := a.currentLocation
	a.currentLocation = norm
	a.currentCoordinate = location.Coordinate
	a.travelHistory = append(a.travelHistory, fmt.Sprintf("%s -> %s", previous, norm))

	created := a.ensureOutgoing(norm, previous)

	if created {
		if err := a.saveConfig(); err != nil {
			return fmt.Sprintf("Walking...\n\nArrived.\n\nCurrent Location\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s\n\nWarning: failed to save config: %v", location.Name, a.formatPossibleJourneys(), a.formatTravelHistory(), err)
		}
		return fmt.Sprintf("Walking...\n\nArrived.\n\nCurrent Location\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s\n\nPersisted auto-generated route(s) to %s", location.Name, a.formatPossibleJourneys(), a.formatTravelHistory(), dataFile())
	}

	return fmt.Sprintf("Walking...\n\nArrived.\n\nCurrent Location\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s", location.Name, a.formatPossibleJourneys(), a.formatTravelHistory())
}

func (a *App) Cost() string {
	return "Travel cost is estimated and currently local-only."
}

// ensureOutgoing prompts or auto-generates a new location when the current one
// has no outgoing edges beyond the one leading back to cameFrom.
func (a *App) ensureOutgoing(id, cameFrom string) bool {
	for _, e := range a.universe.Edges[id] {
		if e.To != cameFrom {
			return false
		}
	}
	if a.interactiveReader != nil {
		return a.interactiveEnsureOutgoing(id)
	}
	return autoGenerateNearby(a, id)
}

func (a *App) routeUnavailableDiagnostics(target string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Route unavailable to %s.\n", target))
	b.WriteString(fmt.Sprintf("Current location id: %s\n", a.currentLocation))
	sl := startLocation()
	if _, ok := a.universe.GetLocation(sl); ok {
		b.WriteString(fmt.Sprintf("%s is present in universe\n", sl))
	} else {
		b.WriteString(fmt.Sprintf("%s is NOT present in universe\n", sl))
	}
	b.WriteString("Outgoing from current location:\n")
	for _, e := range a.universe.Edges[a.currentLocation] {
		b.WriteString(fmt.Sprintf("- %s (%s)\n", e.To, e.Mode))
	}
	b.WriteString("\nKnown location IDs:\n")
	for id := range a.universe.Locations {
		b.WriteString(fmt.Sprintf("- %s\n", id))
	}
	return b.String()
}

func pathDistance(path []reality.Edge) float64 {
	total := 0.0
	for _, edge := range path {
		total += edge.Distance
	}
	return total
}

func pathCost(path []reality.Edge) float64 {
	total := 0.0
	for _, edge := range path {
		total += edge.Cost
	}
	return total
}

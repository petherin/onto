package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/petherin/onto/internal/reality"
)

type App struct {
	universe          *reality.Universe
	currentLocation   string
	currentCoordinate reality.Coordinate
	travelHistory     []string
	interactiveReader *bufio.Reader
}

func NewApp() *App {
	universe := reality.NewUniverse()
	// attempt to load configuration from data/locations.json
	type cfg struct {
		Locations []reality.Location `json:"locations"`
		Edges     []reality.Edge     `json:"edges"`
	}

	var c cfg
	data, err := os.ReadFile("data/locations.json")
	if err == nil {
		if err := json.Unmarshal(data, &c); err == nil {
			for _, loc := range c.Locations {
				universe.AddLocation(loc)
			}
			for _, edge := range c.Edges {
				universe.AddEdge(edge)
			}
		}
	}

	// if config did not populate anything, fall back to a small default map
	if len(universe.Locations) == 0 {
		base := reality.NewCoordinate()

		home := reality.Location{
			ID:          "home",
			Name:        "Home",
			Description: "A quiet residential location.",
			Coordinate:  base,
		}

		station := reality.Location{
			ID:          "station",
			Name:        "Station",
			Description: "Leeds Station.",
			Coordinate:  coordinateFor("Station", base),
		}

		park := reality.Location{
			ID:          "park",
			Name:        "Park",
			Description: "A green public park.",
			Coordinate:  coordinateFor("Park", base),
		}

		cityCentre := reality.Location{
			ID:          "city-centre",
			Name:        "City Centre",
			Description: "The centre of town.",
			Coordinate:  coordinateFor("City Centre", base),
		}

		universe.AddLocation(home)
		universe.AddLocation(station)
		universe.AddLocation(park)
		universe.AddLocation(cityCentre)

		universe.AddEdge(reality.Edge{From: "home", To: "station", Mode: reality.Walk, Distance: 1.6, Cost: 1, Description: "Walk to the station"})
		universe.AddEdge(reality.Edge{From: "home", To: "park", Mode: reality.Walk, Distance: 0.8, Cost: 1, Description: "Walk to the park"})
		universe.AddEdge(reality.Edge{From: "station", To: "city-centre", Mode: reality.Rail, Distance: 2.0, Cost: 3, Description: "Take the rail line"})
		universe.AddEdge(reality.Edge{From: "city-centre", To: "home", Mode: reality.Walk, Distance: 2.4, Cost: 2, Description: "Walk home"})
	}

	// ensure a `home` location exists so users can always `route home` or `travel home`
	if _, ok := universe.GetLocation("home"); !ok {
		base := reality.NewCoordinate()
		home := reality.Location{
			ID:          "home",
			Name:        "Home",
			Description: "Home base (auto-added)",
			Coordinate:  base,
		}
		universe.AddLocation(home)
	}

	// choose initial location (prefer home)
	start := "home"
	if _, ok := universe.GetLocation(start); !ok {
		// pick any available location as fallback
		for id := range universe.Locations {
			start = id
			break
		}
	}

	loc, _ := universe.GetLocation(start)

	return &App{
		universe:          universe,
		currentLocation:   start,
		currentCoordinate: loc.Coordinate,
		travelHistory:     []string{},
	}
}

func coordinateFor(name string, base reality.Coordinate) reality.Coordinate {
	coord := base
	coord.Location = name
	coord.City = "Leeds"
	return coord
}

func (a *App) Execute(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return ""
	}

	// command is parts[0], args are the rest (join to support multi-word destinations)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	switch cmd {
	case "help":
		return a.Help()
	case "where":
		return a.Where()
	case "look":
		return a.Look()
	case "ls":
		return a.List()
	case "route":
		if args == "" {
			return "Usage: route <destination>"
		}
		return a.Route(args)
	case "travel":
		if args == "" {
			return "Usage: travel <destination>"
		}
		return a.Travel(args)
	case "cost":
		return a.Cost()
	case "exit":
		return "Goodbye."
	default:
		if suggestion := a.suggestCommand(parts[0]); suggestion != "" {
			return fmt.Sprintf("Unknown command: %s\n\nDid you mean '%s'?\n\n%s", parts[0], suggestion, a.Help())
		}
		return fmt.Sprintf("Unknown command: %s\n\n%s", parts[0], a.Help())
	}
}

func (a *App) Help() string {
	return strings.Join([]string{
		"Usage",
		"",
		"where                  Show your current reality coordinate",
		"look                  Describe your current location",
		"ls                    List nearby connected locations",
		"route <destination>   Plan a route to a known place",
		"travel <destination>  Move to a known place",
		"cost                  Show travel cost information",
		"exit                  Leave the CLI",
		"",
		"Example destinations:",
		"home, station, park, city-centre",
		"",
		"Example commands:",
		"route station",
		"travel station",
		"route park",
	}, "\n")
}

func (a *App) Where() string {
	coord := a.currentCoordinate
	return fmt.Sprintf("Reality Coordinate\nUniverse : %s\nTimeline : %s\nQuantum  : %s\nPlanet   : %s\nCountry  : %s\nRegion   : %s\nCity     : %s\nLocation : %s\nObserver : %s\n\nPossible journeys\n%s\n\nRecent travel history\n%s", coord.Universe, coord.Timeline, coord.Quantum, coord.Planet, coord.Country, coord.Region, coord.City, coord.Location, coord.Observer, a.formatPossibleJourneys(), a.formatTravelHistory())
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
		lines = append(lines, fmt.Sprintf("- %s", edge.To))
	}
	if len(lines) == 0 {
		return "No nearby locations."
	}
	return strings.Join(lines, "\n")
}

func (a *App) Route(target string) string {
	// normalize user input: allow 'city centre' to match 'city-centre'
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
		steps = append(steps, fmt.Sprintf("%s (%s)", displayName(edge.To), string(edge.Mode)))
	}

	return fmt.Sprintf("Route\n%s\n\nDistance\n%.1f km\n\nTravel Cost\n%.0f", strings.Join(steps, "\n"), pathDistance(path), pathCost(path))
}

func (a *App) Travel(target string) string {
	// normalize multi-word input to ID form
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

	// ensure a `home` location exists so users can always `route home` or `travel home`
	// attempt to auto-generate outgoing nodes; ensureOutgoing returns true when it created something
	created := a.ensureOutgoing(target)

	if created {
		if err := a.saveConfig(); err != nil {
			return fmt.Sprintf("Walking...\n\nArrived.\n\nCurrent Location\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s\n\nWarning: failed to save config: %v", location.Name, a.formatPossibleJourneys(), a.formatTravelHistory(), err)
		}
		// print confirmation to the interactive user
		return fmt.Sprintf("Walking...\n\nArrived.\n\nCurrent Location\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s\n\nPersisted auto-generated route(s) to data/locations.json", location.Name, a.formatPossibleJourneys(), a.formatTravelHistory())
	}

	return fmt.Sprintf("Walking...\n\nArrived.\n\nCurrent Location\n%s\n\nPossible journeys\n%s\n\nTravel history\n%s", location.Name, a.formatPossibleJourneys(), a.formatTravelHistory())
}

func (a *App) Cost() string {
	return "Travel cost is estimated and currently local-only."
}

func (a *App) Run() {
	fmt.Println("Onto Explorer v0.1")
	fmt.Println()
	fmt.Println("Type 'help' to see the available commands.")
	fmt.Println("You can navigate between: home, station, park, city-centre")
	fmt.Println()
	fmt.Println("Current Position")
	fmt.Println(a.Where())
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	a.interactiveReader = reader
	for {
		fmt.Print(Prompt())
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		response := a.Execute(strings.TrimSpace(input))
		if response == "Goodbye." {
			fmt.Println(response)
			break
		}
		if response != "" {
			fmt.Println(response)
		}
	}
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

func displayName(id string) string {
	switch id {
	case "home":
		return "Home"
	case "station":
		return "Station"
	case "park":
		return "Park"
	case "city-centre":
		return "City Centre"
	default:
		return id
	}
}

func (a *App) formatPossibleJourneys() string {
	var lines []string
	for _, edge := range a.universe.Edges[a.currentLocation] {
		lines = append(lines, fmt.Sprintf("- %s (%s, %.0f cost)", displayName(edge.To), string(edge.Mode), edge.Cost))
	}
	if len(lines) == 0 {
		return "None"
	}
	return strings.Join(lines, "\n")
}

func (a *App) formatTravelHistory() string {
	if len(a.travelHistory) == 0 {
		return "None yet"
	}
	return strings.Join(a.travelHistory, "\n")
}

func (a *App) suggestDestination(target string) string {
	if target == "" {
		return ""
	}

	best := ""
	bestDistance := 999
	// consider both IDs and display names for suggestions
	// prepare normalized target forms
	lowerTarget := strings.ToLower(target)
	compactTarget := strings.ReplaceAll(lowerTarget, " ", "")

	for id, loc := range a.universe.Locations {
		// compare against raw id
		distanceID := levenshteinDistance(lowerTarget, strings.ToLower(id))
		if distanceID < bestDistance {
			bestDistance = distanceID
			best = id
		}

		// compare against compact id (remove hyphens/underscores)
		compactID := strings.ReplaceAll(strings.ToLower(id), "-", "")
		compactID = strings.ReplaceAll(compactID, "_", "")
		distanceCompact := levenshteinDistance(compactTarget, compactID)
		if distanceCompact < bestDistance {
			bestDistance = distanceCompact
			best = id
		}

		// compare against display name
		distanceName := levenshteinDistance(lowerTarget, strings.ToLower(loc.Name))
		if distanceName < bestDistance {
			bestDistance = distanceName
			best = id
		}
	}

	// allow a slightly larger threshold for longer names
	if bestDistance <= 2 {
		return best
	}
	if bestDistance <= 3 && len(target) > 6 {
		return best
	}
	return ""
}

// ensureOutgoing auto-generates a nearby location and edge if the given
// location has no outgoing edges, to keep exploration possible.
func (a *App) ensureOutgoing(id string) bool {
	// if there are outgoing edges already, nothing to do
	edges := a.universe.Edges[id]
	if len(edges) > 0 {
		return false
	}

	if a.interactiveReader != nil {
		return a.interactiveEnsureOutgoing(id)
	}

	return autoGenerateNearby(a, id)
}

// saveConfig writes the current universe locations and edges to data/locations.json
// saveConfig moved to internal/cli/config.go

func (a *App) routeUnavailableDiagnostics(target string) string {
	// produce helpful debug info when a route cannot be found
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Route unavailable to %s.\n", target))
	b.WriteString(fmt.Sprintf("Current location id: %s\n", a.currentLocation))
	if _, ok := a.universe.GetLocation("home"); ok {
		b.WriteString("Home is present in universe\n")
	} else {
		b.WriteString("Home is NOT present in universe\n")
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

func (a *App) suggestCommand(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return ""
	}

	candidates := []string{"help", "where", "look", "ls", "route", "travel", "cost", "exit"}
	best := ""
	bestDistance := 999
	for _, candidate := range candidates {
		distance := levenshteinDistance(input, candidate)
		if distance < bestDistance {
			bestDistance = distance
			best = candidate
		}
	}

	if bestDistance <= 2 {
		return best
	}
	return ""
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min(values ...int) int {
	best := values[0]
	for _, value := range values[1:] {
		if value < best {
			best = value
		}
	}
	return best
}

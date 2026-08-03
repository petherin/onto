package cli

import (
	"bufio"
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
	universe, err := LoadUniverse(dataFile())
	if err != nil {
		// fall back to a small default map
		universe = reality.NewUniverse()
		base := reality.NewCoordinate()
		universe.AddLocation(reality.Location{ID: "home", Name: "Home", Description: "A quiet residential location.", Coordinate: base})
		universe.AddLocation(reality.Location{ID: "station", Name: "Station", Description: "Leeds Station.", Coordinate: coordinateFor("Station", base)})
		universe.AddLocation(reality.Location{ID: "park", Name: "Park", Description: "A green public park.", Coordinate: coordinateFor("Park", base)})
		universe.AddLocation(reality.Location{ID: "city-centre", Name: "City Centre", Description: "The centre of town.", Coordinate: coordinateFor("City Centre", base)})
		universe.AddEdge(reality.Edge{From: "home", To: "station", Mode: reality.Walk, Distance: 1.6, Cost: 1, Description: "Walk to the station"})
		universe.AddEdge(reality.Edge{From: "home", To: "park", Mode: reality.Walk, Distance: 0.8, Cost: 1, Description: "Walk to the park"})
		universe.AddEdge(reality.Edge{From: "station", To: "city-centre", Mode: reality.Rail, Distance: 2.0, Cost: 3, Description: "Take the rail line"})
		universe.AddEdge(reality.Edge{From: "city-centre", To: "home", Mode: reality.Walk, Distance: 2.4, Cost: 2, Description: "Walk home"})
	}

	// ensure the start location always exists so 'route' and 'travel' to it always work
	sl := startLocation()
	if _, ok := universe.GetLocation(sl); !ok {
		base := reality.NewCoordinate()
		universe.AddLocation(reality.Location{ID: sl, Name: sl, Description: "Start location (auto-added)", Coordinate: base})
	}

	start := startLocation()
	if _, ok := universe.GetLocation(start); !ok {
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
	case "shift":
		return a.Shift()
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
		"shift                 Jump to the nearest quantum branch of your current location",
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

func (a *App) Run() {
	fmt.Println(AppVersion)
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



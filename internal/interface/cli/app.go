package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/infrastructure/generator"
	infranav "github.com/petherin/onto/internal/infrastructure/navigation"
	"github.com/petherin/onto/internal/infrastructure/persistence"
)

type App struct {
	universe          *universe.Universe
	session           *exploration.Session
	repo              universe.Repository
	pathfinder        navigation.Pathfinder
	interactiveReader *bufio.Reader
}

func NewApp() *App {
	repo := persistence.NewJSONRepository(dataFile())
	u, err := repo.Load()
	if err != nil {
		u = buildDefaultUniverse()
	}

	sl := startLocation()
	if _, ok := u.GetLocation(sl); !ok {
		base := universe.NewCoordinate()
		u.AddLocation(universe.Location{ID: sl, Name: sl, Description: "Start location (auto-added)", Coordinate: base})
	}

	start := sl
	if _, ok := u.GetLocation(start); !ok {
		if ids := u.AllLocationIDs(); len(ids) > 0 {
			start = ids[0]
		}
	}

	loc, _ := u.GetLocation(start)
	return &App{
		universe:   u,
		session:    exploration.NewSession(start, loc.Coordinate),
		repo:       repo,
		pathfinder: infranav.NewBFSPathfinder(),
	}
}

func (a *App) Execute(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	parts := strings.Fields(trimmed)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	switch cmd {
	case cmdHelp:
		return a.Help()
	case cmdWhere:
		return a.Where()
	case cmdLook:
		return a.Look()
	case cmdList:
		return a.List()
	case cmdRoute:
		if args == "" {
			return "Usage: route <destination>"
		}
		return a.Route(args)
	case cmdTravel:
		if args == "" {
			return "Usage: travel <destination>"
		}
		return a.Travel(args)
	case cmdCost:
		return a.Cost()
	case cmdShift:
		if args == argBack {
			return a.ShiftBack()
		}
		return a.Shift()
	case cmdExit:
		return msgGoodbye
	default:
		if suggestion := a.suggestCommand(cmd); suggestion != "" {
			return fmt.Sprintf("Unknown command: %s\n\nDid you mean '%s'?\n\n%s", cmd, suggestion, a.Help())
		}
		return fmt.Sprintf("Unknown command: %s\n\n%s", cmd, a.Help())
	}
}

func (a *App) Help() string {
	return strings.Join([]string{
		"Usage",
		"",
		"where                  Show your current reality coordinate",
		"look                   Describe your current location",
		"ls                     List nearby connected locations",
		"route <destination>    Plan a route to a known place",
		"travel <destination>   Move to a known place",
		"cost                   Show travel cost information",
		"shift                  Jump forward to the next quantum branch",
		"shift back             Return to the previous quantum branch",
		"exit                   Leave the CLI",
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

func (a *App) Travel(target string) string {
	var handler universe.LocationGenerator
	if a.interactiveReader != nil {
		handler = &InteractiveHandler{reader: a.interactiveReader, gen: generator.New()}
	} else {
		handler = generator.New()
	}

	cmd := &commands.TravelCommand{
		Universe:       a.universe,
		Session:        a.session,
		Repo:           a.repo,
		Pathfinder:     a.pathfinder,
		DeadEndHandler: handler,
	}

	result, err := cmd.Execute(target)
	if err != nil {
		if suggestion := a.suggestDestination(target); suggestion != "" {
			return fmt.Sprintf(fmtUnknownDestSuggest, target, suggestion)
		}
		return a.routeUnavailableDiagnostics(target)
	}

	return a.formatTravelResult(result, target)
}

func (a *App) Shift() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Repo: a.repo}
	result, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Shift failed: %v", err)
	}
	return a.formatShiftResult(result)
}

func (a *App) ShiftBack() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
	result, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Cannot shift back: %v", err)
	}
	return a.formatShiftResult(result)
}

func (a *App) Route(target string) string {
	q := &queries.RouteQuery{Universe: a.universe, Session: a.session, Pathfinder: a.pathfinder}
	result, err := q.Execute(target)
	if err != nil {
		if suggestion := a.suggestDestination(target); suggestion != "" {
			return fmt.Sprintf(fmtUnknownDestSuggest, target, suggestion)
		}
		return fmt.Sprintf("Route unavailable to %s.", target)
	}
	return a.formatRouteResult(result)
}

func (a *App) Cost() string {
	return "Travel cost is estimated and currently local-only."
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
		if response == msgGoodbye {
			fmt.Println(response)
			break
		}
		if response != "" {
			fmt.Println(response)
		}
	}
}

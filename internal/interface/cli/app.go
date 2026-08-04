// Package cli is the delivery mechanism for the Onto application. It owns the
// terminal run loop, command dispatch, result formatting, and user-facing
// helpers such as fuzzy command/destination suggestion and the interactive
// dead-end handler. The domain and application layers have no knowledge that
// this package exists.
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

// App is the top-level CLI object. It holds the wired-up universe, session,
// repository, and pathfinder and exposes one method per user command.
type App struct {
	universe          *universe.Aggregate
	session           *exploration.Entity
	repo              universe.Repository
	pathfinder        navigation.PathfinderService
	interactiveReader *bufio.Reader
}

// NewApp loads (or initialises) the universe from disk and returns a ready App.
func NewApp() *App {
	repo := persistence.NewJSONRepository(dataFile())
	u, err := repo.Load()
	if err != nil {
		u = buildDefaultUniverse()
	}

	sl := startLocation()
	if _, ok := u.GetLocation(sl); !ok {
		base := universe.DefaultCoordinateVO()
		u.AddLocation(universe.LocationEntity{ID: sl, Name: sl, Description: "Start location (auto-added)", Coordinate: base})
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
		session:    exploration.NewEntity(start, loc.Coordinate),
		repo:       repo,
		pathfinder: infranav.NewBFSPathfinder(),
	}
}

// Execute dispatches a raw input string to the appropriate command handler and
// returns the formatted output string.
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
	case cmdHome:
		return a.GoHome()
	case cmdCost:
		return a.Cost()
	case cmdShift:
		if args == argBack {
			return a.ShiftBack()
		}
		return a.Shift()
	case cmdJump:
		if args == argBack {
			return a.JumpBack()
		}
		return a.Jump()
	case cmdExit:
		return msgGoodbye
	default:
		if suggestion := a.suggestCommand(cmd); suggestion != "" {
			return fmt.Sprintf("Unknown command: %s\n\nDid you mean '%s'?\n\n%s", cmd, suggestion, a.Help())
		}
		return fmt.Sprintf("Unknown command: %s\n\n%s", cmd, a.Help())
	}
}

// Help returns the usage text listing all available commands.
func (a *App) Help() string {
	return strings.Join([]string{
		"Usage",
		"",
		"where                  Show your current reality coordinate",
		"look                   Describe your current location",
		"ls                     List nearby connected locations",
		"route <destination>    Plan a route to a known place",
		"travel <destination>   Move to a known place",
		"home                   Return home (jumps back timelines, shifts back quantum, then travels)",
		"cost                   Show travel cost information",
		"shift                  Jump forward to the next quantum branch",
		"shift back             Return to the previous quantum branch",
		"jump                   Jump forward to the next timeline branch",
		"jump back              Return to the previous timeline branch",
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

// Travel attempts physical movement to target, reporting the specific reason if
// the destination exists but cannot be reached (e.g. across a quantum boundary).
func (a *App) Travel(target string) string {
	var handler universe.LocationGeneratorService
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
		norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
		if _, known := a.universe.GetLocation(norm); !known {
			// Destination not found — try a fuzzy suggestion.
			if suggestion := a.suggestDestination(target); suggestion != "" {
				return fmt.Sprintf(fmtUnknownDestSuggest, target, suggestion)
			}
			return a.routeUnavailableDiagnostics(target)
		}
		// Destination exists but is unreachable — surface the real reason.
		return err.Error()
	}

	return a.formatTravelResult(result)
}

// Shift jumps the session forward to the next quantum branch of the current location.
func (a *App) Shift() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Repo: a.repo}
	result, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Shift failed: %v", err)
	}
	return a.formatShiftResult(result)
}

// ShiftBack returns the session to the previous quantum branch.
func (a *App) ShiftBack() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
	result, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Cannot shift back: %v", err)
	}
	return a.formatShiftResult(result)
}

// Jump moves the session forward to the next timeline branch of the current location.
func (a *App) Jump() string {
	cmd := &commands.JumpCommand{Universe: a.universe, Session: a.session, Repo: a.repo}
	result, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Jump failed: %v", err)
	}
	return a.formatJumpResult(result)
}

// JumpBack returns the session to the previous timeline branch.
func (a *App) JumpBack() string {
	cmd := &commands.JumpCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
	result, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Cannot jump back: %v", err)
	}
	return a.formatJumpResult(result)
}

// GoHome returns the session to the start location, unwinding timeline jumps
// then quantum shifts then travelling physically. It first shows the plan and
// estimated cost and asks for confirmation before doing anything.
func (a *App) GoHome() string {
	if a.session.CurrentLocation == defaultStartLocation &&
		a.session.QuantumLevel() == 0 &&
		a.session.TimelineLevel() == 0 {
		return "You are already home."
	}

	// --- Build the plan (read-only, nothing moves yet) ---
	var planLines []string
	estimatedCost := 0.0

	tl := a.session.TimelineLevel()
	for i := tl; i > 0; i-- {
		planLines = append(planLines, fmt.Sprintf("  jump back  (timeline %s → T%d)  cost %.0f", fmt.Sprintf("T%d", i), i-1, universe.TimelineShiftCost))
		estimatedCost += universe.TimelineShiftCost
	}

	ql := a.session.QuantumLevel()
	for i := ql; i > 0; i-- {
		planLines = append(planLines, fmt.Sprintf("  shift back (quantum Q%d → Q%d)  cost %.0f", i, i-1, universe.QuantumShiftCost))
		estimatedCost += universe.QuantumShiftCost
	}

	// Estimate physical leg cost via the route query (doesn't move the session).
	if a.session.CurrentLocation != defaultStartLocation {
		q := &queries.RouteQuery{Universe: a.universe, Session: a.session, Pathfinder: a.pathfinder}
		if rr, err := q.Execute(defaultStartLocation); err == nil {
			for _, edge := range rr.Steps {
				planLines = append(planLines, fmt.Sprintf("  travel     %s → %s (%s)  cost %.0f", a.locationName(edge.From), a.locationName(edge.To), edge.Mode, edge.Cost))
			}
			estimatedCost += rr.Cost
		} else {
			planLines = append(planLines, fmt.Sprintf("  travel     home (route unavailable: %v)", err))
		}
	}

	plan := strings.Join(planLines, "\n")

	// Print the plan and prompt directly (same pattern as InteractiveHandler).
	fmt.Printf("\nRoute home\n%s\n\nEstimated cost: %.0f\n\nProceed? [y/N]: ", plan, estimatedCost)

	if a.interactiveReader != nil {
		line, _ := a.interactiveReader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(line)) != "y" {
			return "Cancelled."
		}
	}

	// --- Execute the plan ---
	var steps []string

	for a.session.TimelineLevel() > 0 {
		cmd := &commands.JumpCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
		result, err := cmd.Execute()
		if err != nil {
			return fmt.Sprintf("Failed while jumping back to base timeline: %v\n\nSteps taken so far:\n%s", err, strings.Join(steps, "\n"))
		}
		steps = append(steps, fmt.Sprintf("Timeline jump back → %s at %s", result.NextTimeline, result.Location.Name))
	}

	for a.session.QuantumLevel() > 0 {
		cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
		result, err := cmd.Execute()
		if err != nil {
			return fmt.Sprintf("Failed while shifting back to base quantum: %v\n\nSteps taken so far:\n%s", err, strings.Join(steps, "\n"))
		}
		steps = append(steps, fmt.Sprintf("Quantum shift back → %s at %s", result.NextQuantum, result.Location.Name))
	}

	if a.session.CurrentLocation != defaultStartLocation {
		travelResult := a.Travel(defaultStartLocation)
		steps = append(steps, fmt.Sprintf("Travelled home → %s", defaultStartLocation))
		return fmt.Sprintf("Heading home...\n\nSteps taken\n%s\n\n%s", strings.Join(steps, "\n"), travelResult)
	}

	return fmt.Sprintf("Heading home...\n\nSteps taken\n%s\n\nYou are home.\n\nCumulative journey cost\n%.0f", strings.Join(steps, "\n"), a.session.CumulativeCost)
}

// Route plans and displays a route to target without moving the session.
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

// Cost returns informational text about how travel costs are calculated.
func (a *App) Cost() string {
	return "Travel cost is estimated and currently local-only."
}

// Run starts the interactive read-eval-print loop, displaying the welcome
// message and current position before reading lines from stdin.
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
		fmt.Print(a.Prompt())
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

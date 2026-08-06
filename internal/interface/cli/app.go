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

	"github.com/chzyer/readline"
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
		// GoHome returns the plan string; confirmation and execution are handled
		// by the run loops (Run / runPlain) to keep I/O out of this method.
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
	case cmdDrift:
		return a.Drift()
	case cmdAlign:
		return a.Align()
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
		"drift                  Enter the next consensus divergence",
		"align                  Return one level toward shared consensus",
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

	result, saveErr := cmd.Execute(target)
	if result == nil {
		// Domain error — no movement occurred.
		norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
		if _, known := a.universe.GetLocation(norm); !known {
			// Check if the target is the next quantum or timeline branch (not yet
			// created — it only exists after 'shift' or 'jump' runs).
			if norm == a.session.NextQuantumID() {
				return fmt.Sprintf("%s is a quantum branch — use 'shift' to enter it", target)
			}
			if norm == a.session.NextTimelineID() {
				return fmt.Sprintf("%s is a timeline branch — use 'jump' to enter it", target)
			}
			// Destination not found — try a fuzzy suggestion.
			if suggestion := a.suggestDestination(target); suggestion != "" {
				return fmt.Sprintf(fmtUnknownDestSuggest, target, suggestion)
			}
			return a.routeUnavailableDiagnostics(target)
		}
		// Destination exists but is unreachable — surface the real reason.
		return saveErr.Error()
	}

	// Travel succeeded; saveErr is non-nil only when auto-generated routes
	// could not be persisted — the formatter appends a warning in that case.
	return a.formatTravelResult(result, saveErr)
}

// Shift jumps the session forward to the next quantum branch of the current location.
func (a *App) Shift() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Repo: a.repo}
	result, saveErr := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Shift failed: %v", saveErr)
	}
	return a.formatShiftResult(result, saveErr)
}

// ShiftBack returns the session to the previous quantum branch.
func (a *App) ShiftBack() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
	result, saveErr := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot shift back: %v", saveErr)
	}
	return a.formatShiftResult(result, saveErr)
}

// Jump moves the session forward to the next timeline branch of the current location.
func (a *App) Jump() string {
	cmd := &commands.JumpCommand{Universe: a.universe, Session: a.session, Repo: a.repo}
	result, saveErr := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Jump failed: %v", saveErr)
	}
	return a.formatJumpResult(result, saveErr)
}

// JumpBack returns the session to the previous timeline branch.
func (a *App) JumpBack() string {
	cmd := &commands.JumpCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
	result, saveErr := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot jump back: %v", saveErr)
	}
	return a.formatJumpResult(result, saveErr)
}

// Drift enters the next consensus divergence.
func (a *App) Drift() string {
	cmd := &commands.DriftCommand{Universe: a.universe, Session: a.session, Repo: a.repo}
	result, saveErr := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Drift failed: %v", saveErr)
	}
	return a.formatDriftResult(result, saveErr)
}

// Align returns the session one level toward shared consensus.
func (a *App) Align() string {
	cmd := &commands.DriftCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
	result, saveErr := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot align: %v", saveErr)
	}
	return a.formatDriftResult(result, saveErr)
}

// GoHome builds and returns the route plan for returning the session to the
// start location (no movement occurs). The run loop is responsible for
// printing the plan, reading the user's confirmation, and calling
// GoHomeConfirm to execute it.
func (a *App) GoHome() string {
	if a.session.Location() == defaultStartLocation &&
		a.session.QuantumLevel() == 0 &&
		a.session.TimelineLevel() == 0 &&
		a.session.ConsensusLevel() == 0 {
		return msgAlreadyHome
	}

	var planLines []string
	estimatedCost := 0.0
	plannedLocation := a.session.Location()

	consensus := a.session.ConsensusLevel()
	for i := consensus; i > 0; i-- {
		planLines = append(planLines, fmt.Sprintf("  align      (consensus %d → %d)  cost %.0f", i, i-1, universe.ConsensusShiftCost))
		estimatedCost += universe.ConsensusShiftCost
		if next, ok := a.lowerContextDestination(plannedLocation, universe.ConsensusShift); ok {
			plannedLocation = next
		}
	}

	tl := a.session.TimelineLevel()
	for i := tl; i > 0; i-- {
		planLines = append(planLines, fmt.Sprintf("  jump back  (timeline %s → T%d)  cost %.0f", fmt.Sprintf("T%d", i), i-1, universe.TimelineShiftCost))
		estimatedCost += universe.TimelineShiftCost
		if next, ok := a.lowerContextDestination(plannedLocation, universe.TimelineShift); ok {
			plannedLocation = next
		}
	}

	ql := a.session.QuantumLevel()
	for i := ql; i > 0; i-- {
		planLines = append(planLines, fmt.Sprintf("  shift back (quantum Q%d → Q%d)  cost %.0f", i, i-1, universe.QuantumShiftCost))
		estimatedCost += universe.QuantumShiftCost
		if next, ok := a.lowerContextDestination(plannedLocation, universe.QuantumShift); ok {
			plannedLocation = next
		}
	}

	// Estimate physical travel after all contextual returns have completed.
	if plannedLocation != defaultStartLocation {
		if path, ok := a.pathfinder.FindRoute(a.universe, plannedLocation, defaultStartLocation); ok {
			for _, edge := range path {
				if !edge.Mode.IsPhysical() {
					planLines = append(planLines, "  travel     home (route unavailable after contextual returns)")
					break
				}
				planLines = append(planLines, fmt.Sprintf("  travel     %s → %s (%s)  cost %.0f", a.locationName(edge.From), a.locationName(edge.To), edge.Mode, edge.Cost))
				estimatedCost += edge.Cost
			}
		} else {
			planLines = append(planLines, "  travel     home (route unavailable)")
		}
	}

	return fmt.Sprintf("Route home\n%s\n\nEstimated cost: %.0f\n\nProceed? [y/N]:", strings.Join(planLines, "\n"), estimatedCost)
}

func (a *App) lowerContextDestination(from string, mode universe.TravelModeVO) (string, bool) {
	current, ok := a.universe.GetLocation(from)
	if !ok {
		return "", false
	}
	for _, edge := range a.universe.EdgesFrom(from) {
		if edge.Mode != mode {
			continue
		}
		dest, ok := a.universe.GetLocation(edge.To)
		if ok && isLowerContextTransition(mode, current.Coordinate, dest.Coordinate) {
			return dest.ID, true
		}
	}
	return "", false
}

func isLowerContextTransition(mode universe.TravelModeVO, current, dest universe.CoordinateVO) bool {
	switch mode {
	case universe.ConsensusShift:
		return dest.Consensus < current.Consensus
	case universe.TimelineShift:
		return dest.TimelineLevel() < current.TimelineLevel()
	case universe.QuantumShift:
		return dest.QuantumLevel() < current.QuantumLevel()
	default:
		return false
	}
}

// GoHomeConfirm executes the home journey plan produced by GoHome. It unwinds
// timeline jumps, quantum shifts, and then travels physically to the start
// location. The run loop calls this after the user confirms.
func (a *App) GoHomeConfirm() string {
	var steps []string

	for a.session.ConsensusLevel() > 0 {
		cmd := &commands.DriftCommand{Universe: a.universe, Session: a.session, Repo: a.repo, Back: true}
		result, err := cmd.Execute()
		if err != nil {
			return fmt.Sprintf("Failed while aligning with shared consensus: %v\n\nSteps taken so far:\n%s", err, strings.Join(steps, "\n"))
		}
		steps = append(steps, fmt.Sprintf("Consensus alignment → level %d at %s", result.Consensus, result.Location.Name))
	}

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

	if a.session.Location() != defaultStartLocation {
		travelResult := a.Travel(defaultStartLocation)
		steps = append(steps, fmt.Sprintf("Travelled home → %s", defaultStartLocation))
		return fmt.Sprintf("Heading home...\n\nSteps taken\n%s\n\n%s", strings.Join(steps, "\n"), travelResult)
	}

	return fmt.Sprintf("Heading home...\n\nSteps taken\n%s\n\nYou are home.\n\nCumulative journey cost\n%.0f", strings.Join(steps, "\n"), a.session.CumulativeCost())
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
	return fmt.Sprintf("Travel costs: local routes vary; quantum shifts %.0f; timeline jumps %.0f; consensus drifts %.0f.",
		universe.QuantumShiftCost, universe.TimelineShiftCost, universe.ConsensusShiftCost)
}

// Run starts the interactive read-eval-print loop with readline (tab
// completion, history, arrow-key editing). Falls back to a plain bufio loop
// when stdout is not a TTY (e.g. in tests or pipes).
func (a *App) Run() {
	a.printWelcome()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          a.Prompt(),
		AutoComplete:    NewCompleter(a),
		HistoryFile:     os.TempDir() + "/onto_history",
		InterruptPrompt: "^C",
		EOFPrompt:       cmdExit,
	})
	if err != nil {
		// Not a TTY or readline unavailable — fall back to plain reader.
		a.runPlain()
		return
	}
	defer func() { _ = rl.Close() }()

	// Use os.Stdin directly for the dead-end interactive handler; readline
	// restores the terminal to cooked mode between Readline() calls, so plain
	// bufio reads are safe at that point.
	a.interactiveReader = bufio.NewReader(os.Stdin)

	for {
		rl.SetPrompt(a.Prompt())
		line, err := rl.Readline()
		if err != nil { // EOF or ^C
			break
		}
		trimmed := strings.TrimSpace(line)

		// home requires a two-step confirm → execute pattern; handle it here
		// so the plan and prompt are printed before reading the confirmation.
		if fields := strings.Fields(trimmed); len(fields) > 0 && fields[0] == cmdHome {
			plan := a.GoHome()
			fmt.Println(plan)
			if plan != msgAlreadyHome {
				rl.SetPrompt("")
				confirm, _ := rl.Readline()
				if strings.ToLower(strings.TrimSpace(confirm)) == "y" {
					fmt.Println(a.GoHomeConfirm())
				} else {
					fmt.Println("Cancelled.")
				}
			}
			continue
		}

		response := a.Execute(trimmed)
		if response == msgGoodbye {
			fmt.Println(response)
			break
		}
		if response != "" {
			fmt.Println(response)
		}
	}
}

// runPlain is the non-readline fallback REPL used when stdin is not a TTY.
func (a *App) runPlain() {
	reader := bufio.NewReader(os.Stdin)
	a.interactiveReader = reader
	for {
		fmt.Print(a.Prompt())
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		trimmed := strings.TrimSpace(input)

		// home requires a two-step confirm → execute pattern; handle it here
		// so the plan and prompt are printed before reading the confirmation.
		if fields := strings.Fields(trimmed); len(fields) > 0 && fields[0] == cmdHome {
			plan := a.GoHome()
			fmt.Println(plan)
			if plan != msgAlreadyHome {
				confirm, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(confirm)) == "y" {
					fmt.Println(a.GoHomeConfirm())
				} else {
					fmt.Println("Cancelled.")
				}
			}
			continue
		}

		response := a.Execute(trimmed)
		if response == msgGoodbye {
			fmt.Println(response)
			break
		}
		if response != "" {
			fmt.Println(response)
		}
	}
}

func (a *App) printWelcome() {
	fmt.Println(AppVersion)
	fmt.Println()
	fmt.Println("Type 'help' to see the available commands. Press Tab to complete commands and destinations.")
	fmt.Println()
	fmt.Println("Current Position")
	fmt.Println(a.Where())
	fmt.Println()
}

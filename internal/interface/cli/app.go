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
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/infrastructure/persistence"
)

// App is the top-level CLI object. It holds the wired-up universe, session,
// repository, and pathfinder and exposes one method per user command.
type App struct {
	universe          *universe.Aggregate
	session           *exploration.Entity
	repo              universe.Repository
	pathfinder        navigation.PathfinderService
	locationGenerator universe.LocationGeneratorService
	interactiveReader *bufio.Reader
	dirty             bool
}

// NewApp loads (or initialises) the universe from disk and returns a ready App.
func NewApp() *App {
	app, err := NewAppWithError()
	if err != nil {
		panic(err)
	}
	return app
}

// NewAppWithError loads (or initialises) the universe and returns a ready App.
func NewAppWithError() (*App, error) {
	repo := persistence.NewJSONRepository(dataFile())
	u, err := repo.Load()
	if err != nil {
		u, err = buildDefaultUniverse()
		if err != nil {
			return nil, fmt.Errorf("build default universe: %w", err)
		}
	}

	sl := startLocation()
	if _, ok := u.GetLocation(sl); !ok {
		base := universe.DefaultCoordinateVO()
		if err := u.AddLocation(universe.LocationEntity{ID: sl, Name: sl, Description: "Start location (auto-added)", Coordinate: base}); err != nil {
			return nil, fmt.Errorf("add start location: %w", err)
		}
	}

	start := sl
	if _, ok := u.GetLocation(start); !ok {
		if ids := u.AllLocationIDs(); len(ids) > 0 {
			start = ids[0]
		}
	}

	loc, _ := u.GetLocation(start)
	return &App{
		universe:          u,
		session:           exploration.NewEntity(start, loc.Coordinate),
		repo:              repo,
		pathfinder:        navigation.NewBFSPathfinder(),
		locationGenerator: universe.NewSequentialLocationGenerator(),
	}, nil
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

	if len(parts) == 1 {
		if number, err := strconv.Atoi(cmd); err == nil {
			return a.ExecuteJourney(number)
		}
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
	case cmdUniverse:
		if args == argBack {
			return a.UniverseBack()
		}
		return a.Universe()
	case cmdStructure:
		if args == argBack {
			return a.StructureBack()
		}
		return a.Structure()
	case cmdSimulate:
		if args == argBack {
			return a.SimulateBack()
		}
		return a.Simulate()
	case cmdDrift:
		return a.Drift()
	case cmdAlign:
		return a.Align()
	case cmdObserve:
		if args == "" {
			return "Usage: observe <observer>"
		}
		if args == argBack {
			return a.ObserveBack()
		}
		return a.Observe(args)
	case cmdTime:
		if args == "" {
			return "Usage: time <RFC3339> or time back"
		}
		if args == argBack {
			return a.TimeBack()
		}
		return a.Time(args)
	case cmdSave:
		if args != "" {
			return "Usage: save"
		}
		msg, err := a.Save()
		if err != nil {
			return err.Error()
		}
		return msg
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
		"universe               Shift forward to the next bubble universe",
		"universe back          Return to the previous bubble universe",
		"structure              Shift forward to the next mathematical structure",
		"structure back         Return to the previous mathematical structure",
		"simulate               Enter the next nested simulation layer",
		"simulate back          Exit one simulation layer toward base reality",
		"drift                  Enter the next consensus divergence",
		"align                  Return one level toward shared consensus",
		"observe <observer>     Change observer perspective",
		"observe back           Return to the previous observer perspective",
		"time <RFC3339>         Enter a temporal branch",
		"time back              Return to the previous temporal branch",
		"save                   Persist the current universe graph to disk",
		"<number>               Take a numbered possible journey",
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
	cmd := &commands.TravelCommand{
		Universe:   a.universe,
		Session:    a.session,
		Pathfinder: a.pathfinder,
	}

	result, err := cmd.Execute(target)
	if result == nil {
		// Domain error — no movement occurred.
		norm := strings.ToLower(strings.ReplaceAll(target, " ", "-"))
		if _, known := a.universe.GetLocation(norm); !known {
			// Check if the target is the next quantum, timeline, or universe
			// branch (not yet created — it only exists after 'shift', 'jump',
			// or 'universe' runs).
			if norm == a.session.NextQuantumID() {
				return fmt.Sprintf("%s is a quantum branch — use 'shift' to enter it", target)
			}
			if norm == a.session.NextTimelineID() {
				return fmt.Sprintf("%s is a timeline branch — use 'jump' to enter it", target)
			}
			if norm == a.session.NextUniverseID() {
				return fmt.Sprintf("%s is a bubble universe — use 'universe' to enter it", target)
			}
			if norm == a.session.NextMathematicsID() {
				return fmt.Sprintf("%s is a mathematical structure — use 'structure' to enter it", target)
			}
			if norm == a.session.NextSimulationID() {
				return fmt.Sprintf("%s is a nested simulation — use 'simulate' to enter it", target)
			}
			// Destination not found — try a fuzzy suggestion.
			if suggestion := a.suggestDestination(target); suggestion != "" {
				return fmt.Sprintf(fmtUnknownDestSuggest, target, suggestion)
			}
			return a.routeUnavailableDiagnostics(target)
		}
		// Destination exists but is unreachable — surface the real reason.
		return err.Error()
	}

	output := a.formatTravelResult(result)
	if !result.DeadEndHandled {
		return output
	}
	location, err := (&commands.GenerateNearbyLocationCommand{
		Universe:  a.universe,
		Generator: a.locationGenerator,
		OriginID:  result.Location.ID,
	}).Execute()
	if err != nil {
		return fmt.Sprintf("%s\n\nUnable to generate a nearby location: %v", output, err)
	}
	a.markDirty()
	return fmt.Sprintf("%s\n\nAuto-generated: %s (%s)", output, location.Name, location.ID)
}

// Shift jumps the session forward to the next quantum branch of the current location.
func (a *App) Shift() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Shift failed: %v", err)
	}
	a.markDirty()
	return a.formatShiftResult(result)
}

// ShiftBack returns the session to the previous quantum branch.
func (a *App) ShiftBack() string {
	cmd := &commands.ShiftCommand{Universe: a.universe, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot shift back: %v", err)
	}
	a.markDirty()
	return a.formatShiftResult(result)
}

// Jump moves the session forward to the next timeline branch of the current location.
func (a *App) Jump() string {
	cmd := &commands.JumpCommand{Universe: a.universe, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Jump failed: %v", err)
	}
	a.markDirty()
	return a.formatJumpResult(result)
}

// JumpBack returns the session to the previous timeline branch.
func (a *App) JumpBack() string {
	cmd := &commands.JumpCommand{Universe: a.universe, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot jump back: %v", err)
	}
	a.markDirty()
	return a.formatJumpResult(result)
}

// Universe shifts the session forward to the next bubble universe of the current location.
func (a *App) Universe() string {
	cmd := &commands.UniverseCommand{Universe: a.universe, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Universe shift failed: %v", err)
	}
	a.markDirty()
	return a.formatUniverseResult(result)
}

// UniverseBack returns the session to the previous bubble universe.
func (a *App) UniverseBack() string {
	cmd := &commands.UniverseCommand{Universe: a.universe, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return to previous universe: %v", err)
	}
	a.markDirty()
	return a.formatUniverseResult(result)
}

// Structure shifts the session forward to the next mathematical structure.
func (a *App) Structure() string {
	cmd := &commands.StructureCommand{Universe: a.universe, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Mathematical structure shift failed: %v", err)
	}
	a.markDirty()
	return a.formatStructureResult(result)
}

// StructureBack returns the session to the previous mathematical structure.
func (a *App) StructureBack() string {
	cmd := &commands.StructureCommand{Universe: a.universe, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return to previous mathematical structure: %v", err)
	}
	a.markDirty()
	return a.formatStructureResult(result)
}

// Simulate enters the next nested simulation layer.
func (a *App) Simulate() string {
	cmd := &commands.SimulateCommand{Universe: a.universe, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Simulation entry failed: %v", err)
	}
	a.markDirty()
	return a.formatSimulateResult(result)
}

// SimulateBack exits one simulation layer toward base reality.
func (a *App) SimulateBack() string {
	cmd := &commands.SimulateCommand{Universe: a.universe, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot exit simulation: %v", err)
	}
	a.markDirty()
	return a.formatSimulateResult(result)
}

// Drift enters the next consensus divergence.
func (a *App) Drift() string {
	cmd := &commands.DriftCommand{Universe: a.universe, Session: a.session}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Drift failed: %v", err)
	}
	a.markDirty()
	return a.formatDriftResult(result)
}

// Align returns the session one level toward shared consensus.
func (a *App) Align() string {
	cmd := &commands.DriftCommand{Universe: a.universe, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot align: %v", err)
	}
	a.markDirty()
	return a.formatDriftResult(result)
}

// Observe changes the current observer perspective.
func (a *App) Observe(observer string) string {
	cmd := &commands.ObserveCommand{Universe: a.universe, Session: a.session, Observer: observer}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Observer shift failed: %v", err)
	}
	a.markDirty()
	return a.formatObserveResult(result)
}

// ObserveBack returns to the previous observer perspective.
func (a *App) ObserveBack() string {
	cmd := &commands.ObserveCommand{Universe: a.universe, Session: a.session, Back: true}
	result, err := cmd.Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return observer perspective: %v", err)
	}
	a.markDirty()
	return a.formatObserveResult(result)
}

// Time enters the temporal branch at target.
func (a *App) Time(target string) string {
	result, err := (&commands.TimeCommand{Universe: a.universe, Session: a.session, Target: target}).Execute()
	if result == nil {
		return fmt.Sprintf("Time shift failed: %v", err)
	}
	a.markDirty()
	return a.formatTimeResult(result)
}

// TimeBack returns to the preceding temporal branch.
func (a *App) TimeBack() string {
	result, err := (&commands.TimeCommand{Universe: a.universe, Session: a.session, Back: true}).Execute()
	if result == nil {
		return fmt.Sprintf("Cannot return through time: %v", err)
	}
	a.markDirty()
	return a.formatTimeResult(result)
}

// ExecuteJourney executes the one-based journey option currently shown by ls.
func (a *App) ExecuteJourney(number int) string {
	options, _ := a.journeyOptions(a.universe.EdgesFrom(a.session.Location()))
	if number < 1 || number > len(options) {
		return fmt.Sprintf("No possible journey numbered %d. Use 'ls' to view available journeys.", number)
	}

	option := options[number-1]
	switch option.kind {
	case journeyTravel:
		return a.Travel(option.target)
	case journeyShift:
		return a.Shift()
	case journeyShiftBack:
		return a.ShiftBack()
	case journeyJump:
		return a.Jump()
	case journeyJumpBack:
		return a.JumpBack()
	case journeyUniverse:
		return a.Universe()
	case journeyUniverseBack:
		return a.UniverseBack()
	case journeyStructure:
		return a.Structure()
	case journeyStructureBack:
		return a.StructureBack()
	case journeySimulate:
		return a.Simulate()
	case journeySimulateBack:
		return a.SimulateBack()
	case journeyDrift:
		return a.Drift()
	case journeyAlign:
		return a.Align()
	case journeyObserveBack:
		return a.ObserveBack()
	case journeyTimeBack:
		return a.TimeBack()
	default:
		return "Selected journey is unavailable."
	}
}

// GoHome builds and returns the route plan for returning the session to the
// start location (no movement occurs). The run loop is responsible for
// printing the plan, reading the user's confirmation, and calling
// GoHomeConfirm to execute it.
func (a *App) GoHome() string {
	cmd := &commands.ReturnHomeCommand{
		Universe:        a.universe,
		Session:         a.session,
		Pathfinder:      a.pathfinder,
		HomeID:          defaultStartLocation,
		DefaultObserver: universe.DefaultCoordinateVO().Observer,
	}
	steps, cost := cmd.Plan()
	if len(steps) == 0 {
		return msgAlreadyHome
	}
	var lines []string
	for _, step := range steps {
		line := fmt.Sprintf("  %-10s", step.Action)
		if step.Detail != "" {
			line += " (" + step.Detail + ")"
		}
		if step.Cost != 0 {
			line += fmt.Sprintf("  cost %.0f", step.Cost)
		}
		lines = append(lines, line)
	}
	return fmt.Sprintf("Route home\n%s\n\nEstimated cost: %.0f\n\nProceed? [y/N]:", strings.Join(lines, "\n"), cost)
}

// GoHomeConfirm executes the home journey plan produced by GoHome. It unwinds
// timeline jumps, quantum shifts, and then travels physically to the start
// location. The run loop calls this after the user confirms.
func (a *App) GoHomeConfirm() string {
	cmd := &commands.ReturnHomeCommand{
		Universe:        a.universe,
		Session:         a.session,
		Pathfinder:      a.pathfinder,
		HomeID:          defaultStartLocation,
		DefaultObserver: universe.DefaultCoordinateVO().Observer,
	}
	commandSteps, err := cmd.Execute()
	if err != nil {
		return fmt.Sprintf("Failed while returning home: %v", err)
	}
	a.markDirty()
	var lines []string
	for _, step := range commandSteps {
		switch step.Action {
		case "observe back":
			lines = append(lines, fmt.Sprintf("Observer return → %s", step.Detail))
		case "align":
			lines = append(lines, fmt.Sprintf("Consensus alignment → level %s", step.Detail))
		default:
			lines = append(lines, fmt.Sprintf("%s %s", step.Action, step.Detail))
		}
	}
	return fmt.Sprintf("Heading home...\n\nSteps taken\n%s\n\nYou are home.\n\nCumulative journey cost\n%.0f", strings.Join(lines, "\n"), a.session.CumulativeCost())
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

// Cost returns the session's total running travel cost.
func (a *App) Cost() string {
	return fmt.Sprintf("Total journey cost: %.0f", a.session.CumulativeCost())
}

func (a *App) markDirty() {
	a.dirty = true
}

// SaveIfDirty persists the universe to disk if there are unsaved mutations
// (from shift/jump/drift/observe/time/travel/dead-end/home commands), and
// clears the dirty flag on success. It is a no-op if nothing has changed.
func (a *App) SaveIfDirty() error {
	if !a.dirty {
		return nil
	}
	if err := a.repo.Save(a.universe); err != nil {
		return err
	}
	a.dirty = false
	return nil
}

// Save unconditionally persists the universe to disk, regardless of the
// dirty flag, and reports a user-facing confirmation message. It backs the
// explicit "save" command.
func (a *App) Save() (string, error) {
	if err := a.repo.Save(a.universe); err != nil {
		return "", err
	}
	a.dirty = false
	return msgSaved, nil
}

func (a *App) warnIfSaveBeforeExitFails() {
	if err := a.SaveIfDirty(); err != nil {
		fmt.Printf(fmtExitSaveWarning+"\n", err)
	}
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
			a.warnIfSaveBeforeExitFails()
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
			a.warnIfSaveBeforeExitFails()
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
			a.warnIfSaveBeforeExitFails()
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
			a.warnIfSaveBeforeExitFails()
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

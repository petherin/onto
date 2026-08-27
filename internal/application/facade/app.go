// Package facade provides the application facade: a single App type that
// dispatches user commands to the application and domain layers and formats
// results as strings. It is delivery-mechanism agnostic — no readline, no I/O,
// no terminal assumptions. CLI, TUI, web, and test code all depend on this
// package rather than on each other.
package facade

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// AppVersion is the human-readable application version string, shown at
// startup by every delivery mechanism (CLI, TUI, web).
const AppVersion = "Onto Explorer v0.1"

// DefaultBudget is the starting spending pool used by the standard game. It
// comfortably covers reaching the default objective and returning home while
// keeping the most expensive transitions (universe, mathematical structure)
// out of reach, so the budget is felt.
const DefaultBudget = 1000.0

// TargetReachedMessage is appended to command output the moment the objective
// coordinate is first visited.
const TargetReachedMessage = "Objective reached — now return home to win."

// WinMessage is appended to command output the moment the objective is
// completed (target reached and back at the start location).
const WinMessage = "You reached your objective and returned home. You win!"

// MaxStars is the top efficiency rating awarded on a win: three stars for a run
// played at or under par.
const MaxStars = 3

// DefaultTarget derives the standard single-objective coordinate from the start
// coordinate: the second quantum branch (Q2) of home. Reaching it requires two
// quantum shifts, and winning requires shifting back home again.
func DefaultTarget(start universe.CoordinateVO) universe.CoordinateVO {
	target := start
	target.Quantum = "Q2"
	return target
}

// DefaultQuestChain derives the standard multi-objective quest from the start
// coordinate: first the second quantum branch (Q2), then one simulation layer
// deep (sim:1) back on the prime branch. The two waypoints sit on different
// reality axes, so completing the chain exercises more than one kind of
// transition, and the optimal round trip stays within DefaultBudget.
func DefaultQuestChain(start universe.CoordinateVO) []universe.CoordinateVO {
	first := start
	first.Quantum = "Q2"
	second := start
	second.Simulation = 1
	return []universe.CoordinateVO{first, second}
}

// Option configures optional App behaviour (currently the game rules) at
// construction. Options are applied before the session is built so budget and
// objectives take effect immediately.
type Option func(*App)

// WithBudget installs a finite spending pool that blocks unaffordable moves.
func WithBudget(budget float64) Option {
	return func(a *App) { a.budget = budget }
}

// WithTarget installs a single objective coordinate for the win condition. It is
// shorthand for a quest chain of length one.
func WithTarget(target universe.CoordinateVO) Option {
	return WithTargets(target)
}

// WithTargets installs an ordered quest chain of objective coordinates. The
// waypoints must be reached in order before returning home wins.
func WithTargets(targets ...universe.CoordinateVO) Option {
	return func(a *App) {
		a.targets = append([]universe.CoordinateVO(nil), targets...)
	}
}

// SessionEntity returns the underlying exploration session.
// Intended for test introspection and TUI/web state access.
func (a *App) SessionEntity() *exploration.Entity { return a.session }

// Aggregate returns the underlying universe aggregate.
// Intended for test introspection and web state access.
func (a *App) Aggregate() *universe.Aggregate { return a.univ }

// IsDirty reports whether unsaved mutations exist.
func (a *App) IsDirty() bool { return a.dirty }

// Prompt returns the context-aware CLI prompt string.
func (a *App) Prompt() string {
	addr := a.session.Coordinate().ShortOntoAddress()
	addr = addr[len("onto://"):]
	return fmt.Sprintf("[%s] > ", addr)
}

// PhysicalDestinationIDs returns the IDs of locations reachable by physical
// travel from the current location, in the same physical reality slice.
// Used by the CLI tab-completer.
func (a *App) PhysicalDestinationIDs() []string {
	current := a.session.Coordinate()
	var ids []string
	for _, edge := range a.univ.EdgesFrom(a.session.Location()) {
		if !edge.Mode.IsPhysical() {
			continue
		}
		dest, ok := a.univ.GetLocation(edge.To)
		if ok && current.SamePhysicalReality(dest.Coordinate) {
			ids = append(ids, dest.ID)
		}
	}
	return ids
}

// App is the application facade. It owns the wired-up universe, session,
// repository, and domain services and exposes one method per user command.
// All methods return plain strings; formatting is done here so every delivery
// mechanism (CLI, TUI, web) sees the same output without duplication.
type App struct {
	univ              *universe.Aggregate
	session           *exploration.Entity
	repo              universe.Repository
	pathfinder        navigation.PathfinderService
	locationGenerator universe.LocationGeneratorService
	homeID            string
	dirty             bool

	// Game configuration seed, captured at construction and reapplied on Reset so
	// a reset session starts a fresh game. budget of 0 means unlimited; a non-empty
	// targets chain gates the win condition. These are construction/reset config
	// only — the live game state is the session, so read a.session (not these
	// fields) when reporting current progress or objectives.
	budget  float64
	targets []universe.CoordinateVO

	// initialLocations/initialEdges snapshot the universe as it was at
	// construction (startup), so Reset can rebuild the starting map after
	// reality transitions have grown it. LocationEntity and EdgeVO are value
	// types, so these slices are an independent copy of the graph.
	initialLocations []universe.LocationEntity
	initialEdges     []universe.EdgeVO
}

// New assembles an App from already-wired dependencies. Callers (cmd/ entry
// points) provide a loaded universe, a repository, the start location ID, and
// the domain services.
func New(
	u *universe.Aggregate,
	repo universe.Repository,
	startID string,
	pathfinder navigation.PathfinderService,
	gen universe.LocationGeneratorService,
	opts ...Option,
) (*App, error) {
	loc, ok := u.GetLocation(startID)
	if !ok {
		return nil, fmt.Errorf("start location %q not found in universe", startID)
	}
	a := &App{
		univ:              u,
		repo:              repo,
		pathfinder:        pathfinder,
		locationGenerator: gen,
		homeID:            startID,
		initialLocations:  u.AllLocations(),
		initialEdges:      u.AllEdgesFlat(),
	}
	for _, opt := range opts {
		opt(a)
	}
	a.session = a.newSession(startID, loc.Coordinate)
	return a, nil
}

// newSession builds a session at the given position and applies the configured
// game rules (budget and quest chain), so New and Reset stay consistent.
func (a *App) newSession(location string, coord universe.CoordinateVO) *exploration.Entity {
	s := exploration.NewEntity(location, coord)
	if a.budget > 0 {
		s.SetBudget(a.budget)
	}
	if len(a.targets) > 0 {
		s.SetTargets(a.targets)
	}
	return s
}

// Reset rebuilds the universe to the state captured at construction (the
// starting map) and returns the session to the start location in base reality,
// discarding every location and edge that reality transitions created this
// session. It marks the app dirty so the cleared map can be saved over the
// grown one.
func (a *App) Reset() string {
	fresh := universe.NewAggregate()
	for _, loc := range a.initialLocations {
		if err := fresh.AddLocation(loc); err != nil {
			return fmt.Sprintf("Failed to reset: %v", err)
		}
	}
	for _, e := range a.initialEdges {
		if err := fresh.AddEdge(e); err != nil {
			return fmt.Sprintf("Failed to reset: %v", err)
		}
	}
	loc, ok := fresh.GetLocation(a.homeID)
	if !ok {
		return fmt.Sprintf("Failed to reset: start location %q missing", a.homeID)
	}
	a.univ = fresh
	a.session = a.newSession(a.homeID, loc.Coordinate)
	a.dirty = true
	return "Map reset to the starting realities."
}

// Execute dispatches a raw input string to the appropriate command handler and
// appends any objective-reached or win banner triggered by the command.
func (a *App) Execute(input string) string {
	doneBefore, wonBefore := a.session.ObjectiveIndex(), a.session.Won()
	out := a.dispatch(input)
	return out + a.goalBanner(doneBefore, wonBefore)
}

// goalBanner returns the messages for goal-state transitions since the given
// prior state: one line for each objective reached (naming the next objective in
// a multi-step chain, or announcing the chain complete on the last), and the win
// banner with its rating once the objective is complete. doneBefore is the
// number of chain objectives reached before the action. Delivery mechanisms that
// move the session outside Execute (e.g. the two-step home confirmation) call
// this too.
func (a *App) goalBanner(doneBefore int, wonBefore bool) string {
	var b strings.Builder
	doneNow := a.session.ObjectiveIndex()
	count := a.session.ObjectiveCount()
	targets := a.session.Targets()
	for i := doneBefore; i < doneNow; i++ {
		if i+1 < count {
			fmt.Fprintf(&b, "\n\nObjective %d of %d reached — next: reach %s.", i+1, count, targets[i+1].ShortOntoAddress())
		} else {
			b.WriteString("\n\n" + TargetReachedMessage)
		}
	}
	if a.session.Won() && !wonBefore {
		b.WriteString("\n\n" + WinMessage)
		b.WriteString("\n" + a.ratingLine())
	}
	return b.String()
}

// objectivePar returns the optimal cost for the current objective: the minimal
// reality-axis cost to visit every quest-chain waypoint in order from the start
// coordinate and return. It reads the live session's chain (the single source of
// truth for the running game) so par and progress can never drift apart. It is 0
// when no objective is set. Objectives that move physical location are not
// accounted for, since objectives currently vary only reality axes.
func (a *App) objectivePar() float64 {
	targets := a.session.Targets()
	if len(targets) == 0 {
		return 0
	}
	start, ok := a.univ.GetLocation(a.homeID)
	if !ok {
		return 0
	}
	prev := start.Coordinate
	var total float64
	for _, t := range targets {
		total += universe.TransitionCost(prev, t)
		prev = t
	}
	total += universe.TransitionCost(prev, start.Coordinate)
	return total
}

// starsForCost rates a completed run against par: three stars for playing at or
// under par (optimal), two for finishing within twice par, and one for any
// slower win. A non-positive par (no meaningful objective) yields 0.
func starsForCost(cost, par float64) int {
	switch {
	case par <= 0:
		return 0
	case cost <= par:
		return MaxStars
	case cost <= par*2:
		return 2
	default:
		return 1
	}
}

// starBar renders a 0..MaxStars rating as filled and empty stars.
func starBar(n int) string {
	if n < 0 {
		n = 0
	}
	if n > MaxStars {
		n = MaxStars
	}
	return strings.Repeat("★", n) + strings.Repeat("☆", MaxStars-n)
}

// ratingLine reports the efficiency rating for a completed run: the star bar
// alongside the actual cost and par.
func (a *App) ratingLine() string {
	par := a.objectivePar()
	cost := a.session.CumulativeCost()
	return fmt.Sprintf("Rating: %s  (%.0f cost / par %.0f)", starBar(starsForCost(cost, par)), cost, par)
}

// dispatch routes a raw input string to the appropriate command handler.
func (a *App) dispatch(input string) string {
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
	case "home":
		return a.GoHome()
	case "cost":
		return a.Cost()
	case "shift":
		if args == "back" {
			return a.ShiftBack()
		}
		return a.Shift()
	case "jump":
		if args == "back" {
			return a.JumpBack()
		}
		return a.Jump()
	case "universe":
		if args == "back" {
			return a.UniverseBack()
		}
		return a.Universe()
	case "structure":
		if args == "back" {
			return a.StructureBack()
		}
		return a.Structure()
	case "simulate":
		if args == "back" {
			return a.SimulateBack()
		}
		return a.Simulate()
	case "drift":
		return a.Drift()
	case "align":
		return a.Align()
	case "observe":
		if args == "" {
			return "Usage: observe <observer>"
		}
		if args == "back" {
			return a.ObserveBack()
		}
		return a.Observe(args)
	case "time":
		if args == "" {
			return "Usage: time <RFC3339> or time back"
		}
		if args == "back" {
			return a.TimeBack()
		}
		return a.Time(args)
	case "save":
		if args != "" {
			return "Usage: save"
		}
		msg, err := a.Save()
		if err != nil {
			return err.Error()
		}
		return msg
	case "exit":
		return "Goodbye."
	default:
		if suggestion := a.suggestCommand(cmd); suggestion != "" {
			return fmt.Sprintf("Unknown command: %s\n\nDid you mean '%s'?\n\n%s", cmd, suggestion, a.Help())
		}
		return fmt.Sprintf("Unknown command: %s\n\n%s", cmd, a.Help())
	}
}

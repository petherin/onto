// Package facade provides the application facade: a single App type that
// dispatches user commands to the application and domain layers and formats
// results as strings. It is delivery-mechanism agnostic — no readline, no I/O,
// no terminal assumptions. CLI, web, and test code all depend on this
// package rather than on each other.
package facade

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// AppVersion is the human-readable application version string, shown at
// startup by every delivery mechanism (CLI, web).
const AppVersion = "Onto Explorer v0.1"

// DefaultBudget is the starting spending pool used by the standard game. It
// comfortably covers reaching the default objective and returning home while
// keeping the most expensive transitions (universe, mathematical structure)
// out of reach, so the budget is felt.
const DefaultBudget = 1000.0

// TargetReachedMessage is appended to command output the moment an objective's
// coordinate is first reached; the objective is only banked once the traveller
// returns home.
const TargetReachedMessage = "Objective reached — now return home to complete it."

// WinMessage is appended to command output the moment the objective is
// completed (target reached and back at the start location).
const WinMessage = "You reached your objective and returned home. You win!"

// BudgetExhaustedMarker labels a finite budget that has been spent down to
// nothing. It distinguishes an exhausted budget (a limit is in force but no
// money is left) from an unlimited one (no limit in force at all), which is
// simply not shown.
const BudgetExhaustedMarker = "exhausted"

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

// QuestSizeMin and QuestSizeMax bound how many objectives a randomly-built quest
// draws from the objective pool (see WithObjectivePool). Fewer are used when the
// pool itself is smaller than QuestSizeMin.
const (
	QuestSizeMin = 2
	QuestSizeMax = 4
)

// WithTargets installs an ordered quest chain of objective coordinates. The
// waypoints must be reached in order before returning home wins.
func WithTargets(targets ...universe.CoordinateVO) Option {
	return func(a *App) {
		a.targets = append([]universe.CoordinateVO(nil), targets...)
	}
}

// WithObjectivePool installs a pool of candidate objectives from which a random
// quest chain is built on session start (and re-rolled by NewQuest). It only has
// an effect when no explicit chain is set via WithTargets, which takes
// precedence. An empty pool is ignored.
func WithObjectivePool(pool ...universe.CoordinateVO) Option {
	return func(a *App) {
		a.objectivePool = append([]universe.CoordinateVO(nil), pool...)
	}
}

// SessionEntity returns the underlying exploration session.
// Intended for test introspection and web state access.
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
// mechanism (CLI, web) sees the same output without duplication.
type App struct {
	univ              *universe.Aggregate
	session           *exploration.Entity
	repo              universe.Repository
	pathfinder        navigation.PathfinderService
	locationGenerator universe.LocationGeneratorService
	homeID            string
	dirty             bool

	// Game configuration seed, captured at construction and reapplied on Reset so
	// a reset session starts a fresh game. budget of 0 means unlimited (no limit in
	// force — distinct from a finite budget later spent down to nothing, which is
	// still in force but exhausted); see budgetInForce. A non-empty targets chain
	// gates the win condition. These are construction/reset config only — the live
	// game state is the session, so read a.session (not these fields) when
	// reporting current progress or objectives.
	budget  float64
	targets []universe.CoordinateVO
	// objectivePool is the set of candidate objectives from which a random quest
	// chain is built when no explicit targets are configured, and re-rolled by
	// NewQuest. Empty means quests are fixed (targets or the default chain).
	objectivePool []universe.CoordinateVO

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

// QuestBuildAttempts bounds how many random quests buildRandomQuest draws while
// looking for one completable within the budget. If none of these attempts fits,
// generation gives up rather than looping forever.
const QuestBuildAttempts = 50

// NoAffordableQuestMessage is returned by NewQuest when no quest the objective
// pool can produce fits within the budget.
const NoAffordableQuestMessage = "No quest in the objective pool fits the current budget — try a larger budget (ONTO_BUDGET) or cheaper objectives (ONTO_OBJECTIVES)."

// budgetInForce reports whether a finite spending limit constrains this game. A
// configured budget of zero (the default) means unlimited spending — no limit is
// in force — as opposed to a finite budget that has been spent down to nothing,
// which is still "in force" but exhausted. Quest affordability keys off this so
// "no limit" is never confused with "no money left".
func (a *App) budgetInForce() bool { return a.budget > 0 }

// newSession builds a session at the given position and applies the configured
// game rules (budget and quest chain), so New and Reset stay consistent. An
// explicit targets chain wins; failing that, a random affordable quest is drawn
// from the objective pool when one is configured. If no pooled quest fits the
// budget the session simply starts with no objective.
func (a *App) newSession(location string, coord universe.CoordinateVO) *exploration.Entity {
	s := exploration.NewEntity(location, coord)
	if a.budgetInForce() {
		s.SetBudget(a.budget)
	}
	switch {
	case len(a.targets) > 0:
		s.SetTargets(a.targets)
	case len(a.objectivePool) > 0:
		if chain, ok := a.buildRandomQuest(a.budget); ok {
			s.SetTargets(chain)
		}
	}
	return s
}

// NewQuest re-rolls the quest chain from the configured objective pool, resetting
// objective progress (and the win flag) while keeping the current position and
// running cost — so re-rolling mid-journey starts a fresh objective without a
// full reset. With no pool configured (a fixed quest, or game mode off) it is a
// no-op that reports the quest is fixed. Affordability is measured against the
// budget still remaining (not the original pool), so once a finite budget has
// been spent down to nothing no quest fits and the current quest is left
// unchanged with the reason reported.
func (a *App) NewQuest() string {
	if len(a.objectivePool) == 0 {
		return "No objective pool configured — the quest is fixed."
	}
	chain, ok := a.buildRandomQuest(a.session.RemainingBudget())
	if !ok {
		return NoAffordableQuestMessage
	}
	a.session.SetTargets(chain)
	var b strings.Builder
	fmt.Fprintf(&b, "New quest — %d objectives:", len(chain))
	for i, t := range chain {
		fmt.Fprintf(&b, "\n  %d. %s", i+1, t.ShortOntoAddress())
	}
	b.WriteString("\nReach each in order, returning home after each; the last return home wins.")
	return b.String()
}

// buildRandomQuest samples a quest chain from the objective pool that can be
// completed within the given budget: between QuestSizeMin and QuestSizeMax
// distinct objectives (fewer when the pool is smaller), in random order, without
// replacement. It draws up to QuestBuildAttempts candidates and returns the first
// whose round-trip par is within budget. When no finite budget is in force
// (unlimited spending — see budgetInForce) affordability never binds, so the
// first draw is always accepted; this is distinct from a finite budget under
// which no draw fits. Callers pass the budget affordability should key off — the
// full budget for a fresh session, the remaining budget for a mid-game re-roll —
// so an exhausted budget (0 remaining) accepts nothing. It returns ok=false when
// the pool is empty or, with a budget in force, no attempt fits it — so the
// caller can report that no affordable quest exists rather than installing an
// impossible one.
func (a *App) buildRandomQuest(budget float64) ([]universe.CoordinateVO, bool) {
	if len(a.objectivePool) == 0 {
		return nil, false
	}
	for attempt := 0; attempt < QuestBuildAttempts; attempt++ {
		chain := sampleQuest(a.objectivePool)
		if !a.budgetInForce() || a.chainPar(chain) <= budget {
			return chain, true
		}
	}
	return nil, false
}

// sampleQuest draws one random quest chain from the pool: between QuestSizeMin and
// QuestSizeMax distinct objectives (fewer when the pool is smaller), in random
// order. Sampling is without replacement, so no objective repeats within a chain.
func sampleQuest(pool []universe.CoordinateVO) []universe.CoordinateVO {
	n := len(pool)
	if n == 0 {
		return nil
	}
	shuffled := append([]universe.CoordinateVO(nil), pool...)
	rand.Shuffle(n, func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	size := QuestSizeMin + rand.Intn(QuestSizeMax-QuestSizeMin+1)
	if size > n {
		size = n
	}
	return shuffled[:size]
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
	doneBefore, reachedBefore, wonBefore := a.session.ObjectiveIndex(), a.session.ReachedTarget(), a.session.Won()
	out := a.dispatch(input)
	return out + a.goalBanner(doneBefore, reachedBefore, wonBefore)
}

// goalBanner returns the messages for goal-state transitions since the given
// prior state. Each objective is a round trip, so reaching its waypoint prompts
// the return-home step, and only returning home banks it: that announces the
// next objective (in a multi-step chain) or, on the last, the win banner with
// its rating. doneBefore/reachedBefore/wonBefore capture the completed-objective
// count, current-objective-reached flag, and win state before the action.
// Delivery mechanisms that move the session outside Execute (e.g. the two-step
// home confirmation) call this too.
func (a *App) goalBanner(doneBefore int, reachedBefore, wonBefore bool) string {
	var b strings.Builder
	doneNow := a.session.ObjectiveIndex()
	count := a.session.ObjectiveCount()
	targets := a.session.Targets()
	if a.session.ReachedTarget() && !reachedBefore {
		b.WriteString("\n\n" + TargetReachedMessage)
	}
	for i := doneBefore; i < doneNow; i++ {
		if i+1 < count {
			fmt.Fprintf(&b, "\n\nObjective %d of %d complete — next: reach %s.", i+1, count, targets[i+1].ShortOntoAddress())
		}
	}
	if a.session.Won() && !wonBefore {
		b.WriteString("\n\n" + WinMessage)
		b.WriteString("\n" + a.ratingLine())
	}
	return b.String()
}

// objectivePar returns the optimal cost for the live session's objective. It
// reads the session's chain (the single source of truth for the running game) so
// par and progress can never drift apart, and delegates to chainPar. It is 0 when
// no objective is set.
func (a *App) objectivePar() float64 {
	return a.chainPar(a.session.Targets())
}

// chainPar returns the optimal cost for an arbitrary quest chain. Each objective
// is an independent round trip, so par is the sum, over every waypoint, of the
// cost to travel out to it from the start and back. Each leg costs its
// reality-axis transitions (TransitionCost) plus the physical-travel distance
// between the two locations (physicalLegCost); the two are orthogonal and
// additive. It is used both to report the live par and to score candidate quests
// before they are installed (so generation can reject chains that cannot be
// completed within the budget). It is 0 for an empty chain.
func (a *App) chainPar(chain []universe.CoordinateVO) float64 {
	if len(chain) == 0 {
		return 0
	}
	start, ok := a.univ.GetLocation(a.homeID)
	if !ok {
		return 0
	}
	base := start.Coordinate
	var total float64
	for _, t := range chain {
		total += universe.TransitionCost(base, t) + a.physicalLegCost(base, base.Location, t.Location)
		total += universe.TransitionCost(t, base) + a.physicalLegCost(base, t.Location, base.Location)
	}
	return total
}

// physicalLegCost returns the optimal physical-travel cost between two physical
// locations (named by their Location field) within the base reality slice, using
// the same pathfinder the travel command uses. Physical travel and reality
// transitions are orthogonal — reality shifts preserve physical location and
// physical edges never cross a reality boundary — and every reality slice mirrors
// the base physical graph, so measuring a leg in the base slice gives its cost in
// any slice. It is 0 when the endpoints resolve to the same node, either cannot be
// resolved to a graph node, or no route connects them.
func (a *App) physicalLegCost(base universe.CoordinateVO, fromName, toName string) float64 {
	from, okFrom := a.univ.FindInReality(base, fromName)
	to, okTo := a.univ.FindInReality(base, toName)
	if !okFrom || !okTo || from.ID == to.ID {
		return 0
	}
	path, ok := a.pathfinder.FindRoute(a.univ, from.ID, to.ID)
	if !ok {
		return 0
	}
	return navigation.PathCost(path)
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

// dispatch routes a raw input string to the appropriate command handler. It is
// a flat switch over the fixed set of commands; funlen is silenced because
// splitting a command table into sub-dispatchers would obscure, not clarify, it.
//
//nolint:funlen // flat command-routing switch over the fixed command set
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
	case "mathematical":
		if args == "back" {
			return a.MathematicalBack()
		}
		return a.Mathematical()
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
	case "quest":
		if args != "" {
			return "Usage: quest"
		}
		return a.NewQuest()
	case "exit":
		return "Goodbye."
	default:
		if suggestion := a.suggestCommand(cmd); suggestion != "" {
			return fmt.Sprintf("Unknown command: %s\n\nDid you mean '%s'?\n\n%s", cmd, suggestion, a.Help())
		}
		return fmt.Sprintf("Unknown command: %s\n\n%s", cmd, a.Help())
	}
}

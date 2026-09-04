package facade

import (
	"github.com/petherin/onto/internal/domain/universe"
)

// DefaultBudget is the starting spending pool used by the standard game. It
// comfortably covers reaching the default objective and returning home while
// keeping the most expensive transitions (universe, mathematical structure)
// out of reach, so the budget is felt.
const DefaultBudget = 1000.0

// BudgetExhaustedMarker labels a finite budget that has been spent down to
// nothing. It distinguishes an exhausted budget (a limit is in force but no
// money is left) from an unlimited one (no limit in force at all), which is
// simply not shown.
const BudgetExhaustedMarker = "exhausted"

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

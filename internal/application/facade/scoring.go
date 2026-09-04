package facade

import (
	"fmt"
	"strings"

	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// TargetReachedMessage is appended to command output the moment an objective's
// coordinate is first reached; the objective is only banked once the traveller
// returns home.
const TargetReachedMessage = "Objective reached — now return home to complete it."

// WinMessage is appended to command output the moment the objective is
// completed (target reached and back at the start location).
const WinMessage = "You reached your objective and returned home. You win!"

// MaxStars is the top efficiency rating awarded on a win: three stars for a run
// played at or under par.
const MaxStars = 3

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

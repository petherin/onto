package facade

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
)

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

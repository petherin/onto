package facade

import (
	"fmt"
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// poolCoords builds n distinct candidate objectives (home at successive quantum
// branches) for objective-pool tests. Each round-trip par is well under the
// default budget (Q1 = 40 … Q6 = 240), so any quest drawn from them is
// affordable.
func poolCoords(n int) []universe.CoordinateVO {
	pool := make([]universe.CoordinateVO, n)
	for i := range pool {
		c := universe.DefaultCoordinateVO()
		c.Quantum = fmt.Sprintf("Q%d", i+1)
		pool[i] = c
	}
	return pool
}

// unaffordablePoolCoords builds n distinct candidate objectives that each sit one
// bubble-universe out, so a single one round-trips at 2*UniverseShiftCost (10000)
// — far beyond the default budget. No quest drawn from these can ever fit, so
// they exercise the affordability gate.
func unaffordablePoolCoords(n int) []universe.CoordinateVO {
	pool := make([]universe.CoordinateVO, n)
	for i := range pool {
		c := universe.DefaultCoordinateVO()
		c.Universe = "U1"
		c.Quantum = fmt.Sprintf("Q%d", i+1) // keep the addresses distinct
		pool[i] = c
	}
	return pool
}

// assertQuestFromPool asserts the live quest is a valid draw from the pool:
// between QuestSizeMin and QuestSizeMax distinct objectives, each drawn from the
// given pool of addresses.
func assertQuestFromPool(t *testing.T, app *App, poolAddrs map[string]bool) {
	t.Helper()
	snap := app.Snapshot()
	assert.True(t, snap.HasTarget)
	assert.GreaterOrEqual(t, snap.ObjectiveCount, QuestSizeMin)
	assert.LessOrEqual(t, snap.ObjectiveCount, QuestSizeMax)
	seen := map[string]bool{}
	for _, o := range snap.Objectives {
		assert.True(t, poolAddrs[o.Address], "objective %s should come from the pool", o.Address)
		assert.False(t, seen[o.Address], "objectives within a quest must be distinct")
		seen[o.Address] = true
	}
}

// TestNewQuest_BuildsFromPool covers the objective-pool flow: a random quest is
// built on start, and the 'quest' command re-rolls a fresh one (resetting
// progress) that is still a valid draw from the pool.
func TestNewQuest_BuildsFromPool(t *testing.T) {
	pool := poolCoords(6)
	poolAddrs := map[string]bool{}
	for _, c := range pool {
		poolAddrs[c.OntoAddress()] = true
	}

	app := newGameApp(t, WithBudget(DefaultBudget), WithObjectivePool(pool...))
	assertQuestFromPool(t, app, poolAddrs)

	out := app.Execute("quest")
	assert.Contains(t, out, "New quest")
	assert.Equal(t, 0, app.Snapshot().ObjectivesDone, "a re-roll resets objective progress")
	assertQuestFromPool(t, app, poolAddrs)
}

// TestNewQuest_NoPoolIsFixed covers that 'quest' is a no-op when no pool is
// configured: a fixed chain stays put and an explanatory message is returned.
func TestNewQuest_NoPoolIsFixed(t *testing.T) {
	target := universe.DefaultCoordinateVO()
	target.Quantum = "Q2"

	app := newGameApp(t, WithTargets(target))
	out := app.Execute("quest")
	assert.Contains(t, out, "No objective pool configured")
	assert.Equal(t, 1, app.Snapshot().ObjectiveCount, "the fixed quest is unchanged")
}

// TestNewQuest_NoAffordableQuestLeavesQuestUnchanged covers the affordability
// gate: when every objective the pool can produce costs more than the budget,
// the session starts with no objective and NewQuest reports the reason without
// installing an impossible quest.
func TestNewQuest_NoAffordableQuestLeavesQuestUnchanged(t *testing.T) {
	pool := unaffordablePoolCoords(5)

	app := newGameApp(t, WithBudget(DefaultBudget), WithObjectivePool(pool...))

	snap := app.Snapshot()
	assert.False(t, snap.HasTarget, "an all-unaffordable pool starts with no objective")
	assert.Equal(t, 0, snap.ObjectiveCount)
	assert.Equal(t, 0.0, snap.Par)

	out := app.Execute("quest")
	assert.Contains(t, out, NoAffordableQuestMessage)
	assert.False(t, app.Snapshot().HasTarget, "a failed re-roll leaves the current quest unchanged")
}

// TestNewQuest_MixedPoolAlwaysAffordable covers that when the pool mixes cheap
// and unaffordable objectives, generation resamples until it finds a quest that
// fits the budget: every start yields an objective whose par is within budget,
// never an impossible one.
func TestNewQuest_MixedPoolAlwaysAffordable(t *testing.T) {
	pool := append(poolCoords(8), unaffordablePoolCoords(2)...)

	for i := 0; i < 30; i++ {
		app := newGameApp(t, WithBudget(DefaultBudget), WithObjectivePool(pool...))
		snap := app.Snapshot()
		require.True(t, snap.HasTarget, "a mixed pool must always yield an affordable quest")
		assert.LessOrEqual(t, snap.Par, DefaultBudget, "a generated quest must fit the budget")
	}
}

// TestNewQuest_ExhaustedBudgetBlocksReroll covers that a re-roll is judged
// against the budget still remaining, not the original pool: a quest that fitted
// at full budget no longer fits once the budget has been spent below its par, so
// 'quest' reports no affordable quest and leaves the current one unchanged.
func TestNewQuest_ExhaustedBudgetBlocksReroll(t *testing.T) {
	// A single Q1 objective round-trips at 2*QuantumShiftCost (40); a budget of
	// three shifts (60) covers it at the start.
	pool := poolCoords(1)
	app := newGameApp(t, WithBudget(3*universe.QuantumShiftCost), WithObjectivePool(pool...))
	require.True(t, app.Snapshot().HasTarget, "the quest fits the full budget at the start")

	// Spend two shifts (40), leaving 20 remaining — below the 40 a quest costs,
	// but still under the original 60.
	app.Execute("shift")
	app.Execute("shift")
	require.Equal(t, float64(universe.QuantumShiftCost), app.Snapshot().RemainingBudget)

	out := app.Execute("quest")
	assert.Contains(t, out, NoAffordableQuestMessage, "no quest fits what is left of the budget")
	assert.Equal(t, 1, app.Snapshot().ObjectiveCount, "a failed re-roll leaves the current quest unchanged")
}

// The default multi-objective quest chain (Q2 then sim:1) is a sequence of round
// trips reached in order: reaching a waypoint prompts the return-home step,
// returning home banks that objective and names the next, and being home
// mid-chain (after only the first objective) is not a win. Playing it optimally
// (out to Q2 and back, then one simulation layer in and out) matches par (140)
// and earns three stars.
func TestQuestChain_DefaultChainOptimalRun(t *testing.T) {
	chain := DefaultQuestChain(universe.DefaultCoordinateVO())
	app := newGameApp(t, WithBudget(DefaultBudget), WithTargets(chain...))

	// Par sums the per-objective round trips: Q2 out (20+20) and back (20+20) =
	// 80, plus one simulation layer in (10) and out (50) = 60, total 140.
	assert.Equal(t, 140.0, app.Snapshot().Par)
	assert.Equal(t, 2, app.Snapshot().ObjectiveCount)
	assert.Equal(t, 0, app.Snapshot().ObjectivesDone)

	app.Execute("shift")        // -> Q1
	out := app.Execute("shift") // -> Q2 (first waypoint reached)
	assert.Contains(t, out, TargetReachedMessage, "reaching the first objective prompts the return home")
	assert.NotContains(t, out, "Objective 1 of 2 complete", "the objective is not banked until home")
	assert.Equal(t, 0, app.Snapshot().ObjectivesDone, "reaching a waypoint does not bank it until home")
	assert.True(t, app.Snapshot().ReachedTarget)
	assert.False(t, app.Snapshot().Won)

	app.Execute("shift back")       // -> Q1
	out = app.Execute("shift back") // -> home: first objective banked
	assert.Contains(t, out, "Objective 1 of 2 complete", "returning home banks the objective and names the next")
	assert.NotContains(t, out, WinMessage, "the chain is not finished")
	assert.False(t, app.Snapshot().Won)
	assert.Equal(t, 1, app.Snapshot().ObjectivesDone)
	assert.False(t, app.Snapshot().ReachedTarget)

	out = app.Execute("simulate") // -> sim:1 (second and final waypoint reached)
	assert.Contains(t, out, TargetReachedMessage, "reaching the last waypoint prompts the return home")
	assert.True(t, app.Snapshot().ReachedTarget)
	assert.False(t, app.Snapshot().Won)

	out = app.Execute("simulate back") // -> home; chain complete, wins at par
	assert.Contains(t, out, WinMessage)
	assert.Contains(t, out, "Rating:")
	snap := app.Snapshot()
	assert.True(t, snap.Won)
	assert.Equal(t, "home", app.SessionEntity().Location())
	assert.Equal(t, 140.0, snap.CumulativeCost)
	assert.Equal(t, MaxStars, snap.Stars, "an at-par chain run earns three stars")
}

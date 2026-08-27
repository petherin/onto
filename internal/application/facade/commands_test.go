package facade

import (
	"fmt"
	"strings"
	"testing"

	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGameApp builds an App on the shared test universe with the given game
// options, starting at home.
func newGameApp(t *testing.T, opts ...Option) *App {
	t.Helper()
	u := mocks.NewTestUniverse()
	repo := mocks.NewMockRepository(t)
	app, err := New(u, repo, "home",
		navigation.NewBFSPathfinder(),
		universe.NewSequentialLocationGenerator(),
		opts...,
	)
	require.NoError(t, err)
	return app
}

// TestNeedsHomeConfirm asserts the helper treats only an actionable route plan
// (one terminated by HomeConfirmPrompt) as needing confirmation, and never the
// terminal "already home" / "no route home" messages that GoHome can also
// return. All three delivery layers gate their confirm flow on this.
func TestNeedsHomeConfirm(t *testing.T) {
	tests := []struct {
		name string
		plan string
		want bool
	}{
		{
			name: "actionable plan",
			plan: "Route home\nHome -> Station\n\nEstimated cost: 2\n\n" + HomeConfirmPrompt,
			want: true,
		},
		{name: "bare prompt", plan: HomeConfirmPrompt, want: true},
		{name: "already home", plan: MsgAlreadyHome, want: false},
		{
			name: "no route home",
			plan: "No route home from Park. There is no path back to home from here.",
			want: false,
		},
		{name: "empty", plan: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NeedsHomeConfirm(tt.plan))
		})
	}
}

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

// A move that costs more than the remaining budget must be rejected without
// spending anything or moving the session.
func TestBudgetGate_BlocksUnaffordableTransition(t *testing.T) {
	app := newGameApp(t, WithBudget(universe.QuantumShiftCost-1))

	out := app.Execute("shift")

	assert.Contains(t, out, "Not enough budget")
	assert.Equal(t, "home", app.SessionEntity().Location(), "a blocked move must not move the session")
	assert.Equal(t, 0.0, app.SessionEntity().CumulativeCost(), "a blocked move must not spend budget")
}

// Returning home must always succeed, even when the walk home costs more than
// the remaining budget: ordinary travel is gated, but the home walk is not. The
// budget is spent down and reported as empty (0), never negative.
func TestHome_AlwaysAllowedEvenWhenOverBudget(t *testing.T) {
	// A budget of 1 covers the walk out to the station but not the walk back.
	app := newGameApp(t, WithBudget(1))

	require.NotContains(t, app.Execute("travel station"), "Not enough budget")
	assert.Equal(t, "station", app.SessionEntity().Location())

	// The return home walk costs another 1, exceeding the budget, but must still
	// complete rather than fail with an insufficient-budget error.
	require.True(t, NeedsHomeConfirm(app.GoHome()))
	out := app.GoHomeConfirm()

	assert.NotContains(t, out, "Failed while returning home")
	assert.Contains(t, out, "You are home")
	assert.Equal(t, "home", app.SessionEntity().Location())
	assert.Equal(t, 0.0, app.Snapshot().RemainingBudget, "an over-budget return clamps remaining to zero, never negative")
}

// An affordable move proceeds and draws down the remaining budget.
func TestBudgetGate_AllowsAffordableTransition(t *testing.T) {
	app := newGameApp(t, WithBudget(100))

	out := app.Execute("shift")

	assert.NotContains(t, out, "Not enough budget")
	assert.Equal(t, universe.QuantumShiftCost, app.SessionEntity().CumulativeCost())
	assert.Equal(t, 100-universe.QuantumShiftCost, app.Snapshot().RemainingBudget)
}

// Reaching the objective and returning home wins: the target-reached and win
// banners appear at the right moments and the snapshot reflects the win.
func TestWin_ReachTargetThenReturnHome(t *testing.T) {
	target := DefaultTarget(universe.DefaultCoordinateVO())
	app := newGameApp(t, WithBudget(DefaultBudget), WithTarget(target))

	assert.False(t, app.Snapshot().ReachedTarget)

	app.Execute("shift")        // -> Q1
	out := app.Execute("shift") // -> Q2 (the target)
	assert.Contains(t, out, TargetReachedMessage)
	assert.True(t, app.Snapshot().ReachedTarget)
	assert.False(t, app.Snapshot().Won)

	app.Execute("shift back")       // -> Q1
	out = app.Execute("shift back") // -> home
	assert.Contains(t, out, WinMessage)
	assert.True(t, app.Snapshot().Won)
	assert.Equal(t, "home", app.SessionEntity().Location())
}

// A win reports par and an efficiency rating: playing the objective optimally
// (straight out to Q2 and straight back) matches par and earns three stars, and
// the win banner carries the rating line.
func TestWin_StarRatingOptimalRun(t *testing.T) {
	target := DefaultTarget(universe.DefaultCoordinateVO())
	app := newGameApp(t, WithBudget(DefaultBudget), WithTarget(target))

	// Par is known and shown before any win.
	assert.Equal(t, 80.0, app.Snapshot().Par)
	assert.Equal(t, 0, app.Snapshot().Stars)

	app.Execute("shift")
	app.Execute("shift")
	app.Execute("shift back")
	out := app.Execute("shift back") // wins at par (cost 80)

	assert.Contains(t, out, "Rating:")
	snap := app.Snapshot()
	assert.True(t, snap.Won)
	assert.Equal(t, 80.0, snap.CumulativeCost)
	assert.Equal(t, MaxStars, snap.Stars, "an at-par run earns three stars")
}

// A wasteful win (extra detours that overspend par) earns fewer stars. A large
// budget lets the detour complete so the rating, not the budget gate, is what is
// exercised.
func TestWin_StarRatingDetourEarnsFewerStars(t *testing.T) {
	target := DefaultTarget(universe.DefaultCoordinateVO())
	app := newGameApp(t, WithBudget(10000), WithTarget(target))

	app.Execute("shift")
	app.Execute("shift") // at target Q2 (cost 40)
	// Waste cost without changing the win path: jump out and back (2 x 800).
	app.Execute("jump")
	app.Execute("jump back")
	app.Execute("shift back")
	app.Execute("shift back") // home; total cost 80 + 1600 = 1680, far over par

	snap := app.Snapshot()
	require.True(t, snap.Won)
	assert.Greater(t, snap.CumulativeCost, snap.Par*2)
	assert.Equal(t, 1, snap.Stars, "a run over twice par earns one star")
}

// The win banner must fire only once, on the transition into the won state.
func TestWin_BannerFiresOnlyOnce(t *testing.T) {
	target := DefaultTarget(universe.DefaultCoordinateVO())
	app := newGameApp(t, WithBudget(DefaultBudget), WithTarget(target))

	app.Execute("shift")
	app.Execute("shift")
	app.Execute("shift back")
	app.Execute("shift back") // wins here

	// A further command (that does not change the won state) must not re-announce.
	out := app.Execute("where")
	assert.False(t, strings.Contains(out, WinMessage), "win is announced once, not on every later command")
	assert.True(t, app.Snapshot().Won)
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

// Par accounts for physical travel, not just reality shifts. An objective at the
// station (a plain walk from home in the test universe, cost 1 each way) has no
// reality-axis component, so its par is the round-trip walk: 1 out + 1 back = 2.
func TestObjectivePar_IncludesPhysicalTravel(t *testing.T) {
	base := universe.DefaultCoordinateVO()
	station := base
	station.Location = "Station"

	app := newGameApp(t, WithBudget(DefaultBudget), WithTarget(station))

	assert.Equal(t, 2.0, app.Snapshot().Par)
}

// Physical travel and reality shifts are orthogonal and additive in par. An
// objective at the station one quantum branch out costs, each way, a quantum
// shift (20) plus the walk (1): 2*(20+1) = 42.
func TestObjectivePar_PhysicalAndRealityLegsSum(t *testing.T) {
	base := universe.DefaultCoordinateVO()
	target := base
	target.Location = "Station"
	target.Quantum = "Q1"

	app := newGameApp(t, WithBudget(DefaultBudget), WithTarget(target))

	assert.Equal(t, 2*(universe.QuantumShiftCost+1), app.Snapshot().Par)
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

// The budget display distinguishes three states cleanly: an unlimited game (no
// budget in force) shows no budget line at all; a finite budget with money left
// shows the remaining amount without the exhausted marker; a finite budget spent
// down to nothing shows the marker so "no limit" is never confused with "no money
// left". A large detour spends past the budget without blocking (the return-home
// walk is never gated), driving remaining to zero.
func TestBudgetDisplay_DistinguishesUnlimitedFromExhausted(t *testing.T) {
	// Unlimited: no budget line.
	unlimited := newGameApp(t)
	assert.NotContains(t, unlimited.Where(), "Budget remaining", "an unlimited game shows no budget line")

	// Finite with money left: the line appears without the exhausted marker.
	funded := newGameApp(t, WithBudget(100))
	out := funded.Where()
	assert.Contains(t, out, "Budget remaining")
	assert.NotContains(t, out, BudgetExhaustedMarker, "a funded budget is not exhausted")

	// Exhausted: spend the whole budget, then confirm the marker appears.
	spent := newGameApp(t, WithBudget(universe.QuantumShiftCost))
	spent.Execute("shift") // spends exactly the budget
	require.Equal(t, 0.0, spent.Snapshot().RemainingBudget)
	assert.Contains(t, spent.Where(), BudgetExhaustedMarker, "a spent-down budget is labelled exhausted")
}

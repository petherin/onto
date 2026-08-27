package facade

import (
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

// The default multi-objective quest chain (Q2 then sim:1) must be reached in
// order: reaching the first waypoint announces the next, reaching the last
// announces the return-home step, and being home mid-chain is not a win. Playing
// it optimally (out to Q2 and back, then one simulation layer in and out) matches
// par (140) and earns three stars.
func TestQuestChain_DefaultChainOptimalRun(t *testing.T) {
	chain := DefaultQuestChain(universe.DefaultCoordinateVO())
	app := newGameApp(t, WithBudget(DefaultBudget), WithTargets(chain...))

	// Par is the whole round trip: two shifts out (40) + two back (40) + one
	// simulation layer in (10) and out (50) = 140.
	assert.Equal(t, 140.0, app.Snapshot().Par)
	assert.Equal(t, 2, app.Snapshot().ObjectiveCount)
	assert.Equal(t, 0, app.Snapshot().ObjectivesDone)

	app.Execute("shift")        // -> Q1
	out := app.Execute("shift") // -> Q2 (first waypoint)
	assert.Contains(t, out, "Objective 1 of 2 reached", "reaching a mid-chain waypoint names the next")
	assert.NotContains(t, out, TargetReachedMessage, "the chain is not fully reached yet")
	assert.Equal(t, 1, app.Snapshot().ObjectivesDone)
	assert.False(t, app.Snapshot().ReachedTarget)

	app.Execute("shift back")       // -> Q1
	out = app.Execute("shift back") // -> home (but chain not finished)
	assert.NotContains(t, out, WinMessage, "being home before the chain is done is not a win")
	assert.False(t, app.Snapshot().Won)

	out = app.Execute("simulate") // -> sim:1 (second and final waypoint)
	assert.Contains(t, out, TargetReachedMessage, "the last waypoint announces the return-home step")
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

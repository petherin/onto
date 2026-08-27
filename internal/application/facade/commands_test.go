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

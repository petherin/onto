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
		universe.NewClusterLocationGenerator(),
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

// Cost reports the session's running travel spend as a whole number: zero before
// any move, then the accumulated edge cost once the traveller has walked.
func TestCost(t *testing.T) {
	app := newGameApp(t)

	assert.Equal(t, "Total journey cost: 0", app.Cost(), "a fresh journey has spent nothing")

	require.NotContains(t, app.Execute("travel station"), "Not enough budget")
	assert.Equal(t, "Total journey cost: 1", app.Cost(), "the walk to the station adds its cost")
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

// newDeadEndApp builds an App whose start location is a genuine physical dead
// end (a well fallen into from home) with only a non-physical drift back out,
// mirroring the seed world's well. The session starts in the well.
func newDeadEndApp(t *testing.T) *App {
	t.Helper()
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()
	well := base
	well.Location = "Well"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home", Coordinate: base}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "well", Name: "Well", Coordinate: well}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "well", Mode: universe.Walk, Cost: 1}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "well", To: "home", Mode: universe.ConsensusShift, Cost: universe.ConsensusShiftCost}))

	repo := mocks.NewMockRepository(t)
	app, err := New(u, repo, "well",
		navigation.NewBFSPathfinder(),
		universe.NewClusterLocationGenerator(),
	)
	require.NoError(t, err)
	return app
}

// TestNonPhysicalMove_FromDeadEnd_GeneratesEscapeOrBlocks covers the core
// behaviour: drifting out of a physical dead end lands the traveller in another
// reality, where a deterministic per-reality roll decides whether a physical way
// out (a nearby node) is generated or the reality is reported as blocked. The
// asserted outcome matches HasPhysicalEscape for the landed coordinate.
func TestNonPhysicalMove_FromDeadEnd_GeneratesEscapeOrBlocks(t *testing.T) {
	app := newDeadEndApp(t)

	out := app.Execute("drift")

	landed := app.Snapshot()
	expectEscape := universe.HasPhysicalEscape(app.session.Coordinate(), universe.ConsensusShiftCost)
	if expectEscape {
		assert.Contains(t, out, "A way out:", "an escapable reality generates a nearby node")
	} else {
		assert.Contains(t, out, "No way out:", "a blocked reality reports no physical route")
	}
	assert.Equal(t, 1, landed.Consensus, "the drift moved the session into consensus divergence 1")
}

// TestNonPhysicalMove_FromDeadEnd_IsDeterministic confirms the escape verdict for
// a given reality is reproducible: resetting and drifting again into the same
// reality produces the same outcome (escape generated or blocked).
func TestNonPhysicalMove_FromDeadEnd_IsDeterministic(t *testing.T) {
	first := newDeadEndApp(t).Execute("drift")
	second := newDeadEndApp(t).Execute("drift")

	firstBlocked := strings.Contains(first, "No way out:")
	secondBlocked := strings.Contains(second, "No way out:")
	assert.Equal(t, firstBlocked, secondBlocked, "the same reality yields the same escape verdict")
}

// TestNonPhysicalMove_Back_DoesNotGenerateEscape confirms unwinding a contextual
// move (align) never spawns an escape node — only forward moves into a new
// reality do.
func TestNonPhysicalMove_Back_DoesNotGenerateEscape(t *testing.T) {
	app := newDeadEndApp(t)
	app.Execute("drift") // consensus 0 -> 1

	out := app.Execute("align") // consensus 1 -> 0, back at the well

	assert.NotContains(t, out, "A way out:", "aligning back does not generate an escape")
	assert.NotContains(t, out, "No way out:", "aligning back does not report escape status")
}

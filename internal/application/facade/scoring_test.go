package facade

import (
	"strings"
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// TestStarsForCost covers the rating tiers and their boundaries, including the
// two-star band and the non-positive-par guard that the win-path tests do not
// exercise directly.
func TestStarsForCost(t *testing.T) {
	tests := []struct {
		name string
		cost float64
		par  float64
		want int
	}{
		{"under par earns three", 50, 100, MaxStars},
		{"at par earns three", 100, 100, MaxStars},
		{"just over par earns two", 101, 100, 2},
		{"at twice par earns two", 200, 100, 2},
		{"just over twice par earns one", 201, 100, 1},
		{"zero par yields none", 50, 0, 0},
		{"negative par yields none", 50, -10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, starsForCost(tt.cost, tt.par))
		})
	}
}

// TestStarBar renders each rating and clamps out-of-range counts to the
// 0..MaxStars band, so a stray negative or oversized count never produces a
// malformed bar.
func TestStarBar(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"none", 0, "☆☆☆"},
		{"one", 1, "★☆☆"},
		{"two", 2, "★★☆"},
		{"three", 3, "★★★"},
		{"negative clamps to none", -1, "☆☆☆"},
		{"over max clamps to full", 5, "★★★"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, starBar(tt.n))
		})
	}
}

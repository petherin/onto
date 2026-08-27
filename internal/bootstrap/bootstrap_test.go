package bootstrap

import (
	"testing"

	"github.com/petherin/onto/internal/application/facade"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig_GameEnv covers how ONTO_GAME and ONTO_BUDGET are read:
// game mode is on by default and only an explicit falsey value disables it,
// while a positive ONTO_BUDGET overrides the pool and anything else falls back
// to 0 (meaning "use the default budget").
func TestDefaultConfig_GameEnv(t *testing.T) {
	tests := []struct {
		name       string
		game       string
		budget     string
		wantGame   bool
		wantBudget float64
	}{
		{name: "defaults (unset)", wantGame: true, wantBudget: 0},
		{name: "game off via 0", game: "0", wantGame: false},
		{name: "game off via false", game: "false", wantGame: false},
		{name: "game off via off", game: "OFF", wantGame: false},
		{name: "game on via 1", game: "1", wantGame: true},
		{name: "budget override", budget: "250", wantGame: true, wantBudget: 250},
		{name: "budget ignored when non-positive", budget: "-5", wantGame: true, wantBudget: 0},
		{name: "budget ignored when unparseable", budget: "lots", wantGame: true, wantBudget: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ONTO_GAME", tt.game)
			t.Setenv("ONTO_BUDGET", tt.budget)

			cfg := DefaultConfig()
			assert.Equal(t, tt.wantGame, cfg.Game)
			assert.Equal(t, tt.wantBudget, cfg.Budget)
		})
	}
}

// TestGameOptions covers the option builder shared by both entry points: no
// options when the game is off, and a budget + target when it is on (the target
// deriving from the start coordinate). A positive configured budget wins over
// the default.
func TestGameOptions(t *testing.T) {
	state := testState(t)

	// Game off: no options, so a session built from them has no budget/target.
	assert.Empty(t, GameOptions(Config{Game: false}, state))

	// Game on, default budget: budget in force and an objective set.
	app := newApp(t, state, GameOptions(Config{Game: true}, state))
	snap := app.Snapshot()
	assert.True(t, snap.HasBudget)
	assert.Equal(t, facade.DefaultBudget, snap.Budget)
	assert.True(t, snap.HasTarget)

	// Game on with an explicit budget: the override is applied.
	app = newApp(t, state, GameOptions(Config{Game: true, Budget: 250}, state))
	assert.Equal(t, 250.0, app.Snapshot().Budget)
}

// testState builds a State on the shared default universe, starting at home.
func testState(t *testing.T) State {
	t.Helper()
	u, err := buildDefaultUniverse()
	require.NoError(t, err)
	return State{Universe: u, StartID: "home"}
}

// newApp builds a facade.App from a State and options for assertions.
func newApp(t *testing.T, state State, opts []facade.Option) *facade.App {
	t.Helper()
	app, err := facade.New(
		state.Universe,
		nil,
		state.StartID,
		navigation.NewBFSPathfinder(),
		universe.NewSequentialLocationGenerator(),
		opts...,
	)
	require.NoError(t, err)
	return app
}

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
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

// TestDefaultConfig_Quest covers how ONTO_QUEST is parsed: a comma-separated
// list of Onto Addresses becomes an ordered chain of coordinates, whitespace and
// blank entries are tolerated, and an unset value yields nil (use the default
// chain).
func TestDefaultConfig_Quest(t *testing.T) {
	first := universe.DefaultCoordinateVO()
	first.Quantum = "Q2"
	second := universe.DefaultCoordinateVO()
	second.Simulation = 1

	// Two addresses, with surrounding spaces and a trailing blank entry.
	t.Setenv("ONTO_QUEST", " "+first.OntoAddress()+" , "+second.OntoAddress()+" , ")
	cfg := DefaultConfig()
	require.Len(t, cfg.Quest, 2)
	assert.Equal(t, "Q2", cfg.Quest[0].Quantum)
	assert.Equal(t, 1, cfg.Quest[1].Simulation)

	// Unset: nil, so GameOptions falls back to the default chain.
	t.Setenv("ONTO_QUEST", "")
	assert.Nil(t, DefaultConfig().Quest)
}

// TestGameOptions covers the option builder shared by both entry points: no
// options when the game is off, and a budget + target when it is on (the target
// deriving from the start coordinate). A positive configured budget wins over
// the default, and a configured quest wins over the default chain.
func TestGameOptions(t *testing.T) {
	state := testState(t)

	// Game off: no options, so a session built from them has no budget/target.
	assert.Empty(t, GameOptions(Config{Game: false}, state))

	// Game on, default budget: budget in force and the default two-waypoint chain.
	app := newApp(t, state, GameOptions(Config{Game: true}, state))
	snap := app.Snapshot()
	assert.True(t, snap.HasBudget)
	assert.Equal(t, facade.DefaultBudget, snap.Budget)
	assert.True(t, snap.HasTarget)
	assert.Equal(t, 2, snap.ObjectiveCount)

	// Game on with an explicit budget: the override is applied.
	app = newApp(t, state, GameOptions(Config{Game: true, Budget: 250}, state))
	assert.Equal(t, 250.0, app.Snapshot().Budget)

	// Game on with a configured quest: it replaces the default chain, so a
	// single-waypoint quest yields exactly one objective.
	home, ok := state.Universe.GetLocation(state.StartID)
	require.True(t, ok)
	waypoint := facade.DefaultTarget(home.Coordinate)
	app = newApp(t, state, GameOptions(Config{Game: true, Quest: []universe.CoordinateVO{waypoint}}, state))
	qs := app.Snapshot()
	assert.True(t, qs.HasTarget)
	assert.Equal(t, 1, qs.ObjectiveCount)
}

// TestDefaultConfig_Objectives covers ONTO_OBJECTIVES parsing: a comma-separated
// pool of Onto Addresses becomes a coordinate pool (whitespace and blank entries
// tolerated), and an unset value yields nil.
func TestDefaultConfig_Objectives(t *testing.T) {
	a := universe.DefaultCoordinateVO()
	a.Quantum = "Q1"
	b := universe.DefaultCoordinateVO()
	b.Simulation = 1

	t.Setenv("ONTO_OBJECTIVES", " "+a.OntoAddress()+" , "+b.OntoAddress()+" , ")
	cfg := DefaultConfig()
	require.Len(t, cfg.Objectives, 2)
	assert.Equal(t, "Q1", cfg.Objectives[0].Quantum)
	assert.Equal(t, 1, cfg.Objectives[1].Simulation)

	t.Setenv("ONTO_OBJECTIVES", "")
	assert.Nil(t, DefaultConfig().Objectives)
}

// TestGameOptions_ObjectivePool covers the pool branch: with game on and an
// objective pool, a random quest of QuestSizeMin..QuestSizeMax objectives is
// built; a fixed ONTO_QUEST chain still takes precedence over the pool; and game
// mode off yields no options at all even when a pool is configured.
func TestGameOptions_ObjectivePool(t *testing.T) {
	state := testState(t)
	pool := questPool(6)

	app := newApp(t, state, GameOptions(Config{Game: true, Objectives: pool}, state))
	snap := app.Snapshot()
	assert.True(t, snap.HasTarget)
	assert.GreaterOrEqual(t, snap.ObjectiveCount, facade.QuestSizeMin)
	assert.LessOrEqual(t, snap.ObjectiveCount, facade.QuestSizeMax)

	// A fixed quest wins over the pool.
	home, ok := state.Universe.GetLocation(state.StartID)
	require.True(t, ok)
	fixed := []universe.CoordinateVO{facade.DefaultTarget(home.Coordinate)}
	app = newApp(t, state, GameOptions(Config{Game: true, Quest: fixed, Objectives: pool}, state))
	assert.Equal(t, 1, app.Snapshot().ObjectiveCount)

	// Game off: no quest generation at all.
	assert.Empty(t, GameOptions(Config{Game: false, Objectives: pool}, state))
}

// questPool builds n distinct candidate objectives for pool tests.
func questPool(n int) []universe.CoordinateVO {
	pool := make([]universe.CoordinateVO, n)
	for i := range pool {
		c := universe.DefaultCoordinateVO()
		c.Quantum = fmt.Sprintf("Q%d", i+1)
		pool[i] = c
	}
	return pool
}

// TestLoadDotEnv covers the minimal .env loader: it sets variables that are not
// already present, strips matching quotes, ignores comments/blank/malformed
// lines, never overrides a real environment variable, and treats a missing file
// as a no-op.
func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# a comment\n\nONTO_TEST_A=alpha\nONTO_TEST_B=\"quoted value\"\nONTO_TEST_PRESET=fromfile\nBADLINE\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	t.Setenv("ONTO_TEST_PRESET", "fromenv")
	t.Cleanup(func() {
		_ = os.Unsetenv("ONTO_TEST_A")
		_ = os.Unsetenv("ONTO_TEST_B")
	})

	loadDotEnv(path)
	assert.Equal(t, "alpha", os.Getenv("ONTO_TEST_A"))
	assert.Equal(t, "quoted value", os.Getenv("ONTO_TEST_B"))
	assert.Equal(t, "fromenv", os.Getenv("ONTO_TEST_PRESET"), "a real env var is never overridden")

	// Missing file is a silent no-op.
	loadDotEnv(filepath.Join(dir, "does-not-exist.env"))
}

// TestBuildDefaultUniverse_WellIsSealedVault confirms the hand-placed well is a
// first-class member of the trap system: it carries TrapSealedVault (so it
// surfaces identically to a generated sealed vault) yet, being in base reality,
// has no physical exit — its only way out is the non-physical drift.
func TestBuildDefaultUniverse_WellIsSealedVault(t *testing.T) {
	u, err := buildDefaultUniverse()
	require.NoError(t, err)

	well, ok := u.GetLocation("well")
	require.True(t, ok)
	assert.Equal(t, universe.TrapSealedVault, well.Trap)
	assert.False(t, universe.HasPhysicalExit(u, "well"), "the well is a sealed sink")
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
		universe.NewClusterLocationGenerator(),
		opts...,
	)
	require.NoError(t, err)
	return app
}

package facade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReset rebuilds the universe to its construction snapshot: reality
// transitions that grew the map (new branch locations and edges) are discarded,
// the session returns home in base reality, and the app is marked dirty so the
// cleared map can be saved over the grown one.
func TestReset(t *testing.T) {
	app := newGameApp(t)

	initialLocations := len(app.Aggregate().AllLocations())
	initialEdges := len(app.Aggregate().AllEdgesFlat())
	home, ok := app.Aggregate().GetLocation("home")
	require.True(t, ok)

	// Grow the universe: each forward shift branches a new quantum location.
	app.Execute("shift")
	app.Execute("shift")
	require.Greater(t, len(app.Aggregate().AllLocations()), initialLocations, "shifting must grow the map")
	require.NotEqual(t, "home", app.SessionEntity().Location(), "shifting moves the session off home")

	out := app.Reset()

	assert.Equal(t, "Map reset to the starting realities.", out)
	assert.Equal(t, initialLocations, len(app.Aggregate().AllLocations()), "reset discards generated locations")
	assert.Equal(t, initialEdges, len(app.Aggregate().AllEdgesFlat()), "reset discards generated edges")
	assert.Equal(t, "home", app.SessionEntity().Location(), "reset returns the session home")
	assert.Equal(t, home.Coordinate, app.SessionEntity().Coordinate(), "reset returns the session to base reality")
	assert.True(t, app.IsDirty(), "reset marks the app dirty so the cleared map can be saved")
}

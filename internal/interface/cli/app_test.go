package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppWhere(t *testing.T) {
	app := NewApp()
	output := app.Execute("where")

	assert.Contains(t, output, "Reality Coordinate")
	assert.Contains(t, output, "Earth")
}

func TestAppRouteToStation(t *testing.T) {
	app := NewApp()
	output := app.Execute("route station")

	assert.Contains(t, output, "Route")
	assert.Contains(t, output, "Station")
}

func TestAppTravelToStation(t *testing.T) {
	app := NewApp()
	output := app.Execute("travel station")

	assert.Contains(t, output, "Arrived")
}

func TestAppTravelShowsPossibleJourneys(t *testing.T) {
	app := NewApp()
	output := app.Execute("travel station")

	assert.Contains(t, output, "Possible journeys")
}

func TestAppSuggestsSimilarDestination(t *testing.T) {
	app := NewApp()
	output := app.Execute("route parl")

	assert.Contains(t, output, "Did you mean")
	assert.Contains(t, output, "park")
}

func TestAppShift_CreatesQuantumBranch(t *testing.T) {
	app := NewApp()
	output := app.Execute("shift")

	assert.Contains(t, output, "Q1")
	assert.Contains(t, output, "quantum")
}

func TestAppShift_UpdatesLocation(t *testing.T) {
	app := NewApp()
	app.Execute("shift")
	output := app.Execute("where")

	assert.Contains(t, output, "home-q1")
}

func TestAppHelp_ListsAllCommands(t *testing.T) {
	app := NewApp()
	output := app.Execute("help")

	for _, cmd := range []string{"where", "look", "ls", "route", "travel", "shift", "exit"} {
		assert.Contains(t, output, cmd, "help should list command %q", cmd)
	}
}

func TestAppUnknownCommand_SuggestsAlternative(t *testing.T) {
	app := NewApp()
	output := app.Execute("wher") // typo of "where"

	assert.Contains(t, output, "Did you mean")
	assert.Contains(t, output, "where")
}

func TestAppEmptyInput_ReturnsEmpty(t *testing.T) {
	app := NewApp()
	output := app.Execute("")

	require.Empty(t, output)
}

func TestAppLook_ReturnsDescription(t *testing.T) {
	app := NewApp()
	output := app.Execute("look")

	assert.Contains(t, output, "Home")
}

func TestAppList_ShowsQuantumOption(t *testing.T) {
	app := NewApp()
	output := app.Execute("ls")

	assert.Contains(t, output, "shift")
}

func TestGoHome_WhenAlreadyHome_ReportsIt(t *testing.T) {
	app := NewApp()
	output := app.Execute("home")

	assert.Contains(t, output, "already home")
}

func TestGoHome_FromStation_ReturnsPlan(t *testing.T) {
	app := NewApp()
	app.Execute("travel station")
	output := app.Execute("home")

	// Should show the plan and ask for confirmation; no movement yet.
	assert.Contains(t, output, "home")
}

func TestTravel_ToQuantumBranchID_SuggestsShift(t *testing.T) {
	app := NewApp()
	// home-q1 doesn't exist yet; it's the next quantum branch.
	output := app.Execute("travel home-q1")

	assert.Contains(t, output, "shift")
}

func TestTravel_ToTimelineBranchID_SuggestsJump(t *testing.T) {
	app := NewApp()
	// home-t1 doesn't exist yet; it's the next timeline branch.
	output := app.Execute("travel home-t1")

	assert.Contains(t, output, "jump")
}

func TestShift_BranchHasPhysicalDestinations(t *testing.T) {
	app := NewApp()
	app.Execute("shift")
	output := app.Execute("ls")

	// home-q1 should inherit physical edges from home (walk to station etc.)
	assert.Contains(t, output, "walk")
}

func TestJump_BranchHasPhysicalDestinations(t *testing.T) {
	app := NewApp()
	app.Execute("jump")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
}

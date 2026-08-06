package cli

import (
	"path/filepath"
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

func TestAppDriftAndAlign(t *testing.T) {
	app := NewApp()

	driftOutput := app.Execute("drift")
	assert.Contains(t, driftOutput, "Consensus divergence entered: level 1")
	assert.Contains(t, app.Execute("where"), "Consensus: 1")

	alignOutput := app.Execute("align")
	assert.Contains(t, alignOutput, "Shared consensus approached: level 0")
}

func TestAppDrift_CanTravelWithinConsensusDivergence(t *testing.T) {
	app := NewApp()
	app.Execute("drift")

	output := app.Execute("travel station-c1")

	assert.Contains(t, output, "Arrived")
	assert.Contains(t, app.Execute("where"), "Consensus: 1")
}

func TestAppContextualLocation_OffersTransitionsAndReturns(t *testing.T) {
	app := NewApp()
	app.Execute("drift")
	app.Execute("travel station-c1")

	list := app.Execute("ls")
	assert.Contains(t, list, "use 'shift'")
	assert.Contains(t, list, "use 'jump'")
	assert.Contains(t, list, "use 'drift'")
	assert.Contains(t, list, "use 'align'")

	assert.Contains(t, app.Execute("shift"), "Quantum branch entered")
	assert.Contains(t, app.Execute("shift back"), "Quantum branch exited")
	assert.Contains(t, app.Execute("jump"), "Timeline branch entered")
	assert.Contains(t, app.Execute("jump back"), "Timeline branch exited")
	assert.Contains(t, app.Execute("align"), "Shared consensus approached")
}

func TestGoHome_UnwindsConsensusDivergence(t *testing.T) {
	app := NewApp()
	app.Execute("drift")

	plan := app.GoHome()
	assert.Contains(t, plan, "align      (consensus 1 \u2192 0)")

	result := app.GoHomeConfirm()
	assert.Contains(t, result, "Consensus alignment → level 0")
	assert.Equal(t, 0, app.session.ConsensusLevel())
}

func TestGoHome_SeparatesConsensusAndPhysicalCosts(t *testing.T) {
	app := NewApp()
	app.Execute("drift")
	app.Execute("travel station-c1")

	plan := app.GoHome()

	assert.Contains(t, plan, "align      (consensus 1 → 0)  cost 5")
	assert.Contains(t, plan, "Estimated cost: 10")
	assert.NotContains(t, plan, "Station (consensus 1) → Station")
}

func TestAppHelp_ListsAllCommands(t *testing.T) {
	app := NewApp()
	output := app.Execute("help")

	for _, cmd := range []string{"where", "look", "ls", "route", "travel", "shift", "drift", "align", "exit"} {
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
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := NewApp()
	// home-q1 doesn't exist yet; it's the next quantum branch.
	output := app.Execute("travel home-q1")

	assert.Contains(t, output, "shift")
}

func TestTravel_ToTimelineBranchID_SuggestsJump(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := NewApp()
	// home-t1 doesn't exist yet; it's the next timeline branch.
	output := app.Execute("travel home-t1")

	assert.Contains(t, output, "jump")
}

func TestShift_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := NewApp()
	app.Execute("shift")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (Q1)")
}

func TestList_ShowsTravelablePhysicalLocationIDs(t *testing.T) {
	app := NewApp()
	app.Execute("travel station")
	app.Execute("shift")

	output := app.Execute("ls")

	assert.Contains(t, output, "City Centre (Q1) [city-centre-q1]")
}

func TestJump_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := NewApp()
	app.Execute("jump")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (T1)")
}

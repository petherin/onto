package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/petherin/onto/internal/application/facade"
	"github.com/petherin/onto/internal/bootstrap"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "onto-cli-test-*")
	if err != nil {
		panic(err)
	}
	dataPath := filepath.Join(tempDir, "locations.json")
	if err := os.Setenv("ONTO_DATA_FILE", dataPath); err != nil {
		_ = os.RemoveAll(tempDir)
		panic(err)
	}

	code := m.Run()
	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}

// newTestApp bootstraps a *App using the current environment config (which
// TestMain or individual t.Setenv calls pre-configure). It is the test
// replacement for the old NewApp() factory that lived in the cli package.
func newTestApp(t *testing.T) *App {
	t.Helper()
	state, err := bootstrap.Bootstrap(bootstrap.DefaultConfig())
	require.NoError(t, err)
	f, err := facade.New(
		state.Universe,
		state.Repo,
		state.StartID,
		navigation.NewBFSPathfinder(),
		universe.NewClusterLocationGenerator(),
	)
	require.NoError(t, err)
	return New(f)
}

func TestAppWhere(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("where")

	assert.Contains(t, output, "Reality Coordinate")
	assert.Contains(t, output, "Earth")
}

func newMockedApp(t *testing.T) (*App, *mocks.MockRepository) {
	t.Helper()
	u := mocks.NewTestUniverse()
	repo := mocks.NewMockRepository(t)
	f, err := facade.New(u, repo, "home", navigation.NewBFSPathfinder(), universe.NewClusterLocationGenerator())
	require.NoError(t, err)
	return New(f), repo
}

func TestAppRouteToStation(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("route station")

	assert.Contains(t, output, "Route")
	assert.Contains(t, output, "Station")
}

func TestAppTravelToStation(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("travel station")

	assert.Contains(t, output, "Arrived")
}

func TestAppTravelShowsPossibleJourneys(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("travel station")

	assert.Contains(t, output, "Possible journeys")
}

func TestAppNumberedJourneyTravelsToDisplayedDestination(t *testing.T) {
	app := newTestApp(t)

	list := app.Execute("ls")
	assert.Contains(t, list, "1. Station")

	output := app.Execute("1")
	assert.Contains(t, output, "Arrived.")
	assert.Equal(t, "station", app.app.SessionEntity().Location())
}

func TestAppNumberedJourneyExecutesContextualReturn(t *testing.T) {
	app := newTestApp(t)
	app.Execute("shift")

	number := 0
	options, _ := app.app.JourneyOptions(app.app.Aggregate().EdgesFrom(app.app.SessionEntity().Location()))
	for i, option := range options {
		if option.Kind == facade.JourneyShiftBack {
			number = i + 1
			break
		}
	}
	require.NotZero(t, number)
	output := app.Execute(fmt.Sprintf("%d", number))

	assert.Contains(t, output, "Quantum branch exited")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
}

func TestAppNumberedJourneyRejectsUnknownNumber(t *testing.T) {
	app := newTestApp(t)

	output := app.Execute("99")

	assert.Contains(t, output, "No possible journey numbered 99")
}

func TestAppSuggestsSimilarDestination(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("route parl")

	assert.Contains(t, output, "Did you mean")
	assert.Contains(t, output, "park")
}

func TestAppShift_CreatesQuantumBranch(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("shift")

	assert.Contains(t, output, "Q1")
	assert.Contains(t, output, "quantum")
}

func TestAppShift_UpdatesLocation(t *testing.T) {
	app := newTestApp(t)
	app.Execute("shift")
	output := app.Execute("where")

	assert.Contains(t, output, "home-q1")
}

func TestAppUniverse_CreatesUniverseBranch(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("universe")

	assert.Contains(t, output, "U1")
	assert.Contains(t, output, "universe")
}

func TestAppUniverse_UpdatesLocation(t *testing.T) {
	app := newTestApp(t)
	app.Execute("universe")
	output := app.Execute("where")

	assert.Contains(t, output, "home-u1")
}

func TestAppUniverseAndBack(t *testing.T) {
	app := newTestApp(t)

	universeOutput := app.Execute("universe")
	assert.Contains(t, universeOutput, "Bubble universe entered: U1")

	backOutput := app.Execute("universe back")
	assert.Contains(t, backOutput, "Bubble universe exited: Origin")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
}

func TestAppStructure_CreatesMathematicsBranch(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("structure")

	assert.Contains(t, output, "M1")
	assert.Contains(t, output, "Mathematical structure entered")
}

func TestAppStructure_UpdatesLocation(t *testing.T) {
	app := newTestApp(t)
	app.Execute("structure")
	output := app.Execute("where")

	assert.Contains(t, output, "home-m1")
	assert.Contains(t, output, "Mathematics: M1")
}

func TestAppStructureAndBack(t *testing.T) {
	app := newTestApp(t)

	structureOutput := app.Execute("structure")
	assert.Contains(t, structureOutput, "Mathematical structure entered: M1")

	backOutput := app.Execute("structure back")
	assert.Contains(t, backOutput, "Mathematical structure exited: Classical")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
}

func TestAppSimulate_CreatesSimulationBranch(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("simulate")

	assert.Contains(t, output, "Simulation layer entered: depth 1")
	assert.Contains(t, output, "simulation")
}

func TestAppSimulate_UpdatesLocation(t *testing.T) {
	app := newTestApp(t)
	app.Execute("simulate")
	output := app.Execute("where")

	assert.Equal(t, "home-s1", app.app.SessionEntity().Location())
	assert.Contains(t, output, "Simulation: 1")
	assert.Contains(t, output, "/sim:1@")
}

func TestAppSimulateAndBack(t *testing.T) {
	app := newTestApp(t)

	enter := app.Execute("simulate")
	assert.Contains(t, enter, "Simulation layer entered: depth 1")

	exit := app.Execute("simulate back")
	assert.Contains(t, exit, "Simulation layer exited: depth 0")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
}

func TestAppDriftAndAlign(t *testing.T) {
	app := newTestApp(t)

	driftOutput := app.Execute("drift")
	assert.Contains(t, driftOutput, "Consensus divergence entered: level 1")
	assert.Contains(t, app.Execute("where"), "Consensus: 1")

	alignOutput := app.Execute("align")
	assert.Contains(t, alignOutput, "Shared consensus approached: level 0")
}

func TestAppObserveAndReturn(t *testing.T) {
	app := newTestApp(t)

	observeOutput := app.Execute("observe Machine")
	assert.Contains(t, observeOutput, "Observer perspective entered: Machine")
	assert.Contains(t, app.Execute("where"), "Observer : Machine")

	returnOutput := app.Execute("observe back")
	assert.Contains(t, returnOutput, "Observer perspective restored: Human")
	assert.NotContains(t, app.Execute("ls"), "Return to the previous observer perspective")
}

func TestAppTimeAndReturn(t *testing.T) {
	app := newTestApp(t)

	output := app.Execute("time 2042-01-02T03:04:05Z")
	assert.Contains(t, output, "Temporal branch entered")
	assert.Contains(t, app.Execute("where"), "2042-01-02T03:04:05Z")

	output = app.Execute("time back")
	assert.Contains(t, output, "Temporal branch exited")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
}

func TestAppDrift_CanTravelWithinConsensusDivergence(t *testing.T) {
	app := newTestApp(t)
	app.Execute("drift")

	output := app.Execute("travel station-c1")

	assert.Contains(t, output, "Arrived")
	assert.Contains(t, app.Execute("where"), "Consensus: 1")
}

func TestAppContextualLocation_OffersTransitionsAndReturns(t *testing.T) {
	app := newTestApp(t)
	app.Execute("drift")
	app.Execute("travel station-c1")

	list := app.Execute("ls")
	assert.Contains(t, list, "— shift)")
	assert.Contains(t, list, "— jump)")
	assert.Contains(t, list, "— universe)")
	assert.Contains(t, list, "— drift)")
	assert.Contains(t, list, "(align)")

	assert.Contains(t, app.Execute("shift"), "Quantum branch entered")
	assert.Contains(t, app.Execute("shift back"), "Quantum branch exited")
	assert.Contains(t, app.Execute("jump"), "Arrived in a distant Hubble volume")
	assert.Contains(t, app.Execute("jump back"), "Returned from the distant Hubble volume")
	assert.Contains(t, app.Execute("universe"), "Bubble universe entered")
	assert.Contains(t, app.Execute("universe back"), "Bubble universe exited")
	assert.Contains(t, app.Execute("align"), "Shared consensus approached")
}

func TestGoHome_UnwindsConsensusDivergence(t *testing.T) {
	app := newTestApp(t)
	app.Execute("drift")

	plan := app.GoHome()
	assert.Contains(t, plan, "align      (consensus 1 \u2192 0)")

	result := app.GoHomeConfirm()
	assert.Contains(t, result, "Consensus alignment → level 0")
	assert.Equal(t, 0, app.app.SessionEntity().ConsensusLevel())
}

func TestGoHome_UnwindsUniverseTransition(t *testing.T) {
	app := newTestApp(t)
	app.Execute("universe")

	plan := app.GoHome()
	assert.Contains(t, plan, "universe back")

	result := app.GoHomeConfirm()
	assert.Contains(t, result, "universe back")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
	assert.Equal(t, 0, app.app.SessionEntity().UniverseLevel())
}

func TestGoHome_UnwindsMathematicsTransition(t *testing.T) {
	app := newTestApp(t)
	app.Execute("structure")

	plan := app.GoHome()
	assert.Contains(t, plan, "structure back")

	result := app.GoHomeConfirm()
	assert.Contains(t, result, "structure back")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
	assert.Equal(t, 0, app.app.SessionEntity().MathematicsLevel())
}

func TestGoHome_UnwindsSimulationTransition(t *testing.T) {
	app := newTestApp(t)
	app.Execute("simulate")

	plan := app.GoHome()
	assert.Contains(t, plan, "simulate back")

	result := app.GoHomeConfirm()
	assert.Contains(t, result, "simulate back")
	assert.Equal(t, "home", app.app.SessionEntity().Location())
	assert.Equal(t, 0, app.app.SessionEntity().SimulationLevel())
}

func TestGoHome_UnwindsObserverPerspective(t *testing.T) {
	app := newTestApp(t)
	app.Execute("observe Cat")

	plan := app.GoHome()
	assert.Contains(t, plan, "observe back (Cat → Human)  cost 2")

	result := app.GoHomeConfirm()
	assert.Contains(t, result, "Observer return → Human")
	assert.Equal(t, "Human", app.app.SessionEntity().Coordinate().Observer)
}

func TestGoHome_SeparatesConsensusAndPhysicalCosts(t *testing.T) {
	app := newTestApp(t)
	app.Execute("drift")
	app.Execute("travel station-c1")

	plan := app.GoHome()

	// The consensus unwind (align, cost 5) is listed separately from the physical
	// walk home (station → home, cost 1 in the two-way seed world) — 6 in total.
	assert.Contains(t, plan, "align      (consensus 1 → 0)  cost 5")
	assert.Contains(t, plan, "travel     (station -> home)  cost 1")
	assert.Contains(t, plan, "Estimated cost: 6")
	assert.NotContains(t, plan, "Station (consensus 1) → Station")
}

func TestGoHome_UnwindsNestedContextsBeforeTime(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	app.Execute("time 2027-01-01T00:00:00Z")
	app.Execute("shift")
	app.Execute("jump")
	app.Execute("universe")
	app.Execute("drift")
	app.Execute("observe dog")

	plan := app.GoHome()

	assert.NotContains(t, plan, "return path unavailable")
	assert.Contains(t, plan, "time back")
	assert.Contains(t, plan, "universe back")
}

func TestGoHome_UnwindsRecordedTransitionOrder(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	app.Execute("shift")
	app.Execute("time 2027-01-01T00:00:00Z")
	app.Execute("jump")
	app.Execute("universe")
	app.Execute("observe dog")

	plan := app.GoHome()

	observer := strings.Index(plan, "observe back")
	universeBack := strings.Index(plan, "universe back")
	jump := strings.Index(plan, "jump back")
	time := strings.Index(plan, "time back")
	shift := strings.Index(plan, "shift back")
	assert.True(t, observer < universeBack && universeBack < jump && jump < time && time < shift, plan)
}

func TestAppHelp_ListsAllCommands(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("help")

	for _, cmd := range []string{"where", "look", "ls", "route", "travel", "shift", "universe", "drift", "align", "observe", "save", "exit"} {
		assert.Contains(t, output, cmd, "help should list command %q", cmd)
	}
}

func TestAppShift_DoesNotSaveImmediately(t *testing.T) {
	app, repo := newMockedApp(t)

	output := app.Execute("shift")

	assert.Contains(t, output, "Quantum branch entered")
	repo.AssertNotCalled(t, "Save", testifymock.Anything)
	assert.True(t, app.app.IsDirty())
}

func TestAppSaveCommand_SavesAndReportsSuccess(t *testing.T) {
	app, repo := newMockedApp(t)
	app.Execute("shift")
	repo.EXPECT().Save(app.app.Aggregate()).Return(nil).Once()

	output := app.Execute("save")

	assert.Equal(t, msgSaved, output)
	assert.False(t, app.app.IsDirty())
}

func TestAppSaveIfDirty_OnlySavesWhenDirty(t *testing.T) {
	t.Run("dirty", func(t *testing.T) {
		app, repo := newMockedApp(t)
		app.Execute("shift") // marks dirty without saving
		repo.EXPECT().Save(app.app.Aggregate()).Return(nil).Once()

		err := app.SaveIfDirty()

		require.NoError(t, err)
		assert.False(t, app.app.IsDirty())
	})

	t.Run("clean", func(t *testing.T) {
		app, repo := newMockedApp(t)

		err := app.SaveIfDirty()

		require.NoError(t, err)
		repo.AssertNotCalled(t, "Save", testifymock.Anything)
	})
}

func TestAppUnknownCommand_SuggestsAlternative(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("wher") // typo of "where"

	assert.Contains(t, output, "Did you mean")
	assert.Contains(t, output, "where")
}

func TestAppEmptyInput_ReturnsEmpty(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("")

	require.Empty(t, output)
}

func TestAppLook_ReturnsDescription(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("look")

	assert.Contains(t, output, "Home")
}

func TestAppList_ShowsQuantumOption(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("ls")

	assert.Contains(t, output, "shift")
	assert.Contains(t, output, "time <RFC3339>")
}

func TestGoHome_WhenAlreadyHome_ReportsIt(t *testing.T) {
	app := newTestApp(t)
	output := app.Execute("home")

	assert.Contains(t, output, "already home")
}

func TestGoHome_FromStation_ReturnsPlan(t *testing.T) {
	app := newTestApp(t)
	app.Execute("travel station")
	output := app.Execute("home")

	// Should show the plan and ask for confirmation; no movement yet.
	assert.Contains(t, output, "home")
}

// The well is a genuine physical dead end: you fall in from the park and there
// is no physical way back out (its only exit is a non-physical drift). GoHome is
// the safety hatch, so it must offer a real escape plan rather than falsely
// claiming the traveller is already home — the escapability itself is asserted by
// TestGoHome_EscapesWellSinkViaSafetyHatch.
func TestGoHome_FromDeadEnd_DoesNotClaimAlreadyHome(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	app.Execute("travel park")
	app.Execute("travel well")
	output := app.Execute("home")

	assert.NotContains(t, output, "already home")
	assert.True(t, facade.NeedsHomeConfirm(output),
		"the safety hatch must offer a completable escape plan from the well")
}

// Travelling into the well must not auto-generate a nearby "ladder": the well is
// a designed physical sink (its only exit is a non-physical drift), so unlike an
// ordinary leaf it must not spawn a walkable escape node on arrival.
func TestTravelToWell_DoesNotAutoGenerateEscape(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	app.Execute("travel park")
	before := len(app.app.GraphSnapshot().Nodes)

	output := app.Execute("travel well")

	assert.NotContains(t, output, "Auto-generated")
	assert.Len(t, app.app.GraphSnapshot().Nodes, before,
		"the well is a designed sink and must not spawn a nearby escape node")
}

// return home is the safety hatch: it must always get the traveller back, even
// from a genuine physical sink like the well whose only exit is a non-physical
// drift. The plan must include that escape (so it does not advertise a journey it
// cannot finish), and confirming it must actually land the session at home rather
// than stranding it in the well family.
func TestGoHome_EscapesWellSinkViaSafetyHatch(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	app.Execute("travel park")
	app.Execute("travel well")
	require.Equal(t, "well", app.app.SessionEntity().Location())

	plan := app.GoHome()
	assert.True(t, facade.NeedsHomeConfirm(plan), "a completable plan must be offered, not a terminal message")
	assert.Contains(t, plan, "escape", "the plan must include the non-physical escape out of the well")

	result := app.GoHomeConfirm()
	assert.NotContains(t, result, "Failed while returning home")
	assert.Equal(t, "home", app.app.SessionEntity().Location(),
		"the safety hatch must land the traveller home, not strand them in the well")
}

// idFromWayOut extracts the first generated location ID from an "A way out: ..."
// escape message, whose list entries read "Name (id)". Tests travel onto the
// generated ladder by ID because auto-generated names are no longer fixed.
func idFromWayOut(out string) string {
	open := strings.Index(out, "(")
	close := strings.Index(out, ")")
	if open < 0 || close < 0 || close < open {
		return ""
	}
	return out[open+1 : close]
}

// A harder reproduction of the reported strand: escape the well by drifting into
// nested realities until one offers a physical ladder, walk onto it, then head
// home. Even after unwinding the recorded context lands the traveller back in the
// well family (a base-reality sink), the safety hatch must still get them home.
func TestGoHome_EscapesWellAfterNestedLadder(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	app.Execute("travel park")
	app.Execute("travel well")

	// Drift between realities until one yields a ladder, then walk onto it.
	laddered := false
	for i := 0; i < 30 && !laddered; i++ {
		out := app.Execute("drift")
		if strings.Contains(out, "A way out") {
			app.Execute("travel " + idFromWayOut(out))
			laddered = true
		}
	}
	require.True(t, laddered, "expected some drifted reality to offer a physical ladder")

	result := app.GoHomeConfirm()
	assert.NotContains(t, result, "Failed while returning home")
	assert.Equal(t, "home", app.app.SessionEntity().Location(),
		"the safety hatch must land the traveller home from a nested well ladder")
}

func TestTravel_ToQuantumBranchID_SuggestsShift(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	// home-q1 doesn't exist yet; it's the next quantum branch.
	output := app.Execute("travel home-q1")

	assert.Contains(t, output, "shift")
}

func TestTravel_ToTimelineBranchID_SuggestsJump(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	// home-t1 doesn't exist yet; it's the next timeline branch.
	output := app.Execute("travel home-t1")

	assert.Contains(t, output, "jump")
}

func TestTravel_ToUniverseBranchID_SuggestsUniverse(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	// home-u1 doesn't exist yet; it's the next bubble universe.
	output := app.Execute("travel home-u1")

	assert.Contains(t, output, "universe")
}

func TestTravel_ToMathematicsBranchID_SuggestsStructure(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	// home-m1 doesn't exist yet; it's the next mathematical structure.
	output := app.Execute("travel home-m1")

	assert.Contains(t, output, "structure")
}

func TestTravel_ToSimulationBranchID_SuggestsSimulate(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := newTestApp(t)
	// home-s1 doesn't exist yet; it's the next simulation layer.
	output := app.Execute("travel home-s1")

	assert.Contains(t, output, "simulate")
}

func TestShift_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := newTestApp(t)
	app.Execute("shift")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (Q1)")
}

func TestList_ShowsTravelablePhysicalLocationIDs(t *testing.T) {
	app := newTestApp(t)
	app.Execute("travel station")
	app.Execute("shift")

	output := app.Execute("ls")

	assert.Contains(t, output, "City Centre (Q1) (rail, 3 — travel city-centre-q1)")
}

func TestJump_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := newTestApp(t)
	app.Execute("jump")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (T1)")
}

func TestUniverse_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := newTestApp(t)
	app.Execute("universe")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (U1)")
}

func TestStructure_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := newTestApp(t)
	app.Execute("structure")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (M1)")
	assert.Contains(t, output, "— structure)")
}

func TestSimulate_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := newTestApp(t)
	app.Execute("simulate")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (sim:1)")
	assert.Contains(t, output, "— simulate)")
	assert.Contains(t, output, "simulate back")
}

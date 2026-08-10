package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/petherin/onto/internal/domain/exploration"
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

func TestAppWhere(t *testing.T) {
	app := NewApp()
	output := app.Execute("where")

	assert.Contains(t, output, "Reality Coordinate")
	assert.Contains(t, output, "Earth")
}

func newMockedApp(t *testing.T) (*App, *mocks.MockRepository) {
	t.Helper()
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	repo := mocks.NewMockRepository(t)
	return &App{
		universe:          u,
		session:           exploration.NewEntity("home", loc.Coordinate),
		repo:              repo,
		pathfinder:        navigation.NewBFSPathfinder(),
		locationGenerator: universe.NewSequentialLocationGenerator(),
	}, repo
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

func TestAppNumberedJourneyTravelsToDisplayedDestination(t *testing.T) {
	app := NewApp()

	list := app.Execute("ls")
	assert.Contains(t, list, "1. Station")

	output := app.Execute("1")
	assert.Contains(t, output, "Arrived.")
	assert.Equal(t, "station", app.session.Location())
}

func TestAppNumberedJourneyExecutesContextualReturn(t *testing.T) {
	app := NewApp()
	app.Execute("shift")

	number := 0
	options, _ := app.journeyOptions(app.universe.EdgesFrom(app.session.Location()))
	for i, option := range options {
		if option.kind == journeyShiftBack {
			number = i + 1
			break
		}
	}
	require.NotZero(t, number)
	output := app.Execute(fmt.Sprintf("%d", number))

	assert.Contains(t, output, "Quantum branch exited")
	assert.Equal(t, "home", app.session.Location())
}

func TestAppNumberedJourneyRejectsUnknownNumber(t *testing.T) {
	app := NewApp()

	output := app.Execute("99")

	assert.Contains(t, output, "No possible journey numbered 99")
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

func TestAppObserveAndReturn(t *testing.T) {
	app := NewApp()

	observeOutput := app.Execute("observe Machine")
	assert.Contains(t, observeOutput, "Observer perspective entered: Machine")
	assert.Contains(t, app.Execute("where"), "Observer : Machine")

	returnOutput := app.Execute("observe back")
	assert.Contains(t, returnOutput, "Observer perspective restored: Human")
	assert.NotContains(t, app.Execute("ls"), "Return to the previous observer perspective")
}

func TestAppTimeAndReturn(t *testing.T) {
	app := NewApp()

	output := app.Execute("time 2042-01-02T03:04:05Z")
	assert.Contains(t, output, "Temporal branch entered")
	assert.Contains(t, app.Execute("where"), "2042-01-02T03:04:05Z")

	output = app.Execute("time back")
	assert.Contains(t, output, "Temporal branch exited")
	assert.Equal(t, "home", app.session.Location())
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
	assert.Contains(t, list, "— shift)")
	assert.Contains(t, list, "— jump)")
	assert.Contains(t, list, "— drift)")
	assert.Contains(t, list, "(align)")

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

func TestGoHome_UnwindsObserverPerspective(t *testing.T) {
	app := NewApp()
	app.Execute("observe Cat")

	plan := app.GoHome()
	assert.Contains(t, plan, "observe back (Cat → Human)  cost 2")

	result := app.GoHomeConfirm()
	assert.Contains(t, result, "Observer return → Human")
	assert.Equal(t, "Human", app.session.Coordinate().Observer)
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

func TestGoHome_UnwindsNestedContextsBeforeTime(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := NewApp()
	app.Execute("time 2027-01-01T00:00:00Z")
	app.Execute("shift")
	app.Execute("jump")
	app.Execute("drift")
	app.Execute("observe dog")

	plan := app.GoHome()

	assert.NotContains(t, plan, "return path unavailable")
	assert.Contains(t, plan, "time back")
}

func TestGoHome_UnwindsRecordedTransitionOrder(t *testing.T) {
	t.Setenv("ONTO_DATA_FILE", filepath.Join(t.TempDir(), "locations.json"))
	app := NewApp()
	app.Execute("shift")
	app.Execute("time 2027-01-01T00:00:00Z")
	app.Execute("jump")
	app.Execute("observe dog")

	plan := app.GoHome()

	observer := strings.Index(plan, "observe back")
	jump := strings.Index(plan, "jump back")
	time := strings.Index(plan, "time back")
	shift := strings.Index(plan, "shift back")
	assert.True(t, observer < jump && jump < time && time < shift, plan)
}

func TestAppHelp_ListsAllCommands(t *testing.T) {
	app := NewApp()
	output := app.Execute("help")

	for _, cmd := range []string{"where", "look", "ls", "route", "travel", "shift", "drift", "align", "observe", "save", "exit"} {
		assert.Contains(t, output, cmd, "help should list command %q", cmd)
	}
}

func TestAppShift_DoesNotSaveImmediately(t *testing.T) {
	app, repo := newMockedApp(t)

	output := app.Execute("shift")

	assert.Contains(t, output, "Quantum branch entered")
	repo.AssertNotCalled(t, "Save", testifymock.Anything)
	assert.True(t, app.dirty)
}

func TestAppSaveCommand_SavesAndReportsSuccess(t *testing.T) {
	app, repo := newMockedApp(t)
	app.Execute("shift")
	repo.EXPECT().Save(app.universe).Return(nil).Once()

	output := app.Execute("save")

	assert.Equal(t, msgSaved, output)
	assert.False(t, app.dirty)
}

func TestAppSaveIfDirty_OnlySavesWhenDirty(t *testing.T) {
	t.Run("dirty", func(t *testing.T) {
		app, repo := newMockedApp(t)
		app.dirty = true
		repo.EXPECT().Save(app.universe).Return(nil).Once()

		err := app.SaveIfDirty()

		require.NoError(t, err)
		assert.False(t, app.dirty)
	})

	t.Run("clean", func(t *testing.T) {
		app, repo := newMockedApp(t)

		err := app.SaveIfDirty()

		require.NoError(t, err)
		repo.AssertNotCalled(t, "Save", testifymock.Anything)
	})
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
	assert.Contains(t, output, "time <RFC3339>")
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

	assert.Contains(t, output, "City Centre (Q1) (rail, 3 — travel city-centre-q1)")
}

func TestJump_BranchHasContextualPhysicalDestinations(t *testing.T) {
	app := NewApp()
	app.Execute("jump")
	output := app.Execute("ls")

	assert.Contains(t, output, "walk")
	assert.Contains(t, output, "Station (T1)")
}

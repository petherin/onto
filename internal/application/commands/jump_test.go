package commands_test

import (
	"errors"
	"testing"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newJumpFixture(t *testing.T) (*universe.UniverseAggregate, *exploration.ExplorationEntity, *mocks.MockRepository) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewExplorationEntity("home", loc.Coordinate)
	repo := mocks.NewMockRepository(t)
	return u, sess, repo
}

func TestJumpCommand_CreatesNewTimelineLocation(t *testing.T) {
	u, sess, repo := newJumpFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "T1", result.NextTimeline)
	assert.Equal(t, "home-t1", result.Location.ID)

	_, exists := u.GetLocation("home-t1")
	assert.True(t, exists, "new timeline location should be added to universe")
}

func TestJumpCommand_AddsTimelineEdgesBothWays(t *testing.T) {
	u, sess, repo := newJumpFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo}
	_, err := cmd.Execute()
	require.NoError(t, err)

	// forward edge: home → home-t1
	found := false
	for _, e := range u.EdgesFrom("home") {
		if e.To == "home-t1" && e.Mode == universe.TimelineShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected forward timeline edge from home to home-t1")

	// reverse edge: home-t1 → home
	found = false
	for _, e := range u.EdgesFrom("home-t1") {
		if e.To == "home" && e.Mode == universe.TimelineShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected reverse timeline edge from home-t1 to home")
}

func TestJumpCommand_JumpsToExistingTimelineLocation(t *testing.T) {
	u, sess, repo := newJumpFixture(t)

	// pre-populate T1 location
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "home-t1", Mode: universe.TimelineShift, Cost: universe.TimelineShiftCost})
	u.AddEdge(universe.EdgeVO{From: "home-t1", To: "home", Mode: universe.TimelineShift, Cost: universe.TimelineShiftCost})

	initialEdgeCount := len(u.EdgesFrom("home"))
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-t1", result.Location.ID)
	assert.Equal(t, initialEdgeCount, len(u.EdgesFrom("home")), "no new edges expected when location already exists")
}

func TestJumpCommand_TimelineIncrements(t *testing.T) {
	u, _, repo := newJumpFixture(t)

	// simulate already being in T1
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord})
	sess := exploration.NewExplorationEntity("home-t1", t1Coord)

	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "T2", result.NextTimeline)
	assert.Equal(t, "home-t1-t2", result.Location.ID)
}

func TestJumpCommand_PersistsAfterJump(t *testing.T) {
	u, sess, repo := newJumpFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.True(t, result.Persisted)
	assert.NoError(t, result.SaveErr)
}

func TestJumpCommand_SaveError(t *testing.T) {
	u, sess, repo := newJumpFixture(t)
	repo.EXPECT().Save(u).Return(errors.New("write failed"))

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.False(t, result.Persisted)
	assert.EqualError(t, result.SaveErr, "write failed")
}

func TestJumpCommand_UpdatesSession(t *testing.T) {
	u, sess, repo := newJumpFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo}
	_, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-t1", sess.CurrentLocation)
	assert.Contains(t, sess.TravelHistory, "home -> home-t1 (timeline shift)")
}

// ── Jump back ────────────────────────────────────────────────────────────────

func TestJumpBack_ReturnsToLowerBranch(t *testing.T) {
	u, _, repo := newJumpFixture(t)

	// Place session in T1
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord})
	u.AddEdge(universe.EdgeVO{From: "home-t1", To: "home", Mode: universe.TimelineShift, Cost: universe.TimelineShiftCost})
	sess := exploration.NewExplorationEntity("home-t1", t1Coord)

	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home", result.Location.ID)
	assert.Equal(t, "Prime", result.NextTimeline)
	assert.True(t, result.Reversed)
	assert.Equal(t, "home", sess.CurrentLocation)
}

func TestJumpBack_AtBaseLevel_ReturnsError(t *testing.T) {
	u, sess, repo := newJumpFixture(t)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo, Back: true}
	_, err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Prime")
	repo.AssertNotCalled(t, "Save")
}

func TestJumpBack_NoReverseEdge_ReturnsError(t *testing.T) {
	u, _, repo := newJumpFixture(t)

	// T1 session but no reverse timeline edge in the graph
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord})
	sess := exploration.NewExplorationEntity("home-t1", t1Coord)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Repo: repo, Back: true}
	_, err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no timeline path back")
	repo.AssertNotCalled(t, "Save")
}

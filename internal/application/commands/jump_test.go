package commands_test

import (
	"testing"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newJumpFixture() (*universe.Aggregate, *exploration.Entity) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	return u, sess
}

func TestJumpCommand_CreatesNewTimelineLocation(t *testing.T) {
	u, sess := newJumpFixture()

	cmd := &commands.JumpCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "T1", result.NextTimeline)
	assert.Equal(t, "home-t1", result.Location.ID)

	_, exists := u.GetLocation("home-t1")
	assert.True(t, exists, "new timeline location should be added to universe")
}

func TestJumpCommand_AddsTimelineEdgesBothWays(t *testing.T) {
	u, sess := newJumpFixture()

	cmd := &commands.JumpCommand{Universe: u, Session: sess}
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
	u, sess := newJumpFixture()

	// pre-populate T1 location
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "home-t1", Mode: universe.TimelineShift, Cost: universe.TimelineShiftCost}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-t1", To: "home", Mode: universe.TimelineShift, Cost: universe.TimelineShiftCost}))

	initialEdgeCount := len(u.EdgesFrom("home"))
	cmd := &commands.JumpCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-t1", result.Location.ID)
	assert.Equal(t, initialEdgeCount, len(u.EdgesFrom("home")), "no new edges expected when location already exists")
}

func TestJumpCommand_TimelineIncrements(t *testing.T) {
	u, _ := newJumpFixture()

	// simulate already being in T1
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord}))
	sess := exploration.NewEntity("home-t1", t1Coord)

	cmd := &commands.JumpCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "T2", result.NextTimeline)
	assert.Equal(t, "home-t2", result.Location.ID)
}

func TestJumpCommand_UpdatesSession(t *testing.T) {
	u, sess := newJumpFixture()

	cmd := &commands.JumpCommand{Universe: u, Session: sess}
	_, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-t1", sess.Location())
	assert.Contains(t, sess.History(), "home -> home-t1 (timeline shift)")
}

// ── Jump back ────────────────────────────────────────────────────────────────

func TestJumpBack_ReturnsToLowerBranch(t *testing.T) {
	u, _ := newJumpFixture()

	// Place session in T1
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-t1", To: "home", Mode: universe.TimelineShift, Cost: universe.TimelineShiftCost}))
	sess := exploration.NewEntity("home-t1", t1Coord)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home", result.Location.ID)
	assert.Equal(t, "Prime", result.NextTimeline)
	assert.True(t, result.Reversed)
	assert.Equal(t, "home", sess.Location())
}

func TestJumpBack_AtBaseLevel_ReturnsError(t *testing.T) {
	u, sess := newJumpFixture()

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Back: true}
	_, err := cmd.Execute()

	require.ErrorIs(t, err, commands.ErrAlreadyAtBaseTimeline)
}

func TestJumpBack_NoReverseEdge_BackfillsPath(t *testing.T) {
	u, _ := newJumpFixture()

	// T1 session but no reverse timeline edge — EnsureLowerContext reconstructs it.
	t1Coord := universe.DefaultCoordinateVO()
	t1Coord.Timeline = "T1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-t1", Name: "Home (T1)", Coordinate: t1Coord}))
	sess := exploration.NewEntity("home-t1", t1Coord)

	cmd := &commands.JumpCommand{Universe: u, Session: sess, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, "home", result.Location.ID)
	require.Equal(t, "Prime", result.NextTimeline)
	require.True(t, result.Reversed)
}

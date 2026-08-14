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

func newUniverseFixture() (*universe.Aggregate, *exploration.Entity) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	return u, sess
}

func TestUniverseCommand_CreatesNewUniverseLocation(t *testing.T) {
	u, sess := newUniverseFixture()

	cmd := &commands.UniverseCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "U1", result.NextUniverse)
	assert.Equal(t, "home-u1", result.Location.ID)

	_, exists := u.GetLocation("home-u1")
	assert.True(t, exists, "new universe location should be added to universe")
}

func TestUniverseCommand_AddsUniverseEdgesBothWays(t *testing.T) {
	u, sess := newUniverseFixture()

	cmd := &commands.UniverseCommand{Universe: u, Session: sess}
	_, err := cmd.Execute()
	require.NoError(t, err)

	// forward edge: home → home-u1
	found := false
	for _, e := range u.EdgesFrom("home") {
		if e.To == "home-u1" && e.Mode == universe.UniverseShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected forward universe edge from home to home-u1")

	// reverse edge: home-u1 → home
	found = false
	for _, e := range u.EdgesFrom("home-u1") {
		if e.To == "home" && e.Mode == universe.UniverseShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected reverse universe edge from home-u1 to home")
}

func TestUniverseCommand_ShiftsToExistingUniverseLocation(t *testing.T) {
	u, sess := newUniverseFixture()

	// pre-populate U1 location
	u1Coord := universe.DefaultCoordinateVO()
	u1Coord.Universe = "U1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-u1", Name: "Home (U1)", Coordinate: u1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "home-u1", Mode: universe.UniverseShift, Cost: universe.UniverseShiftCost}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-u1", To: "home", Mode: universe.UniverseShift, Cost: universe.UniverseShiftCost}))

	initialEdgeCount := len(u.EdgesFrom("home"))
	cmd := &commands.UniverseCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-u1", result.Location.ID)
	assert.Equal(t, initialEdgeCount, len(u.EdgesFrom("home")), "no new edges expected when location already exists")
}

func TestUniverseCommand_UniverseIncrements(t *testing.T) {
	u, _ := newUniverseFixture()

	// simulate already being in U1
	u1Coord := universe.DefaultCoordinateVO()
	u1Coord.Universe = "U1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-u1", Name: "Home (U1)", Coordinate: u1Coord}))
	sess := exploration.NewEntity("home-u1", u1Coord)

	cmd := &commands.UniverseCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "U2", result.NextUniverse)
	assert.Equal(t, "home-u2", result.Location.ID)
}

func TestUniverseCommand_UpdatesSession(t *testing.T) {
	u, sess := newUniverseFixture()

	cmd := &commands.UniverseCommand{Universe: u, Session: sess}
	_, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-u1", sess.Location())
	assert.Contains(t, sess.History(), "home -> home-u1 (universe shift)")
}

// ── Universe back ───────────────────────────────────────────────────────────

func TestUniverseBack_ReturnsToLowerBranch(t *testing.T) {
	u, _ := newUniverseFixture()

	// Place session in U1
	u1Coord := universe.DefaultCoordinateVO()
	u1Coord.Universe = "U1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-u1", Name: "Home (U1)", Coordinate: u1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-u1", To: "home", Mode: universe.UniverseShift, Cost: universe.UniverseShiftCost}))
	sess := exploration.NewEntity("home-u1", u1Coord)

	cmd := &commands.UniverseCommand{Universe: u, Session: sess, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home", result.Location.ID)
	assert.Equal(t, "Origin", result.NextUniverse)
	assert.True(t, result.Reversed)
	assert.Equal(t, "home", sess.Location())
}

func TestUniverseBack_AtBaseLevel_ReturnsError(t *testing.T) {
	u, sess := newUniverseFixture()

	cmd := &commands.UniverseCommand{Universe: u, Session: sess, Back: true}
	_, err := cmd.Execute()

	require.ErrorIs(t, err, commands.ErrAlreadyAtBaseUniverse)
}

func TestUniverseBack_NoReverseEdge_BackfillsPath(t *testing.T) {
	u, _ := newUniverseFixture()

	// U1 session but no reverse universe edge — EnsureLowerContext reconstructs it.
	u1Coord := universe.DefaultCoordinateVO()
	u1Coord.Universe = "U1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-u1", Name: "Home (U1)", Coordinate: u1Coord}))
	sess := exploration.NewEntity("home-u1", u1Coord)

	cmd := &commands.UniverseCommand{Universe: u, Session: sess, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, "home", result.Location.ID)
	require.Equal(t, "Origin", result.NextUniverse)
	require.True(t, result.Reversed)
}

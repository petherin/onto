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

func newShiftFixture() (*universe.Aggregate, *exploration.Entity) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	return u, sess
}

func TestShiftCommand_CreatesNewQuantumLocation(t *testing.T) {
	u, sess := newShiftFixture()

	cmd := &commands.ShiftCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Q1", result.NextQuantum)
	assert.Equal(t, "home-q1", result.Location.ID)

	_, exists := u.GetLocation("home-q1")
	assert.True(t, exists, "new quantum location should be added to universe")
}

func TestShiftCommand_AddsQuantumEdgesBothWays(t *testing.T) {
	u, sess := newShiftFixture()

	cmd := &commands.ShiftCommand{Universe: u, Session: sess}
	_, err := cmd.Execute()
	require.NoError(t, err)

	// forward edge: home → home-q1
	found := false
	for _, e := range u.EdgesFrom("home") {
		if e.To == "home-q1" && e.Mode == universe.QuantumShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected forward quantum edge from home to home-q1")

	// reverse edge: home-q1 → home
	found = false
	for _, e := range u.EdgesFrom("home-q1") {
		if e.To == "home" && e.Mode == universe.QuantumShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected reverse quantum edge from home-q1 to home")
}

func TestShiftCommand_ShiftsToExistingQuantumLocation(t *testing.T) {
	u, sess := newShiftFixture()

	// pre-populate Q1 location
	q1Coord := universe.DefaultCoordinateVO()
	q1Coord.Quantum = "Q1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "home-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-q1", To: "home", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost}))

	initialEdgeCount := len(u.EdgesFrom("home"))
	cmd := &commands.ShiftCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-q1", result.Location.ID)
	assert.Equal(t, initialEdgeCount, len(u.EdgesFrom("home")), "no new edges expected when location already exists")
}

func TestShiftCommand_QuantumIncrements(t *testing.T) {
	u, _ := newShiftFixture()

	// simulate already being in Q1
	q1Coord := universe.DefaultCoordinateVO()
	q1Coord.Quantum = "Q1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord}))
	sess := exploration.NewEntity("home-q1", q1Coord)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Q2", result.NextQuantum)
	assert.Equal(t, "home-q2", result.Location.ID)
}

func TestShiftCommand_UpdatesSession(t *testing.T) {
	u, sess := newShiftFixture()

	cmd := &commands.ShiftCommand{Universe: u, Session: sess}
	_, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-q1", sess.Location())
	assert.Contains(t, sess.History(), "home -> home-q1 (quantum shift)")
}

// ── Shift back ────────────────────────────────────────────────────────────────

func TestShiftBack_ReturnsToLowerBranch(t *testing.T) {
	u, _ := newShiftFixture()

	// Place session in Q1
	q1Coord := universe.DefaultCoordinateVO()
	q1Coord.Quantum = "Q1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-q1", To: "home", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost}))
	sess := exploration.NewEntity("home-q1", q1Coord)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home", result.Location.ID)
	assert.Equal(t, "Q0", result.NextQuantum)
	assert.True(t, result.Reversed)
	assert.Equal(t, "home", sess.Location())
}

func TestShiftBack_AtBaseLevel_ReturnsError(t *testing.T) {
	u, sess := newShiftFixture()

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Back: true}
	_, err := cmd.Execute()

	require.ErrorIs(t, err, commands.ErrAlreadyAtBaseQuantum)
}

func TestShiftBack_NoReverseEdge_BackfillsPath(t *testing.T) {
	u, _ := newShiftFixture()

	// Q1 session but no reverse quantum edge — EnsureLowerContext reconstructs it.
	q1Coord := universe.DefaultCoordinateVO()
	q1Coord.Quantum = "Q1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord}))
	sess := exploration.NewEntity("home-q1", q1Coord)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	require.Equal(t, "home", result.Location.ID)
	require.Equal(t, "Q0", result.NextQuantum)
	require.True(t, result.Reversed)
}

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

func newShiftFixture(t *testing.T) (*universe.Universe, *exploration.Session, *mocks.MockRepository) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewSession("home", loc.Coordinate)
	repo := mocks.NewMockRepository(t)
	return u, sess, repo
}

func TestShiftCommand_CreatesNewQuantumLocation(t *testing.T) {
	u, sess, repo := newShiftFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Q1", result.NextQuantum)
	assert.Equal(t, "home-q1", result.Location.ID)

	_, exists := u.GetLocation("home-q1")
	assert.True(t, exists, "new quantum location should be added to universe")
}

func TestShiftCommand_AddsQuantumEdgesBothWays(t *testing.T) {
	u, sess, repo := newShiftFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo}
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
	u, sess, repo := newShiftFixture(t)

	// pre-populate Q1 location
	q1Coord := universe.NewCoordinate()
	q1Coord.Quantum = "Q1"
	u.AddLocation(universe.Location{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord})
	u.AddEdge(universe.Edge{From: "home", To: "home-q1", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost})
	u.AddEdge(universe.Edge{From: "home-q1", To: "home", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost})

	initialEdgeCount := len(u.EdgesFrom("home"))
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-q1", result.Location.ID)
	assert.Equal(t, initialEdgeCount, len(u.EdgesFrom("home")), "no new edges expected when location already exists")
}

func TestShiftCommand_QuantumIncrements(t *testing.T) {
	u, _, repo := newShiftFixture(t)

	// simulate already being in Q1
	q1Coord := universe.NewCoordinate()
	q1Coord.Quantum = "Q1"
	u.AddLocation(universe.Location{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord})
	sess := exploration.NewSession("home-q1", q1Coord)

	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "Q2", result.NextQuantum)
	assert.Equal(t, "home-q1-q2", result.Location.ID)
}

func TestShiftCommand_PersistsAfterShift(t *testing.T) {
	u, sess, repo := newShiftFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.True(t, result.Persisted)
	assert.NoError(t, result.SaveErr)
}

func TestShiftCommand_SaveError(t *testing.T) {
	u, sess, repo := newShiftFixture(t)
	repo.EXPECT().Save(u).Return(errors.New("write failed"))

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.False(t, result.Persisted)
	assert.EqualError(t, result.SaveErr, "write failed")
}

func TestShiftCommand_UpdatesSession(t *testing.T) {
	u, sess, repo := newShiftFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo}
	_, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-q1", sess.CurrentLocation)
	assert.Contains(t, sess.TravelHistory, "home -> home-q1 (quantum shift)")
}

// ── Shift back ────────────────────────────────────────────────────────────────

func TestShiftBack_ReturnsToLowerBranch(t *testing.T) {
	u, _, repo := newShiftFixture(t)

	// Place session in Q1
	q1Coord := universe.NewCoordinate()
	q1Coord.Quantum = "Q1"
	u.AddLocation(universe.Location{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord})
	u.AddEdge(universe.Edge{From: "home-q1", To: "home", Mode: universe.QuantumShift, Cost: universe.QuantumShiftCost})
	sess := exploration.NewSession("home-q1", q1Coord)

	repo.EXPECT().Save(u).Return(nil)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home", result.Location.ID)
	assert.Equal(t, "Q0", result.NextQuantum)
	assert.True(t, result.Reversed)
	assert.Equal(t, "home", sess.CurrentLocation)
}

func TestShiftBack_AtBaseLevel_ReturnsError(t *testing.T) {
	u, sess, repo := newShiftFixture(t)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo, Back: true}
	_, err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Q0")
	repo.AssertNotCalled(t, "Save")
}

func TestShiftBack_NoReverseEdge_ReturnsError(t *testing.T) {
	u, _, repo := newShiftFixture(t)

	// Q1 session but no reverse quantum edge in the graph
	q1Coord := universe.NewCoordinate()
	q1Coord.Quantum = "Q1"
	u.AddLocation(universe.Location{ID: "home-q1", Name: "Home (Q1)", Coordinate: q1Coord})
	sess := exploration.NewSession("home-q1", q1Coord)

	cmd := &commands.ShiftCommand{Universe: u, Session: sess, Repo: repo, Back: true}
	_, err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no quantum path back")
	repo.AssertNotCalled(t, "Save")
}

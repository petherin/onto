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

func newStructureFixture() (*universe.Aggregate, *exploration.Entity) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	return u, sess
}

func TestStructureCommand_CreatesNewMathematicsLocation(t *testing.T) {
	u, sess := newStructureFixture()

	cmd := &commands.StructureCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "M1", result.NextMathematics)
	assert.Equal(t, "home-m1", result.Location.ID)

	_, exists := u.GetLocation("home-m1")
	assert.True(t, exists, "new mathematics location should be added to universe")
}

func TestStructureCommand_AddsMathematicsEdgesBothWays(t *testing.T) {
	u, sess := newStructureFixture()

	cmd := &commands.StructureCommand{Universe: u, Session: sess}
	_, err := cmd.Execute()
	require.NoError(t, err)

	found := false
	for _, e := range u.EdgesFrom("home") {
		if e.To == "home-m1" && e.Mode == universe.MathematicalShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected forward mathematics edge from home to home-m1")

	found = false
	for _, e := range u.EdgesFrom("home-m1") {
		if e.To == "home" && e.Mode == universe.MathematicalShift {
			found = true
			break
		}
	}
	assert.True(t, found, "expected reverse mathematics edge from home-m1 to home")
}

func TestStructureCommand_ShiftsToExistingMathematicsLocation(t *testing.T) {
	u, sess := newStructureFixture()

	m1Coord := universe.DefaultCoordinateVO()
	m1Coord.Mathematics = "M1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-m1", Name: "Home (M1)", Coordinate: m1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "home-m1", Mode: universe.MathematicalShift, Cost: universe.MathematicalShiftCost}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-m1", To: "home", Mode: universe.MathematicalShift, Cost: universe.MathematicalShiftCost}))

	initialEdgeCount := len(u.EdgesFrom("home"))
	cmd := &commands.StructureCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-m1", result.Location.ID)
	assert.Equal(t, initialEdgeCount, len(u.EdgesFrom("home")), "no new edges expected when location already exists")
}

func TestStructureCommand_MathematicsIncrements(t *testing.T) {
	u, _ := newStructureFixture()

	m1Coord := universe.DefaultCoordinateVO()
	m1Coord.Mathematics = "M1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-m1", Name: "Home (M1)", Coordinate: m1Coord}))
	sess := exploration.NewEntity("home-m1", m1Coord)

	cmd := &commands.StructureCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "M2", result.NextMathematics)
	assert.Equal(t, "home-m2", result.Location.ID)
}

func TestStructureCommand_UpdatesSession(t *testing.T) {
	u, sess := newStructureFixture()

	cmd := &commands.StructureCommand{Universe: u, Session: sess}
	_, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home-m1", sess.Location())
	assert.Contains(t, sess.History(), "home -> home-m1 (mathematical structure shift)")
	assert.Equal(t, universe.MathematicalShiftCost, sess.CumulativeCost())
}

// ── Structure back ──────────────────────────────────────────────────────────

func TestStructureBack_ReturnsToLowerBranch(t *testing.T) {
	u, _ := newStructureFixture()

	m1Coord := universe.DefaultCoordinateVO()
	m1Coord.Mathematics = "M1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-m1", Name: "Home (M1)", Coordinate: m1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-m1", To: "home", Mode: universe.MathematicalShift, Cost: universe.MathematicalShiftCost}))
	sess := exploration.NewEntity("home-m1", m1Coord)

	cmd := &commands.StructureCommand{Universe: u, Session: sess, Back: true}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, "home", result.Location.ID)
	assert.Equal(t, "Classical", result.NextMathematics)
	assert.True(t, result.Reversed)
	assert.Equal(t, "home", sess.Location())
}

func TestStructureBack_AtBaseLevel_ReturnsError(t *testing.T) {
	u, sess := newStructureFixture()

	cmd := &commands.StructureCommand{Universe: u, Session: sess, Back: true}
	_, err := cmd.Execute()

	require.ErrorIs(t, err, commands.ErrAlreadyAtBaseMathematics)
}

func TestStructureBack_NoReverseEdge_ReturnsError(t *testing.T) {
	u, _ := newStructureFixture()

	m1Coord := universe.DefaultCoordinateVO()
	m1Coord.Mathematics = "M1"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-m1", Name: "Home (M1)", Coordinate: m1Coord}))
	sess := exploration.NewEntity("home-m1", m1Coord)

	cmd := &commands.StructureCommand{Universe: u, Session: sess, Back: true}
	_, err := cmd.Execute()

	require.ErrorIs(t, err, commands.ErrNoMathematicsPathBack)
}

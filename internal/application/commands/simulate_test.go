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

func newSimulateFixture() (*universe.Aggregate, *exploration.Entity) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	return u, sess
}

func TestSimulateCommand_CreatesNewSimulationLocation(t *testing.T) {
	u, sess := newSimulateFixture()

	cmd := &commands.SimulateCommand{Universe: u, Session: sess}
	result, err := cmd.Execute()

	require.NoError(t, err)
	assert.Equal(t, 1, result.Simulation)
	assert.Equal(t, "home-s1", result.Location.ID)

	_, exists := u.GetLocation("home-s1")
	assert.True(t, exists)
}

func TestSimulateCommand_AddsSimulationEdgesBothWays(t *testing.T) {
	u, sess := newSimulateFixture()

	_, err := (&commands.SimulateCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)

	var forward, reverse universe.EdgeVO
	for _, e := range u.EdgesFrom("home") {
		if e.To == "home-s1" && e.Mode == universe.SimulationEntry {
			forward = e
		}
	}
	for _, e := range u.EdgesFrom("home-s1") {
		if e.To == "home" && e.Mode == universe.SimulationEntry {
			reverse = e
		}
	}
	assert.Equal(t, universe.SimulationEntryCost, forward.Cost)
	assert.Equal(t, universe.SimulationExitCost, reverse.Cost)
}

func TestSimulateCommand_IncrementsDepth(t *testing.T) {
	u, _ := newSimulateFixture()

	s1Coord := universe.DefaultCoordinateVO()
	s1Coord.Simulation = 1
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-s1", Name: "Home (sim:1)", Coordinate: s1Coord}))
	sess := exploration.NewEntity("home-s1", s1Coord)

	result, err := (&commands.SimulateCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	assert.Equal(t, 2, result.Simulation)
	assert.Equal(t, "home-s2", result.Location.ID)
}

func TestSimulateCommand_UpdatesSession(t *testing.T) {
	u, sess := newSimulateFixture()

	_, err := (&commands.SimulateCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)

	assert.Equal(t, "home-s1", sess.Location())
	assert.Equal(t, 1, sess.SimulationLevel())
	assert.Contains(t, sess.History(), "home -> home-s1 (simulation entry)")
	assert.Equal(t, universe.SimulationEntryCost, sess.CumulativeCost())
}

func TestSimulateBack_ReturnsToLowerDepth(t *testing.T) {
	u, _ := newSimulateFixture()

	s1Coord := universe.DefaultCoordinateVO()
	s1Coord.Simulation = 1
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-s1", Name: "Home (sim:1)", Coordinate: s1Coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{
		From: "home-s1", To: "home", Mode: universe.SimulationEntry, Cost: universe.SimulationExitCost,
		Description: "Simulation exit to depth 0",
	}))
	sess := exploration.NewEntity("home-s1", s1Coord)

	result, err := (&commands.SimulateCommand{Universe: u, Session: sess, Back: true}).Execute()
	require.NoError(t, err)
	assert.Equal(t, "home", result.Location.ID)
	assert.Equal(t, 0, result.Simulation)
	assert.True(t, result.Reversed)
	assert.Equal(t, universe.SimulationExitCost, sess.CumulativeCost())
	assert.Contains(t, sess.History()[0], "simulation exit")
}

func TestSimulateBack_AtBaseLevel_ReturnsError(t *testing.T) {
	u, sess := newSimulateFixture()

	_, err := (&commands.SimulateCommand{Universe: u, Session: sess, Back: true}).Execute()
	require.ErrorIs(t, err, commands.ErrAlreadyAtBaseReality)
}

func TestSimulateBack_NoReverseEdge_ReturnsError(t *testing.T) {
	u, _ := newSimulateFixture()

	s1Coord := universe.DefaultCoordinateVO()
	s1Coord.Simulation = 1
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-s1", Name: "Home (sim:1)", Coordinate: s1Coord}))
	sess := exploration.NewEntity("home-s1", s1Coord)

	_, err := (&commands.SimulateCommand{Universe: u, Session: sess, Back: true}).Execute()
	require.ErrorIs(t, err, commands.ErrNoSimulationPathBack)
}

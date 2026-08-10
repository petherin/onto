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

func newDriftFixture() (*universe.Aggregate, *exploration.Entity) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	return u, exploration.NewEntity("home", loc.Coordinate)
}

func TestDriftCommand_CreatesAndEntersConsensusDivergence(t *testing.T) {
	u, sess := newDriftFixture()

	result, err := (&commands.DriftCommand{Universe: u, Session: sess}).Execute()

	require.NoError(t, err)
	assert.Equal(t, 1, result.Consensus)
	assert.Equal(t, "home-c1", result.Location.ID)
	assert.Equal(t, 1, sess.ConsensusLevel())
	assert.Contains(t, sess.History(), "home -> home-c1 (consensus drift)")
}

func TestDriftCommand_AlignsToLowerConsensusLevel(t *testing.T) {
	u, _ := newDriftFixture()
	coord := universe.DefaultCoordinateVO()
	coord.Consensus = 1
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-c1", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-c1", To: "home", Mode: universe.ConsensusShift, Cost: universe.ConsensusShiftCost}))
	sess := exploration.NewEntity("home-c1", coord)
	result, err := (&commands.DriftCommand{Universe: u, Session: sess, Back: true}).Execute()

	require.NoError(t, err)
	assert.True(t, result.Reversed)
	assert.Equal(t, 0, result.Consensus)
	assert.Equal(t, "home", sess.Location())
}

func TestDriftCommand_AlignAtConsensusReturnsError(t *testing.T) {
	u, sess := newDriftFixture()

	_, err := (&commands.DriftCommand{Universe: u, Session: sess, Back: true}).Execute()

	require.ErrorIs(t, err, commands.ErrAlreadyAtConsensus)
}

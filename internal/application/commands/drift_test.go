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

func newDriftFixture(t *testing.T) (*universe.Aggregate, *exploration.Entity, *mocks.MockRepository) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	return u, exploration.NewEntity("home", loc.Coordinate), mocks.NewMockRepository(t)
}

func TestDriftCommand_CreatesAndEntersConsensusDivergence(t *testing.T) {
	u, sess, repo := newDriftFixture(t)
	repo.EXPECT().Save(u).Return(nil)

	result, err := (&commands.DriftCommand{Universe: u, Session: sess, Repo: repo}).Execute()

	require.NoError(t, err)
	assert.Equal(t, 1, result.Consensus)
	assert.Equal(t, "home-c1", result.Location.ID)
	assert.Equal(t, 1, sess.ConsensusLevel())
	assert.Contains(t, sess.History(), "home -> home-c1 (consensus drift)")
}

func TestDriftCommand_AlignsToLowerConsensusLevel(t *testing.T) {
	u, _, repo := newDriftFixture(t)
	coord := universe.DefaultCoordinateVO()
	coord.Consensus = 1
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-c1", Coordinate: coord}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home-c1", To: "home", Mode: universe.ConsensusShift, Cost: universe.ConsensusShiftCost}))
	sess := exploration.NewEntity("home-c1", coord)
	repo.EXPECT().Save(u).Return(nil)

	result, err := (&commands.DriftCommand{Universe: u, Session: sess, Repo: repo, Back: true}).Execute()

	require.NoError(t, err)
	assert.True(t, result.Reversed)
	assert.Equal(t, 0, result.Consensus)
	assert.Equal(t, "home", sess.Location())
}

func TestDriftCommand_AlignAtConsensusReturnsError(t *testing.T) {
	u, sess, repo := newDriftFixture(t)

	_, err := (&commands.DriftCommand{Universe: u, Session: sess, Repo: repo, Back: true}).Execute()

	require.ErrorIs(t, err, commands.ErrAlreadyAtConsensus)
	repo.AssertNotCalled(t, "Save")
}

func TestDriftCommand_SaveErrorReturnsMovementResult(t *testing.T) {
	u, sess, repo := newDriftFixture(t)
	repo.EXPECT().Save(u).Return(errors.New("write failed"))

	result, err := (&commands.DriftCommand{Universe: u, Session: sess, Repo: repo}).Execute()

	require.NotNil(t, result)
	assert.EqualError(t, err, "write failed")
}

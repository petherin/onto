package commands_test

import (
	"testing"
	"time"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeCommand_EntersAndLeavesTemporalBranch(t *testing.T) {
	u := mocks.NewTestUniverse()
	home, _ := u.GetLocation("home")
	session := exploration.NewEntity("home", home.Coordinate)
	target := time.Date(2042, 1, 2, 3, 4, 5, 0, time.UTC)

	result, err := (&commands.TimeCommand{Universe: u, Session: session, Target: target.Format(time.RFC3339)}).Execute()

	require.NoError(t, err)
	assert.Equal(t, target, result.Time)
	assert.Equal(t, "home-at-20420102t030405z", session.Location())
	assert.Equal(t, universe.TimeShiftCost, session.CumulativeCost())

	result, err = (&commands.TimeCommand{Universe: u, Session: session, Back: true}).Execute()

	require.NoError(t, err)
	assert.True(t, result.Reversed)
	assert.Equal(t, "home", session.Location())
	assert.Equal(t, 2*universe.TimeShiftCost, session.CumulativeCost())
}

func TestTimeCommand_RejectsInvalidTimestamp(t *testing.T) {
	u := mocks.NewTestUniverse()
	home, _ := u.GetLocation("home")
	session := exploration.NewEntity("home", home.Coordinate)

	_, err := (&commands.TimeCommand{Universe: u, Session: session, Target: "tomorrow"}).Execute()

	require.ErrorIs(t, err, commands.ErrInvalidTimeTarget)
}

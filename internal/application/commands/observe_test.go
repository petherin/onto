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

func TestObserveCommand_ChangesObserverPerspective(t *testing.T) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	repo := mocks.NewMockRepository(t)
	repo.EXPECT().Save(u).Return(nil)

	result, err := (&commands.ObserveCommand{Universe: u, Session: sess, Repo: repo, Observer: "Machine"}).Execute()

	require.NoError(t, err)
	assert.Equal(t, "Machine", result.Observer)
	assert.Equal(t, "home-o-machine", result.Location.ID)
	assert.Equal(t, "Machine", sess.Coordinate().Observer)
}

func TestObserveCommand_ReturnsToPreviousPerspective(t *testing.T) {
	u := mocks.NewTestUniverse()
	base, _ := u.GetLocation("home")
	machine := base.Coordinate
	machine.Observer = "Machine"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-o-machine", Coordinate: machine}))
	bat := machine
	bat.Observer = "Bat"
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-o-bat", Coordinate: bat}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{
		From:        "home-o-machine",
		To:          "home",
		Mode:        universe.ObserverShift,
		Cost:        universe.ObserverShiftCost,
		Description: "Observer shift back to Human",
	}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{
		From:        "home-o-machine",
		To:          "home-o-bat",
		Mode:        universe.ObserverShift,
		Cost:        universe.ObserverShiftCost,
		Description: "Observer shift to Bat",
	}))
	sess := exploration.NewEntity("home-o-machine", machine)
	repo := mocks.NewMockRepository(t)
	repo.EXPECT().Save(u).Return(nil)

	result, err := (&commands.ObserveCommand{Universe: u, Session: sess, Repo: repo, Back: true}).Execute()

	require.NoError(t, err)
	assert.True(t, result.Reversed)
	assert.Equal(t, "Human", sess.Coordinate().Observer)
}

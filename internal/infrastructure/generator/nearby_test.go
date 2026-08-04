package generator_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/infrastructure/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUniverse(id string) *universe.Aggregate {
	u := universe.NewAggregate()
	u.AddLocation(universe.LocationEntity{ID: id, Name: id, Coordinate: universe.DefaultCoordinateVO()})
	return u
}

func TestNearbyGenerator_Handle_CreatesFirstNearbyLocation(t *testing.T) {
	u := newUniverse("home")
	g := generator.New()
	coord := universe.DefaultCoordinateVO()

	created := g.Handle(u, "home", coord)

	require.True(t, created)
	_, ok := u.GetLocation("home-1")
	assert.True(t, ok, "home-1 should have been created")
}

func TestNearbyGenerator_Handle_AddsBidirectionalWalkEdges(t *testing.T) {
	u := newUniverse("home")
	g := generator.New()

	g.Handle(u, "home", universe.DefaultCoordinateVO())

	outgoing := u.EdgesFrom("home")
	var toNearby bool
	for _, e := range outgoing {
		if e.To == "home-1" && e.Mode == universe.Walk {
			toNearby = true
		}
	}
	assert.True(t, toNearby, "outgoing walk edge to home-1 expected")

	returning := u.EdgesFrom("home-1")
	var toHome bool
	for _, e := range returning {
		if e.To == "home" && e.Mode == universe.Walk {
			toHome = true
		}
	}
	assert.True(t, toHome, "return walk edge from home-1 to home expected")
}

func TestNearbyGenerator_Handle_SkipsExistingIDAndUsesNext(t *testing.T) {
	u := newUniverse("home")
	// Pre-occupy home-1 so the generator has to use home-2.
	u.AddLocation(universe.LocationEntity{ID: "home-1"})
	g := generator.New()

	created := g.Handle(u, "home", universe.DefaultCoordinateVO())

	require.True(t, created)
	_, ok := u.GetLocation("home-2")
	assert.True(t, ok, "home-2 should be created when home-1 is taken")
}

func TestNearbyGenerator_Handle_CoordinateSetsLocationField(t *testing.T) {
	u := newUniverse("home")
	g := generator.New()

	g.Handle(u, "home", universe.DefaultCoordinateVO())

	loc, _ := u.GetLocation("home-1")
	assert.Equal(t, "Nearby 1", loc.Coordinate.Location)
}

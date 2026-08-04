package queries_test

import (
	"testing"

	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouteFixture(t *testing.T) (*universe.Aggregate, *exploration.Entity, *mocks.MockPathfinderService) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	pf := mocks.NewMockPathfinderService(t)
	return u, sess, pf
}

func TestRouteQuery_Success_ReturnsSteps(t *testing.T) {
	u, sess, pf := newRouteFixture(t)

	route := []universe.EdgeVO{{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "station").Return(route, true)

	q := &queries.RouteQuery{Universe: u, Session: sess, Pathfinder: pf}
	result, err := q.Execute("station")

	require.NoError(t, err)
	assert.Equal(t, route, result.Steps)
}

func TestRouteQuery_Success_CalculatesDistanceAndCost(t *testing.T) {
	u, sess, pf := newRouteFixture(t)

	route := []universe.EdgeVO{
		{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1},
		{From: "station", To: "city", Mode: universe.Rail, Distance: 3.0, Cost: 3},
	}
	u.AddLocation(universe.LocationEntity{ID: "city", Name: "City", Coordinate: universe.DefaultCoordinateVO()})
	pf.EXPECT().FindRoute(u, "home", "city").Return(route, true)

	q := &queries.RouteQuery{Universe: u, Session: sess, Pathfinder: pf}
	result, err := q.Execute("city")

	require.NoError(t, err)
	assert.InDelta(t, 4.6, result.Distance, 0.001)
	assert.InDelta(t, 4.0, result.Cost, 0.001)
}

func TestRouteQuery_UnknownDestination(t *testing.T) {
	u, sess, pf := newRouteFixture(t)

	// No FindRoute expectation set — any call would immediately fail the test
	q := &queries.RouteQuery{Universe: u, Session: sess, Pathfinder: pf}
	_, err := q.Execute("atlantis")

	require.ErrorIs(t, err, navigation.ErrUnknownDestination)
}

func TestRouteQuery_NoPath(t *testing.T) {
	u, sess, pf := newRouteFixture(t)

	// island exists in universe but pathfinder finds no route
	u.AddLocation(universe.LocationEntity{ID: "island", Name: "Island", Coordinate: universe.DefaultCoordinateVO()})
	pf.EXPECT().FindRoute(u, "home", "island").Return(nil, false)

	q := &queries.RouteQuery{Universe: u, Session: sess, Pathfinder: pf}
	_, err := q.Execute("island")

	require.ErrorIs(t, err, navigation.ErrNoRoute)
}

func TestRouteQuery_CaseNormalised(t *testing.T) {
	u, sess, pf := newRouteFixture(t)

	route := []universe.EdgeVO{{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1}}
	pf.EXPECT().FindRoute(u, "home", "station").Return(route, true)

	q := &queries.RouteQuery{Universe: u, Session: sess, Pathfinder: pf}
	result, err := q.Execute("Station") // mixed case

	require.NoError(t, err)
	assert.NotEmpty(t, result.Steps)
}

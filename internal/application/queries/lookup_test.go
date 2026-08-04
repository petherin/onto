package queries_test

import (
	"testing"

	"github.com/petherin/onto/internal/application/queries"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLookupFixture() (*universe.Aggregate, *exploration.Entity) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	return u, sess
}

func TestLookupQuery_Where_ReturnsCoordinate(t *testing.T) {
	u, sess := newLookupFixture()
	q := &queries.LookupQuery{Universe: u, Session: sess}

	result := q.Where()

	assert.Equal(t, sess.CurrentCoordinate, result.Coordinate)
}

func TestLookupQuery_Where_ReturnsEdges(t *testing.T) {
	u, sess := newLookupFixture()
	q := &queries.LookupQuery{Universe: u, Session: sess}

	result := q.Where()

	assert.Equal(t, u.EdgesFrom("home"), result.Edges)
}

func TestLookupQuery_Where_ReturnsNextQuantumID(t *testing.T) {
	u, sess := newLookupFixture()
	q := &queries.LookupQuery{Universe: u, Session: sess}

	result := q.Where()

	assert.Equal(t, "home-q1", result.NextQuantum)
}

func TestLookupQuery_Where_ReturnsHistory(t *testing.T) {
	u, sess := newLookupFixture()

	// simulate one prior move
	stationLoc, _ := u.GetLocation("station")
	sess.MoveTo(stationLoc, 1)
	homeLoc, _ := u.GetLocation("home")
	sess.MoveTo(homeLoc, 1)

	q := &queries.LookupQuery{Universe: u, Session: sess}
	result := q.Where()

	require.Len(t, result.History, 2)
	assert.Equal(t, "home -> station", result.History[0])
	assert.Equal(t, "station -> home", result.History[1])
}

func TestLookupQuery_Look_ReturnsNameAndDescription(t *testing.T) {
	u, sess := newLookupFixture()
	q := &queries.LookupQuery{Universe: u, Session: sess}

	result, ok := q.Look()

	require.True(t, ok)
	assert.Equal(t, "Home", result.Name)
	assert.Equal(t, "Start", result.Description)
}

func TestLookupQuery_Look_NotFound(t *testing.T) {
	u := mocks.NewTestUniverse()
	// create a session pointing at a location not in the universe
	sess := exploration.NewEntity("ghost", universe.DefaultCoordinateVO())

	q := &queries.LookupQuery{Universe: u, Session: sess}
	result, ok := q.Look()

	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestLookupQuery_List_ReturnsEdgesAndNextQuantum(t *testing.T) {
	u, sess := newLookupFixture()
	q := &queries.LookupQuery{Universe: u, Session: sess}

	result := q.List()

	assert.Equal(t, u.EdgesFrom("home"), result.Edges)
	assert.Equal(t, "home-q1", result.NextQuantum)
}

func TestLookupQuery_List_AfterTravel(t *testing.T) {
	u, sess := newLookupFixture()

	stationLoc, _ := u.GetLocation("station")
	sess.MoveTo(stationLoc, 1)

	q := &queries.LookupQuery{Universe: u, Session: sess}
	result := q.List()

	assert.Equal(t, u.EdgesFrom("station"), result.Edges)
	assert.Equal(t, "station-q1", result.NextQuantum)
}

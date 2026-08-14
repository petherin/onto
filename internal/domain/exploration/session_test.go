package exploration_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
)

func defaultCoord() universe.CoordinateVO { return universe.DefaultCoordinateVO() }

func TestNewEntity_InitialisesCorrectly(t *testing.T) {
	coord := defaultCoord()
	e := exploration.NewEntity("home", coord)

	assert.Equal(t, "home", e.Location())
	assert.Equal(t, coord, e.Coordinate())
	assert.Empty(t, e.History())
	assert.Equal(t, 0.0, e.CumulativeCost())
}

func TestMoveTo_UpdatesLocationCoordinateAndHistory(t *testing.T) {
	e := exploration.NewEntity("home", defaultCoord())
	stationCoord := defaultCoord()
	stationCoord.Location = "Station"
	loc := universe.LocationEntity{ID: "station", Coordinate: stationCoord}

	e.MoveTo(loc, 5.0)

	assert.Equal(t, "station", e.Location())
	assert.Equal(t, stationCoord, e.Coordinate())
	assert.Equal(t, 5.0, e.CumulativeCost())
	assert.Contains(t, e.History(), "home -> station")
}

func TestMoveTo_AccumulatesCostAcrossMultipleMoves(t *testing.T) {
	e := exploration.NewEntity("home", defaultCoord())
	loc1 := universe.LocationEntity{ID: "a", Coordinate: defaultCoord()}
	loc2 := universe.LocationEntity{ID: "b", Coordinate: defaultCoord()}

	e.MoveTo(loc1, 3.0)
	e.MoveTo(loc2, 7.0)

	assert.Equal(t, 10.0, e.CumulativeCost())
}

func TestTransitionTo_RecordsQuantumShiftInHistory(t *testing.T) {
	e := exploration.NewEntity("home", defaultCoord())
	q1Coord := defaultCoord()
	q1Coord.Quantum = "Q1"
	loc := universe.LocationEntity{ID: "home-q1", Coordinate: q1Coord}

	e.TransitionTo(loc, 20.0, universe.QuantumShift, false)

	assert.Equal(t, "home-q1", e.Location())
	assert.Equal(t, "Q1", e.Coordinate().Quantum)
	assert.Equal(t, 20.0, e.CumulativeCost())
	assert.Contains(t, e.History(), "home -> home-q1 (quantum shift)")
}

func TestTransitionTo_RecordsTimelineShiftInHistory(t *testing.T) {
	e := exploration.NewEntity("home", defaultCoord())
	t1Coord := defaultCoord()
	t1Coord.Timeline = "T1"
	loc := universe.LocationEntity{ID: "home-t1", Coordinate: t1Coord}

	e.TransitionTo(loc, 800.0, universe.TimelineShift, false)

	assert.Equal(t, "home-t1", e.Location())
	assert.Equal(t, "T1", e.Coordinate().Timeline)
	assert.Equal(t, 800.0, e.CumulativeCost())
	assert.Contains(t, e.History(), "home -> home-t1 (timeline shift)")
}

func TestQuantumLevelAndNextID(t *testing.T) {
	tests := []struct {
		name       string
		locationID string
		quantum    string
		wantLevel  int
		wantNextID string
	}{
		{"base Q0", "home", "Q0", 0, "home-q1"},
		{"Q2", "home-q2", "Q2", 2, "home-q3"},
		{"Q3", "home-q3", "Q3", 3, "home-q4"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			coord := defaultCoord()
			coord.Quantum = tc.quantum
			e := exploration.NewEntity(tc.locationID, coord)
			assert.Equal(t, tc.wantLevel, e.QuantumLevel())
			assert.Equal(t, tc.wantNextID, e.NextQuantumID())
		})
	}
}

func TestTimelineLevelAndNextID(t *testing.T) {
	tests := []struct {
		name       string
		locationID string
		timeline   string
		wantLevel  int
		wantNextID string
	}{
		{"Prime", "home", "Prime", 0, "home-t1"},
		{"T1", "home-t1", "T1", 1, "home-t2"},
		{"T2", "home-t2", "T2", 2, "home-t3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			coord := defaultCoord()
			coord.Timeline = tc.timeline
			e := exploration.NewEntity(tc.locationID, coord)
			assert.Equal(t, tc.wantLevel, e.TimelineLevel())
			assert.Equal(t, tc.wantNextID, e.NextTimelineID())
		})
	}
}

func TestMathematicsLevelAndNextID(t *testing.T) {
	tests := []struct {
		name        string
		locationID  string
		mathematics string
		wantLevel   int
		wantNextID  string
	}{
		{"Classical", "home", "Classical", 0, "home-m1"},
		{"M1", "home-m1", "M1", 1, "home-m2"},
		{"M2", "home-m2", "M2", 2, "home-m3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			coord := defaultCoord()
			coord.Mathematics = tc.mathematics
			e := exploration.NewEntity(tc.locationID, coord)
			assert.Equal(t, tc.wantLevel, e.MathematicsLevel())
			assert.Equal(t, tc.wantNextID, e.NextMathematicsID())
		})
	}
}

func TestSimulationLevelAndNextID(t *testing.T) {
	tests := []struct {
		name       string
		locationID string
		simulation int
		wantLevel  int
		wantNextID string
	}{
		{"base", "home", 0, 0, "home-s1"},
		{"s1", "home-s1", 1, 1, "home-s2"},
		{"s2", "home-s2", 2, 2, "home-s3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			coord := defaultCoord()
			coord.Simulation = tc.simulation
			e := exploration.NewEntity(tc.locationID, coord)
			assert.Equal(t, tc.wantLevel, e.SimulationLevel())
			assert.Equal(t, tc.wantNextID, e.NextSimulationID())
		})
	}
}

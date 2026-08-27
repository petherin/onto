package exploration_test

import (
	"testing"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/stretchr/testify/assert"
)

func defaultCoord() universe.CoordinateVO { return universe.DefaultCoordinateVO() }

// Reversing transitions in a different order than they were entered must still
// keep the context stack consistent with the coordinate. A reversed transition
// removes the most recent entry for that mode wherever it sits in the stack,
// not only when it happens to be on top — otherwise the stack desyncs and a
// later 'home' tries to unwind an axis that is already at base.
func TestTransitionTo_ReverseOutOfOrder_KeepsStackConsistent(t *testing.T) {
	u1 := defaultCoord()
	u1.Universe = "U1"
	u1q1 := u1
	u1q1.Quantum = "Q1"
	q1 := defaultCoord()
	q1.Quantum = "Q1"

	e := exploration.NewEntity("home", defaultCoord())
	e.TransitionTo(universe.LocationEntity{ID: "home-u1", Coordinate: u1}, 0, universe.UniverseShift, false)
	e.TransitionTo(universe.LocationEntity{ID: "home-u1-q1", Coordinate: u1q1}, 0, universe.QuantumShift, false)

	// Unwind the universe axis first, even though quantum is on top of the stack.
	e.TransitionTo(universe.LocationEntity{ID: "home-q1", Coordinate: q1}, 0, universe.UniverseShift, true)
	transitions := e.ContextTransitions()
	assert.Len(t, transitions, 1, "universe entry should be removed even from under the quantum entry")
	assert.Equal(t, universe.QuantumShift, transitions[0].Mode)

	// Unwind the remaining quantum axis; the stack must be empty again.
	e.TransitionTo(universe.LocationEntity{ID: "home", Coordinate: defaultCoord()}, 0, universe.QuantumShift, true)
	assert.Empty(t, e.ContextTransitions())
}

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

func TestBudget_UnlimitedByDefault(t *testing.T) {
	e := exploration.NewEntity("home", defaultCoord())

	assert.False(t, e.HasBudget())
	assert.True(t, e.CanAfford(1_000_000), "no budget means every move is affordable")
	assert.Equal(t, 0.0, e.RemainingBudget())
}

func TestBudget_TracksRemainingAndAffordability(t *testing.T) {
	e := exploration.NewEntity("home", defaultCoord())
	e.SetBudget(100)

	assert.True(t, e.HasBudget())
	assert.Equal(t, 100.0, e.RemainingBudget())
	assert.True(t, e.CanAfford(100), "a move spending the whole budget is affordable")
	assert.False(t, e.CanAfford(101), "a move exceeding the budget is not affordable")

	e.MoveTo(universe.LocationEntity{ID: "a", Coordinate: defaultCoord()}, 60)
	assert.Equal(t, 40.0, e.RemainingBudget())
	assert.True(t, e.CanAfford(40))
	assert.False(t, e.CanAfford(41))
}

// Returning home is always permitted even when it costs more than the budget
// covers, so cumulative cost can exceed the budget. The remaining budget must
// then report as empty (0) rather than a negative value.
func TestBudget_RemainingClampsToZeroWhenOverspent(t *testing.T) {
	e := exploration.NewEntity("home", defaultCoord())
	e.SetBudget(10)

	e.MoveTo(universe.LocationEntity{ID: "a", Coordinate: defaultCoord()}, 25)

	assert.Equal(t, 0.0, e.RemainingBudget(), "remaining never goes negative")
	assert.False(t, e.CanAfford(1), "an overspent budget cannot afford any further move")
}

func TestWinCondition_ReachTargetThenReturnHome(t *testing.T) {
	target := defaultCoord()
	target.Quantum = "Q2"

	e := exploration.NewEntity("home", defaultCoord())
	e.SetTarget(target)
	assert.True(t, e.HasTarget())
	assert.False(t, e.ReachedTarget(), "target set at home is not reached from the start")
	assert.False(t, e.Won())

	// Shift out to Q1 then Q2 (the target); reached but not yet won.
	q1 := defaultCoord()
	q1.Quantum = "Q1"
	e.TransitionTo(universe.LocationEntity{ID: "home-q1", Coordinate: q1}, 20, universe.QuantumShift, false)
	assert.False(t, e.ReachedTarget())

	q2 := defaultCoord()
	q2.Quantum = "Q2"
	e.TransitionTo(universe.LocationEntity{ID: "home-q2", Coordinate: q2}, 20, universe.QuantumShift, false)
	assert.True(t, e.ReachedTarget(), "arriving at the target coordinate marks it reached")
	assert.False(t, e.Won(), "still away from home")

	// Shift back down to Q1 then home; returning home after reaching wins.
	e.TransitionTo(universe.LocationEntity{ID: "home-q1", Coordinate: q1}, 20, universe.QuantumShift, true)
	assert.False(t, e.Won())
	e.TransitionTo(universe.LocationEntity{ID: "home", Coordinate: defaultCoord()}, 20, universe.QuantumShift, true)
	assert.True(t, e.Won(), "reached target and returned to the start location")
}

func TestWinCondition_ReturnHomeWithoutReachingDoesNotWin(t *testing.T) {
	target := defaultCoord()
	target.Quantum = "Q2"

	e := exploration.NewEntity("home", defaultCoord())
	e.SetTarget(target)

	// Move to a station and back home without ever reaching the target.
	stationCoord := defaultCoord()
	stationCoord.Location = "Station"
	e.MoveTo(universe.LocationEntity{ID: "station", Coordinate: stationCoord}, 1)
	e.MoveTo(universe.LocationEntity{ID: "home", Coordinate: defaultCoord()}, 1)

	assert.False(t, e.ReachedTarget())
	assert.False(t, e.Won())
}

// A multi-objective quest chain must be completed in order: a later waypoint
// reached out of order does not advance the chain, returning home before the
// whole chain is done does not win, and finishing the chain then coming home
// does. The chain here is Q2 then one simulation layer deep (sim:1).
func TestQuestChain_OrderedProgressThenReturnHome(t *testing.T) {
	q1 := defaultCoord()
	q1.Quantum = "Q1"
	q2 := defaultCoord()
	q2.Quantum = "Q2"
	sim1 := defaultCoord()
	sim1.Simulation = 1

	e := exploration.NewEntity("home", defaultCoord())
	e.SetTargets([]universe.CoordinateVO{q2, sim1})
	assert.True(t, e.HasTarget())
	assert.Equal(t, 2, e.ObjectiveCount())
	assert.Equal(t, 0, e.ObjectiveIndex())
	assert.False(t, e.ReachedTarget())
	assert.Equal(t, q2.OntoAddress(), e.Target().OntoAddress(), "the current target is the first waypoint")

	// Reaching the second waypoint out of order must not advance the chain.
	e.TransitionTo(universe.LocationEntity{ID: "home-sim1", Coordinate: sim1}, 10, universe.SimulationEntry, false)
	assert.Equal(t, 0, e.ObjectiveIndex(), "waypoints must be reached in order")
	e.TransitionTo(universe.LocationEntity{ID: "home", Coordinate: defaultCoord()}, 50, universe.SimulationEntry, true)

	// Reach the first waypoint (via Q1): the chain advances to the second.
	e.TransitionTo(universe.LocationEntity{ID: "home-q1", Coordinate: q1}, 20, universe.QuantumShift, false)
	e.TransitionTo(universe.LocationEntity{ID: "home-q2", Coordinate: q2}, 20, universe.QuantumShift, false)
	assert.Equal(t, 1, e.ObjectiveIndex())
	assert.False(t, e.ReachedTarget(), "one of two waypoints is not the whole chain")
	assert.Equal(t, sim1.OntoAddress(), e.Target().OntoAddress(), "the current target advances to the second waypoint")

	// Returning home after only the first waypoint must not win.
	e.TransitionTo(universe.LocationEntity{ID: "home-q1", Coordinate: q1}, 20, universe.QuantumShift, true)
	e.TransitionTo(universe.LocationEntity{ID: "home", Coordinate: defaultCoord()}, 20, universe.QuantumShift, true)
	assert.False(t, e.Won(), "home before finishing the chain does not win")

	// Reach the second waypoint, then return home: the chain is complete and won.
	e.TransitionTo(universe.LocationEntity{ID: "home-sim1", Coordinate: sim1}, 10, universe.SimulationEntry, false)
	assert.Equal(t, 2, e.ObjectiveIndex())
	assert.True(t, e.ReachedTarget(), "both waypoints reached")
	assert.False(t, e.Won(), "still inside the simulation, away from home")
	e.TransitionTo(universe.LocationEntity{ID: "home", Coordinate: defaultCoord()}, 50, universe.SimulationEntry, true)
	assert.True(t, e.Won(), "whole chain reached and returned home wins")
}

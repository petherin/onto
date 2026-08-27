package commands_test

import (
	"testing"

	"github.com/petherin/onto/internal/application/commands"
	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestReturnHomePlan_RecordedStackShowsDecreasingLevels(t *testing.T) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)

	// Nested multi-axis excursion ending with a physical side-trip.
	_, err := (&commands.SimulateCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.UniverseCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.StructureCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.DriftCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.DriftCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.DriftCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.JumpCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.ShiftCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)

	// Side-step along a physical edge inside the branched context (the base
	// "station" ID is a different reality and is not walkable from here).
	var stepped bool
	for _, e := range u.EdgesFrom(sess.Location()) {
		if !e.Mode.IsPhysical() {
			continue
		}
		_, err = (&commands.TravelCommand{
			Universe: u, Session: sess, Pathfinder: navigation.NewBFSPathfinder(),
		}).Execute(e.To)
		require.NoError(t, err)
		stepped = true
		break
	}
	require.True(t, stepped, "expected a physical neighbour inside the branch")

	cmd := &commands.ReturnHomeCommand{
		Universe:   u,
		Session:    sess,
		Pathfinder: navigation.NewBFSPathfinder(),
		HomeID:     "home",
	}
	steps, _ := cmd.Plan()
	require.NotEmpty(t, steps)

	// LIFO unwind: shift, jump, three aligns, structure, universe, simulate.
	require.Equal(t, "shift back", steps[0].Action)
	require.Equal(t, "quantum Q1 → Q0", steps[0].Detail)

	require.Equal(t, "jump back", steps[1].Action)
	require.Equal(t, "timeline T1 → Prime", steps[1].Detail)

	require.Equal(t, "align", steps[2].Action)
	require.Equal(t, "consensus 3 → 2", steps[2].Detail)
	require.Equal(t, "consensus 2 → 1", steps[3].Detail)
	require.Equal(t, "consensus 1 → 0", steps[4].Detail)

	require.Equal(t, "structure back", steps[5].Action)
	require.Equal(t, "mathematics M1 → Classical", steps[5].Detail)

	require.Equal(t, "universe back", steps[6].Action)
	require.Equal(t, "universe U1 → Origin", steps[6].Detail)

	require.Equal(t, "simulate back", steps[7].Action)
	require.Equal(t, "simulation 1 → 0", steps[7].Detail)

	for _, step := range steps {
		require.NotContains(t, step.Detail, " → 3", "step %q should not stall at consensus 3: %q", step.Action, step.Detail)
		require.NotContains(t, step.Detail, "Q1 → Q1")
		require.NotContains(t, step.Detail, "T1 → T1")
		require.NotContains(t, step.Detail, "U1 → U1")
		require.NotContains(t, step.Detail, "M1 → M1")
		require.NotContains(t, step.Detail, "simulation 1 → 1")
		require.NotContains(t, step.Detail, "consensus 3 → 3")
		require.NotContains(t, step.Detail, "consensus 2 → 2")
		require.NotContains(t, step.Detail, "consensus 1 → 1")
	}

	// Residual physical trip should not still carry exotic axis suffixes.
	for _, step := range steps {
		if step.Action != "travel" {
			continue
		}
		require.NotContains(t, step.Detail, "-s")
		require.NotContains(t, step.Detail, "-q")
		require.NotContains(t, step.Detail, "-c")
		require.NotContains(t, step.Detail, "-u")
		require.NotContains(t, step.Detail, "-m")
		require.NotContains(t, step.Detail, "-t1")
	}
}

// Reproduces the reported "home" failure: a user enters several contextual
// branches and then manually unwinds them with the *back commands in a
// different order than they were entered. Each back command lowers its axis
// whenever that level is > 0, independently of the recorded stack order, so an
// order-sensitive pop would leave stale entries behind. A later ReturnHome
// would then try to unwind an axis already at base and fail with
// "already at base universe". After the fix the stack stays consistent, so the
// session is genuinely back home with an empty stack and ReturnHome is a no-op.
func TestReturnHomeExecute_AfterOutOfOrderBack_NoStaleTransitions(t *testing.T) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)

	// Enter universe then quantum branches: stack is [universe, quantum].
	_, err := (&commands.UniverseCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	_, err = (&commands.ShiftCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	require.Len(t, sess.ContextTransitions(), 2)

	// Unwind out of order: universe back first (quantum is on top of the stack).
	_, err = (&commands.UniverseCommand{Universe: u, Session: sess, Back: true}).Execute()
	require.NoError(t, err)
	require.Equal(t, 0, sess.UniverseLevel())
	require.Len(t, sess.ContextTransitions(), 1, "stale universe entry must not remain")

	_, err = (&commands.ShiftCommand{Universe: u, Session: sess, Back: true}).Execute()
	require.NoError(t, err)
	require.Equal(t, 0, sess.QuantumLevel())

	// Back at base reality with no outstanding transitions.
	require.Equal(t, "home", sess.Location())
	require.Empty(t, sess.ContextTransitions())

	// ReturnHome must not attempt to unwind anything and must not error.
	cmd := &commands.ReturnHomeCommand{
		Universe:   u,
		Session:    sess,
		Pathfinder: navigation.NewBFSPathfinder(),
		HomeID:     "home",
	}
	steps, err := cmd.Execute()
	require.NoError(t, err)
	require.Empty(t, steps)
	require.Equal(t, "home", sess.Location())
}

// expandNearbyAndTravel generates the next nearby location off originID, commits
// it and its bidirectional edges, then walks the session there via TravelCommand
// — mirroring what the facade does when a user travels past the edge of a
// branch's graph. It returns the new location's ID.
func expandNearbyAndTravel(t *testing.T, u *universe.Aggregate, sess *exploration.Entity, pf navigation.PathfinderService, originID string) string {
	t.Helper()
	origin, ok := u.GetLocation(originID)
	require.True(t, ok)
	loc, out, back, err := universe.NewNearbyLocation(u, originID, origin.Coordinate)
	require.NoError(t, err)
	require.NoError(t, u.AddLocation(loc))
	require.NoError(t, u.AddEdge(out))
	require.NoError(t, u.AddEdge(back))
	_, err = (&commands.TravelCommand{Universe: u, Session: sess, Pathfinder: pf}).Execute(loc.ID)
	require.NoError(t, err)
	return loc.ID
}

// TestReturnHomeExecute_NearbyInsideBranch_TravelsHome reproduces the reported
// "no route: home" failure end-to-end through the commands. A user shifts into
// universe U1 and then expands dead ends *inside* U1 — nearby nodes that have no
// counterpart in the Origin reality. On return-home, unwinding the universe used
// to land the session on an isolated Origin counterpart, so the final walk home
// failed. The fix mirrors the branch node's physical edges down, so ReturnHome
// completes and the session ends at home.
func TestReturnHomeExecute_NearbyInsideBranch_TravelsHome(t *testing.T) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	pf := navigation.NewBFSPathfinder()

	// Enter U1 (records a universe transition and clones the physical graph).
	_, err := (&commands.UniverseCommand{Universe: u, Session: sess}).Execute()
	require.NoError(t, err)
	require.Equal(t, "home-u1", sess.Location())

	// Expand two dead ends while inside U1 and walk into them.
	firstID := expandNearbyAndTravel(t, u, sess, pf, sess.Location())
	secondID := expandNearbyAndTravel(t, u, sess, pf, firstID)
	require.Equal(t, "home-1-u1", firstID)
	require.Equal(t, "home-1-1-u1", secondID)
	require.Equal(t, "home-1-1-u1", sess.Location())

	cmd := &commands.ReturnHomeCommand{
		Universe:   u,
		Session:    sess,
		Pathfinder: pf,
		HomeID:     "home",
	}
	steps, err := cmd.Execute()
	require.NoError(t, err)
	require.NotEmpty(t, steps)
	require.Equal(t, "home", sess.Location())
	require.Equal(t, 0, sess.UniverseLevel())
	require.Empty(t, sess.ContextTransitions())
}

// TestReturnHomeExecute_NearbyInsideObserverBranch_TravelsHome is the
// observer-axis analogue of the universe case above, and the concrete bug the
// depth-5 model test first surfaced. A nearby dead-end spawned inside an
// observer branch (home-1-o-cat) has only physical edges — no observer-return
// edge — and observer returns are edge-defined, so they cannot self-heal by ID
// arithmetic like the numeric axes. Return-home used to fail with "no observer
// path back from here"; the fix reconstructs the enclosing counterpart from the
// recorded transition origin, so ReturnHome now completes back to home.
func TestReturnHomeExecute_NearbyInsideObserverBranch_TravelsHome(t *testing.T) {
	u := mocks.NewTestUniverse()
	loc, _ := u.GetLocation("home")
	sess := exploration.NewEntity("home", loc.Coordinate)
	pf := navigation.NewBFSPathfinder()

	// Enter an observer branch, then expand a dead end inside it and walk there.
	_, err := (&commands.ObserveCommand{Universe: u, Session: sess, Observer: "Cat"}).Execute()
	require.NoError(t, err)
	require.Equal(t, "home-o-cat", sess.Location())

	nearbyID := expandNearbyAndTravel(t, u, sess, pf, sess.Location())
	require.Equal(t, "home-1-o-cat", nearbyID)
	require.Equal(t, "home-1-o-cat", sess.Location())

	cmd := &commands.ReturnHomeCommand{
		Universe:   u,
		Session:    sess,
		Pathfinder: pf,
		HomeID:     "home",
	}
	steps, err := cmd.Execute()
	require.NoError(t, err)
	require.NotEmpty(t, steps)
	require.Equal(t, "home", sess.Location())
	require.Equal(t, universe.DefaultCoordinateVO().Observer, sess.Coordinate().Observer)
	require.Empty(t, sess.ContextTransitions())
}

func TestReturnHomeExecute_CreatesMissingReversePath(t *testing.T) {
	u := mocks.NewTestUniverse()
	base := universe.DefaultCoordinateVO()
	sim := base
	sim.Simulation = 1

	require.NoError(t, u.AddLocation(universe.LocationEntity{
		ID: "spur", Name: "Spur", Coordinate: base,
	}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{
		ID: "spur-s1", Name: "Spur (sim:1)", Coordinate: sim,
	}))
	// Forward edge only — no reverse path back to base reality.
	require.NoError(t, u.AddEdge(universe.EdgeVO{
		From: "spur", To: "spur-s1", Mode: universe.SimulationEntry, Cost: universe.SimulationEntryCost,
	}))
	// Connect spur into the physical graph so home is reachable after exit.
	require.NoError(t, u.AddEdge(universe.EdgeVO{
		From: "home", To: "spur", Mode: universe.Walk, Distance: 1, Cost: 1,
	}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{
		From: "spur", To: "home", Mode: universe.Walk, Distance: 1, Cost: 1,
	}))

	sess := exploration.NewEntity("spur", base)
	loc, ok := u.GetLocation("spur-s1")
	require.True(t, ok)
	sess.TransitionTo(loc, universe.SimulationEntryCost, universe.SimulationEntry, false)
	require.Equal(t, "spur-s1", sess.Location())
	require.Len(t, sess.ContextTransitions(), 1)

	cmd := &commands.ReturnHomeCommand{
		Universe:   u,
		Session:    sess,
		Pathfinder: navigation.NewBFSPathfinder(),
		HomeID:     "home",
	}

	steps, _ := cmd.Plan()
	require.Equal(t, "simulate back", steps[0].Action)
	require.Equal(t, "simulation 1 → 0", steps[0].Detail)

	_, err := cmd.Execute()
	require.NoError(t, err)
	require.Equal(t, "home", sess.Location())
	require.Equal(t, 0, sess.SimulationLevel())
}

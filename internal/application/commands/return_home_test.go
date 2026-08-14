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

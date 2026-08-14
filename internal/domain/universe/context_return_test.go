package universe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLowerContextIDAndCoordinate(t *testing.T) {
	id, ok := LowerContextID("park-1-m1-u3-t1-c3-s1-q1", SimulationEntry)
	require.True(t, ok)
	// Canonical axis order is m, u, q, t, c, s — lowering simulation drops -s1.
	require.Equal(t, "park-1-m1-u3-q1-t1-c3", id)

	coord := CoordinateVO{
		Mathematics: "M1",
		Universe:    "U3",
		Timeline:    "T1",
		Quantum:     "Q1",
		Consensus:   3,
		Simulation:  1,
	}
	lower, ok := LowerContextCoordinate(coord, SimulationEntry)
	require.True(t, ok)
	require.Equal(t, 0, lower.Simulation)
	require.Equal(t, 3, lower.Consensus)
	require.Equal(t, "Q1", lower.Quantum)
}

func TestEnsureLowerContextCreatesMissingReverse(t *testing.T) {
	u := NewAggregate()
	higher := LocationEntity{
		ID:   "park-1-s1",
		Name: "Park spur",
		Coordinate: CoordinateVO{
			Location:   "Park spur",
			Simulation: 1,
			Quantum:    "Q0",
			Timeline:   "Prime",
			Universe:   "Origin",
			Mathematics: "Classical",
		},
	}
	require.NoError(t, u.AddLocation(higher))

	dest, err := EnsureLowerContext(u, higher.ID, SimulationEntry)
	require.NoError(t, err)
	require.Equal(t, "park-1", dest)

	lower, ok := u.GetLocation(dest)
	require.True(t, ok)
	require.Equal(t, 0, lower.Coordinate.Simulation)

	// Reverse edge must exist so subsequent *back commands can traverse it.
	var found bool
	for _, e := range u.EdgesFrom(higher.ID) {
		if e.To == dest && e.Mode == SimulationEntry {
			found = true
			require.Equal(t, SimulationExitCost, e.Cost)
		}
	}
	require.True(t, found)
}

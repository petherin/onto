package universe

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTransitionCost covers the per-axis step costing: symmetric axes cost the
// same up or down, simulation uses the cheap entry cost inward and the dearer
// exit cost outward, unrelated axes contribute nothing, and multiple differing
// axes sum.
func TestTransitionCost(t *testing.T) {
	base := DefaultCoordinateVO()

	q2 := base
	q2.Quantum = "Q2"

	sim2 := base
	sim2.Simulation = 2

	machine := base
	machine.Observer = "Machine"

	future := base
	future.Time = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	mixed := base
	mixed.Quantum = "Q1"
	mixed.Timeline = "T1"

	tests := []struct {
		name     string
		from, to CoordinateVO
		want     float64
	}{
		{name: "no difference", from: base, to: base, want: 0},
		{name: "quantum two up", from: base, to: q2, want: 2 * QuantumShiftCost},
		{name: "quantum two down", from: q2, to: base, want: 2 * QuantumShiftCost},
		{name: "simulation entry (cheap inward)", from: base, to: sim2, want: 2 * SimulationEntryCost},
		{name: "simulation exit (dear outward)", from: sim2, to: base, want: 2 * SimulationExitCost},
		{name: "observer changed (flat, one shift)", from: base, to: machine, want: ObserverShiftCost},
		{name: "observer restored (symmetric)", from: machine, to: base, want: ObserverShiftCost},
		{name: "observer with another axis sums", from: base, to: func() CoordinateVO { c := machine; c.Quantum = "Q1"; return c }(), want: QuantumShiftCost + ObserverShiftCost},
		{name: "time changed (flat, one shift)", from: base, to: future, want: TimeShiftCost},
		{name: "time restored (symmetric)", from: future, to: base, want: TimeShiftCost},
		{name: "time with another axis sums", from: base, to: func() CoordinateVO { c := future; c.Quantum = "Q1"; return c }(), want: QuantumShiftCost + TimeShiftCost},
		{name: "mixed axes sum", from: base, to: mixed, want: QuantumShiftCost + TimelineShiftCost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TransitionCost(tt.from, tt.to))
		})
	}
}

// TestTransitionCost_RoundTripIsPar confirms the round trip to the default
// objective (Q2 and back) costs 80 — the par the game rates against.
func TestTransitionCost_RoundTripIsPar(t *testing.T) {
	start := DefaultCoordinateVO()
	target := start
	target.Quantum = "Q2"

	par := TransitionCost(start, target) + TransitionCost(target, start)
	assert.Equal(t, 4*QuantumShiftCost, par)
	assert.Equal(t, 80.0, par)
}

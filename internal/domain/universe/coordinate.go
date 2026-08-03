package universe

import (
	"fmt"
	"time"
)

type Coordinate struct {
	Meta        string
	Mathematics string
	Universe    string
	Timeline    string
	Quantum     string
	Simulation  int
	Galaxy      string
	System      string
	Planet      string
	Country     string
	Region      string
	City        string
	Location    string
	Observer    string
	Time        time.Time
}

// QuantumLevel returns the numeric quantum level encoded in the Quantum field (Q0 → 0, Q1 → 1, …).
func (c Coordinate) QuantumLevel() int {
	n := 0
	if len(c.Quantum) > 1 && c.Quantum[0] == 'Q' {
		fmt.Sscanf(c.Quantum[1:], "%d", &n)
	}
	return n
}

// TimelineLevel returns the numeric timeline level ("Prime" → 0, "T1" → 1, "T2" → 2, …).
func (c Coordinate) TimelineLevel() int {
	if c.Timeline == "Prime" || c.Timeline == "" {
		return 0
	}
	n := 0
	if len(c.Timeline) > 1 && c.Timeline[0] == 'T' {
		fmt.Sscanf(c.Timeline[1:], "%d", &n)
	}
	return n
}

func NewCoordinate() Coordinate {
	return Coordinate{
		Meta:        "Origin",
		Mathematics: "Classical",
		Universe:    "Origin",
		Timeline:    "Prime",
		Quantum:     "Q0",
		Simulation:  0,
		Galaxy:      "Milky Way",
		System:      "Solar System",
		Planet:      "Earth",
		Country:     "United Kingdom",
		Region:      "Yorkshire",
		City:        "Leeds",
		Location:    "Home",
		Observer:    "Human",
	}
}

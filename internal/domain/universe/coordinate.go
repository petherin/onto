package universe

import (
	"fmt"
	"time"
)

// CoordinateVO is a value object representing a full reality position vector.
// Each field narrows scope from the widest (Meta, Universe) down to the most
// local (Location, Observer). Zero values are valid — a CoordinateVO with only
// Planet and City set still describes a meaningful position within a physical world.
type CoordinateVO struct {
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
func (c CoordinateVO) QuantumLevel() int {
	n := 0
	if len(c.Quantum) > 1 && c.Quantum[0] == 'Q' {
		_, _ = fmt.Sscanf(c.Quantum[1:], "%d", &n)
	}
	return n
}

// TimelineLevel returns the numeric timeline level ("Prime" → 0, "T1" → 1, "T2" → 2, …).
func (c CoordinateVO) TimelineLevel() int {
	if c.Timeline == "Prime" || c.Timeline == "" {
		return 0
	}
	n := 0
	if len(c.Timeline) > 1 && c.Timeline[0] == 'T' {
		_, _ = fmt.Sscanf(c.Timeline[1:], "%d", &n)
	}
	return n
}

// DefaultCoordinateVO returns the default starting CoordinateVO: Earth, United
// Kingdom, Yorkshire, Leeds, Home, Observer: Human, Prime timeline, Q0 quantum level.
func DefaultCoordinateVO() CoordinateVO {
	return CoordinateVO{
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

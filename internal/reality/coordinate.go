package reality

import "time"

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

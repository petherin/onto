package cli

import "github.com/petherin/onto/internal/domain/universe"

func buildDefaultUniverse() *universe.Universe {
	u := universe.NewUniverse()
	base := universe.NewCoordinate()

	u.AddLocation(universe.Location{ID: "home", Name: "Home", Description: "A quiet residential location.", Coordinate: base})
	u.AddLocation(universe.Location{ID: "station", Name: "Station", Description: "Leeds Station.", Coordinate: coordFor("Station", base)})
	u.AddLocation(universe.Location{ID: "park", Name: "Park", Description: "A green public park.", Coordinate: coordFor("Park", base)})
	u.AddLocation(universe.Location{ID: "city-centre", Name: "City Centre", Description: "The centre of town.", Coordinate: coordFor("City Centre", base)})

	u.AddEdge(universe.Edge{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1, Description: "Walk to the station"})
	u.AddEdge(universe.Edge{From: "home", To: "park", Mode: universe.Walk, Distance: 0.8, Cost: 1, Description: "Walk to the park"})
	u.AddEdge(universe.Edge{From: "station", To: "city-centre", Mode: universe.Rail, Distance: 2.0, Cost: 3, Description: "Take the rail line"})
	u.AddEdge(universe.Edge{From: "city-centre", To: "home", Mode: universe.Walk, Distance: 2.4, Cost: 2, Description: "Walk home"})

	return u
}

func coordFor(name string, base universe.Coordinate) universe.Coordinate {
	coord := base
	coord.Location = name
	coord.City = "Leeds"
	return coord
}

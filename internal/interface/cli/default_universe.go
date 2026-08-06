package cli

import "github.com/petherin/onto/internal/domain/universe"

func buildDefaultUniverse() (*universe.Aggregate, error) {
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()

	locations := []universe.LocationEntity{
		{ID: "home", Name: "Home", Description: "A quiet residential location.", Coordinate: base},
		{ID: "station", Name: "Station", Description: "Leeds Station.", Coordinate: coordFor("Station", base)},
		{ID: "park", Name: "Park", Description: "A green public park.", Coordinate: coordFor("Park", base)},
		{ID: "city-centre", Name: "City Centre", Description: "The centre of town.", Coordinate: coordFor("City Centre", base)},
	}
	for _, location := range locations {
		if err := u.AddLocation(location); err != nil {
			return nil, err
		}
	}
	edges := []universe.EdgeVO{
		{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1, Description: "Walk to the station"},
		{From: "home", To: "park", Mode: universe.Walk, Distance: 0.8, Cost: 1, Description: "Walk to the park"},
		{From: "station", To: "city-centre", Mode: universe.Rail, Distance: 2.0, Cost: 3, Description: "Take the rail line"},
		{From: "city-centre", To: "home", Mode: universe.Walk, Distance: 2.4, Cost: 2, Description: "Walk home"},
	}
	for _, edge := range edges {
		if err := u.AddEdge(edge); err != nil {
			return nil, err
		}
	}

	return u, nil
}

func coordFor(name string, base universe.CoordinateVO) universe.CoordinateVO {
	coord := base
	coord.Location = name
	coord.City = "Leeds"
	return coord
}

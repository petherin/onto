// Package mocks contains generated mock implementations of domain interfaces
// (produced by mockery) and shared test fixtures such as NewTestUniverse.
// Nothing in this package should be imported outside of test files.
package mocks

import "github.com/petherin/onto/internal/domain/universe"

// NewTestUniverse builds a minimal two-location universe suitable for most tests.
//
//	home --walk--> station --walk--> home
func NewTestUniverse() *universe.Aggregate {
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()
	station := base
	station.Location = "Station"

	u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home", Description: "Start", Coordinate: base})
	u.AddLocation(universe.LocationEntity{ID: "station", Name: "Station", Description: "Train station", Coordinate: station})
	u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1})
	u.AddEdge(universe.EdgeVO{From: "station", To: "home", Mode: universe.Walk, Distance: 1.6, Cost: 1})
	return u
}

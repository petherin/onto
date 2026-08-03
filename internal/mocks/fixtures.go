// Package mocks contains generated mock implementations of domain interfaces
// (produced by mockery) and shared test fixtures such as NewTestUniverse.
// Nothing in this package should be imported outside of test files.
package mocks

import "github.com/petherin/onto/internal/domain/universe"

// NewTestUniverse builds a minimal two-location universe suitable for most tests.
//
//	home --walk--> station --walk--> home
func NewTestUniverse() *universe.Universe {
	u := universe.NewUniverse()
	base := universe.NewCoordinate()
	station := base
	station.Location = "Station"

	u.AddLocation(universe.Location{ID: "home", Name: "Home", Description: "Start", Coordinate: base})
	u.AddLocation(universe.Location{ID: "station", Name: "Station", Description: "Train station", Coordinate: station})
	u.AddEdge(universe.Edge{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1})
	u.AddEdge(universe.Edge{From: "station", To: "home", Mode: universe.Walk, Distance: 1.6, Cost: 1})
	return u
}

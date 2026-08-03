// Package universe is the core domain model. It defines the aggregate root
// (Universe), entities (Location), value objects (Coordinate, Edge,
// TravelMode), domain interfaces (Repository, LocationGenerator), and domain
// services (BranchQuantum, BranchTimeline) that encode the rules of reality
// navigation. Nothing in this package may import other internal packages.
package universe

// Universe is the aggregate root of the domain. It owns all Locations and
// directed Edges and exposes them only through its methods, keeping the
// internal maps unexported to preserve encapsulation.
type Universe struct {
	locations map[string]Location
	edges     map[string][]Edge
}

// NewUniverse returns an empty, ready-to-use Universe.
func NewUniverse() *Universe {
	return &Universe{
		locations: make(map[string]Location),
		edges:     make(map[string][]Edge),
	}
}

// AddLocation inserts or replaces a location in the universe.
func (u *Universe) AddLocation(location Location) {
	u.locations[location.ID] = location
}

// AddEdge appends a directed edge to the universe graph.
func (u *Universe) AddEdge(edge Edge) {
	u.edges[edge.From] = append(u.edges[edge.From], edge)
}

// GetLocation looks up a location by its ID, returning the location and a
// boolean indicating whether it was found.
func (u *Universe) GetLocation(id string) (Location, bool) {
	location, ok := u.locations[id]
	return location, ok
}

// EdgesFrom returns all edges originating from the given location ID.
func (u *Universe) EdgesFrom(id string) []Edge {
	return u.edges[id]
}

// AllLocations returns all locations in the universe as a slice.
func (u *Universe) AllLocations() []Location {
	locs := make([]Location, 0, len(u.locations))
	for _, l := range u.locations {
		locs = append(locs, l)
	}
	return locs
}

// AllLocationIDs returns every location ID in the universe.
func (u *Universe) AllLocationIDs() []string {
	ids := make([]string, 0, len(u.locations))
	for id := range u.locations {
		ids = append(ids, id)
	}
	return ids
}

// AllEdgesFlat returns every edge in the universe as a flat slice.
func (u *Universe) AllEdgesFlat() []Edge {
	var result []Edge
	for _, list := range u.edges {
		result = append(result, list...)
	}
	return result
}

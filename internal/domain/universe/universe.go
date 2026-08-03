package universe

type Universe struct {
	locations map[string]Location
	edges     map[string][]Edge
}

func NewUniverse() *Universe {
	return &Universe{
		locations: make(map[string]Location),
		edges:     make(map[string][]Edge),
	}
}

func (u *Universe) AddLocation(location Location) {
	u.locations[location.ID] = location
}

func (u *Universe) AddEdge(edge Edge) {
	u.edges[edge.From] = append(u.edges[edge.From], edge)
}

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

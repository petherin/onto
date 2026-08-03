// Package universe is the core domain model. It defines the aggregate root
// (UniverseAggregate), entities (LocationEntity), value objects (CoordinateVO,
// EdgeVO, TravelModeVO), domain interfaces (UniverseRepository,
// LocationGeneratorService), and domain services (BranchQuantumService,
// BranchTimelineService) that encode the rules of reality navigation.
// Nothing in this package may import other internal packages.
package universe

// UniverseAggregate is the aggregate root of the domain. It owns all
// LocationEntity values and directed EdgeVO values, exposing them only through
// its methods so that internal invariants are enforced by the struct itself.
type UniverseAggregate struct {
	locations map[string]LocationEntity
	edges     map[string][]EdgeVO
}

// NewUniverseAggregate returns an empty, ready-to-use UniverseAggregate.
func NewUniverseAggregate() *UniverseAggregate {
	return &UniverseAggregate{
		locations: make(map[string]LocationEntity),
		edges:     make(map[string][]EdgeVO),
	}
}

// AddLocation inserts or replaces a LocationEntity in the aggregate.
func (u *UniverseAggregate) AddLocation(location LocationEntity) {
	u.locations[location.ID] = location
}

// AddEdge appends a directed EdgeVO to the aggregate graph.
func (u *UniverseAggregate) AddEdge(edge EdgeVO) {
	u.edges[edge.From] = append(u.edges[edge.From], edge)
}

// GetLocation looks up a LocationEntity by its ID, returning the entity and a
// boolean indicating whether it was found.
func (u *UniverseAggregate) GetLocation(id string) (LocationEntity, bool) {
	location, ok := u.locations[id]
	return location, ok
}

// EdgesFrom returns all EdgeVO values originating from the given location ID.
func (u *UniverseAggregate) EdgesFrom(id string) []EdgeVO {
	return u.edges[id]
}

// AllLocations returns all LocationEntity values in the aggregate as a slice.
func (u *UniverseAggregate) AllLocations() []LocationEntity {
	locs := make([]LocationEntity, 0, len(u.locations))
	for _, l := range u.locations {
		locs = append(locs, l)
	}
	return locs
}

// AllLocationIDs returns every location ID in the aggregate.
func (u *UniverseAggregate) AllLocationIDs() []string {
	ids := make([]string, 0, len(u.locations))
	for id := range u.locations {
		ids = append(ids, id)
	}
	return ids
}

// AllEdgesFlat returns every EdgeVO in the aggregate as a flat slice.
func (u *UniverseAggregate) AllEdgesFlat() []EdgeVO {
	var result []EdgeVO
	for _, list := range u.edges {
		result = append(result, list...)
	}
	return result
}

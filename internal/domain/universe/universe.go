// Package universe is the core domain model. It defines the aggregate root
// (Aggregate), entities (LocationEntity), value objects (CoordinateVO,
// EdgeVO, TravelModeVO), domain interfaces (Repository,
// LocationGeneratorService), and domain services (BranchQuantumService,
// BranchTimelineService) that encode the rules of reality navigation.
// Nothing in this package may import other internal packages.
package universe

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidLocation       = errors.New("invalid location")
	ErrLocationAlreadyExists = errors.New("location already exists")
	ErrInvalidEdge           = errors.New("invalid edge")
	ErrUnknownEdgeEndpoint   = errors.New("edge endpoint does not exist")
	ErrDuplicateEdge         = errors.New("duplicate edge")
	ErrPhysicalRealityCross  = errors.New("physical edge crosses reality boundary")
)

// Aggregate is the aggregate root of the domain. It owns all
// LocationEntity values and directed EdgeVO values, exposing them only through
// its methods so that internal invariants are enforced by the struct itself.
type Aggregate struct {
	locations map[string]LocationEntity
	edges     map[string][]EdgeVO
}

// NewAggregate returns an empty, ready-to-use Aggregate.
func NewAggregate() *Aggregate {
	return &Aggregate{
		locations: make(map[string]LocationEntity),
		edges:     make(map[string][]EdgeVO),
	}
}

// AddLocation adds a uniquely identified location to the aggregate.
func (u *Aggregate) AddLocation(location LocationEntity) error {
	if !validLocationID(location.ID) {
		return fmt.Errorf("%w: %q", ErrInvalidLocation, location.ID)
	}
	if _, exists := u.locations[location.ID]; exists {
		return fmt.Errorf("%w: %s", ErrLocationAlreadyExists, location.ID)
	}
	u.locations[location.ID] = location
	return nil
}

// AddEdge adds a directed edge after enforcing graph and reality invariants.
func (u *Aggregate) AddEdge(edge EdgeVO) error {
	if edge.From == "" || edge.To == "" || edge.From == edge.To || !edge.Mode.IsKnown() ||
		edge.Distance < 0 || edge.Cost < 0 {
		return fmt.Errorf("%w: %+v", ErrInvalidEdge, edge)
	}
	from, fromOK := u.GetLocation(edge.From)
	to, toOK := u.GetLocation(edge.To)
	if !fromOK || !toOK {
		return fmt.Errorf("%w: %s -> %s", ErrUnknownEdgeEndpoint, edge.From, edge.To)
	}
	if edge.Mode.IsPhysical() && !from.Coordinate.SamePhysicalReality(to.Coordinate) {
		return fmt.Errorf("%w: %s -> %s", ErrPhysicalRealityCross, edge.From, edge.To)
	}
	for _, existing := range u.edges[edge.From] {
		if existing.To == edge.To && existing.Mode == edge.Mode {
			return fmt.Errorf("%w: %s -> %s (%s)", ErrDuplicateEdge, edge.From, edge.To, edge.Mode)
		}
	}
	u.edges[edge.From] = append(u.edges[edge.From], edge)
	return nil
}

func validLocationID(id string) bool {
	if id == "" || strings.TrimSpace(id) != id {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	return true
}

// GetLocation looks up a LocationEntity by its ID, returning the entity and a
// boolean indicating whether it was found.
func (u *Aggregate) GetLocation(id string) (LocationEntity, bool) {
	location, ok := u.locations[id]
	return location, ok
}

// EdgesFrom returns all EdgeVO values originating from the given location ID.
func (u *Aggregate) EdgesFrom(id string) []EdgeVO {
	return u.edges[id]
}

// AllLocations returns all LocationEntity values in the aggregate as a slice.
func (u *Aggregate) AllLocations() []LocationEntity {
	locs := make([]LocationEntity, 0, len(u.locations))
	for _, l := range u.locations {
		locs = append(locs, l)
	}
	return locs
}

// AllLocationIDs returns every location ID in the aggregate.
func (u *Aggregate) AllLocationIDs() []string {
	ids := make([]string, 0, len(u.locations))
	for id := range u.locations {
		ids = append(ids, id)
	}
	return ids
}

// AllEdgesFlat returns every EdgeVO in the aggregate as a flat slice.
func (u *Aggregate) AllEdgesFlat() []EdgeVO {
	var result []EdgeVO
	for _, list := range u.edges {
		result = append(result, list...)
	}
	return result
}

// Package universe is the core domain model. It defines the aggregate root
// (Aggregate), entities (LocationEntity), value objects (CoordinateVO,
// EdgeVO, TravelModeVO), the Repository interface, and contextual branch
// functions that encode the rules of reality navigation.
// Nothing in this package may import other internal packages.
package universe

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidLocation reports a location with an invalid identifier.
	ErrInvalidLocation = errors.New("invalid location")
	// ErrLocationAlreadyExists reports a duplicate location identity.
	ErrLocationAlreadyExists = errors.New("location already exists")
	// ErrInvalidEdge reports an edge with invalid values.
	ErrInvalidEdge = errors.New("invalid edge")
	// ErrUnknownEdgeEndpoint reports an edge whose endpoint is absent.
	ErrUnknownEdgeEndpoint = errors.New("edge endpoint does not exist")
	// ErrDuplicateEdge reports an existing edge with the same source, target, and mode.
	ErrDuplicateEdge = errors.New("duplicate edge")
	// ErrPhysicalRealityCross reports physical travel between different reality contexts.
	ErrPhysicalRealityCross = errors.New("physical edge crosses reality boundary")
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
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
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

// FindInReality resolves a location by name or ID within a specific reality,
// i.e. among the copies of the world that share coord's non-physical axes. A
// plain place name ("park") therefore resolves to the copy that sits in the
// caller's current reality (e.g. "park-m1-u1-t1-c1"), not the base-reality
// original. An exact same-reality ID match wins first; otherwise the physical
// Location field is matched. Returns false when no same-reality match exists.
func (u *Aggregate) FindInReality(coord CoordinateVO, name string) (LocationEntity, bool) {
	norm := normaliseName(name)
	if loc, ok := u.locations[norm]; ok && loc.Coordinate.SamePhysicalReality(coord) {
		return loc, true
	}
	for _, loc := range u.locations {
		if loc.Coordinate.SamePhysicalReality(coord) && normaliseName(loc.Coordinate.Location) == norm {
			return loc, true
		}
	}
	return LocationEntity{}, false
}

// normaliseName lower-cases and hyphenates a place name so a human-typed
// destination ("City Centre") matches a canonical location ID/field ("city-centre").
func normaliseName(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}

// AllEdgesFlat returns every EdgeVO in the aggregate as a flat slice.
func (u *Aggregate) AllEdgesFlat() []EdgeVO {
	var result []EdgeVO
	for _, list := range u.edges {
		result = append(result, list...)
	}
	return result
}

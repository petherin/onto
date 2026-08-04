// Package persistence implements the universe.Repository interface using
// a JSON file on disk. The serialised format stores locations and edges as flat
// slices so the file is human-readable and easy to seed by hand.
package persistence

import (
	"encoding/json"
	"os"

	"github.com/petherin/onto/internal/domain/universe"
)

type serialized struct {
	Locations []universe.LocationEntity `json:"locations"`
	Edges     []universe.EdgeVO         `json:"edges"`
}

// JSONRepository implements universe.Repository using a JSON file on disk.
type JSONRepository struct {
	path string
}

// NewJSONRepository returns a JSONRepository that reads and writes the file at path.
func NewJSONRepository(path string) *JSONRepository {
	return &JSONRepository{path: path}
}

// Load reads the JSON file and reconstructs an Aggregate from it.
func (r *JSONRepository) Load() (*universe.Aggregate, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, err
	}

	var s serialized
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	defaults := universe.DefaultCoordinateVO()
	u := universe.NewAggregate()
	for i := range s.Locations {
		s.Locations[i].Coordinate = mergeCoordinate(s.Locations[i].Coordinate, defaults)
		u.AddLocation(s.Locations[i])
	}
	for _, e := range s.Edges {
		u.AddEdge(e)
	}
	return u, nil
}

// mergeCoordinate fills any empty string fields in c with the corresponding
// value from defaults. This repairs coordinates written by older versions of
// the code that did not persist the full coordinate.
func mergeCoordinate(c, defaults universe.CoordinateVO) universe.CoordinateVO {
	if c.Meta == "" {
		c.Meta = defaults.Meta
	}
	if c.Mathematics == "" {
		c.Mathematics = defaults.Mathematics
	}
	if c.Universe == "" {
		c.Universe = defaults.Universe
	}
	if c.Timeline == "" {
		c.Timeline = defaults.Timeline
	}
	if c.Quantum == "" {
		c.Quantum = defaults.Quantum
	}
	if c.Galaxy == "" {
		c.Galaxy = defaults.Galaxy
	}
	if c.System == "" {
		c.System = defaults.System
	}
	if c.Planet == "" {
		c.Planet = defaults.Planet
	}
	if c.Country == "" {
		c.Country = defaults.Country
	}
	if c.Region == "" {
		c.Region = defaults.Region
	}
	if c.Observer == "" {
		c.Observer = defaults.Observer
	}
	return c
}

// Save serialises the Aggregate to indented JSON and writes it to disk atomically.
func (r *JSONRepository) Save(u *universe.Aggregate) error {
	s := serialized{
		Locations: u.AllLocations(),
		Edges:     u.AllEdgesFlat(),
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

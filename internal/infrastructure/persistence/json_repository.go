// Package persistence implements the universe.Repository interface using a
// JSON file on disk. The serialised format stores locations and edges as flat
// slices so the file is human-readable and easy to seed by hand.
package persistence

import (
	"encoding/json"
	"os"

	"github.com/petherin/onto/internal/domain/universe"
)

type serialized struct {
	Locations []universe.Location `json:"locations"`
	Edges     []universe.Edge     `json:"edges"`
}

// JSONRepository implements universe.Repository using a JSON file on disk.
type JSONRepository struct {
	path string
}

// NewJSONRepository returns a JSONRepository that reads and writes the file at path.
func NewJSONRepository(path string) *JSONRepository {
	return &JSONRepository{path: path}
}

// Load reads the JSON file and reconstructs a Universe from it.
func (r *JSONRepository) Load() (*universe.Universe, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, err
	}

	var s serialized
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	u := universe.NewUniverse()
	for _, loc := range s.Locations {
		u.AddLocation(loc)
	}
	for _, e := range s.Edges {
		u.AddEdge(e)
	}
	return u, nil
}

// Save serialises the Universe to indented JSON and writes it to disk atomically.
func (r *JSONRepository) Save(u *universe.Universe) error {
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

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

func NewJSONRepository(path string) *JSONRepository {
	return &JSONRepository{path: path}
}

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

func (r *JSONRepository) Save(u *universe.Universe) error {
	var s serialized
	for _, loc := range u.Locations {
		s.Locations = append(s.Locations, loc)
	}
	for _, list := range u.Edges {
		for _, e := range list {
			s.Edges = append(s.Edges, e)
		}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

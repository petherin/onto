package cli

import (
	"encoding/json"
	"os"

	"github.com/petherin/onto/internal/reality"
)

// SaveUniverse writes the universe to the given path as JSON.
func SaveUniverse(u *reality.Universe, path string) error {
	type cfg struct {
		Locations []reality.Location `json:"locations"`
		Edges     []reality.Edge     `json:"edges"`
	}

	var c cfg
	for _, loc := range u.Locations {
		c.Locations = append(c.Locations, loc)
	}
	for _, list := range u.Edges {
		for _, e := range list {
			c.Edges = append(c.Edges, e)
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadUniverse attempts to read a universe from the given path. If the file
// does not exist or cannot be read, returns an error.
func LoadUniverse(path string) (*reality.Universe, error) {
	type cfg struct {
		Locations []reality.Location `json:"locations"`
		Edges     []reality.Edge     `json:"edges"`
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c cfg
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	u := reality.NewUniverse()
	for _, loc := range c.Locations {
		u.AddLocation(loc)
	}
	for _, e := range c.Edges {
		u.AddEdge(e)
	}
	return u, nil
}

// helper for App to save current universe
func (a *App) saveConfig() error {
	return SaveUniverse(a.universe, dataFile())
}

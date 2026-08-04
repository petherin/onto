package persistence_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "locations.json")
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "universe.json")
	repo := persistence.NewJSONRepository(path)

	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home", Coordinate: coord})
	u.AddLocation(universe.LocationEntity{ID: "station", Name: "Station", Coordinate: coord})
	u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1})

	require.NoError(t, repo.Save(u))

	loaded, err := repo.Load()
	require.NoError(t, err)

	_, okHome := loaded.GetLocation("home")
	_, okStation := loaded.GetLocation("station")
	assert.True(t, okHome)
	assert.True(t, okStation)

	edges := loaded.EdgesFrom("home")
	require.Len(t, edges, 1)
	assert.Equal(t, "station", edges[0].To)
	assert.Equal(t, universe.Walk, edges[0].Mode)
}

func TestLoad_FileNotFound(t *testing.T) {
	repo := persistence.NewJSONRepository("/no/such/file.json")
	_, err := repo.Load()
	assert.Error(t, err)
}

func TestLoad_MergeCoordinate_FillsMissingFieldsFromDefaults(t *testing.T) {
	// Simulate a stale file that only has City and Location set.
	type rawLocation struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Coordinate struct {
			City     string `json:"City"`
			Location string `json:"Location"`
		} `json:"coordinate"`
	}
	type payload struct {
		Locations []rawLocation       `json:"locations"`
		Edges     []universe.EdgeVO   `json:"edges"`
	}

	raw := payload{
		Locations: []rawLocation{
			{ID: "home", Name: "Home", Coordinate: struct {
				City     string `json:"City"`
				Location string `json:"Location"`
			}{"Leeds", "Home"}},
		},
	}
	path := writeTempJSON(t, raw)

	loaded, err := persistence.NewJSONRepository(path).Load()
	require.NoError(t, err)

	loc, ok := loaded.GetLocation("home")
	require.True(t, ok)

	defaults := universe.DefaultCoordinateVO()
	assert.Equal(t, defaults.Planet, loc.Coordinate.Planet, "Planet should be merged from defaults")
	assert.Equal(t, defaults.Country, loc.Coordinate.Country, "Country should be merged from defaults")
	assert.Equal(t, defaults.Timeline, loc.Coordinate.Timeline, "Timeline should be merged from defaults")
	assert.Equal(t, defaults.Quantum, loc.Coordinate.Quantum, "Quantum should be merged from defaults")
	assert.Equal(t, "Leeds", loc.Coordinate.City, "City explicitly set should be preserved")
}

func TestLoad_MergeCoordinate_PreservesExistingValues(t *testing.T) {
	u := universe.NewAggregate()
	coord := universe.DefaultCoordinateVO()
	coord.Planet = "Mars"
	coord.Timeline = "T3"
	u.AddLocation(universe.LocationEntity{ID: "base", Name: "Martian Base", Coordinate: coord})

	path := filepath.Join(t.TempDir(), "mars.json")
	repo := persistence.NewJSONRepository(path)
	require.NoError(t, repo.Save(u))

	loaded, err := repo.Load()
	require.NoError(t, err)

	loc, _ := loaded.GetLocation("base")
	assert.Equal(t, "Mars", loc.Coordinate.Planet, "non-default planet must not be overwritten")
	assert.Equal(t, "T3", loc.Coordinate.Timeline, "non-default timeline must not be overwritten")
}

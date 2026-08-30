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
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home", Name: "Home", Coordinate: coord}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "station", Name: "Station", Coordinate: coord}))
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "home-1", Name: "The Quiet Wharf", Coordinate: coord, Generated: true}))
	require.NoError(t, u.AddEdge(universe.EdgeVO{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1}))

	require.NoError(t, repo.Save(u))

	loaded, err := repo.Load()
	require.NoError(t, err)

	_, okHome := loaded.GetLocation("home")
	_, okStation := loaded.GetLocation("station")
	assert.True(t, okHome)
	assert.True(t, okStation)

	// A hand-seeded location keeps Generated false; an auto-generated one must
	// round-trip with the flag set so the validator keeps skipping it.
	home, _ := loaded.GetLocation("home")
	assert.False(t, home.Generated, "hand-seeded location must not be marked Generated")
	generated, okGenerated := loaded.GetLocation("home-1")
	require.True(t, okGenerated)
	assert.True(t, generated.Generated, "auto-generated flag must survive save/load")

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
		Locations []rawLocation     `json:"locations"`
		Edges     []universe.EdgeVO `json:"edges"`
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
	require.NoError(t, u.AddLocation(universe.LocationEntity{ID: "base", Name: "Martian Base", Coordinate: coord}))

	path := filepath.Join(t.TempDir(), "mars.json")
	repo := persistence.NewJSONRepository(path)
	require.NoError(t, repo.Save(u))

	loaded, err := repo.Load()
	require.NoError(t, err)

	loc, _ := loaded.GetLocation("base")
	assert.Equal(t, "Mars", loc.Coordinate.Planet, "non-default planet must not be overwritten")
	assert.Equal(t, "T3", loc.Coordinate.Timeline, "non-default timeline must not be overwritten")
}

// TestLoad_RepairsNonCanonicalIDs reproduces a save written by an older version
// where a nearby location generated inside a universe branch got the corrupt ID
// "park-u1-1" (a bare index after the -u1 axis suffix). On load the repository
// must re-canonicalise it to "park-1-u1", remap every edge endpoint, and thereby
// let universe back / return-home step down from it again.
func TestLoad_RepairsNonCanonicalIDs(t *testing.T) {
	origin := universe.DefaultCoordinateVO()
	origin.Location = "Park"

	u1 := origin
	u1.Universe = "U1"

	// park -> park-u1 (universe branch) -> park-u1-1 (corrupt nearby in-branch).
	payload := struct {
		Locations []universe.LocationEntity `json:"locations"`
		Edges     []universe.EdgeVO         `json:"edges"`
	}{
		Locations: []universe.LocationEntity{
			{ID: "park", Name: "Park", Coordinate: origin},
			{ID: "park-u1", Name: "Park", Coordinate: u1},
			{ID: "park-u1-1", Name: "Nearby 1", Coordinate: u1},
		},
		Edges: []universe.EdgeVO{
			{From: "park", To: "park-u1", Mode: universe.UniverseShift, Cost: universe.UniverseShiftCost},
			{From: "park-u1", To: "park", Mode: universe.UniverseShift, Cost: universe.UniverseShiftCost},
			{From: "park-u1", To: "park-u1-1", Mode: universe.Walk, Cost: 1},
			{From: "park-u1-1", To: "park-u1", Mode: universe.Walk, Cost: 1},
		},
	}
	path := writeTempJSON(t, payload)

	loaded, err := persistence.NewJSONRepository(path).Load()
	require.NoError(t, err)

	// The corrupt ID is gone; the canonical one is present.
	_, okOld := loaded.GetLocation("park-u1-1")
	assert.False(t, okOld, "corrupt ID must be renamed away")
	repaired, okNew := loaded.GetLocation("park-1-u1")
	require.True(t, okNew, "canonical ID must exist after repair")
	assert.Equal(t, "U1", repaired.Coordinate.Universe)

	// Edges were remapped onto the canonical ID.
	edges := loaded.EdgesFrom("park-u1")
	var found bool
	for _, e := range edges {
		if e.To == "park-1-u1" {
			found = true
		}
		assert.NotEqual(t, "park-u1-1", e.To, "no edge may still point at the corrupt ID")
	}
	assert.True(t, found, "edge from park-u1 must now target park-1-u1")

	// universe back now resolves from the repaired node instead of failing.
	destID, err := universe.EnsureLowerContext(loaded, "park-1-u1", universe.UniverseShift)
	require.NoError(t, err)
	assert.Equal(t, "park-1", destID)
}

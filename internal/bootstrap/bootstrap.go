// Package bootstrap is the application bootstrap layer. It is the only place
// that knows about both the infrastructure implementations and the domain types
// — it wires them together and hands the assembled state to a delivery
// mechanism (CLI, TUI, web, …). Delivery-mechanism packages must not import
// this package; only cmd/ entry points (the Composition Root) should call
// Bootstrap.
package bootstrap

import (
	"fmt"
	"os"

	"github.com/petherin/onto/internal/domain/universe"
	"github.com/petherin/onto/internal/infrastructure/persistence"
)

const (
	defaultDataFile      = "data/locations.json"
	defaultStartLocation = "home"
)

// Config holds the runtime configuration for the application.
type Config struct {
	DataFile      string
	StartLocation string
}

// DefaultConfig builds a Config from environment variables, falling back to
// sensible defaults. Override with ONTO_DATA_FILE and ONTO_START_LOCATION.
func DefaultConfig() Config {
	dataFile := os.Getenv("ONTO_DATA_FILE")
	if dataFile == "" {
		dataFile = defaultDataFile
	}
	startLoc := os.Getenv("ONTO_START_LOCATION")
	if startLoc == "" {
		startLoc = defaultStartLocation
	}
	return Config{DataFile: dataFile, StartLocation: startLoc}
}

// State is the fully-assembled domain state returned by Bootstrap.
type State struct {
	Universe *universe.Aggregate
	Repo     universe.Repository
	StartID  string
}

// Bootstrap loads (or initialises) the universe from disk and returns a State
// ready to be handed to a delivery-mechanism constructor (cli.New, tui.Run, …).
func Bootstrap(cfg Config) (State, error) {
	repo := persistence.NewJSONRepository(cfg.DataFile)
	u, err := repo.Load()
	if err != nil {
		u, err = buildDefaultUniverse()
		if err != nil {
			return State{}, fmt.Errorf("build default universe: %w", err)
		}
	}

	sl := cfg.StartLocation
	if _, ok := u.GetLocation(sl); !ok {
		base := universe.DefaultCoordinateVO()
		if addErr := u.AddLocation(universe.LocationEntity{
			ID:          sl,
			Name:        sl,
			Description: "Start location (auto-added)",
			Coordinate:  base,
		}); addErr != nil {
			return State{}, fmt.Errorf("add start location: %w", addErr)
		}
	}

	// Fallback: if the requested start location still doesn't resolve, use the
	// first available location so the app always starts somewhere valid.
	start := sl
	if _, ok := u.GetLocation(start); !ok {
		if ids := u.AllLocationIDs(); len(ids) > 0 {
			start = ids[0]
		}
	}

	return State{Universe: u, Repo: repo, StartID: start}, nil
}

// buildDefaultUniverse constructs the hard-coded starter universe used when no
// saved data file exists yet.
func buildDefaultUniverse() (*universe.Aggregate, error) {
	u := universe.NewAggregate()
	base := universe.DefaultCoordinateVO()

	locations := []universe.LocationEntity{
		{ID: "home", Name: "Home", Description: "A quiet residential location.", Coordinate: base},
		{ID: "station", Name: "Station", Description: "Leeds Station.", Coordinate: coordFor("Station", base)},
		{ID: "park", Name: "Park", Description: "A green public park.", Coordinate: coordFor("Park", base)},
		{ID: "city-centre", Name: "City Centre", Description: "The centre of town.", Coordinate: coordFor("City Centre", base)},
	}
	for _, loc := range locations {
		if err := u.AddLocation(loc); err != nil {
			return nil, err
		}
	}

	edges := []universe.EdgeVO{
		{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1, Description: "Walk to the station"},
		{From: "home", To: "park", Mode: universe.Walk, Distance: 0.8, Cost: 1, Description: "Walk to the park"},
		{From: "station", To: "city-centre", Mode: universe.Rail, Distance: 2.0, Cost: 3, Description: "Take the rail line"},
		{From: "city-centre", To: "home", Mode: universe.Walk, Distance: 2.4, Cost: 2, Description: "Walk home"},
	}
	for _, e := range edges {
		if err := u.AddEdge(e); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func coordFor(name string, base universe.CoordinateVO) universe.CoordinateVO {
	coord := base
	coord.Location = name
	coord.City = "Leeds"
	return coord
}

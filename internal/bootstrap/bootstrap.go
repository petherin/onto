// Package bootstrap is the application bootstrap layer. It is the only place
// that knows about both the infrastructure implementations and the domain types
// — it wires them together and hands the assembled state to a delivery
// mechanism (CLI, web, …). Delivery-mechanism packages must not import
// this package; only cmd/ entry points (the Composition Root) should call
// Bootstrap.
package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/petherin/onto/internal/application/facade"
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
	// Game enables game mode (a spending budget and a win objective). It is on
	// by default; set ONTO_GAME to a falsey value to disable it. Budget is the
	// starting spending pool; 0 means use facade.DefaultBudget.
	Game   bool
	Budget float64
	// Quest is the fixed, ordered quest chain parsed from ONTO_QUEST
	// (comma-separated Onto Addresses). When set it overrides Objectives and the
	// default chain.
	Quest []universe.CoordinateVO
	// Objectives is the pool of candidate objectives parsed from ONTO_OBJECTIVES
	// (comma-separated Onto Addresses). When ONTO_QUEST is unset, a random quest
	// (2-4 distinct objectives) is built from this pool on start and re-rolled by
	// the 'quest' command. Empty means fall back to the default chain.
	Objectives []universe.CoordinateVO
}

// DefaultConfig builds a Config from environment variables, falling back to
// sensible defaults. It first loads a .env file from the working directory (if
// present) for any variables not already set in the real environment, so real
// environment variables always take precedence. Override with ONTO_DATA_FILE,
// ONTO_START_LOCATION, ONTO_GAME, ONTO_BUDGET, ONTO_QUEST and ONTO_OBJECTIVES.
func DefaultConfig() Config {
	loadDotEnv(".env")
	dataFile := os.Getenv("ONTO_DATA_FILE")
	if dataFile == "" {
		dataFile = defaultDataFile
	}
	startLoc := os.Getenv("ONTO_START_LOCATION")
	if startLoc == "" {
		startLoc = defaultStartLocation
	}
	return Config{
		DataFile:      dataFile,
		StartLocation: startLoc,
		Game:          gameEnabled(os.Getenv("ONTO_GAME")),
		Budget:        budgetOverride(os.Getenv("ONTO_BUDGET")),
		Quest:         parseAddressList(os.Getenv("ONTO_QUEST")),
		Objectives:    parseAddressList(os.Getenv("ONTO_OBJECTIVES")),
	}
}

// loadDotEnv reads simple KEY=VALUE lines from a .env file and sets any variable
// not already present in the environment (so real environment variables always
// win). A missing file is not an error. Blank lines and lines beginning with '#'
// are ignored; surrounding whitespace and a single pair of matching quotes
// around the value are stripped. Only the first '=' splits key from value, so
// values may themselves contain '='.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		_ = os.Setenv(key, val)
	}
}

// gameEnabled interprets ONTO_GAME. Game mode is on by default (empty value);
// only an explicit falsey value (0, false, no, off) turns it off.
func gameEnabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// budgetOverride parses ONTO_BUDGET. An unset, unparseable, or non-positive
// value yields 0, which GameOptions treats as "use facade.DefaultBudget".
func budgetOverride(v string) float64 {
	budget, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || budget <= 0 {
		return 0
	}
	return budget
}

// parseAddressList parses a comma-separated list of Onto Addresses (used for
// both ONTO_QUEST and ONTO_OBJECTIVES) into coordinates. Blank entries are
// skipped and an entry that fails to parse is dropped (the rest are kept). An
// unset value, or one with no usable addresses, yields nil.
func parseAddressList(v string) []universe.CoordinateVO {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	var targets []universe.CoordinateVO
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		coord, err := universe.ParseOntoAddress(part)
		if err != nil {
			continue
		}
		targets = append(targets, coord)
	}
	return targets
}

// GameOptions turns the resolved game configuration into facade options. When
// game mode is disabled it returns no options at all — no budget, no objective,
// and no objective pool — so quest generation is skipped entirely and the
// session runs with unlimited spending. When game mode is on it applies the
// budget and resolves the objective in precedence order: a fixed chain from
// cfg.Quest (ONTO_QUEST); else a random quest built from the cfg.Objectives pool
// (ONTO_OBJECTIVES), which the 'quest' command can re-roll; else the default
// chain derived from the start coordinate. Both cmd/ entry points share this so
// the CLI and web enable the game identically.
func GameOptions(cfg Config, state State) []facade.Option {
	if !cfg.Game {
		return nil
	}
	budget := cfg.Budget
	if budget <= 0 {
		budget = facade.DefaultBudget
	}
	opts := []facade.Option{facade.WithBudget(budget)}
	switch {
	case len(cfg.Quest) > 0:
		opts = append(opts, facade.WithTargets(cfg.Quest...))
	case len(cfg.Objectives) > 0:
		opts = append(opts, facade.WithObjectivePool(cfg.Objectives...))
	default:
		if loc, ok := state.Universe.GetLocation(state.StartID); ok {
			opts = append(opts, facade.WithTargets(facade.DefaultQuestChain(loc.Coordinate)...))
		}
	}
	return opts
}

// State is the fully-assembled domain state returned by Bootstrap.
type State struct {
	Universe *universe.Aggregate
	Repo     universe.Repository
	StartID  string
}

// Bootstrap loads (or initialises) the universe from disk and returns a State
// ready to be handed to a delivery-mechanism constructor (cli.New, web.Run, …).
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
		// The well is the hand-placed canonical sealed vault: no physical exit, one
		// non-physical drift out. It is the one trap seeded directly into base
		// reality (SelectTrap only spawns traps in nested realities), but it carries
		// the same TrapType so it surfaces identically to a generated sealed vault.
		{ID: "well", Name: "Well", Description: "The bottom of an old stone well. The walls are sheer — there is no walking out of here.", Coordinate: coordFor("Well", base), Trap: universe.TrapSealedVault},
		{ID: "kirkstall-abbey", Name: "Kirkstall Abbey", Description: "The ruins of a Cistercian abbey by the River Aire. The lane ends here among the cloisters.", Coordinate: coordFor("Kirkstall Abbey", base)},
	}
	for _, loc := range locations {
		if err := u.AddLocation(loc); err != nil {
			return nil, err
		}
	}

	// Physical edges are directional (AddEdge stores an edge only under its From
	// node), so every walk/rail leg is paired with its return so the starter world
	// is fully two-way: if you can walk somewhere, you can walk back. The return
	// legs mirror the outbound distance/cost. Auto-generated nearby locations
	// already come with both directions (NewNearbyCluster), and isDeadEnd ignores
	// the edge you arrived on, so a genuine leaf like Kirkstall Abbey — reachable
	// only by a single there-and-back walk — still counts as a dead end (its one
	// physical edge leads back the way you came) yet keeps a physical exit, so
	// travel expands it into a fresh nearby cluster on arrival.
	edges := []universe.EdgeVO{
		{From: "home", To: "station", Mode: universe.Walk, Distance: 1.6, Cost: 1, Description: "Walk to the station"},
		{From: "station", To: "home", Mode: universe.Walk, Distance: 1.6, Cost: 1, Description: "Walk home from the station"},
		{From: "home", To: "park", Mode: universe.Walk, Distance: 0.8, Cost: 1, Description: "Walk to the park"},
		{From: "park", To: "home", Mode: universe.Walk, Distance: 0.8, Cost: 1, Description: "Walk home from the park"},
		{From: "station", To: "city-centre", Mode: universe.Rail, Distance: 2.0, Cost: 3, Description: "Take the rail line"},
		{From: "city-centre", To: "station", Mode: universe.Rail, Distance: 2.0, Cost: 3, Description: "Take the rail line back"},
		{From: "city-centre", To: "home", Mode: universe.Walk, Distance: 2.4, Cost: 2, Description: "Walk home"},
		{From: "home", To: "city-centre", Mode: universe.Walk, Distance: 2.4, Cost: 2, Description: "Walk to the city centre"},
		// Kirkstall Abbey is a genuine leaf dead end: a single there-and-back walk
		// from the city centre along the river, with no onward physical route. On
		// arrival isDeadEnd is true (its only physical edge leads back) while
		// HasPhysicalExit stays true, so travel auto-expands it into a nearby cluster.
		{From: "city-centre", To: "kirkstall-abbey", Mode: universe.Walk, Distance: 3.0, Cost: 2, Description: "Walk the towpath to the abbey"},
		{From: "kirkstall-abbey", To: "city-centre", Mode: universe.Walk, Distance: 3.0, Cost: 2, Description: "Walk back to the city centre"},
		// The well is a genuine physical dead end: you fall in from the park
		// (a one-way physical drop, no walking back up), so travel and physical
		// reachability treat it as a sink and return-home reports no walkable
		// route out. The only way out is contextual — a non-physical drift back to
		// the surface — so the traveller is never truly soft-locked. Because the
		// exit is non-physical, isDeadEnd still sees no onward physical route and a
		// nearby node is generated on arrival, exactly as for any other dead end.
		{From: "park", To: "well", Mode: universe.Walk, Distance: 0.1, Cost: 1, Description: "Fall down the well"},
		{From: "well", To: "park", Mode: universe.ConsensusShift, Cost: universe.ConsensusShiftCost, Description: "Drift out of the well back to the surface"},
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

// Package facade provides the application facade: a single App type that
// dispatches user commands to the application and domain layers and formats
// results as strings. It is delivery-mechanism agnostic — no readline, no I/O,
// no terminal assumptions. CLI, web, and test code all depend on this
// package rather than on each other.
package facade

import (
	"fmt"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// AppVersion is the human-readable application version string, shown at
// startup by every delivery mechanism (CLI, web).
const AppVersion = "Onto Explorer v0.1"

// SessionEntity returns the underlying exploration session.
// Intended for test introspection and web state access.
func (a *App) SessionEntity() *exploration.Entity { return a.session }

// Aggregate returns the underlying universe aggregate.
// Intended for test introspection and web state access.
func (a *App) Aggregate() *universe.Aggregate { return a.univ }

// IsDirty reports whether unsaved mutations exist.
func (a *App) IsDirty() bool { return a.dirty }

// Prompt returns the context-aware CLI prompt string.
func (a *App) Prompt() string {
	addr := a.session.Coordinate().ShortOntoAddress()
	addr = addr[len("onto://"):]
	return fmt.Sprintf("[%s] > ", addr)
}

// PhysicalDestinationIDs returns the IDs of locations reachable by physical
// travel from the current location, in the same physical reality slice.
// Used by the CLI tab-completer.
func (a *App) PhysicalDestinationIDs() []string {
	current := a.session.Coordinate()
	var ids []string
	for _, edge := range a.univ.EdgesFrom(a.session.Location()) {
		if !edge.Mode.IsPhysical() {
			continue
		}
		dest, ok := a.univ.GetLocation(edge.To)
		if ok && current.SamePhysicalReality(dest.Coordinate) {
			ids = append(ids, dest.ID)
		}
	}
	return ids
}

// App is the application facade. It owns the wired-up universe, session,
// repository, and domain services and exposes one method per user command.
// All methods return plain strings; formatting is done here so every delivery
// mechanism (CLI, web) sees the same output without duplication.
type App struct {
	univ              *universe.Aggregate
	session           *exploration.Entity
	repo              universe.Repository
	pathfinder        navigation.PathfinderService
	locationGenerator universe.LocationGeneratorService
	homeID            string
	dirty             bool

	// Game configuration seed, captured at construction and reapplied on Reset so
	// a reset session starts a fresh game. budget of 0 means unlimited (no limit in
	// force — distinct from a finite budget later spent down to nothing, which is
	// still in force but exhausted); see budgetInForce. A non-empty targets chain
	// gates the win condition. These are construction/reset config only — the live
	// game state is the session, so read a.session (not these fields) when
	// reporting current progress or objectives.
	budget  float64
	targets []universe.CoordinateVO
	// objectivePool is the set of candidate objectives from which a random quest
	// chain is built when no explicit targets are configured, and re-rolled by
	// NewQuest. Empty means quests are fixed (targets or the default chain).
	objectivePool []universe.CoordinateVO

	// initialLocations/initialEdges snapshot the universe as it was at
	// construction (startup), so Reset can rebuild the starting map after
	// reality transitions have grown it. LocationEntity and EdgeVO are value
	// types, so these slices are an independent copy of the graph.
	initialLocations []universe.LocationEntity
	initialEdges     []universe.EdgeVO
}

// New assembles an App from already-wired dependencies. Callers (cmd/ entry
// points) provide a loaded universe, a repository, the start location ID, and
// the domain services.
func New(
	u *universe.Aggregate,
	repo universe.Repository,
	startID string,
	pathfinder navigation.PathfinderService,
	gen universe.LocationGeneratorService,
	opts ...Option,
) (*App, error) {
	loc, ok := u.GetLocation(startID)
	if !ok {
		return nil, fmt.Errorf("start location %q not found in universe", startID)
	}
	a := &App{
		univ:              u,
		repo:              repo,
		pathfinder:        pathfinder,
		locationGenerator: gen,
		homeID:            startID,
		initialLocations:  u.AllLocations(),
		initialEdges:      u.AllEdgesFlat(),
	}
	for _, opt := range opts {
		opt(a)
	}
	a.session = a.newSession(startID, loc.Coordinate)
	return a, nil
}

// Reset rebuilds the universe to the state captured at construction (the
// starting map) and returns the session to the start location in base reality,
// discarding every location and edge that reality transitions created this
// session. It marks the app dirty so the cleared map can be saved over the
// grown one.
func (a *App) Reset() string {
	fresh := universe.NewAggregate()
	for _, loc := range a.initialLocations {
		if err := fresh.AddLocation(loc); err != nil {
			return fmt.Sprintf("Failed to reset: %v", err)
		}
	}
	for _, e := range a.initialEdges {
		if err := fresh.AddEdge(e); err != nil {
			return fmt.Sprintf("Failed to reset: %v", err)
		}
	}
	loc, ok := fresh.GetLocation(a.homeID)
	if !ok {
		return fmt.Sprintf("Failed to reset: start location %q missing", a.homeID)
	}
	a.univ = fresh
	a.session = a.newSession(a.homeID, loc.Coordinate)
	a.dirty = true
	return "Map reset to the starting realities."
}

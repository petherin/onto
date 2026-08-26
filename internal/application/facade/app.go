// Package facade provides the application facade: a single App type that
// dispatches user commands to the application and domain layers and formats
// results as strings. It is delivery-mechanism agnostic — no readline, no I/O,
// no terminal assumptions. CLI, TUI, web, and test code all depend on this
// package rather than on each other.
package facade

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/petherin/onto/internal/domain/exploration"
	"github.com/petherin/onto/internal/domain/navigation"
	"github.com/petherin/onto/internal/domain/universe"
)

// AppVersion is the human-readable application version string, shown at
// startup by every delivery mechanism (CLI, TUI, web).
const AppVersion = "Onto Explorer v0.1"

// SessionEntity returns the underlying exploration session.
// Intended for test introspection and TUI/web state access.
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
// mechanism (CLI, TUI, web) sees the same output without duplication.
type App struct {
	univ              *universe.Aggregate
	session           *exploration.Entity
	repo              universe.Repository
	pathfinder        navigation.PathfinderService
	locationGenerator universe.LocationGeneratorService
	homeID            string
	dirty             bool

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
) (*App, error) {
	loc, ok := u.GetLocation(startID)
	if !ok {
		return nil, fmt.Errorf("start location %q not found in universe", startID)
	}
	return &App{
		univ:              u,
		session:           exploration.NewEntity(startID, loc.Coordinate),
		repo:              repo,
		pathfinder:        pathfinder,
		locationGenerator: gen,
		homeID:            startID,
		initialLocations:  u.AllLocations(),
		initialEdges:      u.AllEdgesFlat(),
	}, nil
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
	a.session = exploration.NewEntity(a.homeID, loc.Coordinate)
	a.dirty = true
	return "Map reset to the starting realities."
}

// Execute dispatches a raw input string to the appropriate command handler.
func (a *App) Execute(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	parts := strings.Fields(trimmed)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	if len(parts) == 1 {
		if number, err := strconv.Atoi(cmd); err == nil {
			return a.ExecuteJourney(number)
		}
	}

	switch cmd {
	case "help":
		return a.Help()
	case "where":
		return a.Where()
	case "look":
		return a.Look()
	case "ls":
		return a.List()
	case "route":
		if args == "" {
			return "Usage: route <destination>"
		}
		return a.Route(args)
	case "travel":
		if args == "" {
			return "Usage: travel <destination>"
		}
		return a.Travel(args)
	case "home":
		return a.GoHome()
	case "cost":
		return a.Cost()
	case "shift":
		if args == "back" {
			return a.ShiftBack()
		}
		return a.Shift()
	case "jump":
		if args == "back" {
			return a.JumpBack()
		}
		return a.Jump()
	case "universe":
		if args == "back" {
			return a.UniverseBack()
		}
		return a.Universe()
	case "structure":
		if args == "back" {
			return a.StructureBack()
		}
		return a.Structure()
	case "simulate":
		if args == "back" {
			return a.SimulateBack()
		}
		return a.Simulate()
	case "drift":
		return a.Drift()
	case "align":
		return a.Align()
	case "observe":
		if args == "" {
			return "Usage: observe <observer>"
		}
		if args == "back" {
			return a.ObserveBack()
		}
		return a.Observe(args)
	case "time":
		if args == "" {
			return "Usage: time <RFC3339> or time back"
		}
		if args == "back" {
			return a.TimeBack()
		}
		return a.Time(args)
	case "save":
		if args != "" {
			return "Usage: save"
		}
		msg, err := a.Save()
		if err != nil {
			return err.Error()
		}
		return msg
	case "exit":
		return "Goodbye."
	default:
		if suggestion := a.suggestCommand(cmd); suggestion != "" {
			return fmt.Sprintf("Unknown command: %s\n\nDid you mean '%s'?\n\n%s", cmd, suggestion, a.Help())
		}
		return fmt.Sprintf("Unknown command: %s\n\n%s", cmd, a.Help())
	}
}

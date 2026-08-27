package facade

import (
	"time"

	"github.com/petherin/onto/internal/domain/navigation"
)

// SessionSnapshot is a read-only view of the current session state,
// used by delivery mechanisms (web, TUI) that need structured data. It
// carries every navigable axis so a GUI can render a coordinate HUD without
// re-parsing formatted command output.
type SessionSnapshot struct {
	Location string
	// OntoAddress is the full canonical Onto Address; ShortOntoAddress is the
	// compact form that omits default axes. Both carry the onto:// scheme.
	OntoAddress      string
	ShortOntoAddress string
	Mathematics      string
	Universe         string
	Timeline         string
	Quantum          string
	Simulation       int
	Consensus        int
	Observer         string
	Time             time.Time
	Planet           string
	City             string
	CumulativeCost   float64
	History          []string

	// Game state. HasBudget reports whether a finite spending pool is in force;
	// when true, Budget is the pool and RemainingBudget is what is left.
	// HasTarget reports whether an objective is set; when true, TargetAddress /
	// TargetShortAddress locate it, ReachedTarget marks that it has been
	// visited, and Won marks the objective complete (reached and back home).
	HasBudget          bool
	Budget             float64
	RemainingBudget    float64
	HasTarget          bool
	TargetAddress      string
	TargetShortAddress string
	ReachedTarget      bool
	Won                bool
}

// Snapshot returns a read-only view of the current session state.
func (a *App) Snapshot() SessionSnapshot {
	coord := a.session.Coordinate()
	return SessionSnapshot{
		Location:         a.session.Location(),
		OntoAddress:      coord.OntoAddress(),
		ShortOntoAddress: coord.ShortOntoAddress(),
		Mathematics:      coord.Mathematics,
		Universe:         coord.Universe,
		Timeline:         coord.Timeline,
		Quantum:          coord.Quantum,
		Simulation:       coord.Simulation,
		Consensus:        coord.Consensus,
		Observer:         coord.Observer,
		Time:             coord.Time,
		Planet:           coord.Planet,
		City:             coord.City,
		CumulativeCost:   a.session.CumulativeCost(),
		History:          a.session.History(),

		HasBudget:          a.session.HasBudget(),
		Budget:             a.session.Budget(),
		RemainingBudget:    a.session.RemainingBudget(),
		HasTarget:          a.session.HasTarget(),
		TargetAddress:      a.session.Target().OntoAddress(),
		TargetShortAddress: a.session.Target().ShortOntoAddress(),
		ReachedTarget:      a.session.ReachedTarget(),
		Won:                a.session.Won(),
	}
}

// NodeSnapshot is a read-only view of a single location in the universe graph.
type NodeSnapshot struct {
	ID          string
	Name        string
	Description string
	Mathematics string
	Universe    string
	Timeline    string
	Quantum     string
	Simulation  int
	Consensus   int
	Observer    string
	Planet      string
	City        string
	Location    string
	OntoAddress string
	// Depth is the location's reality nesting depth (see
	// universe.CoordinateVO.NestingDepth): 0 for base reality, growing by one
	// per reality transition away from it. GUIs use it to arrange nested
	// realities by how deep they sit.
	Depth int
	// Reachable reports whether this location can be reached from the current
	// session location using ordinary travel (a physical, same-reality route).
	// Nodes that would need a shift/jump/observe, or that have no path at all,
	// are false. The current location itself is false.
	Reachable bool
}

// EdgeSnapshot is a read-only view of a single directed edge.
type EdgeSnapshot struct {
	From     string
	To       string
	Mode     string
	Cost     float64
	Distance float64
}

// GraphSnapshot is a read-only view of the full universe graph.
type GraphSnapshot struct {
	Nodes []NodeSnapshot
	Edges []EdgeSnapshot
}

// GraphSnapshot returns a read-only view of all locations and edges.
func (a *App) GraphSnapshot() GraphSnapshot {
	locs := a.univ.AllLocations()
	reachable := navigation.ReachableFrom(a.univ, a.session.Location())
	nodes := make([]NodeSnapshot, 0, len(locs))
	for _, loc := range locs {
		nodes = append(nodes, NodeSnapshot{
			ID:          loc.ID,
			Name:        loc.Name,
			Description: loc.Description,
			Mathematics: loc.Coordinate.Mathematics,
			Universe:    loc.Coordinate.Universe,
			Timeline:    loc.Coordinate.Timeline,
			Quantum:     loc.Coordinate.Quantum,
			Simulation:  loc.Coordinate.Simulation,
			Consensus:   loc.Coordinate.Consensus,
			Observer:    loc.Coordinate.Observer,
			Planet:      loc.Coordinate.Planet,
			City:        loc.Coordinate.City,
			Location:    loc.Coordinate.Location,
			OntoAddress: loc.Coordinate.OntoAddress(),
			Depth:       loc.Coordinate.NestingDepth(),
			Reachable:   reachable[loc.ID],
		})
	}
	flat := a.univ.AllEdgesFlat()
	edges := make([]EdgeSnapshot, 0, len(flat))
	for _, e := range flat {
		edges = append(edges, EdgeSnapshot{
			From:     e.From,
			To:       e.To,
			Mode:     string(e.Mode),
			Cost:     e.Cost,
			Distance: e.Distance,
		})
	}
	return GraphSnapshot{Nodes: nodes, Edges: edges}
}

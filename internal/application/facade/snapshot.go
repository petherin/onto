package facade

// SessionSnapshot is a read-only view of the current session state,
// used by delivery mechanisms (web, TUI) that need structured data.
type SessionSnapshot struct {
	Location       string
	OntoAddress    string
	Timeline       string
	Quantum        string
	CumulativeCost float64
	History        []string
}

// Snapshot returns a read-only view of the current session state.
func (a *App) Snapshot() SessionSnapshot {
	coord := a.session.Coordinate()
	return SessionSnapshot{
		Location:       a.session.Location(),
		OntoAddress:    coord.OntoAddress(),
		Timeline:       coord.Timeline,
		Quantum:        coord.Quantum,
		CumulativeCost: a.session.CumulativeCost(),
		History:        a.session.History(),
	}
}

// NodeSnapshot is a read-only view of a single location in the universe graph.
type NodeSnapshot struct {
	ID          string
	Name        string
	Description string
	Timeline    string
	Quantum     string
	Planet      string
	City        string
	Location    string
	OntoAddress string
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
	nodes := make([]NodeSnapshot, 0, len(locs))
	for _, loc := range locs {
		nodes = append(nodes, NodeSnapshot{
			ID:          loc.ID,
			Name:        loc.Name,
			Description: loc.Description,
			Timeline:    loc.Coordinate.Timeline,
			Quantum:     loc.Coordinate.Quantum,
			Planet:      loc.Coordinate.Planet,
			City:        loc.Coordinate.City,
			Location:    loc.Coordinate.Location,
			OntoAddress: loc.Coordinate.OntoAddress(),
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

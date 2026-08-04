package web

import (
	"encoding/json"
	"net/http"
)

// nodeJSON is the wire representation of a location for the 3D graph.
type nodeJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Timeline    string `json:"timeline"`
	Quantum     string `json:"quantum"`
	Planet      string `json:"planet"`
	City        string `json:"city"`
	Location    string `json:"location"`
	OntoAddress string `json:"ontoAddress"`
}

// edgeJSON is the wire representation of a directed graph edge.
type edgeJSON struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Mode     string  `json:"mode"`
	Cost     float64 `json:"cost"`
	Distance float64 `json:"distance"`
}

// universeJSON is the full graph snapshot sent to the frontend.
type universeJSON struct {
	Nodes []nodeJSON `json:"nodes"`
	Edges []edgeJSON `json:"edges"`
}

// stateJSON is the current session state sent to the frontend.
type stateJSON struct {
	Location      string   `json:"location"`
	OntoAddress   string   `json:"ontoAddress"`
	Timeline      string   `json:"timeline"`
	Quantum       string   `json:"quantum"`
	CumulativeCost float64 `json:"cumulativeCost"`
	History       []string `json:"history"`
}

// commandRequest is the body of a POST /api/command request.
type commandRequest struct {
	Command string `json:"command"`
}

// commandResponse is the body of a POST /api/command response.
type commandResponse struct {
	Output  string       `json:"output"`
	State   stateJSON    `json:"state"`
	Universe universeJSON `json:"universe"`
}

func (s *Server) handleUniverse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.buildUniverse())
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.buildState())
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	output := s.app.Execute(req.Command)

	writeJSON(w, commandResponse{
		Output:   output,
		State:    s.buildState(),
		Universe: s.buildUniverse(),
	})
}

func (s *Server) buildState() stateJSON {
	snap := s.app.Snapshot()
	return stateJSON{
		Location:       snap.Location,
		OntoAddress:    snap.OntoAddress,
		Timeline:       snap.Timeline,
		Quantum:        snap.Quantum,
		CumulativeCost: snap.CumulativeCost,
		History:        snap.History,
	}
}

func (s *Server) buildUniverse() universeJSON {
	graph := s.app.GraphSnapshot()
	nodes := make([]nodeJSON, 0, len(graph.Nodes))
	for _, n := range graph.Nodes {
		nodes = append(nodes, nodeJSON{
			ID:          n.ID,
			Name:        n.Name,
			Description: n.Description,
			Timeline:    n.Timeline,
			Quantum:     n.Quantum,
			Planet:      n.Planet,
			City:        n.City,
			Location:    n.Location,
			OntoAddress: n.OntoAddress,
		})
	}
	edges := make([]edgeJSON, 0, len(graph.Edges))
	for _, e := range graph.Edges {
		edges = append(edges, edgeJSON{
			From:     e.From,
			To:       e.To,
			Mode:     e.Mode,
			Cost:     e.Cost,
			Distance: e.Distance,
		})
	}
	return universeJSON{Nodes: nodes, Edges: edges}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// Package web implements an optional browser-based delivery mechanism for the
// Onto application: a small HTTP server that serves a single-page "Reality
// Map" and exposes the application facade over JSON. Like the CLI and TUI
// packages, it delegates all command execution and state to *facade.App and
// contains no domain logic of its own.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/petherin/onto/internal/application/facade"
)

//go:embed static/*
var staticFS embed.FS

const (
	msgGoodbye = "Goodbye."
	cmdHome    = "home"
)

// Server wraps a *facade.App and serialises access to it. Only one browser
// session is expected, but the mutex keeps concurrent requests (and the
// two-step 'home' confirmation) consistent.
type Server struct {
	app *facade.App
	mu  sync.Mutex

	// awaitingHomeConfirm mirrors the CLI/TUI two-step 'home' flow: GoHome
	// shows the plan, and the next 'y' confirms via GoHomeConfirm.
	awaitingHomeConfirm bool
}

// stateDTO is the JSON payload returned to the browser after every request.
type stateDTO struct {
	Version             string                 `json:"version"`
	Prompt              string                 `json:"prompt"`
	Look                string                 `json:"look"`
	Response            string                 `json:"response"`
	Dirty               bool                   `json:"dirty"`
	AwaitingHomeConfirm bool                   `json:"awaitingHomeConfirm"`
	Session             facade.SessionSnapshot `json:"session"`
	Graph               facade.GraphSnapshot   `json:"graph"`
}

// NewServer builds a Server ready to be mounted with Handler.
func NewServer(app *facade.App) *Server { return &Server{app: app} }

// Handler returns the HTTP handler serving the SPA and JSON API.
func (s *Server) Handler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("web: embed static: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/execute", s.handleExecute)
	mux.HandleFunc("/api/save", s.handleSave)
	return mux
}

// Run starts the server on addr and blocks. It saves any unsaved mutations
// best-effort when the process is interrupted (handled by the caller).
func Run(app *facade.App, addr string) error {
	s := NewServer(app)
	log.Printf("Onto web running at %s", browsableURL(addr))
	return http.ListenAndServe(addr, s.Handler())
}

// browsableURL turns a net/http listen address into a clickable URL. A
// host-less address such as ":8090" (or "0.0.0.0:8090") listens on all
// interfaces, but terminals can't linkify it, so we substitute "localhost".
func browsableURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeState(w, "")
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeState(w, s.execute(body.Command))
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.app.SaveIfDirty(); err != nil {
		s.writeState(w, "Warning: failed to save: "+err.Error())
		return
	}
	s.writeState(w, "Saved.")
}

// execute runs one command line, mirroring the TUI's two-step 'home' flow.
// It never terminates the process; an 'exit' command just saves and reports.
func (s *Server) execute(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if s.awaitingHomeConfirm {
		s.awaitingHomeConfirm = false
		if strings.EqualFold(line, "y") {
			return s.app.GoHomeConfirm()
		}
		return "Cancelled."
	}
	fields := strings.Fields(line)
	if fields[0] == cmdHome {
		plan := s.app.GoHome()
		if facade.NeedsHomeConfirm(plan) {
			s.awaitingHomeConfirm = true
		}
		return plan
	}
	response := s.app.Execute(line)
	if response == msgGoodbye {
		if err := s.app.SaveIfDirty(); err != nil {
			return "Warning: failed to save: " + err.Error()
		}
	}
	return response
}

// writeState serialises the current facade state (plus an optional response
// string) to the response writer as JSON. Callers must hold s.mu.
func (s *Server) writeState(w http.ResponseWriter, response string) {
	dto := stateDTO{
		Version:             facade.AppVersion,
		Prompt:              s.app.Prompt(),
		Look:                s.app.Look(),
		Response:            response,
		Dirty:               s.app.IsDirty(),
		AwaitingHomeConfirm: s.awaitingHomeConfirm,
		Session:             s.app.Snapshot(),
		Graph:               s.app.GraphSnapshot(),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dto); err != nil {
		log.Printf("web: encode state: %v", err)
	}
}

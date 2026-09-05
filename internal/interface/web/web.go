// Package web implements an optional browser-based delivery mechanism for the
// Onto application: a small HTTP server that serves a single-page "Reality
// Map" and exposes the application facade over JSON. Like the CLI package, it
// delegates all command execution and state to *facade.App and contains no
// domain logic of its own.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
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

	// awaitingHomeConfirm mirrors the CLI two-step 'home' flow: GoHome
	// shows the plan, and the next 'y' confirms via GoHomeConfirm.
	awaitingHomeConfirm bool
}

// stateDTO is the JSON payload returned to the browser after every request.
type stateDTO struct {
	Version             string                 `json:"version"`
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
	// The API routes live on their own mux so the "/" file server can't shadow
	// them: method-in-pattern routing lets ServeMux answer a wrong-method /api
	// request with 405, but only when no catch-all (the file server) also
	// matches the path — hence the dedicated sub-mux mounted at "/api/".
	api := http.NewServeMux()
	api.HandleFunc("GET /api/state", s.handleState)
	api.HandleFunc("POST /api/execute", s.handleExecute)
	api.HandleFunc("POST /api/save", s.handleSave)
	api.HandleFunc("POST /api/reset", s.handleReset)

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.Handle("/api/", api)
	return corsMiddleware(mux)
}

// corsMiddleware adds CORS headers so the SPA can call this API from a different
// origin. In the split MiniStack layout the SPA is served from S3 (onto.world)
// while the API runs on ECS behind an ALB (api.onto.world), so the browser makes
// cross-origin requests and needs these headers (and a preflight answer for the
// JSON POSTs). The allowed origin defaults to "*" and can be pinned to a single
// origin with ONTO_ALLOWED_ORIGIN. Same-origin dev (make web) is unaffected.
func corsMiddleware(next http.Handler) http.Handler {
	origin := os.Getenv("ONTO_ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Add("Vary", "Origin")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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

func (s *Server) handleSave(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.app.SaveIfDirty(); err != nil {
		s.writeState(w, "Warning: failed to save: "+err.Error())
		return
	}
	s.writeState(w, "Saved.")
}

// handleReset performs a full server-side reset back to the starting map,
// discarding every branch reality transitions created. It also clears any
// pending home confirmation, since the session is being returned to base.
func (s *Server) handleReset(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.awaitingHomeConfirm = false
	s.writeState(w, s.app.Reset())
}

// execute runs one command line, mirroring the CLI's two-step 'home' flow.
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

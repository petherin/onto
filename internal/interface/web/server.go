// Package web is the HTTP delivery mechanism for the Onto application.
// It wraps the cli.App in a JSON API and serves the Three.js frontend.
// The domain and application layers are unaware of this package.
package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/petherin/onto/internal/interface/cli"
)

//go:embed static
var staticFiles embed.FS

// Server owns the HTTP mux and the wrapped application.
type Server struct {
	app *cli.App
	mux *http.ServeMux
}

// NewServer creates a Server backed by the given App and registers all routes.
func NewServer(app *cli.App) *Server {
	s := &Server{
		app: app,
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Static frontend — strip the leading "static/" from the embedded path.
	sub, _ := fs.Sub(staticFiles, "static")
	s.mux.Handle("/", http.FileServer(http.FS(sub)))

	// JSON API
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/universe", s.handleUniverse)
	s.mux.HandleFunc("/api/command", s.handleCommand)
}

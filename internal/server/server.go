// Package server exposes the orgo HTTP API (see docs/DESIGN.md) and serves
// the embedded single-page frontend. It holds an atomically-swappable
// reference to the current roam.Index snapshot; internal/watch (via main)
// calls SetIndex after every re-index, and the SSE hub notifies connected
// browsers to reload.
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/naok-000/orgo/internal/roam"
)

// Server implements http.Handler for the full orgo API + static frontend.
type Server struct {
	idx     atomic.Pointer[roam.Index]
	version string
	hub     *sseHub
	mux     *http.ServeMux
}

// New builds a Server serving the given initial index snapshot.
func New(initial *roam.Index, version string) *Server {
	s := &Server{version: version, hub: newSSEHub()}
	s.idx.Store(initial)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/meta", s.handleMeta)
	mux.HandleFunc("GET /api/graph", s.handleGraph)
	mux.HandleFunc("GET /api/notes", s.handleNotes)
	mux.HandleFunc("GET /api/note/{id}", s.handleNote)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.Handle("/", http.FileServer(http.FS(spaFS{distSub()})))

	s.mux = mux
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Index returns the currently active index snapshot.
func (s *Server) Index() *roam.Index {
	return s.idx.Load()
}

// SetIndex swaps in a new index snapshot and notifies every connected
// /api/events client to reload.
func (s *Server) SetIndex(idx *roam.Index) {
	s.idx.Store(idx)
	s.hub.Broadcast()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("server: encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// nonNil returns s unchanged, except that a nil slice becomes an empty one
// so it marshals as JSON "[]" rather than "null".
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Package server exposes the orgo HTTP API (see docs/DESIGN.md) and serves
// the embedded single-page frontend. It holds an atomically-swappable
// reference to the current roam.Index snapshot; internal/watch (via main)
// calls SetIndex after every re-index, and the SSE hub notifies connected
// browsers to reload.
package server

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/naok-000/orgo/internal/roam"
)

// Server implements http.Handler for the full orgo API + static frontend.
type Server struct {
	idx     atomic.Pointer[roam.Index]
	version string
	hub     *sseHub
	mux     *http.ServeMux

	// allowedHosts, when non-nil, is the set of Host header values
	// (lowercased) requests must carry; anything else is rejected with 403.
	// See RestrictHost. It is written once before the server starts
	// handling requests and only read afterwards, so it needs no locking.
	allowedHosts map[string]bool
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

// RestrictHost enables Host-header validation (DNS-rebinding protection)
// when addr — the address the server was asked to bind — is a loopback
// address. A malicious web page can point an attacker-controlled DNS name
// at 127.0.0.1 and read a loopback-only service cross-origin; validating
// the Host header defeats that. Allowed values are "localhost",
// "127.0.0.1", "[::1]", and the configured address itself, each with or
// without the ":port" suffix. For non-loopback addresses (an explicitly
// exposed server) no restriction is applied. Must be called before the
// server starts handling requests.
func (s *Server) RestrictHost(addr string, port int) {
	if !isLoopbackAddr(addr) {
		return
	}
	names := map[string]bool{
		"localhost":          true,
		"127.0.0.1":          true,
		"[::1]":              true,
		hostHeaderForm(addr): true,
	}
	allowed := make(map[string]bool, 2*len(names))
	p := strconv.Itoa(port)
	for name := range names {
		allowed[name] = true
		allowed[name+":"+p] = true
	}
	s.allowedHosts = allowed
}

func isLoopbackAddr(addr string) bool {
	if strings.EqualFold(addr, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(addr, "[]"))
	return ip != nil && ip.IsLoopback()
}

// hostHeaderForm converts a configured bind address into the form it takes
// in a Host header: lowercased, and bracketed if it is an IPv6 literal.
func hostHeaderForm(addr string) string {
	h := strings.ToLower(strings.Trim(addr, "[]"))
	if ip := net.ParseIP(h); ip != nil && ip.To4() == nil {
		return "[" + h + "]"
	}
	return h
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.allowedHosts != nil && !s.allowedHosts[strings.ToLower(r.Host)] {
		writeError(w, http.StatusForbidden, "unexpected Host header")
		return
	}
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

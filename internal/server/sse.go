package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const heartbeatInterval = 25 * time.Second

// sseHub fans out "reload" notifications to every connected /api/events
// client.
type sseHub struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{subs: make(map[chan struct{}]struct{})}
}

// subscribe registers a new listener and returns a channel that receives a
// value on every Broadcast, plus a function to unregister it.
func (h *sseHub) subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
	return ch, cancel
}

// Broadcast notifies every connected listener. Listeners that already have a
// pending notification are left alone (coalescing is fine: a reload is a
// reload).
func (h *sseHub) Broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, cancel := s.hub.subscribe()
	defer cancel()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			if _, err := fmt.Fprint(w, "event: reload\ndata: {}\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

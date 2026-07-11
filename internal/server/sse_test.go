package server

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/naok-000/orgo/internal/roam"
)

// TestSSEReloadOnSetIndex drives the SSE endpoint through a real HTTP
// server (httptest.ResponseRecorder doesn't support the streaming/flushing
// semantics SSE needs), connects a client, calls SetIndex to simulate a
// watcher-triggered re-index, and asserts a "reload" event arrives.
func TestSSEReloadOnSetIndex(t *testing.T) {
	s := New(testIndex(t), "v")
	httpSrv := httptest.NewServer(s)
	defer httpSrv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpSrv.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	// Give the handler a moment to register its subscription before we
	// broadcast, then trigger a reindex.
	time.Sleep(50 * time.Millisecond)
	s.SetIndex(roam.NewIndex("/somewhere", nil))

	reader := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(4 * time.Second)
	var sawEvent, sawData bool
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "event: reload" {
			sawEvent = true
		}
		if sawEvent && line == "data: {}" {
			sawData = true
			break
		}
	}
	if !sawEvent || !sawData {
		t.Fatalf("did not observe a full reload SSE event (event=%v data=%v)", sawEvent, sawData)
	}
}

func TestSSEHandlesClientDisconnect(t *testing.T) {
	s := New(testIndex(t), "v")
	httpSrv := httptest.NewServer(s)
	defer httpSrv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpSrv.URL+"/api/events", nil)
	if err != nil {
		cancel()
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("Do: %v", err)
	}

	// Simulate a client disconnect. This should not hang the server or leak
	// the subscription forever; a subsequent broadcast must not panic.
	resp.Body.Close()
	cancel()

	time.Sleep(100 * time.Millisecond)
	s.SetIndex(roam.NewIndex("/somewhere-else", nil)) // must not panic
}

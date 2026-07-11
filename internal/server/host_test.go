package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func doRequestWithHost(t *testing.T, s *Server, target, host string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestRestrictHostAllowsLoopbackNames(t *testing.T) {
	s := New(testIndex(t), "v")
	s.RestrictHost("127.0.0.1", 35911)

	good := []string{
		"localhost",
		"localhost:35911",
		"127.0.0.1",
		"127.0.0.1:35911",
		"[::1]",
		"[::1]:35911",
		"LOCALHOST:35911", // Host headers are case-insensitive
	}
	for _, host := range good {
		rec := doRequestWithHost(t, s, "/api/meta", host)
		if rec.Code != http.StatusOK {
			t.Errorf("Host %q: status = %d, want 200", host, rec.Code)
		}
	}
}

func TestRestrictHostRejectsForeignHosts(t *testing.T) {
	s := New(testIndex(t), "v")
	s.RestrictHost("127.0.0.1", 35911)

	bad := []string{
		"evil.example:35911", // classic DNS-rebinding Host
		"evil.example",
		"localhost.evil.example:35911",
		"localhost:9999", // wrong port
		"",
	}
	for _, host := range bad {
		rec := doRequestWithHost(t, s, "/api/meta", host)
		if rec.Code != http.StatusForbidden {
			t.Errorf("Host %q: status = %d, want 403", host, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Host %q: Content-Type = %q, want application/json", host, ct)
		}
	}
}

// TestRestrictHostAppliesToAllRoutes: the check must cover static/SPA
// routes too, not just /api.
func TestRestrictHostAppliesToAllRoutes(t *testing.T) {
	s := New(testIndex(t), "v")
	s.RestrictHost("127.0.0.1", 35911)

	for _, target := range []string{"/", "/graph", "/api/notes", "/api/does-not-exist"} {
		rec := doRequestWithHost(t, s, target, "evil.example:35911")
		if rec.Code != http.StatusForbidden {
			t.Errorf("target %q: status = %d, want 403", target, rec.Code)
		}
	}
	// And legitimate hosts still reach those routes.
	rec := doRequestWithHost(t, s, "/", "localhost:35911")
	if rec.Code != http.StatusOK {
		t.Errorf("static route with good host: status = %d, want 200", rec.Code)
	}
}

func TestRestrictHostDisabledForNonLoopbackAddr(t *testing.T) {
	s := New(testIndex(t), "v")
	s.RestrictHost("0.0.0.0", 35911) // deliberately exposed: no restriction

	rec := doRequestWithHost(t, s, "/api/meta", "some.lan.name:35911")
	if rec.Code != http.StatusOK {
		t.Errorf("non-loopback bind should not restrict Host, got %d", rec.Code)
	}
}

func TestRestrictHostIPv6ConfiguredAddr(t *testing.T) {
	s := New(testIndex(t), "v")
	s.RestrictHost("::1", 35911)

	rec := doRequestWithHost(t, s, "/api/meta", "[::1]:35911")
	if rec.Code != http.StatusOK {
		t.Errorf("IPv6 loopback Host should be allowed, got %d", rec.Code)
	}
	rec = doRequestWithHost(t, s, "/api/meta", "evil.example:35911")
	if rec.Code != http.StatusForbidden {
		t.Errorf("foreign Host should be rejected for ::1 bind, got %d", rec.Code)
	}
}

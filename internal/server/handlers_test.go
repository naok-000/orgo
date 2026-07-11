package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/naok-000/orgo/internal/roam"
)

func testIndex(t *testing.T) *roam.Index {
	t.Helper()
	idx, err := roam.Scan("../../testdata/notes")
	if err != nil {
		t.Fatalf("roam.Scan: %v", err)
	}
	return idx
}

func doRequest(t *testing.T, s *Server, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode JSON: %v\nbody: %s", err, rec.Body.String())
	}
	return v
}

func TestHandleMeta(t *testing.T) {
	s := New(testIndex(t), "test-version")
	rec := doRequest(t, s, "GET", "/api/meta")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}

	meta := decodeJSON[metaResponse](t, rec)
	if meta.NoteCount != 9 {
		t.Errorf("noteCount = %d, want 9", meta.NoteCount)
	}
	if meta.Version != "test-version" {
		t.Errorf("version = %q, want test-version", meta.Version)
	}
	if meta.WorkspaceID == "" {
		t.Error("workspaceId should not be empty")
	}
	if meta.Root == "" {
		t.Error("root should not be empty")
	}
	if meta.Diagnostics.DeadLinkCount != 1 {
		t.Errorf("deadLinkCount = %d, want 1", meta.Diagnostics.DeadLinkCount)
	}
	if meta.Diagnostics.DuplicateIDs == nil {
		t.Error("duplicateIds should marshal as [] not null")
	}
}

func TestHandleGraph(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/graph")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	graph := decodeJSON[graphResponse](t, rec)
	if len(graph.Nodes) != 9 {
		t.Errorf("nodes = %d, want 9", len(graph.Nodes))
	}

	// The dead link (orgo-project.org -> 99999999-...) must not appear as an
	// edge; both endpoints must exist for every edge.
	nodeIDs := map[string]bool{}
	for _, n := range graph.Nodes {
		nodeIDs[n.ID] = true
	}
	for _, e := range graph.Edges {
		if !nodeIDs[e.Source] || !nodeIDs[e.Target] {
			t.Errorf("edge %+v references a node not present in nodes", e)
		}
		if e.Source == "99999999-9999-4999-8999-999999999999" || e.Target == "99999999-9999-4999-8999-999999999999" {
			t.Errorf("dead link must not appear as an edge: %+v", e)
		}
	}

	// Elisp heading node: level 1, "tools" tag inherited.
	var elisp *graphNode
	for i := range graph.Nodes {
		if graph.Nodes[i].ID == "66666666-6666-4666-8666-666666666666" {
			elisp = &graph.Nodes[i]
		}
	}
	if elisp == nil {
		t.Fatal("missing Elisp node in graph")
	}
	if elisp.Level != 1 {
		t.Errorf("Elisp level = %d, want 1", elisp.Level)
	}
	found := false
	for _, tag := range elisp.Tags {
		if tag == "tools" {
			found = true
		}
	}
	if !found {
		t.Errorf("Elisp tags = %v, want to include tools", elisp.Tags)
	}
}

func TestHandleNotesSortedByTitle(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/notes")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	notes := decodeJSON[[]noteSummary](t, rec)
	if len(notes) != 9 {
		t.Fatalf("notes = %d, want 9", len(notes))
	}
	for i := 1; i < len(notes); i++ {
		if notes[i-1].Title > notes[i].Title {
			t.Errorf("notes not sorted by title: %q before %q", notes[i-1].Title, notes[i].Title)
		}
	}
}

func TestHandleNoteSuccess(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/note/44444444-4444-4444-8444-444444444444")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	note := decodeJSON[noteDetail](t, rec)
	if note.Title != "org-roam" {
		t.Errorf("title = %q, want org-roam", note.Title)
	}
	wantAliases := []string{"roam", "my zettelkasten tool"}
	if len(note.Aliases) != len(wantAliases) {
		t.Fatalf("aliases = %v, want %v", note.Aliases, wantAliases)
	}
	for i, a := range wantAliases {
		if note.Aliases[i] != a {
			t.Errorf("aliases[%d] = %q, want %q", i, note.Aliases[i], a)
		}
	}
	if note.HTML == "" {
		t.Error("html should not be empty")
	}
	// org-roam.org is linked to by emacs.org, programming.org, orgo-project.org, zettelkasten.org.
	if len(note.Backlinks) < 3 {
		t.Errorf("backlinks = %v, expected several", note.Backlinks)
	}
}

func TestHandleNoteHeadingSubtree(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/note/66666666-6666-4666-8666-666666666666")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	note := decodeJSON[noteDetail](t, rec)
	if note.Title != "Elisp" {
		t.Errorf("title = %q, want Elisp", note.Title)
	}
	if note.Level != 1 {
		t.Errorf("level = %d, want 1", note.Level)
	}
}

func TestHandleNoteNotFound(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/note/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	errBody := decodeJSON[map[string]string](t, rec)
	if errBody["error"] == "" {
		t.Error("expected non-empty error message")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandleSearch(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/search?q=zettelkasten")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	results := decodeJSON[[]searchResult](t, rec)
	if len(results) == 0 {
		t.Fatal("expected at least one search result")
	}
	found := false
	for _, r := range results {
		if r.ID == "77777777-7777-4777-8777-777777777777" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Zettelkasten note in results: %+v", results)
	}
}

func TestHandleSearchEmptyQueryIs400(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/search?q=")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	errBody := decodeJSON[map[string]string](t, rec)
	if errBody["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestHandleSearchMissingQueryIs400(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/search")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUnknownAPIPathIs404JSON(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/api/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestStaticServesIndexAtRoot(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestStaticFallsBackToIndexForUnknownPath(t *testing.T) {
	s := New(testIndex(t), "v")
	rec := doRequest(t, s, "GET", "/graph")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (SPA fallback)", rec.Code)
	}
}

func TestSetIndexSwapsSnapshot(t *testing.T) {
	s := New(testIndex(t), "v")
	before := s.Index().NoteCount()

	empty := roam.NewIndex("/nowhere", nil)
	s.SetIndex(empty)

	if got := s.Index().NoteCount(); got != 0 {
		t.Errorf("noteCount after SetIndex = %d, want 0", got)
	}
	if before == 0 {
		t.Fatal("test setup issue: expected a nonzero initial note count")
	}
}

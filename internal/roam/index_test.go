package roam

import (
	"testing"
	"time"
)

func TestNewIndexDuplicateIDResolvesToSmallestPath(t *testing.T) {
	now := time.Now()
	winner := &Node{ID: "dup", Title: "B wins", File: "b.org", Pos: 1, ModTime: now}
	loser := &Node{ID: "dup", Title: "Z loses", File: "z.org", Pos: 1, ModTime: now}
	idx := NewIndex("/root", []*Node{loser, winner})

	got, ok := idx.Node("dup")
	if !ok {
		t.Fatalf("expected node dup to exist")
	}
	if got.File != "b.org" {
		t.Errorf("winner file = %q, want b.org (lexicographically smallest path)", got.File)
	}
	if len(idx.Diagnostics().DuplicateIDs) != 1 || idx.Diagnostics().DuplicateIDs[0] != "dup" {
		t.Errorf("duplicateIds = %v, want [dup]", idx.Diagnostics().DuplicateIDs)
	}
	if idx.NoteCount() != 1 {
		t.Errorf("noteCount = %d, want 1", idx.NoteCount())
	}
}

func TestNewIndexDuplicateIDSamePathBreaksTiesByPosition(t *testing.T) {
	now := time.Now()
	first := &Node{ID: "dup", Title: "first", File: "a.org", Pos: 5, ModTime: now}
	second := &Node{ID: "dup", Title: "second", File: "a.org", Pos: 50, ModTime: now}
	idx := NewIndex("/root", []*Node{second, first})

	got, _ := idx.Node("dup")
	if got.Title != "first" {
		t.Errorf("winner = %q, want %q (smaller position)", got.Title, "first")
	}
}

func TestNewIndexDeadLinksExcludedFromEdgesAndCounted(t *testing.T) {
	now := time.Now()
	a := &Node{ID: "a", Title: "A", File: "a.org", ModTime: now, Links: []string{"b", "ghost"}}
	b := &Node{ID: "b", Title: "B", File: "b.org", ModTime: now}
	idx := NewIndex("/root", []*Node{a, b})

	edges := idx.Edges()
	if len(edges) != 1 || edges[0] != (Edge{Source: "a", Target: "b"}) {
		t.Errorf("edges = %v, want a single a->b edge", edges)
	}
	if idx.Diagnostics().DeadLinkCount != 1 {
		t.Errorf("deadLinkCount = %d, want 1", idx.Diagnostics().DeadLinkCount)
	}
}

func TestNewIndexEdgesDeduplicated(t *testing.T) {
	now := time.Now()
	a := &Node{ID: "a", Title: "A", File: "a.org", ModTime: now, Links: []string{"b", "b", "b"}}
	b := &Node{ID: "b", Title: "B", File: "b.org", ModTime: now}
	idx := NewIndex("/root", []*Node{a, b})

	if len(idx.Edges()) != 1 {
		t.Errorf("edges = %v, want exactly 1 deduplicated edge", idx.Edges())
	}
	if idx.Degree("a") != 1 || idx.Degree("b") != 1 {
		t.Errorf("degree a=%d b=%d, want 1/1 (deduplicated)", idx.Degree("a"), idx.Degree("b"))
	}
}

func TestNewIndexDegreeCountsLinksAndBacklinks(t *testing.T) {
	now := time.Now()
	a := &Node{ID: "a", Title: "A", File: "a.org", ModTime: now, Links: []string{"b"}}
	b := &Node{ID: "b", Title: "B", File: "b.org", ModTime: now, Links: []string{"c"}}
	c := &Node{ID: "c", Title: "C", File: "c.org", ModTime: now}
	idx := NewIndex("/root", []*Node{a, b, c})

	// b has one outgoing (b->c) and one incoming (a->b) edge.
	if got := idx.Degree("b"); got != 2 {
		t.Errorf("degree(b) = %d, want 2", got)
	}
	if got := idx.Degree("a"); got != 1 {
		t.Errorf("degree(a) = %d, want 1", got)
	}
	orphan := &Node{ID: "d", Title: "D", File: "d.org", ModTime: now}
	idx2 := NewIndex("/root", []*Node{a, b, c, orphan})
	if got := idx2.Degree("d"); got != 0 {
		t.Errorf("degree(orphan) = %d, want 0", got)
	}
}

func TestNewIndexBacklinksSortedByTitle(t *testing.T) {
	now := time.Now()
	target := &Node{ID: "t", Title: "Target", File: "t.org", ModTime: now}
	z := &Node{ID: "z", Title: "Zeta", File: "z.org", ModTime: now, Links: []string{"t"}}
	a := &Node{ID: "y", Title: "Alpha", File: "y.org", ModTime: now, Links: []string{"t"}}
	idx := NewIndex("/root", []*Node{target, z, a})

	bl := idx.Backlinks("t")
	if len(bl) != 2 {
		t.Fatalf("expected 2 backlinks, got %d", len(bl))
	}
	if bl[0].Title != "Alpha" || bl[1].Title != "Zeta" {
		t.Errorf("backlinks = [%s, %s], want [Alpha, Zeta]", bl[0].Title, bl[1].Title)
	}
}

func TestNewIndexNoDuplicatesYieldsEmptyDiagnostics(t *testing.T) {
	now := time.Now()
	a := &Node{ID: "a", Title: "A", File: "a.org", ModTime: now}
	idx := NewIndex("/root", []*Node{a})
	if len(idx.Diagnostics().DuplicateIDs) != 0 {
		t.Errorf("duplicateIds = %v, want empty", idx.Diagnostics().DuplicateIDs)
	}
	if idx.Diagnostics().DeadLinkCount != 0 {
		t.Errorf("deadLinkCount = %d, want 0", idx.Diagnostics().DeadLinkCount)
	}
}

func TestScanFullCorpus(t *testing.T) {
	idx, err := Scan("../../testdata/notes")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if got, want := idx.NoteCount(), 9; got != want {
		t.Fatalf("noteCount = %d, want %d", got, want)
	}

	expectedFileIDs := map[string]string{
		"11111111-1111-4111-8111-111111111111": "programming.org",
		"22222222-2222-4222-8222-222222222222": "go.org",
		"33333333-3333-4333-8333-333333333333": "emacs.org",
		"44444444-4444-4444-8444-444444444444": "org-roam.org",
		"55555555-5555-4555-8555-555555555555": "orgo-project.org",
		"77777777-7777-4777-8777-777777777777": "zettelkasten.org",
		"88888888-8888-4888-8888-888888888888": "orphan.org",
		"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa": "daily/2026-07-11.org",
	}
	for id, file := range expectedFileIDs {
		nd, ok := idx.Node(id)
		if !ok {
			t.Errorf("missing expected node %s (%s)", id, file)
			continue
		}
		if nd.File != file {
			t.Errorf("node %s file = %q, want %q", id, nd.File, file)
		}
		if nd.Level != 0 {
			t.Errorf("node %s level = %d, want 0 (file node)", id, nd.Level)
		}
	}

	elisp, ok := idx.Node("66666666-6666-4666-8666-666666666666")
	if !ok {
		t.Fatal("missing Elisp heading node")
	}
	if elisp.File != "emacs.org" {
		t.Errorf("Elisp file = %q, want emacs.org", elisp.File)
	}
	if elisp.Level != 1 {
		t.Errorf("Elisp level = %d, want 1", elisp.Level)
	}
	if elisp.Title != "Elisp" {
		t.Errorf("Elisp title = %q, want Elisp", elisp.Title)
	}
	found := false
	for _, tag := range elisp.Tags {
		if tag == "tools" {
			found = true
		}
	}
	if !found {
		t.Errorf("Elisp tags = %v, want to include inherited filetag \"tools\"", elisp.Tags)
	}
	if len(elisp.Links) != 1 || elisp.Links[0] != "33333333-3333-4333-8333-333333333333" {
		t.Errorf("Elisp links = %v, want self-link to Emacs attributed here, not the file node", elisp.Links)
	}

	emacsFile, _ := idx.Node("33333333-3333-4333-8333-333333333333")
	if len(emacsFile.Links) != 1 || emacsFile.Links[0] != "44444444-4444-4444-8444-444444444444" {
		t.Errorf("emacs.org file node links = %v, want just [44444444-...] (org-roam); the Elisp self-link must not appear here", emacsFile.Links)
	}

	// not-a-note.org has no :ID: and must be skipped entirely.
	for _, nd := range idx.Nodes() {
		if nd.File == "not-a-note.org" {
			t.Errorf("not-a-note.org should be skipped entirely, found node %+v", nd)
		}
	}

	orgRoam, ok := idx.Node("44444444-4444-4444-8444-444444444444")
	if !ok {
		t.Fatal("missing org-roam.org node")
	}
	wantAliases := []string{"roam", "my zettelkasten tool"}
	if !stringSlicesEqual(orgRoam.Aliases, wantAliases) {
		t.Errorf("org-roam.org aliases = %v, want %v", orgRoam.Aliases, wantAliases)
	}

	zettel, ok := idx.Node("77777777-7777-4777-8777-777777777777")
	if !ok {
		t.Fatal("missing zettelkasten.org node")
	}
	if len(zettel.Refs) != 1 || zettel.Refs[0] != "https://en.wikipedia.org/wiki/Zettelkasten" {
		t.Errorf("zettelkasten.org refs = %v", zettel.Refs)
	}

	// orgo-project.org links to a nonexistent id 99999999-...: excluded from
	// the graph, counted as a diagnostic.
	if idx.Diagnostics().DeadLinkCount != 1 {
		t.Errorf("deadLinkCount = %d, want 1", idx.Diagnostics().DeadLinkCount)
	}
	for _, e := range idx.Edges() {
		if e.Target == "99999999-9999-4999-8999-999999999999" || e.Source == "99999999-9999-4999-8999-999999999999" {
			t.Errorf("dead link target must not appear in edges: %+v", e)
		}
	}

	if len(idx.Diagnostics().DuplicateIDs) != 0 {
		t.Errorf("duplicateIds = %v, want none in this corpus", idx.Diagnostics().DuplicateIDs)
	}

	daily, ok := idx.Node("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	if !ok {
		t.Fatal("missing daily/2026-07-11.org node")
	}
	wantLinks := map[string]bool{
		"55555555-5555-4555-8555-555555555555": false,
		"66666666-6666-4666-8666-666666666666": false,
	}
	for _, l := range daily.Links {
		if _, ok := wantLinks[l]; ok {
			wantLinks[l] = true
		}
	}
	for id, seen := range wantLinks {
		if !seen {
			t.Errorf("daily note should link to %s", id)
		}
	}
}

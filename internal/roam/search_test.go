package roam

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSearchMatchesTitleAliasAndBody(t *testing.T) {
	now := time.Now()
	nodes := []*Node{
		{ID: "1", Title: "Zettelkasten", File: "z.org", ModTime: now, Body: "A method for notes."},
		{ID: "2", Title: "Something else", Aliases: []string{"roam", "my zettelkasten tool"}, File: "o.org", ModTime: now, Body: "Unrelated body."},
		{ID: "3", Title: "Unrelated", File: "u.org", ModTime: now, Body: "This mentions zettelkasten in passing, deep inside the body text."},
		{ID: "4", Title: "No match", File: "n.org", ModTime: now, Body: "Nothing relevant here."},
	}
	idx := NewIndex("/root", nodes)

	results := idx.Search("zettelkasten", 50)
	gotIDs := map[string]bool{}
	for _, r := range results {
		gotIDs[r.ID] = true
	}
	for _, want := range []string{"1", "2", "3"} {
		if !gotIDs[want] {
			t.Errorf("expected node %s in results, got %+v", want, results)
		}
	}
	if gotIDs["4"] {
		t.Errorf("node 4 should not match, got %+v", results)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	now := time.Now()
	nodes := []*Node{{ID: "1", Title: "Programming", File: "p.org", ModTime: now, Body: "Go is fun."}}
	idx := NewIndex("/root", nodes)

	for _, q := range []string{"PROGRAMMING", "programming", "PrOgRaMmInG", "GO IS FUN"} {
		if len(idx.Search(q, 50)) != 1 {
			t.Errorf("query %q should match, got %v", q, idx.Search(q, 50))
		}
	}
}

func TestSearchSnippetAroundBodyMatch(t *testing.T) {
	now := time.Now()
	body := strings.Repeat("padding ", 30) + "the quick brown fox jumps over the lazy dog" + strings.Repeat(" more", 30)
	nodes := []*Node{{ID: "1", Title: "Fox note", File: "f.org", ModTime: now, Body: body}}
	idx := NewIndex("/root", nodes)

	results := idx.Search("quick brown fox", 50)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	snip := results[0].Snippet
	if snip == "" {
		t.Fatal("expected a non-empty snippet")
	}
	if !strings.Contains(snip, "quick brown fox") {
		t.Errorf("snippet = %q, want it to contain the match", snip)
	}
	if len(snip) > 200 {
		t.Errorf("snippet too long (%d chars): %q", len(snip), snip)
	}
}

func TestSearchSnippetEmptyWhenOnlyTitleOrAliasMatches(t *testing.T) {
	now := time.Now()
	nodes := []*Node{{ID: "1", Title: "Unique Title Word", File: "f.org", ModTime: now, Body: "Body has nothing to do with the query."}}
	idx := NewIndex("/root", nodes)

	results := idx.Search("Unique Title Word", 50)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Snippet != "" {
		t.Errorf("snippet = %q, want empty (no body match)", results[0].Snippet)
	}
}

func TestSearchCapsAt50(t *testing.T) {
	now := time.Now()
	var nodes []*Node
	for i := 0; i < 75; i++ {
		nodes = append(nodes, &Node{
			ID:      fmt.Sprintf("id-%02d", i),
			Title:   fmt.Sprintf("Match %02d", i),
			File:    fmt.Sprintf("f%02d.org", i),
			ModTime: now,
		})
	}
	idx := NewIndex("/root", nodes)

	results := idx.Search("Match", 50)
	if len(results) != 50 {
		t.Fatalf("expected 50 results (capped), got %d", len(results))
	}
}

func TestSearchEmptyQueryReturnsNoResults(t *testing.T) {
	now := time.Now()
	idx := NewIndex("/root", []*Node{{ID: "1", Title: "Anything", File: "f.org", ModTime: now}})
	if got := idx.Search("", 50); len(got) != 0 {
		t.Errorf("expected no results for empty query, got %v", got)
	}
}

func TestSearchResultsSortedByTitle(t *testing.T) {
	now := time.Now()
	nodes := []*Node{
		{ID: "1", Title: "Zeta match", File: "z.org", ModTime: now},
		{ID: "2", Title: "Alpha match", File: "a.org", ModTime: now},
		{ID: "3", Title: "Mid match", File: "m.org", ModTime: now},
	}
	idx := NewIndex("/root", nodes)
	results := idx.Search("match", 50)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Alpha match" || results[1].Title != "Mid match" || results[2].Title != "Zeta match" {
		t.Errorf("results not sorted by title: %+v", results)
	}
}

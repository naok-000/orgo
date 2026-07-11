package roam

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// SearchResult is a single /api/search hit.
type SearchResult struct {
	ID      string
	Title   string
	Snippet string
}

const snippetRadius = 75 // ~150 chars total around the match

var whitespaceRe = regexp.MustCompile(`\s+`)

// Search performs a case-insensitive substring search over every node's
// title, aliases, and body (Node.Body: its org source with drawers and
// #+keyword: lines stripped), returning at most limit results sorted by
// title. Callers are expected to reject an empty query before calling
// Search (Search itself just returns no results for one).
func (idx *Index) Search(q string, limit int) []SearchResult {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	lowerQ := strings.ToLower(q)

	var results []SearchResult
	for _, nd := range idx.nodes {
		matched := strings.Contains(strings.ToLower(nd.Title), lowerQ)
		if !matched {
			for _, a := range nd.Aliases {
				if strings.Contains(strings.ToLower(a), lowerQ) {
					matched = true
					break
				}
			}
		}
		snip := snippet(nd.Body, lowerQ)
		if !matched && snip == "" {
			continue
		}
		results = append(results, SearchResult{ID: nd.ID, Title: nd.Title, Snippet: snip})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Title != results[j].Title {
			return results[i].Title < results[j].Title
		}
		return results[i].ID < results[j].ID
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// snippet returns a ~150 character window of body around the first
// case-insensitive occurrence of lowerQ, or "" if body doesn't contain it.
// Whitespace (including newlines and org markup spacing) is collapsed for
// readability; the raw org markup itself is not stripped.
func snippet(body, lowerQ string) string {
	lowerBody := strings.ToLower(body)
	idx := strings.Index(lowerBody, lowerQ)
	if idx == -1 {
		return ""
	}
	start := prevRuneStart(body, max(0, idx-snippetRadius))
	end := nextRuneStart(body, min(len(body), idx+len(lowerQ)+snippetRadius))

	s := whitespaceRe.ReplaceAllString(strings.TrimSpace(body[start:end]), " ")
	if start > 0 {
		s = "…" + s
	}
	if end < len(body) {
		s = s + "…"
	}
	return s
}

func prevRuneStart(s string, i int) int {
	for i > 0 && !utf8.RuneStart(s[i]) {
		i--
	}
	return i
}

func nextRuneStart(s string, i int) int {
	for i < len(s) && !utf8.RuneStart(s[i]) {
		i++
	}
	return i
}

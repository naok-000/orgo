package roam

import (
	"strings"
	"testing"
	"time"
)

func nodeByID(t *testing.T, nodes []*Node, id string) *Node {
	t.Helper()
	for _, n := range nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("no node with id %s in %d nodes", id, len(nodes))
	return nil
}

func TestParseFileSkipsFileWithNoID(t *testing.T) {
	src := "#+title: Not an org-roam note\n\nThis file has no :ID: property.\n"
	nodes := ParseFile("not-a-note.org", []byte(src), time.Now())
	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d: %+v", len(nodes), nodes)
	}
}

func TestParseFileBasicFileNode(t *testing.T) {
	src := `:PROPERTIES:
:ID:       11111111-1111-4111-8111-111111111111
:END:
#+title: Programming
#+filetags: :dev:

Body text with a link to [[id:22222222-2222-4222-8222-222222222222][Go]].
`
	nodes := ParseFile("programming.org", []byte(src), time.Now())
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	n := nodes[0]
	if n.ID != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("id = %q", n.ID)
	}
	if n.Title != "Programming" {
		t.Errorf("title = %q", n.Title)
	}
	if n.Level != 0 {
		t.Errorf("level = %d, want 0", n.Level)
	}
	if len(n.Tags) != 1 || n.Tags[0] != "dev" {
		t.Errorf("tags = %v", n.Tags)
	}
	if len(n.Links) != 1 || n.Links[0] != "22222222-2222-4222-8222-222222222222" {
		t.Errorf("links = %v", n.Links)
	}
}

func TestParseFileBodyStripsMetadataForSearch(t *testing.T) {
	src := `:PROPERTIES:
:ID:       77777777-7777-4777-8777-777777777777
:ROAM_REFS: https://en.wikipedia.org/wiki/Zettelkasten
:END:
#+title: Zettelkasten
#+filetags: :pkm:

A note-taking method built on small, densely linked notes.
`
	nodes := ParseFile("zettelkasten.org", []byte(src), time.Now())
	n := nodeByID(t, nodes, "77777777-7777-4777-8777-777777777777")

	// Source keeps everything (needed to render faithfully).
	if !strings.Contains(n.Source, ":ROAM_REFS:") || !strings.Contains(n.Source, "#+title:") {
		t.Errorf("Source should retain metadata lines verbatim:\n%s", n.Source)
	}
	// Body is for search: metadata stripped, prose kept.
	if strings.Contains(n.Body, ":ROAM_REFS:") || strings.Contains(n.Body, ":ID:") || strings.Contains(n.Body, ":END:") {
		t.Errorf("Body should not contain property drawer lines:\n%s", n.Body)
	}
	if strings.Contains(n.Body, "#+title:") || strings.Contains(n.Body, "#+filetags:") {
		t.Errorf("Body should not contain #+keyword lines:\n%s", n.Body)
	}
	if !strings.Contains(n.Body, "A note-taking method built on small, densely linked notes.") {
		t.Errorf("Body should retain the actual prose:\n%s", n.Body)
	}
}

func TestParseFileTitleFallsBackToFilename(t *testing.T) {
	src := `:PROPERTIES:
:ID:       11111111-1111-4111-8111-111111111111
:END:

No title keyword here.
`
	nodes := ParseFile("my-note.org", []byte(src), time.Now())
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Title != "my-note" {
		t.Errorf("title = %q, want %q", nodes[0].Title, "my-note")
	}
}

func TestParseFileRoamAliasesQuotedWordSplitting(t *testing.T) {
	src := `:PROPERTIES:
:ID:       44444444-4444-4444-8444-444444444444
:ROAM_ALIASES: "roam" "my zettelkasten tool"
:END:
#+title: org-roam
`
	nodes := ParseFile("org-roam.org", []byte(src), time.Now())
	n := nodeByID(t, nodes, "44444444-4444-4444-8444-444444444444")
	want := []string{"roam", "my zettelkasten tool"}
	if !stringSlicesEqual(n.Aliases, want) {
		t.Errorf("aliases = %v, want %v", n.Aliases, want)
	}
}

func TestParseFileRoamRefs(t *testing.T) {
	src := `:PROPERTIES:
:ID:       77777777-7777-4777-8777-777777777777
:ROAM_REFS: https://en.wikipedia.org/wiki/Zettelkasten
:END:
#+title: Zettelkasten
`
	nodes := ParseFile("zettelkasten.org", []byte(src), time.Now())
	n := nodeByID(t, nodes, "77777777-7777-4777-8777-777777777777")
	want := []string{"https://en.wikipedia.org/wiki/Zettelkasten"}
	if !stringSlicesEqual(n.Refs, want) {
		t.Errorf("refs = %v, want %v", n.Refs, want)
	}
}

func TestParseFiletagsForms(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  []string
	}{
		{"colon", "#+filetags: :a:b:", []string{"a", "b"}},
		{"comma", "#+filetags: a, b", []string{"a", "b"}},
		{"space", "#+filetags: a b", []string{"a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := ":PROPERTIES:\n:ID:       11111111-1111-4111-8111-111111111111\n:END:\n" + tc.value + "\n"
			nodes := ParseFile("f.org", []byte(src), time.Now())
			n := nodeByID(t, nodes, "11111111-1111-4111-8111-111111111111")
			if !stringSlicesEqual(n.Tags, tc.want) {
				t.Errorf("tags = %v, want %v", n.Tags, tc.want)
			}
		})
	}
}

// TestParseFileHeadingNodeInheritsFiletagsAndAttributesSelfLink mirrors
// testdata/notes/emacs.org's Elisp heading: it must inherit the file's
// "tools" tag, and its own self-link back to the file's id must be
// attributed to the Elisp node, not the file node (nearest enclosing node).
func TestParseFileHeadingNodeInheritsFiletagsAndAttributesSelfLink(t *testing.T) {
	src := `:PROPERTIES:
:ID:       33333333-3333-4333-8333-333333333333
:END:
#+title: Emacs
#+filetags: :tools:

The extensible editor. See [[id:44444444-4444-4444-8444-444444444444][org-roam]].

* Elisp
:PROPERTIES:
:ID:       66666666-6666-4666-8666-666666666666
:END:

Emacs Lisp extends [[id:33333333-3333-4333-8333-333333333333][Emacs]].

** Use-package

Nested content, not its own node.

* Keybindings

Nothing here links anywhere.
`
	nodes := ParseFile("emacs.org", []byte(src), time.Now())
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (file + Elisp heading), got %d: %+v", len(nodes), nodes)
	}

	file := nodeByID(t, nodes, "33333333-3333-4333-8333-333333333333")
	if len(file.Links) != 1 || file.Links[0] != "44444444-4444-4444-8444-444444444444" {
		t.Errorf("file node links = %v, want [44444444-4444-4444-8444-444444444444]", file.Links)
	}

	elisp := nodeByID(t, nodes, "66666666-6666-4666-8666-666666666666")
	if elisp.Level != 1 {
		t.Errorf("Elisp level = %d, want 1", elisp.Level)
	}
	if elisp.Title != "Elisp" {
		t.Errorf("Elisp title = %q", elisp.Title)
	}
	if !stringSlicesEqual(elisp.Tags, []string{"tools"}) {
		t.Errorf("Elisp tags = %v, want [tools] (inherited from filetags)", elisp.Tags)
	}
	// The self-link under the Elisp heading must attribute to Elisp, not the
	// file node.
	if len(elisp.Links) != 1 || elisp.Links[0] != "33333333-3333-4333-8333-333333333333" {
		t.Errorf("Elisp links = %v, want [33333333-3333-4333-8333-333333333333]", elisp.Links)
	}
}

// TestParseFileNonNodeHeadingFallsBackToFileNode covers the "Keybindings"
// heading in emacs.org, which is not itself a node: content beneath it
// (including any links) belongs to the file node.
func TestParseFileNonNodeHeadingFallsBackToFileNode(t *testing.T) {
	src := `:PROPERTIES:
:ID:       33333333-3333-4333-8333-333333333333
:END:
#+title: Emacs

* Elisp
:PROPERTIES:
:ID:       66666666-6666-4666-8666-666666666666
:END:

Some Elisp text.

* Keybindings

A link here [[id:44444444-4444-4444-8444-444444444444][org-roam]] belongs to the file node.
`
	nodes := ParseFile("emacs.org", []byte(src), time.Now())
	file := nodeByID(t, nodes, "33333333-3333-4333-8333-333333333333")
	if len(file.Links) != 1 || file.Links[0] != "44444444-4444-4444-8444-444444444444" {
		t.Errorf("file node links = %v, want the Keybindings link attributed to file", file.Links)
	}
	elisp := nodeByID(t, nodes, "66666666-6666-4666-8666-666666666666")
	if len(elisp.Links) != 0 {
		t.Errorf("Elisp node should have no links here, got %v", elisp.Links)
	}
}

func TestParseFileHeadingSubtreeSourceStopsAtNextHeading(t *testing.T) {
	src := `:PROPERTIES:
:ID:       33333333-3333-4333-8333-333333333333
:END:
#+title: Emacs

* Elisp
:PROPERTIES:
:ID:       66666666-6666-4666-8666-666666666666
:END:

Elisp body.

** Use-package

Nested subsection body.

* Keybindings

Keybindings body.
`
	nodes := ParseFile("emacs.org", []byte(src), time.Now())
	elisp := nodeByID(t, nodes, "66666666-6666-4666-8666-666666666666")
	if !containsAll(elisp.Source, "Elisp body.", "Use-package", "Nested subsection body.") {
		t.Errorf("Elisp subtree source should include its own body and nested subheadings:\n%s", elisp.Source)
	}
	if containsAll(elisp.Source, "Keybindings body.") {
		t.Errorf("Elisp subtree source should stop before the sibling Keybindings heading:\n%s", elisp.Source)
	}
}

func TestParseFileFileWithoutIDCanStillHaveHeadingNodes(t *testing.T) {
	src := `#+title: No file id here

Preamble link to [[id:44444444-4444-4444-8444-444444444444][org-roam]] is dropped (no node to own it).

* A heading node
:PROPERTIES:
:ID:       11111111-1111-4111-8111-111111111111
:END:

Body.
`
	nodes := ParseFile("f.org", []byte(src), time.Now())
	if len(nodes) != 1 {
		t.Fatalf("expected exactly 1 node (the heading), got %d: %+v", len(nodes), nodes)
	}
	if nodes[0].ID != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("unexpected node: %+v", nodes[0])
	}
}

func TestParseFileHeadingOwnTags(t *testing.T) {
	src := `:PROPERTIES:
:ID:       33333333-3333-4333-8333-333333333333
:END:
#+title: Emacs
#+filetags: :tools:

* Elisp :lang:fun:
:PROPERTIES:
:ID:       66666666-6666-4666-8666-666666666666
:END:

Body.
`
	nodes := ParseFile("emacs.org", []byte(src), time.Now())
	elisp := nodeByID(t, nodes, "66666666-6666-4666-8666-666666666666")
	if elisp.Title != "Elisp" {
		t.Errorf("title = %q, want Elisp (trailing tags stripped)", elisp.Title)
	}
	if !stringSlicesEqual(elisp.Tags, []string{"tools", "lang", "fun"}) {
		t.Errorf("tags = %v, want [tools lang fun]", elisp.Tags)
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

package render

import (
	"strings"
	"testing"
)

func resolverFor(ids ...string) Resolver {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(id string) bool { return set[id] }
}

func TestRenderResolvedIDLink(t *testing.T) {
	src := "See [[id:11111111-1111-4111-8111-111111111111][Programming]] for more."
	got, err := Render(src, resolverFor("11111111-1111-4111-8111-111111111111"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := `<a href="#/note/11111111-1111-4111-8111-111111111111"`
	if !strings.Contains(got, want) || !strings.Contains(got, ">Programming</a>") {
		t.Fatalf("output missing resolved link.\n got: %s\nwant substrings: %s and >Programming</a>", got, want)
	}
}

func TestRenderDeadIDLink(t *testing.T) {
	src := "A ghost link: [[id:99999999-9999-4999-8999-999999999999][ghost note]]."
	got, err := Render(src, resolverFor()) // nothing resolves
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := `<span class="dead-link" title="missing note">ghost note</span>`
	if !strings.Contains(got, want) {
		t.Fatalf("output missing dead-link span.\n got: %s\nwant substring: %s", got, want)
	}
}

func TestRenderIDLinkWithoutDescription(t *testing.T) {
	src := "[[id:11111111-1111-4111-8111-111111111111]]"
	got, err := Render(src, resolverFor("11111111-1111-4111-8111-111111111111"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := `<a href="#/note/11111111-1111-4111-8111-111111111111"`
	if !strings.Contains(got, want) || !strings.Contains(got, ">11111111-1111-4111-8111-111111111111</a>") {
		t.Fatalf("output missing bare id link.\n got: %s\nwant substring: %s", got, want)
	}
}

func TestRenderFileLinkIsInert(t *testing.T) {
	src := "[[file:../images/diagram.png][diagram]]"
	got, err := Render(src, resolverFor())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(got, "<a ") || strings.Contains(got, "<img") {
		t.Fatalf("file: link should not become a clickable link or image, got: %s", got)
	}
	want := `<span class="file-link" title="../images/diagram.png">diagram</span>`
	if !strings.Contains(got, want) {
		t.Fatalf("output missing inert file-link span.\n got: %s\nwant substring: %s", got, want)
	}
}

func TestRenderSrcBlock(t *testing.T) {
	src := "#+begin_src go\npackage main\n#+end_src\n"
	got, err := Render(src, resolverFor())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(got, "package main") {
		t.Fatalf("src block content missing: %s", got)
	}
	if !strings.Contains(got, `class="src`) {
		t.Fatalf("src block should keep its class after sanitizing: %s", got)
	}
}

func TestSanitizerKeepsClassesOnAllowedElements(t *testing.T) {
	src := "[[id:99999999-9999-4999-8999-999999999999][ghost]]"
	got, err := Render(src, resolverFor())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(got, `class="dead-link"`) {
		t.Fatalf("expected dead-link class to survive sanitizing: %s", got)
	}
}

func TestSanitizerStripsScripts(t *testing.T) {
	src := "#+begin_export html\n<script>alert(1)</script>\n#+end_export\n"
	got, err := Render(src, resolverFor())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(got, "<script") {
		t.Fatalf("sanitizer should strip <script> tags, got: %s", got)
	}
}

func TestSanitizerStripsEventHandlers(t *testing.T) {
	src := "#+begin_export html\n<div onclick=\"alert(1)\">hi</div>\n#+end_export\n"
	got, err := Render(src, resolverFor())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(got, "onclick") {
		t.Fatalf("sanitizer should strip inline event handlers, got: %s", got)
	}
}

func TestRenderTitleHeading(t *testing.T) {
	src := "#+title: My Note\n\nHello.\n"
	got, err := Render(src, resolverFor())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(got, "My Note") {
		t.Fatalf("expected title in output: %s", got)
	}
}

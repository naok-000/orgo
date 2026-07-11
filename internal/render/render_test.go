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

// TestRenderIDLinkPathEscapesNonUUIDIDs: ids are not required to be UUIDs;
// an id containing '/' (or other URL-special characters) must be
// percent-encoded so it stays a single path segment of the in-app route.
func TestRenderIDLinkPathEscapesNonUUIDIDs(t *testing.T) {
	src := "[[id:area/project][proj]]"
	got, err := Render(src, resolverFor("area/project"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := `href="#/note/area%2Fproject"`
	if !strings.Contains(got, want) {
		t.Fatalf("output missing percent-encoded id link.\n got: %s\nwant substring: %s", got, want)
	}
	if strings.Contains(got, `href="#/note/area/project"`) {
		t.Fatalf("id must not appear unencoded in href: %s", got)
	}
}

// TestRenderCommentBlockNotExported: org never exports comment blocks; the
// renderer must drop them (matching the indexer, which does not scan them
// for links).
func TestRenderCommentBlockNotExported(t *testing.T) {
	src := "#+begin_comment\nsecret [[id:11111111-1111-4111-8111-111111111111][hidden link]]\n#+end_comment\n\nvisible text\n"
	got, err := Render(src, resolverFor("11111111-1111-4111-8111-111111111111"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(got, "secret") || strings.Contains(got, "hidden link") {
		t.Fatalf("comment block content must not be rendered: %s", got)
	}
	if !strings.Contains(got, "visible text") {
		t.Fatalf("content outside the comment block must still render: %s", got)
	}
}

// TestRenderQuoteBlockLinkIsFollowable: links inside quote blocks are real
// links — the renderer must produce an in-app anchor for them, matching the
// indexer which counts them as edges.
func TestRenderQuoteBlockLinkIsFollowable(t *testing.T) {
	src := "#+begin_quote\nQuoting [[id:11111111-1111-4111-8111-111111111111][a note]].\n#+end_quote\n"
	got, err := Render(src, resolverFor("11111111-1111-4111-8111-111111111111"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(got, `href="#/note/11111111-1111-4111-8111-111111111111"`) {
		t.Fatalf("quote block link should render as an in-app anchor: %s", got)
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

// TestRenderLatexFragmentsSurvive: LaTeX fragments are typeset client-side
// (KaTeX in frontend/src/ui/math.ts), so the render pipeline must pass each
// delimiter style through Render()+sanitization verbatim — as HTML-escaped
// text, delimiters intact.
func TestRenderLatexFragmentsSurvive(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "inline single dollar",
			src:  "Inline $E = mc^2$ here.",
			want: "$E = mc^2$",
		},
		{
			name: "display double dollar",
			src:  `Display $$\int_0^1 x\,dx = \tfrac12$$ here.`,
			want: `$$\int_0^1 x\,dx = \tfrac12$$`,
		},
		{
			name: "inline parens",
			src:  `Paren \(a^2 + b^2 = c^2\) here.`,
			want: `\(a^2 + b^2 = c^2\)`,
		},
		{
			name: "display brackets",
			src:  `Bracket \[\sum_{n=1}^\infty \frac{1}{n^2}\] here.`,
			want: `\[\sum_{n=1}^\infty \frac{1}{n^2}\]`,
		},
		{
			name: "equation environment",
			src:  "\\begin{equation}\nf(x) = x^2\n\\end{equation}\n",
			want: "\\begin{equation}\nf(x) = x^2\n\\end{equation}",
		},
		{
			name: "align environment escapes ampersands",
			src:  "\\begin{align}\na &= b \\\\\nc &= d\n\\end{align}\n",
			want: "\\begin{align}\na &amp;= b \\\\\nc &amp;= d\n\\end{align}",
		},
		{
			name: "inline math escapes angle brackets",
			src:  "Compare $a < b$ here.",
			want: "$a &lt; b$",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Render(tc.src, resolverFor())
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("LaTeX fragment did not survive rendering.\n got: %s\nwant substring: %s", got, tc.want)
			}
		})
	}
}

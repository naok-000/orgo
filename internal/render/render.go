// Package render turns org-roam node source text into sanitized HTML for
// the API to serve. It wraps github.com/niklasfasching/go-org's HTML writer
// to rewrite [[id:...]] links into in-app links (or dead-link markers), then
// runs the result through a bluemonday policy before returning it — go-org
// passes raw HTML export blocks straight through, and note files may
// originate from untrusted sources.
package render

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/niklasfasching/go-org/org"
)

// Resolver reports whether an org-roam node id exists in the current index.
// Render uses it to decide whether an id: link becomes a followable in-app
// link or a "dead-link" span.
type Resolver func(id string) bool

var policy = newPolicy()

func newPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	// go-org emits "class" on source blocks (src-<lang>), and orgo's own
	// writer emits it on dead-link spans; the SPA also relies on being able
	// to style headings/links/code via classes. bluemonday's UGC policy
	// strips "class" everywhere by default ("we are not allowing users to
	// style their own content"), so it needs to be allowed back in on the
	// specific elements orgo/go-org actually put it on. "id" attributes
	// (heading anchors, outline nav targets) are already permitted globally
	// by UGCPolicy's AllowStandardAttributes.
	p.AllowAttrs("class").OnElements("span", "code", "pre", "div", "a")
	return p
}

// Render converts org source into sanitized HTML.
//
// Link handling:
//   - [[id:...]] links are rewritten to "#/note/<id>" (an in-app hash route)
//     when resolve(id) is true, and to a `<span class="dead-link"
//     title="missing note">` otherwise.
//   - [[file:...]] links (including local image references written with an
//     explicit file: prefix) are rendered as inert text carrying the raw
//     path in a title attribute, rather than as a clickable link or <img>:
//     orgo does not serve the scanned org directory as static files, so a
//     literal relative/absolute filesystem path is never something the
//     browser could actually load, and rendering it as a dead <a>/<img>
//     would be misleading. This is a deliberate scope simplification.
//   - Bare relative links without an explicit protocol (e.g. [[./x.png]])
//     are left to go-org's default rendering (a plain relative href/src);
//     whether they resolve depends on how the user's browser/environment is
//     set up. http/https/mailto links are untouched.
func Render(source string, resolve Resolver) (string, error) {
	conf := org.New()
	doc := conf.Parse(strings.NewReader(source), "")
	w := newHTMLWriter(resolve)
	out, err := doc.Write(w)
	if err != nil {
		return "", err
	}
	return policy.Sanitize(out), nil
}

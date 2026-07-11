package render

import (
	"html"
	"net/url"
	"strings"

	"github.com/niklasfasching/go-org/org"
)

// idLinkWriter extends go-org's HTMLWriter to special-case id: and file:
// links and to drop comment blocks. It follows go-org's own extension
// pattern (see org.HTMLWriter's ExtendingWriter field): WriteNodes
// dispatches to hw.ExtendingWriter when set, so setting it to this wrapper
// lets us override individual Write* methods while everything else falls
// through to the embedded HTMLWriter's defaults, including recursively
// (e.g. a link's description text).
type idLinkWriter struct {
	*org.HTMLWriter
	resolve Resolver
}

func newHTMLWriter(resolve Resolver) *idLinkWriter {
	hw := org.NewHTMLWriter()
	w := &idLinkWriter{HTMLWriter: hw, resolve: resolve}
	hw.ExtendingWriter = w
	return w
}

// WriteBlock drops comment blocks: org never exports them, but go-org's
// default writer renders them as a <div class="comment-block"> with fully
// parsed content. Skipping them keeps the rendered HTML in agreement with
// the indexer, which treats comment blocks as literal regions and does not
// scan them for links.
func (w *idLinkWriter) WriteBlock(b org.Block) {
	if strings.EqualFold(b.Name, "COMMENT") {
		return
	}
	w.HTMLWriter.WriteBlock(b)
}

func (w *idLinkWriter) WriteRegularLink(l org.RegularLink) {
	switch l.Protocol {
	case "id":
		w.writeIDLink(l)
	case "file":
		w.writeFileLink(l)
	default:
		w.HTMLWriter.WriteRegularLink(l)
	}
}

func (w *idLinkWriter) writeIDLink(l org.RegularLink) {
	id := strings.TrimSpace(strings.TrimPrefix(l.URL, "id:"))
	text := html.EscapeString(id)
	if l.Description != nil {
		text = w.WriteNodesAsString(l.Description...)
	}
	if w.resolve != nil && w.resolve(id) {
		// Percent-encode the id so non-UUID ids containing '/', '?', '#',
		// etc. survive as a single path segment of the in-app hash route;
		// the frontend decodes the remainder after "#/note/".
		w.WriteString(`<a href="#/note/` + html.EscapeString(url.PathEscape(id)) + `">` + text + `</a>`)
		return
	}
	w.WriteString(`<span class="dead-link" title="missing note">` + text + `</span>`)
}

func (w *idLinkWriter) writeFileLink(l org.RegularLink) {
	rawPath := strings.TrimPrefix(l.URL, "file:")
	text := html.EscapeString(rawPath)
	if l.Description != nil {
		text = w.WriteNodesAsString(l.Description...)
	}
	w.WriteString(`<span class="file-link" title="` + html.EscapeString(rawPath) + `">` + text + `</span>`)
}

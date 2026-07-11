package render

import (
	"html"
	"strings"

	"github.com/niklasfasching/go-org/org"
)

// idLinkWriter extends go-org's HTMLWriter to special-case id: and file:
// links. It follows go-org's own extension pattern (see org.HTMLWriter's
// ExtendingWriter field): WriteNodes dispatches to hw.ExtendingWriter when
// set, so setting it to this wrapper lets us override individual Write*
// methods while everything else falls through to the embedded HTMLWriter's
// defaults, including recursively (e.g. a link's description text).
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
		w.WriteString(`<a href="#/note/` + html.EscapeString(id) + `">` + text + `</a>`)
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

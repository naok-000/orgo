package server

import (
	"embed"
	"errors"
	"io/fs"
)

// dist holds the built frontend (frontend/ -> internal/server/dist/ via
// `npm run build`, per docs/DESIGN.md). Plain "//go:embed dist" silently
// skips any file or directory whose name starts with "." or "_"; "all:"
// disables that so build output using those conventions (or a future
// dotfile) is never silently dropped from the binary.
//
//go:embed all:dist
var dist embed.FS

// distSub returns the embedded frontend build rooted at its own top level
// (i.e. "index.html" instead of "dist/index.html").
func distSub() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// Only possible if the embed directive above is broken, which would
		// fail the build itself first; this can't happen at runtime.
		panic(err)
	}
	return sub
}

// spaFS wraps an fs.FS so that any path which doesn't exist falls back to
// serving index.html, letting the embedded single-page app own client-side
// routing (DESIGN.md: "any non-/api path serves dist files, falling back to
// index.html").
type spaFS struct {
	fs.FS
}

func (s spaFS) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if err == nil {
		return f, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return s.FS.Open("index.html")
	}
	return nil, err
}

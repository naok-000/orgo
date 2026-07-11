# orgo design

orgo = org-roam-ui's browsing/graph features + k1LoW/mo's reading UX,
as a standalone Go binary with an embedded web frontend.

## Decisions

- **Parse org files directly** (org-roam compatibility subset; no SQLite DB,
  no running Emacs). A node is a file, or a heading, with an `:ID:` property.
  Indexed per node: id, title (`#+title` or heading text), level, tags,
  aliases, refs, and outgoing `[[id:…]]` links.
  - Links are attributed to the *nearest enclosing node* (heading node if the
    link sits under one, else the file node).
  - `ROAM_ALIASES` uses org-style quoted-word splitting (aliases may contain
    spaces); `ROAM_REFS` may hold multiple refs.
  - File tags (`#+filetags`) are inherited by heading nodes; heading tags add.
  - Duplicate IDs are detected deterministically (first file wins, sorted
    path order) and reported as diagnostics.
  - Unresolved `id:` links are not silently ignored: they render as
    `class="dead-link"` spans and are counted in diagnostics. `/api/graph`
    only emits edges whose endpoints both exist.
- **Rendering happens server-side** with `niklasfasching/go-org`; `id:` links
  are rewritten to in-app links (`#/note/<id>`). Output is sanitized with
  bluemonday (UGC policy + classes for code highlighting) — go-org passes
  raw HTML export blocks through, and notes may come from untrusted sources.
- **Live reload via SSE** (`/api/events`), fsnotify + debounce re-index.
- **Frontend is a single-page app** built with Vite + TypeScript; built assets
  are committed under `internal/server/dist/` so plain `go install` works.
  CI rebuilds and fails when `dist/` drifts from source. Graph rendering via
  `force-graph` (same family org-roam-ui uses).
- **mo UX subset we adopt:** tabs (open/switch/close, persisted), sidebar note
  list, full-text search (Ctrl-K palette), dark/light theme, live reload,
  content width/font-size controls. We deliberately skip mo's daemon/session/
  group model: orgo serves exactly one org-roam directory per process,
  loopback-only by default.
- **Emacs entry point:** `orgo.el` (installable via `use-package :vc`)
  launches the binary against `org-roam-directory`. Binary install paths:
  GitHub release download, `go install`, or `nix run github:naok-000/orgo`.

## CLI

```
orgo [flags] [dir]        # dir defaults to .
  -p, --port int          # default 35911
      --addr string       # default 127.0.0.1
      --no-browser        # don't auto-open browser
      --version
```

## HTTP API (v1 contract)

| Route                 | Response                                                                 |
|-----------------------|--------------------------------------------------------------------------|
| `GET /api/meta`       | `{"root","workspaceId","version","noteCount"}` (workspaceId = hash of abs root path) |
| `GET /api/graph`      | `{"nodes":[{"id","title","file","level","tags":[…],"nlinks"}],"edges":[{"source","target"}]}` |
| `GET /api/notes`      | `[{"id","title","file","tags":[…],"mtime"}]` (file+heading nodes)        |
| `GET /api/note/{id}`  | `{"id","title","file","level","tags","aliases","refs","html","backlinks":[{"id","title"}]}`; 404 + JSON error if unknown |
| `GET /api/search?q=`  | `[{"id","title","snippet"}]` — case-insensitive over titles/aliases/body |
| `GET /api/events`     | SSE stream; event `reload` (data: `{}`) after any re-index               |
| `GET /`               | embedded SPA (any non-API path → index.html)                             |

## Frontend routes & state (hash-based)

- `#/note/<id>` — the *active* note. The URL represents the active node, not
  the whole tab set; back/forward switches the active tab naturally.
- `#/graph` — global graph; `#/graph/<id>` — local neighborhood of a node.
- Following an in-note link or double-clicking a graph node focuses an
  existing tab for that id or opens a new one. Single-click on a graph node
  selects/highlights only — selection never mutates tabs.
- Tabs, ordering, active tab, theme, and graph preferences persist in
  localStorage, namespaced by `workspaceId` so two roam dirs don't share
  state. Restored tabs whose node vanished show a visible "missing" state.

## Layout

```
main.go                  CLI + wiring
internal/roam/           scanning, org parsing, index, graph (pure, heavily tested)
internal/render/         org→HTML (go-org config, link rewriting, sanitizing)
internal/server/         HTTP handlers, SSE hub, go:embed dist/
internal/watch/          fsnotify + debounce
frontend/                Vite + TS sources (vitest) → builds into internal/server/dist/
orgo.el                  Emacs launcher
testdata/notes/          shared org-roam corpus for tests and demos
.github/workflows/ci.yml go test + staticcheck + frontend build + dist drift check
```

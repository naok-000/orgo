# orgo design

orgo = org-roam-ui's browsing/graph features + k1LoW/mo's reading UX,
as a standalone Go binary with an embedded web frontend.

## Decisions

- **Parse org files directly.** No dependency on org-roam's SQLite DB or a
  running Emacs. A node is a file, or a heading, with an `:ID:` property.
  We index: id, title (`#+title` or heading text), tags (`#+filetags`,
  heading tags), aliases (`ROAM_ALIASES`), refs (`ROAM_REFS`), and all
  `[[id:…]]` links with their source node.
- **Rendering happens server-side** with `niklasfasching/go-org`; `id:` links
  are rewritten to in-app links (`#/note/<id>`), dead ids get a
  `class="dead-link"` span.
- **Live reload via SSE** (`/api/events`), fsnotify + debounce re-index.
- **Frontend is a single-page app** built with Vite + TypeScript, committed
  built assets under `internal/server/dist/` so plain `go install` works.
  Graph via `force-graph` (same family org-roam-ui uses).
- **mo UX subset we adopt:** tabs (open/switch/close, persisted in
  localStorage), sidebar note list (sortable), full-text search (Ctrl-K
  palette), dark/light theme, live reload, font-size/width controls.
  We deliberately skip mo's daemon/session/group model: orgo serves exactly
  one org-roam directory per process.
- **Emacs entry point:** `orgo.el` (installable via `use-package :vc`)
  launches the binary against `org-roam-directory`.

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
| `GET /api/graph`      | `{"nodes":[{"id","title","file","level","tags":[…],"nlinks"}],"edges":[{"source","target"}]}` |
| `GET /api/notes`      | `[{"id","title","file","tags":[…],"mtime"}]` (file+heading nodes)        |
| `GET /api/note/{id}`  | `{"id","title","file","level","tags","aliases","refs","html","backlinks":[{"id","title"}]}`; 404 + JSON error if unknown |
| `GET /api/search?q=`  | `[{"id","title","snippet"}]` — case-insensitive over title/aliases/body  |
| `GET /api/events`     | SSE stream; event `reload` (data: `{}`) after any re-index               |
| `GET /`               | embedded SPA (also any non-API path → index.html)                        |

Edges pointing at unknown ids are dropped from `/api/graph`.

## Frontend routes (hash-based)

- `#/note/<id>` — note open in a tab (opening a link adds/focuses a tab)
- `#/graph` — full graph view
- default `#/` — graph if no tabs, else last active tab

## Layout (Go)

```
main.go                  CLI + wiring
internal/roam/           scanning, org parsing, index, graph (pure, heavily tested)
internal/render/         org→HTML (go-org configuration, link rewriting)
internal/server/         HTTP handlers, SSE hub, go:embed dist/
internal/watch/          fsnotify + debounce
frontend/                Vite + TS sources (vitest tests) → builds into internal/server/dist/
orgo.el                  Emacs launcher
testdata/notes/          shared org-roam corpus for tests and demos
```

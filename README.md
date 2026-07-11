# orgo

Browse your [org-roam](https://www.orgroam.com/) notes in the browser — with
followable links, a tabbed reading UX inspired by [k1LoW/mo](https://github.com/k1LoW/mo),
and an interactive link graph inspired by [org-roam-ui](https://github.com/org-roam/org-roam-ui).

orgo is a single Go binary that parses your org files directly: no running
Emacs, no org-roam SQLite database required.

## Features

- **Note browsing** — org files rendered to HTML; `[[id:…]]` links between
  notes are followable in the browser.
- **LaTeX math** — `$…$`, `$$…$$`, `\(…\)`, `\[…\]`, and environments like
  `\begin{equation}` render with bundled KaTeX (works offline).
- **Tabs** — notes open as tabs you can switch and close, like an editor.
  Tabs survive reloads (per note directory).
- **Graph** — the link structure visualized as an interactive force-directed
  graph: global view, plus a local neighborhood view per note (with a
  "Show full graph" button to get back). Single-click highlights,
  double-click opens the note. Node size and link width are adjustable.
- **Search** — `Ctrl-K` / `Cmd-K` palette searching titles, aliases, and body
  text.
- **Live reload** — save a file in Emacs and the browser updates.
- **Dark / light theme**, adjustable content width and font size.

## Install

orgo is distributed via GitHub only (no MELPA, no brew).

### 1. Get the binary

Any one of:

```console
$ go install github.com/naok-000/orgo@latest     # Go toolchain
$ nix run github:naok-000/orgo -- ~/org-roam     # Nix (flakes)
```

or download a binary from the [releases page](https://github.com/naok-000/orgo/releases)
when available.

### 2. (Optional) Emacs integration

With Emacs 30+, `use-package` installs `orgo.el` straight from GitHub:

```elisp
(use-package orgo
  :vc (:url "https://github.com/naok-000/orgo" :rev :newest))
```

(Emacs 29: `M-x package-vc-install RET https://github.com/naok-000/orgo`,
or use straight.el / elpaca with their GitHub recipes.)

Then:

| Command                     | Effect                                            |
|-----------------------------|---------------------------------------------------|
| `M-x orgo`                  | start the server for `org-roam-directory`, open browser |
| `M-x orgo-open-current-note`| show the node at point in the browser             |
| `M-x orgo-stop`             | stop the server                                   |

`orgo.el` only manages the process — the binary from step 1 must be on your
`exec-path`.

## Usage

```console
$ orgo ~/org-roam
orgo: serving /home/you/org-roam at http://127.0.0.1:35911/
```

```
orgo [flags] [dir]        # dir defaults to .
  -p, --port int          # default 35911
      --addr string       # default 127.0.0.1 (loopback only)
      --no-browser        # don't auto-open the browser
      --version
```

## What counts as a note?

Anything org-roam considers a node, within a documented compatibility subset:
files with a file-level `:ID:` property, and headings with their own `:ID:`.
`#+title`, `#+filetags`, heading tags, `ROAM_ALIASES` and `ROAM_REFS` are
indexed. See [docs/DESIGN.md](./docs/DESIGN.md) for details and limitations.

## Development

Requires [Nix](https://nixos.org/) with flakes and [direnv](https://direnv.net/):

```console
$ direnv allow        # or: nix develop
$ go test ./...       # backend tests
$ cd frontend && npm ci && npm test   # frontend tests
$ npm run build       # rebuilds internal/server/dist (committed)
```

The frontend's built assets are committed so that `go install` works from a
plain checkout; CI fails if they drift from the TypeScript sources.

## License

See [LICENSE](./LICENSE).

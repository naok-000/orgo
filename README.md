# orgo

Browse your [org-roam](https://www.orgroam.com/) notes in the browser — with
followable links, a tabbed reading UX inspired by [k1LoW/mo](https://github.com/k1LoW/mo),
and an interactive link graph inspired by [org-roam-ui](https://github.com/org-roam/org-roam-ui).

## Features

- **Note browsing** — org files rendered to HTML in your browser; `[[id:…]]`
  links between notes are followable.
- **Tabs** — notes open as tabs you can switch between and close, like a code
  editor. Sessions survive reloads.
- **Graph** — the link structure of your notes visualized as an interactive
  force-directed graph, with a local neighborhood view per note.
- **Search** — full-text search across titles and note content.
- **Live reload** — save a file in Emacs and the browser updates.
- **Dark / light theme.**
- **No Emacs required at runtime** — orgo parses your org files directly; it
  does not read the org-roam SQLite database.

## Install

_(work in progress)_

## Usage

```console
$ orgo ~/org-roam
```

Then open http://localhost:35911/.

## Development

Requires [Nix](https://nixos.org/) with flakes and [direnv](https://direnv.net/):

```console
$ direnv allow   # or: nix develop
$ go test ./...
```

## License

See [LICENSE](./LICENSE).

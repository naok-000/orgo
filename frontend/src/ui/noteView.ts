import type { Backlink, NoteDetail } from "../types.ts";
import { el, clear } from "./dom.ts";

export interface NoteViewCallbacks {
  onShowInGraph: (id: string) => void;
}

export class NoteView {
  readonly root: HTMLElement;

  constructor(private readonly callbacks: NoteViewCallbacks) {
    this.root = el("div", { className: "note-view" });
  }

  showLoading(): void {
    clear(this.root);
    this.root.append(el("div", { className: "state state-loading", text: "Loading…" }));
  }

  showMissing(id: string): void {
    clear(this.root);
    this.root.append(
      el("div", { className: "state state-missing" }, [
        el("h2", { text: "Note not found" }),
        el("p", { text: `No note with id "${id}" exists. It may have been removed.` }),
      ]),
    );
  }

  showError(message: string): void {
    clear(this.root);
    this.root.append(
      el("div", { className: "state state-error" }, [
        el("h2", { text: "Failed to load note" }),
        el("p", { text: message }),
      ]),
    );
  }

  showNote(note: NoteDetail): void {
    clear(this.root);

    const tagsEl = note.tags.length
      ? el(
          "div",
          { className: "note-tags" },
          note.tags.map((t) => el("span", { className: "tag", text: t })),
        )
      : null;

    const showInGraphBtn = el("button", {
      className: "show-in-graph",
      attrs: { type: "button" },
      text: "Show in graph",
    });
    showInGraphBtn.addEventListener("click", () => this.callbacks.onShowInGraph(note.id));

    const header = el("header", { className: "note-header" }, [
      el("h1", { className: "note-title", text: note.title }),
      tagsEl,
      showInGraphBtn,
    ]);

    // note.html is trusted, server-sanitized HTML per docs/DESIGN.md.
    const body = el("div", { className: "note-body org-content", html: note.html });

    this.root.append(header, body, this.renderBacklinks(note.backlinks));
  }

  private renderBacklinks(backlinks: Backlink[]): HTMLElement {
    if (backlinks.length === 0) {
      return el("section", { className: "backlinks" }, [
        el("h2", { text: "Backlinks" }),
        el("p", { className: "backlinks-empty", text: "Nothing links here yet." }),
      ]);
    }
    const list = el(
      "ul",
      { className: "backlinks-list" },
      backlinks.map((b) =>
        el("li", {}, [
          el("a", { attrs: { href: `#/note/${encodeURIComponent(b.id)}` }, text: b.title }),
        ]),
      ),
    );
    return el("section", { className: "backlinks" }, [
      el("h2", { text: `Backlinks (${backlinks.length})` }),
      list,
    ]);
  }
}

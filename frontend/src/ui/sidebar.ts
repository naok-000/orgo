import type { NoteSummary } from "../types.ts";
import { filterNotes, sortNotes, type SortMode } from "../notesList.ts";
import { el, clear } from "./dom.ts";

export interface SidebarProps {
  notes: NoteSummary[];
  filter: string;
  sort: SortMode;
  collapsed: boolean;
  activeNoteId: string | null;
}

export interface SidebarCallbacks {
  onFilterChange: (value: string) => void;
  onSortChange: (sort: SortMode) => void;
  onToggleCollapse: () => void;
  onOpenNote: (id: string) => void;
}

export class Sidebar {
  readonly root: HTMLElement;
  private readonly filterInput: HTMLInputElement;
  private readonly sortButton: HTMLButtonElement;
  private readonly collapseButton: HTMLButtonElement;
  private readonly listEl: HTMLUListElement;
  private readonly countEl: HTMLElement;

  constructor(private readonly callbacks: SidebarCallbacks) {
    this.filterInput = el("input", {
      className: "sidebar-filter",
      attrs: {
        type: "text",
        placeholder: "Filter notes…",
        "aria-label": "Filter notes",
      },
      on: {
        input: () => this.callbacks.onFilterChange(this.filterInput.value),
      },
    }) as HTMLInputElement;

    this.sortButton = el("button", {
      className: "sidebar-sort",
      attrs: { type: "button", title: "Toggle sort" },
    }) as HTMLButtonElement;
    this.sortButton.addEventListener("click", () => {
      this.callbacks.onSortChange(this.currentSort === "title" ? "mtime" : "title");
    });

    this.collapseButton = el(
      "button",
      {
        className: "sidebar-collapse",
        attrs: { type: "button", title: "Collapse sidebar", "aria-label": "Collapse sidebar" },
        on: { click: () => this.callbacks.onToggleCollapse() },
      },
      ["«"],
    ) as HTMLButtonElement;

    this.countEl = el("span", { className: "sidebar-count" });
    this.listEl = el("ul", { className: "note-list", attrs: { role: "list" } });

    const header = el("div", { className: "sidebar-header" }, [
      this.filterInput,
      this.sortButton,
      this.collapseButton,
    ]);

    this.root = el("aside", { className: "sidebar" }, [
      header,
      this.countEl,
      this.listEl,
    ]);
  }

  private currentSort: SortMode = "title";

  render(props: SidebarProps): void {
    this.root.classList.toggle("collapsed", props.collapsed);
    this.collapseButton.textContent = props.collapsed ? "»" : "«";
    this.collapseButton.title = props.collapsed ? "Expand sidebar" : "Collapse sidebar";
    if (this.filterInput.value !== props.filter) {
      this.filterInput.value = props.filter;
    }
    this.currentSort = props.sort;
    this.sortButton.textContent = props.sort === "title" ? "A–Z" : "Recent";

    const visible = sortNotes(filterNotes(props.notes, props.filter), props.sort);
    this.countEl.textContent = `${visible.length} / ${props.notes.length} notes`;

    clear(this.listEl);
    if (visible.length === 0) {
      this.listEl.append(
        el("li", { className: "note-list-empty", text: "No notes match." }),
      );
      return;
    }
    for (const note of visible) {
      this.listEl.append(this.renderItem(note, props.activeNoteId === note.id));
    }
  }

  private renderItem(note: NoteSummary, active: boolean): HTMLLIElement {
    const tags = note.tags.length
      ? el(
          "span",
          { className: "note-item-tags" },
          note.tags.map((t) => el("span", { className: "tag", text: t })),
        )
      : null;

    const item = el(
      "li",
      {
        className: `note-item${active ? " active" : ""}`,
        attrs: { role: "listitem" },
      },
      [el("span", { className: "note-item-title", text: note.title }), tags],
    );
    item.addEventListener("click", () => this.callbacks.onOpenNote(note.id));
    return item as HTMLLIElement;
  }
}

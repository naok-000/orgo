import type { SearchResult } from "../types.ts";
import { createSearchController, type SearchFetcher } from "../search.ts";
import { el, clear } from "./dom.ts";

export interface PaletteCallbacks {
  onSelect: (id: string) => void;
}

export class CommandPalette {
  readonly root: HTMLElement;
  private readonly input: HTMLInputElement;
  private readonly listEl: HTMLElement;
  private readonly controller: ReturnType<typeof createSearchController>;
  private results: SearchResult[] = [];
  private selectedIndex = 0;
  private isOpen = false;

  constructor(
    private readonly callbacks: PaletteCallbacks,
    fetcher: SearchFetcher,
  ) {
    this.controller = createSearchController(
      fetcher,
      (_q, results) => {
        this.results = results;
        this.selectedIndex = 0;
        this.renderResults();
      },
      () => {
        this.results = [];
        this.renderResults();
      },
    );

    this.input = el("input", {
      className: "palette-input",
      attrs: {
        type: "text",
        placeholder: "Search notes…",
        "aria-label": "Search notes",
      },
    }) as HTMLInputElement;
    this.input.addEventListener("input", () => this.controller.query(this.input.value));
    this.input.addEventListener("keydown", (e) => this.handleKeydown(e));

    this.listEl = el("ul", { className: "palette-results" });

    const panel = el("div", { className: "palette-panel" }, [this.input, this.listEl]);
    this.root = el("div", { className: "palette-overlay hidden" }, [panel]);
    this.root.addEventListener("mousedown", (e) => {
      if (e.target === this.root) this.close();
    });
  }

  get opened(): boolean {
    return this.isOpen;
  }

  open(): void {
    this.isOpen = true;
    this.root.classList.remove("hidden");
    this.input.value = "";
    this.results = [];
    this.selectedIndex = 0;
    this.renderResults();
    requestAnimationFrame(() => this.input.focus());
  }

  close(): void {
    this.isOpen = false;
    this.root.classList.add("hidden");
    this.controller.dispose();
  }

  toggle(): void {
    if (this.isOpen) this.close();
    else this.open();
  }

  private handleKeydown(e: KeyboardEvent): void {
    if (e.key === "Escape") {
      e.preventDefault();
      this.close();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      this.move(1);
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      this.move(-1);
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      const r = this.results[this.selectedIndex];
      if (r) {
        this.callbacks.onSelect(r.id);
        this.close();
      }
    }
  }

  private move(delta: number): void {
    if (this.results.length === 0) return;
    this.selectedIndex =
      (this.selectedIndex + delta + this.results.length) % this.results.length;
    this.renderResults();
  }

  private renderResults(): void {
    clear(this.listEl);
    if (this.results.length === 0) {
      this.listEl.append(
        el("li", {
          className: "palette-empty",
          text: this.input.value.trim() ? "No results." : "Type to search…",
        }),
      );
      return;
    }
    this.results.forEach((r, i) => {
      const item = el(
        "li",
        {
          className: `palette-item${i === this.selectedIndex ? " active" : ""}`,
          attrs: { role: "option" },
        },
        [
          el("span", { className: "palette-item-title", text: r.title }),
          el("span", { className: "palette-item-snippet", text: r.snippet }),
        ],
      );
      item.addEventListener("click", () => {
        this.callbacks.onSelect(r.id);
        this.close();
      });
      item.addEventListener("mouseenter", () => {
        this.selectedIndex = i;
        this.renderResults();
      });
      this.listEl.append(item);
    });
  }
}

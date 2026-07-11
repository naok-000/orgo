import type { Tab } from "../store/tabs.ts";
import { tabKey } from "../store/tabs.ts";
import { el, clear } from "./dom.ts";

export interface TabBarCallbacks {
  onFocus: (key: string) => void;
  onClose: (id: string) => void;
}

export class TabBar {
  readonly root: HTMLElement;

  constructor(
    private readonly callbacks: TabBarCallbacks,
    /** Resolve a display title for a note id (may be loading/unknown). */
    private readonly getTitle: (id: string) => string,
  ) {
    this.root = el("div", { className: "tabbar", attrs: { role: "tablist" } });
  }

  render(tabs: Tab[], active: string): void {
    clear(this.root);
    for (const tab of tabs) {
      this.root.append(this.renderTab(tab, tabKey(tab) === active));
    }
  }

  private renderTab(tab: Tab, active: boolean): HTMLElement {
    const key = tabKey(tab);
    const isGraph = tab.kind === "graph";
    const missing = tab.kind === "note" && tab.missing;

    const label = el("span", {
      className: "tab-label",
      text: isGraph ? "Graph" : this.getTitle(tab.id),
    });

    const children: (Node | null)[] = [
      isGraph ? el("span", { className: "tab-icon", text: "◉" }) : null,
      label,
    ];

    if (!isGraph) {
      const closeBtn = el("button", {
        className: "tab-close",
        attrs: { type: "button", title: "Close tab", "aria-label": "Close tab" },
        text: "×",
      });
      closeBtn.addEventListener("click", (e) => {
        e.stopPropagation();
        this.callbacks.onClose(tab.id);
      });
      children.push(closeBtn);
    }

    const el_ = el(
      "div",
      {
        className: [
          "tab",
          active ? "active" : "",
          isGraph ? "tab-graph" : "",
          missing ? "tab-missing" : "",
        ]
          .filter(Boolean)
          .join(" "),
        attrs: { role: "tab", "aria-selected": String(active) },
      },
      children,
    );
    el_.addEventListener("click", () => this.callbacks.onFocus(key));
    if (!isGraph) {
      // middle-click close
      el_.addEventListener("auxclick", (e) => {
        if (e.button === 1) {
          e.preventDefault();
          this.callbacks.onClose(tab.id);
        }
      });
    }
    return el_;
  }
}

// Tab store: pinned "graph" tab + note tabs. Pure state-transition functions
// (heavily unit tested) wrapped by a thin stateful TabStore class that adds
// persistence + pub/sub for the UI layer.

import type { NamespacedStorage } from "./storage.ts";

export type Tab =
  | { kind: "graph" }
  | { kind: "note"; id: string; missing: boolean };

export interface TabsState {
  /** tabs[0] is always the pinned graph tab. */
  tabs: Tab[];
  /** "graph" or a note id. Always refers to an entry present in tabs. */
  active: string;
}

export interface PersistedTabsV1 {
  tabs: Array<{ kind: "graph" } | { kind: "note"; id: string }>;
  active: string;
}

const STORAGE_KEY = "tabs";

export function tabKey(tab: Tab): string {
  return tab.kind === "graph" ? "graph" : tab.id;
}

export function initialTabsState(): TabsState {
  return { tabs: [{ kind: "graph" }], active: "graph" };
}

export function findNoteTab(state: TabsState, id: string): Tab | undefined {
  return state.tabs.find((t) => t.kind === "note" && t.id === id);
}

/**
 * Focus the tab for `id` if it is already open, otherwise open and focus a
 * new one. This single function is what both "click an in-note link" and
 * "browser back/forward to #/note/<id>" call, which is what guarantees back/
 * forward never duplicates tabs (per docs/DESIGN.md).
 */
export function openNoteTab(state: TabsState, id: string): TabsState {
  if (findNoteTab(state, id)) {
    return { ...state, active: id };
  }
  const tabs: Tab[] = [...state.tabs, { kind: "note", id, missing: false }];
  return { tabs, active: id };
}

export function closeNoteTab(state: TabsState, id: string): TabsState {
  const idx = state.tabs.findIndex((t) => t.kind === "note" && t.id === id);
  if (idx === -1) return state;
  const tabs = state.tabs.slice(0, idx).concat(state.tabs.slice(idx + 1));
  let active = state.active;
  if (active === id) {
    const neighbor = tabs[idx] ?? tabs[idx - 1] ?? tabs[0];
    active = neighbor ? tabKey(neighbor) : "graph";
  }
  return { tabs, active };
}

export function focusGraphTab(state: TabsState): TabsState {
  return { ...state, active: "graph" };
}

export function setTabMissing(
  state: TabsState,
  id: string,
  missing: boolean,
): TabsState {
  let changed = false;
  const tabs = state.tabs.map((t) => {
    if (t.kind === "note" && t.id === id && t.missing !== missing) {
      changed = true;
      return { ...t, missing };
    }
    return t;
  });
  return changed ? { ...state, tabs } : state;
}

export function serializeTabs(state: TabsState): PersistedTabsV1 {
  return {
    tabs: state.tabs.map((t) =>
      t.kind === "graph" ? { kind: "graph" as const } : { kind: "note" as const, id: t.id },
    ),
    active: state.active,
  };
}

export function deserializeTabs(data: unknown): TabsState {
  const fallback = initialTabsState();
  if (!data || typeof data !== "object") return fallback;
  const d = data as Partial<PersistedTabsV1>;
  if (!Array.isArray(d.tabs)) return fallback;

  // The graph tab is always synthesized at index 0, regardless of where (or
  // whether) it appears in the persisted data — this keeps the "tabs[0] is
  // always the pinned graph tab" invariant even if storage was hand-edited
  // or written by a future version with a different tab order.
  const noteTabs: Tab[] = [];
  for (const raw of d.tabs) {
    if (!raw || typeof raw !== "object") continue;
    const t = raw as { kind?: unknown; id?: unknown };
    if (t.kind === "note" && typeof t.id === "string") {
      if (!noteTabs.some((x) => x.kind === "note" && x.id === t.id)) {
        noteTabs.push({ kind: "note", id: t.id, missing: false });
      }
    }
  }
  const tabs: Tab[] = [{ kind: "graph" }, ...noteTabs];

  const active =
    typeof d.active === "string" &&
    tabs.some((t) => tabKey(t) === d.active)
      ? d.active
      : "graph";

  return { tabs, active };
}

export class TabStore {
  private state: TabsState;
  private listeners = new Set<(state: TabsState) => void>();

  constructor(private readonly storage?: NamespacedStorage) {
    const persisted = storage?.getJSON<PersistedTabsV1>(STORAGE_KEY);
    this.state = persisted ? deserializeTabs(persisted) : initialTabsState();
  }

  getState(): TabsState {
    return this.state;
  }

  subscribe(fn: (state: TabsState) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private commit(next: TabsState): void {
    this.state = next;
    this.storage?.setJSON(STORAGE_KEY, serializeTabs(next));
    for (const l of this.listeners) l(next);
  }

  openNote(id: string): void {
    this.commit(openNoteTab(this.state, id));
  }

  closeNote(id: string): void {
    this.commit(closeNoteTab(this.state, id));
  }

  focusGraph(): void {
    this.commit(focusGraphTab(this.state));
  }

  setMissing(id: string, missing: boolean): void {
    this.commit(setTabMissing(this.state, id, missing));
  }
}

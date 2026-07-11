import { describe, expect, it } from "vitest";
import {
  TabStore,
  closeNoteTab,
  deserializeTabs,
  focusGraphTab,
  initialTabsState,
  openNoteTab,
  serializeTabs,
  setTabMissing,
  tabKey,
  type TabsState,
} from "./tabs.ts";
import { NamespacedStorage, type StorageLike } from "./storage.ts";

class FakeStorage implements StorageLike {
  private map = new Map<string, string>();
  getItem(key: string): string | null {
    return this.map.has(key) ? this.map.get(key)! : null;
  }
  setItem(key: string, value: string): void {
    this.map.set(key, value);
  }
  removeItem(key: string): void {
    this.map.delete(key);
  }
}

describe("initialTabsState", () => {
  it("starts with only the pinned graph tab, active", () => {
    const state = initialTabsState();
    expect(state.tabs).toEqual([{ kind: "graph" }]);
    expect(state.active).toBe("graph");
  });
});

describe("openNoteTab", () => {
  it("opens and focuses a new tab", () => {
    const state = openNoteTab(initialTabsState(), "n1");
    expect(state.tabs).toEqual([{ kind: "graph" }, { kind: "note", id: "n1", missing: false }]);
    expect(state.active).toBe("n1");
  });

  it("focuses an already-open tab instead of duplicating it", () => {
    let state = openNoteTab(initialTabsState(), "n1");
    state = openNoteTab(state, "n2");
    state = openNoteTab(state, "n1"); // re-visit n1, e.g. via back/forward or a link click
    expect(state.tabs).toHaveLength(3);
    expect(state.active).toBe("n1");
  });
});

describe("closeNoteTab", () => {
  function withTabs(...ids: string[]): TabsState {
    return ids.reduce((s, id) => openNoteTab(s, id), initialTabsState());
  }

  it("removes the tab", () => {
    const state = closeNoteTab(withTabs("n1", "n2"), "n1");
    expect(state.tabs.map(tabKey)).toEqual(["graph", "n2"]);
  });

  it("is a no-op for an id that isn't open", () => {
    const original = withTabs("n1");
    expect(closeNoteTab(original, "does-not-exist")).toBe(original);
  });

  it("activates the next tab when closing the active tab", () => {
    const state = closeNoteTab(withTabs("n1", "n2", "n3"), "n2"); // n2 was active (last opened)
    expect(state.active).toBe("n3");
  });

  it("activates the previous tab when closing the last tab", () => {
    const state = closeNoteTab(withTabs("n1", "n2", "n3"), "n3");
    expect(state.active).toBe("n2");
  });

  it("falls back to the graph tab when the only note tab is closed", () => {
    const state = closeNoteTab(withTabs("n1"), "n1");
    expect(state.active).toBe("graph");
    expect(state.tabs).toEqual([{ kind: "graph" }]);
  });

  it("leaves the active tab untouched when closing a background tab", () => {
    let state = withTabs("n1", "n2");
    state = focusGraphTab(state);
    state = closeNoteTab(state, "n1");
    expect(state.active).toBe("graph");
    expect(state.tabs.map(tabKey)).toEqual(["graph", "n2"]);
  });
});

describe("setTabMissing", () => {
  it("marks a note tab missing without touching others", () => {
    let state = openNoteTab(initialTabsState(), "n1");
    state = openNoteTab(state, "n2");
    state = setTabMissing(state, "n1", true);
    expect(state.tabs).toEqual([
      { kind: "graph" },
      { kind: "note", id: "n1", missing: true },
      { kind: "note", id: "n2", missing: false },
    ]);
  });

  it("is referentially stable when nothing changes", () => {
    const state = openNoteTab(initialTabsState(), "n1");
    expect(setTabMissing(state, "n1", false)).toBe(state);
  });
});

describe("serialize/deserialize round trip", () => {
  it("round-trips tabs and active id, dropping the missing flag", () => {
    let state = openNoteTab(initialTabsState(), "n1");
    state = openNoteTab(state, "n2");
    state = setTabMissing(state, "n1", true);

    const persisted = serializeTabs(state);
    expect(persisted).toEqual({
      tabs: [{ kind: "graph" }, { kind: "note", id: "n1" }, { kind: "note", id: "n2" }],
      active: "n2",
    });

    const restored = deserializeTabs(persisted);
    // missing is recomputed at runtime (starts false), not carried over.
    expect(restored.tabs).toEqual([
      { kind: "graph" },
      { kind: "note", id: "n1", missing: false },
      { kind: "note", id: "n2", missing: false },
    ]);
    expect(restored.active).toBe("n2");
  });

  it("falls back to the initial state for garbage input", () => {
    expect(deserializeTabs(undefined)).toEqual(initialTabsState());
    expect(deserializeTabs(null)).toEqual(initialTabsState());
    expect(deserializeTabs("not an object")).toEqual(initialTabsState());
    expect(deserializeTabs({ tabs: "nope" })).toEqual(initialTabsState());
  });

  it("ensures exactly one graph tab, always first", () => {
    const restored = deserializeTabs({
      tabs: [{ kind: "note", id: "n1" }, { kind: "graph" }, { kind: "graph" }],
      active: "n1",
    });
    expect(restored.tabs[0]).toEqual({ kind: "graph" });
    expect(restored.tabs.filter((t) => t.kind === "graph")).toHaveLength(1);
  });

  it("de-duplicates repeated note ids", () => {
    const restored = deserializeTabs({
      tabs: [{ kind: "note", id: "n1" }, { kind: "note", id: "n1" }],
      active: "n1",
    });
    expect(restored.tabs.filter((t) => t.kind === "note" && t.id === "n1")).toHaveLength(1);
  });

  it("falls back active to graph if the persisted active tab isn't in the tab list", () => {
    const restored = deserializeTabs({
      tabs: [{ kind: "note", id: "n1" }],
      active: "ghost",
    });
    expect(restored.active).toBe("graph");
  });
});

describe("TabStore", () => {
  it("persists state through a NamespacedStorage backend and restores it", () => {
    const backend = new FakeStorage();
    const storage = new NamespacedStorage("ws", backend);

    const store1 = new TabStore(storage);
    store1.openNote("n1");
    store1.openNote("n2");
    store1.focusGraph();

    const store2 = new TabStore(storage);
    expect(store2.getState().tabs.map(tabKey)).toEqual(["graph", "n1", "n2"]);
    expect(store2.getState().active).toBe("graph");
  });

  it("notifies subscribers on every mutation", () => {
    const store = new TabStore();
    const seen: string[] = [];
    const unsubscribe = store.subscribe((s) => seen.push(s.active));

    store.openNote("n1");
    store.openNote("n2");
    store.focusGraph();
    expect(seen).toEqual(["n1", "n2", "graph"]);

    unsubscribe();
    store.openNote("n3");
    expect(seen).toEqual(["n1", "n2", "graph"]); // no more notifications
  });

  it("restoring a tab whose note vanished still shows it until reconciled by the caller", () => {
    // TabStore itself doesn't know which note ids are valid; the app layer
    // calls setMissing() after cross-checking against GET /api/notes. This
    // test documents that restore alone leaves missing=false, and setMissing
    // flips it — exercising the "restored tab whose node vanished" path.
    const backend = new FakeStorage();
    const storage = new NamespacedStorage("ws", backend);
    new TabStore(storage).openNote("ghost-note");

    const restored = new TabStore(storage);
    expect(restored.getState().tabs[1]).toEqual({ kind: "note", id: "ghost-note", missing: false });

    restored.setMissing("ghost-note", true);
    expect(restored.getState().tabs[1]).toEqual({ kind: "note", id: "ghost-note", missing: true });
  });
});

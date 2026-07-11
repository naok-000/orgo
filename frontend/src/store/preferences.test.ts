import { describe, expect, it } from "vitest";
import {
  DEFAULT_GRAPH_PREFS,
  DEFAULT_READING_PREFS,
  PreferencesStore,
  resolveInitialTheme,
  sanitizeGraphPrefs,
  sanitizeReadingPrefs,
} from "./preferences.ts";
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

describe("resolveInitialTheme", () => {
  it("prefers a persisted value over the OS preference", () => {
    expect(resolveInitialTheme("light", true)).toBe("light");
    expect(resolveInitialTheme("dark", false)).toBe("dark");
  });

  it("falls back to the OS preference when nothing is persisted", () => {
    expect(resolveInitialTheme(undefined, true)).toBe("dark");
    expect(resolveInitialTheme(undefined, false)).toBe("light");
  });

  it("ignores a corrupt persisted value", () => {
    expect(resolveInitialTheme("blue", true)).toBe("dark");
  });
});

describe("sanitizeReadingPrefs / sanitizeGraphPrefs", () => {
  it("returns defaults for missing/invalid data", () => {
    expect(sanitizeReadingPrefs(undefined)).toEqual(DEFAULT_READING_PREFS);
    expect(sanitizeReadingPrefs({ width: "huge" })).toEqual(DEFAULT_READING_PREFS);
    expect(sanitizeGraphPrefs(undefined)).toEqual(DEFAULT_GRAPH_PREFS);
    expect(sanitizeGraphPrefs({ depth: 5 })).toEqual(DEFAULT_GRAPH_PREFS);
  });

  it("preserves valid fields", () => {
    expect(sanitizeReadingPrefs({ width: "wide", fontSize: "l" })).toEqual({
      width: "wide",
      fontSize: "l",
    });
    expect(
      sanitizeGraphPrefs({ depth: 2, nodeScale: 1.5, linkWidth: 2.25 }),
    ).toEqual({ depth: 2, nodeScale: 1.5, linkWidth: 2.25 });
  });

  it("round-trips values that are already sane", () => {
    const sane = { depth: 2, nodeScale: 0.5, linkWidth: 3 } as const;
    expect(sanitizeGraphPrefs(sanitizeGraphPrefs(sane))).toEqual(sane);
  });

  it("clamps out-of-range numbers to the slider ranges", () => {
    expect(sanitizeGraphPrefs({ nodeScale: 10 }).nodeScale).toBe(2);
    expect(sanitizeGraphPrefs({ nodeScale: 0.1 }).nodeScale).toBe(0.5);
    expect(sanitizeGraphPrefs({ linkWidth: 99 }).linkWidth).toBe(3);
    expect(sanitizeGraphPrefs({ linkWidth: -1 }).linkWidth).toBe(0.5);
  });

  it("rejects NaN/Infinity/non-numbers, falling back to defaults", () => {
    expect(sanitizeGraphPrefs({ nodeScale: NaN }).nodeScale).toBe(1);
    expect(sanitizeGraphPrefs({ nodeScale: Infinity }).nodeScale).toBe(1);
    expect(sanitizeGraphPrefs({ nodeScale: -Infinity }).nodeScale).toBe(1);
    expect(sanitizeGraphPrefs({ nodeScale: "1.5" }).nodeScale).toBe(1);
    expect(sanitizeGraphPrefs({ linkWidth: NaN }).linkWidth).toBe(1);
    expect(sanitizeGraphPrefs({ linkWidth: "2" }).linkWidth).toBe(1);
    expect(sanitizeGraphPrefs({ linkWidth: true }).linkWidth).toBe(1);
    expect(sanitizeGraphPrefs({ linkWidth: null }).linkWidth).toBe(1);
  });
});

describe("PreferencesStore", () => {
  it("persists theme/reading/graph prefs and restores them", () => {
    const backend = new FakeStorage();
    const store1 = new PreferencesStore(new NamespacedStorage("ws", backend), false);
    store1.setTheme("dark");
    store1.setReading({ width: "wide" });
    store1.setGraph({ depth: 2, nodeScale: 1.5 });

    const store2 = new PreferencesStore(new NamespacedStorage("ws", backend), false);
    expect(store2.getTheme()).toBe("dark");
    expect(store2.getReading()).toEqual({ width: "wide", fontSize: "m" });
    expect(store2.getGraph()).toEqual({ depth: 2, nodeScale: 1.5, linkWidth: 1 });
  });

  it("merges partial updates into existing reading prefs", () => {
    const store = new PreferencesStore(new NamespacedStorage("ws", new FakeStorage()), false);
    store.setReading({ fontSize: "l" });
    expect(store.getReading()).toEqual({ width: "narrow", fontSize: "l" });
    store.setReading({ width: "wide" });
    expect(store.getReading()).toEqual({ width: "wide", fontSize: "l" });
  });

  it("sanitizes the merged result in setGraph, not just restored data", () => {
    const store = new PreferencesStore(new NamespacedStorage("ws", new FakeStorage()), false);
    store.setGraph({ nodeScale: 99 });
    expect(store.getGraph().nodeScale).toBe(2);
    store.setGraph({ linkWidth: NaN });
    expect(store.getGraph().linkWidth).toBe(1);
    // Prior valid fields survive a later partial update with a bad value.
    expect(store.getGraph().nodeScale).toBe(2);
  });

  it("persists the sanitized graph prefs, so restoration matches", () => {
    const backend = new FakeStorage();
    const store1 = new PreferencesStore(new NamespacedStorage("ws", backend), false);
    store1.setGraph({ nodeScale: 0.01, linkWidth: Infinity });

    const store2 = new PreferencesStore(new NamespacedStorage("ws", backend), false);
    expect(store2.getGraph()).toEqual({ depth: 1, nodeScale: 0.5, linkWidth: 1 });
  });
});

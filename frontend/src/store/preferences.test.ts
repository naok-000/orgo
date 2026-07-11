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
    expect(sanitizeGraphPrefs({ depth: 5 })).toEqual({ depth: 1 });
  });

  it("preserves valid fields", () => {
    expect(sanitizeReadingPrefs({ width: "wide", fontSize: "l" })).toEqual({
      width: "wide",
      fontSize: "l",
    });
    expect(sanitizeGraphPrefs({ depth: 2 })).toEqual({ depth: 2 });
  });
});

describe("PreferencesStore", () => {
  it("persists theme/reading/graph prefs and restores them", () => {
    const backend = new FakeStorage();
    const store1 = new PreferencesStore(new NamespacedStorage("ws", backend), false);
    store1.setTheme("dark");
    store1.setReading({ width: "wide" });
    store1.setGraph({ depth: 2 });

    const store2 = new PreferencesStore(new NamespacedStorage("ws", backend), false);
    expect(store2.getTheme()).toBe("dark");
    expect(store2.getReading()).toEqual({ width: "wide", fontSize: "m" });
    expect(store2.getGraph()).toEqual({ depth: 2 });
  });

  it("merges partial updates into existing reading prefs", () => {
    const store = new PreferencesStore(new NamespacedStorage("ws", new FakeStorage()), false);
    store.setReading({ fontSize: "l" });
    expect(store.getReading()).toEqual({ width: "narrow", fontSize: "l" });
    store.setReading({ width: "wide" });
    expect(store.getReading()).toEqual({ width: "wide", fontSize: "l" });
  });
});

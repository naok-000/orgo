import { describe, expect, it } from "vitest";
import { NamespacedStorage, namespacedKey, type StorageLike } from "./storage.ts";

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
  keys(): string[] {
    return [...this.map.keys()];
  }
}

describe("namespacedKey", () => {
  it("prefixes keys with orgo:<workspaceId>:", () => {
    expect(namespacedKey("ws1", "tabs")).toBe("orgo:ws1:tabs");
  });
});

describe("NamespacedStorage", () => {
  it("round-trips JSON values", () => {
    const backend = new FakeStorage();
    const storage = new NamespacedStorage("ws1", backend);
    storage.setJSON("tabs", { a: 1, b: [1, 2, 3] });
    expect(storage.getJSON("tabs")).toEqual({ a: 1, b: [1, 2, 3] });
  });

  it("returns undefined for missing keys", () => {
    const storage = new NamespacedStorage("ws1", new FakeStorage());
    expect(storage.getJSON("nope")).toBeUndefined();
    expect(storage.getString("nope")).toBeUndefined();
  });

  it("returns undefined instead of throwing on corrupt JSON", () => {
    const backend = new FakeStorage();
    backend.setItem("orgo:ws1:tabs", "{not json");
    const storage = new NamespacedStorage("ws1", backend);
    expect(storage.getJSON("tabs")).toBeUndefined();
  });

  it("round-trips string values", () => {
    const storage = new NamespacedStorage("ws1", new FakeStorage());
    storage.setString("theme", "dark");
    expect(storage.getString("theme")).toBe("dark");
  });

  it("removes keys", () => {
    const storage = new NamespacedStorage("ws1", new FakeStorage());
    storage.setString("theme", "dark");
    storage.remove("theme");
    expect(storage.getString("theme")).toBeUndefined();
  });

  it("namespaces by workspaceId so two workspaces never collide", () => {
    const backend = new FakeStorage();
    const wsA = new NamespacedStorage("workspace-a", backend);
    const wsB = new NamespacedStorage("workspace-b", backend);

    wsA.setJSON("tabs", { active: "graph" });
    wsB.setJSON("tabs", { active: "note-1" });

    expect(wsA.getJSON("tabs")).toEqual({ active: "graph" });
    expect(wsB.getJSON("tabs")).toEqual({ active: "note-1" });
    expect(backend.keys().sort()).toEqual(["orgo:workspace-a:tabs", "orgo:workspace-b:tabs"]);
  });

  it("defaults to globalThis.localStorage when no backend is supplied", () => {
    const storage = new NamespacedStorage("default-backend-test");
    storage.setString("k", "v");
    expect(globalThis.localStorage.getItem("orgo:default-backend-test:k")).toBe("v");
    globalThis.localStorage.removeItem("orgo:default-backend-test:k");
  });
});

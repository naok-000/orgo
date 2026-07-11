// Namespaced localStorage wrapper. Keys are namespaced by workspaceId
// (from GET /api/meta) so two org-roam directories opened at different
// times never bleed state into each other, per docs/DESIGN.md.

export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

export function namespacedKey(workspaceId: string, key: string): string {
  return `orgo:${workspaceId}:${key}`;
}

export class NamespacedStorage {
  constructor(
    private readonly workspaceId: string,
    private readonly backend: StorageLike = globalThis.localStorage,
  ) {}

  getJSON<T>(key: string): T | undefined {
    const raw = this.backend.getItem(namespacedKey(this.workspaceId, key));
    if (raw == null) return undefined;
    try {
      return JSON.parse(raw) as T;
    } catch {
      return undefined;
    }
  }

  setJSON<T>(key: string, value: T): void {
    this.backend.setItem(
      namespacedKey(this.workspaceId, key),
      JSON.stringify(value),
    );
  }

  getString(key: string): string | undefined {
    return this.backend.getItem(namespacedKey(this.workspaceId, key)) ?? undefined;
  }

  setString(key: string, value: string): void {
    this.backend.setItem(namespacedKey(this.workspaceId, key), value);
  }

  remove(key: string): void {
    this.backend.removeItem(namespacedKey(this.workspaceId, key));
  }
}

import { describe, expect, it, vi } from "vitest";
import { NoteCache } from "./noteCache.ts";
import type { NoteDetail } from "../types.ts";

function note(id: string, title = id): NoteDetail {
  return {
    id,
    title,
    file: `${id}.org`,
    level: 0,
    tags: [],
    aliases: [],
    refs: [],
    html: `<p>${title}</p>`,
    backlinks: [],
  };
}

function deferred<T>(): {
  promise: Promise<T>;
  resolve: (v: T) => void;
  reject: (e: unknown) => void;
} {
  let resolve!: (v: T) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("NoteCache", () => {
  it("caches successful fetches; peek returns the cached value without fetching", async () => {
    const fetcher = vi.fn().mockResolvedValue(note("n1"));
    const cache = new NoteCache(fetcher);

    expect(cache.peek("n1")).toBeUndefined();
    await cache.fetch("n1");
    expect(cache.peek("n1")).toEqual(note("n1"));
    expect(fetcher).toHaveBeenCalledTimes(1);

    // A second fetch is served from... well, fetch always goes through the
    // in-flight/cache discipline; but peek alone never triggers the fetcher.
    cache.peek("n1");
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("de-duplicates concurrent fetches for the same id", async () => {
    const d = deferred<NoteDetail>();
    const fetcher = vi.fn().mockReturnValue(d.promise);
    const cache = new NoteCache(fetcher);

    const p1 = cache.fetch("n1");
    const p2 = cache.fetch("n1");
    expect(p1).toBe(p2);
    expect(fetcher).toHaveBeenCalledTimes(1);

    d.resolve(note("n1"));
    await p1;
    expect(cache.peek("n1")).toEqual(note("n1"));
  });

  it("does not cache failures, so the next fetch retries", async () => {
    const fetcher = vi
      .fn()
      .mockRejectedValueOnce(new Error("boom"))
      .mockResolvedValueOnce(note("n1"));
    const cache = new NoteCache(fetcher);

    await expect(cache.fetch("n1")).rejects.toThrow("boom");
    expect(cache.peek("n1")).toBeUndefined();

    await expect(cache.fetch("n1")).resolves.toEqual(note("n1"));
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it("invalidate(id) drops one entry only", async () => {
    const fetcher = vi.fn().mockImplementation((id: string) => Promise.resolve(note(id)));
    const cache = new NoteCache(fetcher);
    await cache.fetch("n1");
    await cache.fetch("n2");

    cache.invalidate("n1");
    expect(cache.peek("n1")).toBeUndefined();
    expect(cache.peek("n2")).toEqual(note("n2"));
  });

  it("invalidateAll() empties the cache so every id re-fetches (live reload)", async () => {
    const fetcher = vi.fn().mockImplementation((id: string) => Promise.resolve(note(id)));
    const cache = new NoteCache(fetcher);
    await cache.fetch("n1");
    await cache.fetch("n2");
    expect(fetcher).toHaveBeenCalledTimes(2);

    cache.invalidateAll();
    expect(cache.peek("n1")).toBeUndefined();
    expect(cache.peek("n2")).toBeUndefined();

    await cache.fetch("n1");
    await cache.fetch("n2");
    expect(fetcher).toHaveBeenCalledTimes(4);
  });

  it("after invalidateAll(), a deleted note's re-fetch surfaces the 404 instead of stale content", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(note("n1", "old title"))
      .mockRejectedValueOnce(Object.assign(new Error("note not found"), { status: 404 }));
    const cache = new NoteCache(fetcher);

    await cache.fetch("n1"); // opened before the reload
    expect(cache.peek("n1")?.title).toBe("old title");

    cache.invalidateAll(); // note deleted on disk, server re-indexed

    expect(cache.peek("n1")).toBeUndefined(); // no stale fast path
    await expect(cache.fetch("n1")).rejects.toThrow("note not found");
    expect(cache.peek("n1")).toBeUndefined();
  });

  it("a fetch in flight when invalidateAll() fires does not re-populate the cache", async () => {
    const d = deferred<NoteDetail>();
    const fetcher = vi.fn().mockReturnValueOnce(d.promise);
    const cache = new NoteCache(fetcher);

    const stale = cache.fetch("n1"); // request leaves before the reload event
    cache.invalidateAll(); // reload arrives while it is in flight
    d.resolve(note("n1", "pre-reload content"));
    await stale; // caller still gets a value (their request completed)...

    expect(cache.peek("n1")).toBeUndefined(); // ...but it is not cached
  });

  it("a fetch started after invalidateAll() issues a fresh request instead of joining the stale one", async () => {
    const dStale = deferred<NoteDetail>();
    const fetcher = vi
      .fn()
      .mockReturnValueOnce(dStale.promise)
      .mockResolvedValueOnce(note("n1", "fresh"));
    const cache = new NoteCache(fetcher);

    void cache.fetch("n1").catch(() => {}); // in flight
    cache.invalidateAll();

    const freshPromise = cache.fetch("n1");
    expect(fetcher).toHaveBeenCalledTimes(2); // did not join the stale in-flight request

    dStale.resolve(note("n1", "stale"));
    const fresh = await freshPromise;
    expect(fresh.title).toBe("fresh");
    expect(cache.peek("n1")?.title).toBe("fresh");
  });

  it("a stale in-flight completion does not clobber a newer in-flight entry", async () => {
    const dStale = deferred<NoteDetail>();
    const dFresh = deferred<NoteDetail>();
    const fetcher = vi
      .fn()
      .mockReturnValueOnce(dStale.promise)
      .mockReturnValueOnce(dFresh.promise);
    const cache = new NoteCache(fetcher);

    const stale = cache.fetch("n1");
    cache.invalidateAll();
    const fresh = cache.fetch("n1");

    // The stale request resolves first; its cleanup must not remove the
    // fresh request's in-flight entry (a third fetch would then start).
    dStale.resolve(note("n1", "stale"));
    await stale;
    expect(cache.fetch("n1")).toBe(fresh);
    expect(fetcher).toHaveBeenCalledTimes(2);

    dFresh.resolve(note("n1", "fresh"));
    await fresh;
    expect(cache.peek("n1")?.title).toBe("fresh");
  });
});

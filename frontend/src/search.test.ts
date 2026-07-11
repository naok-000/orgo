import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createSearchController } from "./search.ts";
import type { SearchResult } from "./types.ts";

function result(id: string): SearchResult {
  return { id, title: id, snippet: `snippet for ${id}` };
}

describe("createSearchController", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("debounces rapid queries into a single fetch after the delay", async () => {
    const fetcher = vi.fn().mockResolvedValue([result("a")]);
    const onResult = vi.fn();
    const controller = createSearchController(fetcher, onResult, vi.fn(), 150);

    controller.query("h");
    controller.query("he");
    controller.query("hel");
    vi.advanceTimersByTime(149);
    expect(fetcher).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    await vi.runAllTimersAsync();

    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher).toHaveBeenCalledWith("hel");
    expect(onResult).toHaveBeenCalledWith("hel", [result("a")]);
  });

  it("calls onResult with [] synchronously for an empty query, skipping the fetch", () => {
    const fetcher = vi.fn().mockResolvedValue([result("a")]);
    const onResult = vi.fn();
    const controller = createSearchController(fetcher, onResult, vi.fn(), 150);

    controller.query("   ");
    expect(fetcher).not.toHaveBeenCalled();
    expect(onResult).toHaveBeenCalledWith("   ", []);
  });

  it("drops a stale response that resolves after a newer query", async () => {
    const deferredA = deferred<SearchResult[]>();
    const deferredB = deferred<SearchResult[]>();
    const fetcher = vi.fn().mockImplementationOnce(() => deferredA.promise).mockImplementationOnce(() => deferredB.promise);
    const onResult = vi.fn();
    const controller = createSearchController(fetcher, onResult, vi.fn(), 150);

    controller.query("a");
    await vi.advanceTimersByTimeAsync(150);
    controller.query("ab");
    await vi.advanceTimersByTimeAsync(150);
    expect(fetcher).toHaveBeenCalledTimes(2);

    // Resolve out of order: the newer query ("ab") resolves first, then the
    // stale one ("a") resolves later — its result must be ignored.
    deferredB.resolve([result("ab-result")]);
    await Promise.resolve();
    deferredA.resolve([result("a-result")]);
    await Promise.resolve();

    expect(onResult).toHaveBeenCalledTimes(1);
    expect(onResult).toHaveBeenCalledWith("ab", [result("ab-result")]);
  });

  it("routes fetch errors to onError, scoped to the still-current query", async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error("boom"));
    const onError = vi.fn();
    const controller = createSearchController(fetcher, vi.fn(), onError, 150);

    controller.query("x");
    await vi.advanceTimersByTimeAsync(150);
    await Promise.resolve();

    expect(onError).toHaveBeenCalledWith("x", expect.any(Error));
  });

  it("dispose() cancels a pending debounce so the fetcher never runs", async () => {
    const fetcher = vi.fn().mockResolvedValue([]);
    const controller = createSearchController(fetcher, vi.fn(), vi.fn(), 150);

    controller.query("x");
    controller.dispose();
    await vi.advanceTimersByTimeAsync(500);

    expect(fetcher).not.toHaveBeenCalled();
  });
});

function deferred<T>(): { promise: Promise<T>; resolve: (v: T) => void } {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

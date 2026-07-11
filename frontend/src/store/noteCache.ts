// Note-detail cache with in-flight request de-duplication and whole-cache
// invalidation. Extracted from the app layer so the live-reload semantics
// ("after a reload event, focusing ANY tab must re-fetch, and a deleted
// note must surface its 404") are unit-testable without the DOM.

import type { NoteDetail } from "../types.ts";

export type NoteFetcher = (id: string) => Promise<NoteDetail>;

export class NoteCache {
  private cache = new Map<string, NoteDetail>();
  private inFlight = new Map<string, Promise<NoteDetail>>();
  /** Bumped by invalidateAll(); responses from an older generation are
   *  discarded so a fetch that was already in flight when the reload event
   *  arrived can never re-populate the cache with pre-reload content. */
  private generation = 0;

  constructor(private readonly fetcher: NoteFetcher) {}

  /** Cached value, if any. Never triggers a fetch. */
  peek(id: string): NoteDetail | undefined {
    return this.cache.get(id);
  }

  /**
   * Fetch (or join the in-flight fetch for) a note. Successful results are
   * cached; failures are not, so the next call retries.
   */
  fetch(id: string): Promise<NoteDetail> {
    const existing = this.inFlight.get(id);
    if (existing) return existing;

    const gen = this.generation;
    const p = this.fetcher(id).then(
      (note) => {
        if (this.generation === gen) this.cache.set(id, note);
        if (this.inFlight.get(id) === p) this.inFlight.delete(id);
        return note;
      },
      (err: unknown) => {
        if (this.inFlight.get(id) === p) this.inFlight.delete(id);
        throw err;
      },
    );
    this.inFlight.set(id, p);
    return p;
  }

  /** Drop one cached entry so the next fetch() hits the network. */
  invalidate(id: string): void {
    this.cache.delete(id);
  }

  /**
   * Drop everything (live reload): cached entries, and in-flight joins —
   * fetches started after this point issue fresh requests, and responses
   * from fetches started before it are not cached.
   */
  invalidateAll(): void {
    this.generation++;
    this.cache.clear();
    this.inFlight.clear();
  }
}

// Debounced search controller backing the Ctrl-K command palette.
// Pure w.r.t. the DOM: takes a fetcher function and result/error callbacks,
// owns only a timer + a sequence counter (so a slow, stale response can
// never clobber a faster, newer one).

import type { SearchResult } from "./types.ts";

export interface SearchController {
  query(q: string): void;
  dispose(): void;
}

export type SearchFetcher = (q: string) => Promise<SearchResult[]>;

export function createSearchController(
  fetcher: SearchFetcher,
  onResult: (q: string, results: SearchResult[]) => void,
  onError: (q: string, err: unknown) => void,
  delayMs = 150,
): SearchController {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let seq = 0;

  function query(q: string): void {
    if (timer !== undefined) {
      clearTimeout(timer);
      timer = undefined;
    }
    const mySeq = ++seq;

    if (q.trim() === "") {
      onResult(q, []);
      return;
    }

    timer = setTimeout(() => {
      fetcher(q).then(
        (results) => {
          if (mySeq === seq) onResult(q, results);
        },
        (err: unknown) => {
          if (mySeq === seq) onError(q, err);
        },
      );
    }, delayMs);
  }

  function dispose(): void {
    if (timer !== undefined) clearTimeout(timer);
  }

  return { query, dispose };
}

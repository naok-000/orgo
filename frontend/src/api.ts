// Thin fetch wrappers over the HTTP API documented in docs/DESIGN.md.
// Kept dependency-free (no axios etc.) to keep the bundle lean.

import type {
  GraphResponse,
  Meta,
  NoteDetail,
  NoteSummary,
  SearchResult,
} from "./types.ts";

const BASE = "/api";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`;
    try {
      const body: unknown = await res.json();
      if (
        body &&
        typeof body === "object" &&
        "error" in body &&
        typeof (body as { error: unknown }).error === "string"
      ) {
        message = (body as { error: string }).error;
      }
    } catch {
      // response wasn't JSON; fall back to statusText
    }
    throw new ApiError(res.status, message);
  }
  return res.json() as Promise<T>;
}

export function getMeta(): Promise<Meta> {
  return getJSON<Meta>("/meta");
}

export function getGraph(): Promise<GraphResponse> {
  return getJSON<GraphResponse>("/graph");
}

export function getNotes(): Promise<NoteSummary[]> {
  return getJSON<NoteSummary[]>("/notes");
}

export function getNote(id: string): Promise<NoteDetail> {
  return getJSON<NoteDetail>(`/note/${encodeURIComponent(id)}`);
}

export function search(q: string): Promise<SearchResult[]> {
  return getJSON<SearchResult[]>(`/search?q=${encodeURIComponent(q)}`);
}

export interface EventsHandle {
  close(): void;
}

/**
 * Subscribe to GET /api/events (SSE). Calls onReload() on the `reload`
 * event. EventSource reconnects natively on transport errors; we also
 * surface onConnectionChange so the UI can show a subtle "reconnecting"
 * hint if the server is gone for a while.
 */
export function subscribeEvents(
  onReload: () => void,
  onConnectionChange?: (connected: boolean) => void,
): EventsHandle {
  const es = new EventSource(`${BASE}/events`);
  es.addEventListener("open", () => onConnectionChange?.(true));
  es.addEventListener("reload", () => onReload());
  es.addEventListener("error", () => onConnectionChange?.(false));
  return {
    close(): void {
      es.close();
    },
  };
}

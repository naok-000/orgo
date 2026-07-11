// Pure sidebar list helpers: filter-as-you-type + sort toggle.

import type { NoteSummary } from "./types.ts";

export type SortMode = "title" | "mtime";

export function filterNotes(
  notes: NoteSummary[],
  query: string,
): NoteSummary[] {
  const q = query.trim().toLowerCase();
  if (!q) return notes;
  return notes.filter(
    (n) =>
      n.title.toLowerCase().includes(q) ||
      n.tags.some((t) => t.toLowerCase().includes(q)),
  );
}

export function sortNotes(
  notes: NoteSummary[],
  mode: SortMode,
): NoteSummary[] {
  const copy = notes.slice();
  if (mode === "title") {
    copy.sort((a, b) => a.title.localeCompare(b.title));
  } else {
    // most recently modified first
    copy.sort(
      (a, b) => new Date(b.mtime).getTime() - new Date(a.mtime).getTime(),
    );
  }
  return copy;
}

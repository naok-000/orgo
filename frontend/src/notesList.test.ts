import { describe, expect, it } from "vitest";
import { filterNotes, sortNotes } from "./notesList.ts";
import type { NoteSummary } from "./types.ts";

function note(partial: Partial<NoteSummary>): NoteSummary {
  return { id: "id", title: "title", file: "f.org", tags: [], mtime: "2026-01-01T00:00:00Z", ...partial };
}

describe("filterNotes", () => {
  const notes = [
    note({ id: "1", title: "Go (programming language)", tags: ["dev", "lang"] }),
    note({ id: "2", title: "org-roam", tags: ["tools", "pkm"] }),
    note({ id: "3", title: "Zettelkasten", tags: ["pkm"] }),
  ];

  it("returns all notes for an empty query", () => {
    expect(filterNotes(notes, "")).toEqual(notes);
    expect(filterNotes(notes, "   ")).toEqual(notes);
  });

  it("matches by title, case-insensitively", () => {
    expect(filterNotes(notes, "GO").map((n) => n.id)).toEqual(["1"]);
  });

  it("matches by tag", () => {
    expect(filterNotes(notes, "pkm").map((n) => n.id).sort()).toEqual(["2", "3"]);
  });

  it("returns an empty list when nothing matches", () => {
    expect(filterNotes(notes, "nonexistent")).toEqual([]);
  });
});

describe("sortNotes", () => {
  const notes = [
    note({ id: "b", title: "Banana", mtime: "2026-01-02T00:00:00Z" }),
    note({ id: "a", title: "Apple", mtime: "2026-01-03T00:00:00Z" }),
    note({ id: "c", title: "Cherry", mtime: "2026-01-01T00:00:00Z" }),
  ];

  it("sorts by title", () => {
    expect(sortNotes(notes, "title").map((n) => n.id)).toEqual(["a", "b", "c"]);
  });

  it("sorts by mtime, most recent first", () => {
    expect(sortNotes(notes, "mtime").map((n) => n.id)).toEqual(["a", "b", "c"]);
  });

  it("does not mutate the input array", () => {
    const copy = [...notes];
    sortNotes(notes, "title");
    expect(notes).toEqual(copy);
  });
});

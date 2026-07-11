import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, getGraph, getMeta, getNote, getNotes, search } from "./api.ts";

function jsonResponse(body: unknown, init: { status?: number } = {}): Response {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: { "content-type": "application/json" },
  });
}

describe("api client (mocked fetch)", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("getMeta hits /api/meta and returns the parsed body", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ root: "/notes", workspaceId: "ws1", version: "0.1.0", noteCount: 3 }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const meta = await getMeta();
    expect(fetchMock).toHaveBeenCalledWith("/api/meta");
    expect(meta).toEqual({ root: "/notes", workspaceId: "ws1", version: "0.1.0", noteCount: 3 });
  });

  it("getGraph hits /api/graph", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ nodes: [], edges: [] }));
    vi.stubGlobal("fetch", fetchMock);

    await getGraph();
    expect(fetchMock).toHaveBeenCalledWith("/api/graph");
  });

  it("getNotes hits /api/notes", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse([]));
    vi.stubGlobal("fetch", fetchMock);

    await getNotes();
    expect(fetchMock).toHaveBeenCalledWith("/api/notes");
  });

  it("getNote URL-encodes the id", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        id: "a b",
        title: "t",
        file: "f.org",
        level: 0,
        tags: [],
        aliases: [],
        refs: [],
        html: "<p></p>",
        backlinks: [],
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await getNote("a b");
    expect(fetchMock).toHaveBeenCalledWith("/api/note/a%20b");
  });

  it("search encodes the query string", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse([]));
    vi.stubGlobal("fetch", fetchMock);

    await search("foo bar");
    expect(fetchMock).toHaveBeenCalledWith("/api/search?q=foo%20bar");
  });

  it("throws ApiError with the server-provided message on non-2xx", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ error: "note not found" }, { status: 404 }),
    );
    vi.stubGlobal("fetch", fetchMock);

    try {
      await getNote("missing");
      expect.unreachable();
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(404);
      expect((err as ApiError).message).toBe("note not found");
    }
  });

  it("falls back to statusText when the error body isn't JSON", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("internal error", { status: 500, statusText: "Internal Server Error" }),
    );
    vi.stubGlobal("fetch", fetchMock);

    try {
      await getMeta();
      expect.unreachable();
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(500);
    }
  });
});

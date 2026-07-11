import { describe, expect, it } from "vitest";
import { formatRoute, parseHash, routesEqual, type Route } from "./router.ts";

describe("parseHash", () => {
  it("parses empty / root hashes as empty", () => {
    expect(parseHash("")).toEqual({ type: "empty" });
    expect(parseHash("#")).toEqual({ type: "empty" });
    expect(parseHash("#/")).toEqual({ type: "empty" });
  });

  it("parses a note route", () => {
    expect(parseHash("#/note/abc-123")).toEqual({ type: "note", id: "abc-123" });
  });

  it("works without the leading #", () => {
    expect(parseHash("/note/abc-123")).toEqual({ type: "note", id: "abc-123" });
  });

  it("decodes URI-encoded ids", () => {
    expect(parseHash("#/note/foo%20bar")).toEqual({ type: "note", id: "foo bar" });
  });

  it("parses global graph route", () => {
    expect(parseHash("#/graph")).toEqual({ type: "graph" });
  });

  it("parses local graph route", () => {
    expect(parseHash("#/graph/node-1")).toEqual({ type: "graph-local", id: "node-1" });
  });

  it("falls back to empty for a note route with no id", () => {
    expect(parseHash("#/note/")).toEqual({ type: "empty" });
    expect(parseHash("#/note")).toEqual({ type: "empty" });
  });

  it("falls back to empty for a graph route with an empty id segment", () => {
    expect(parseHash("#/graph/")).toEqual({ type: "graph" });
  });

  it("falls back to empty for unknown shapes rather than throwing", () => {
    expect(parseHash("#/something/else/entirely")).toEqual({ type: "empty" });
    expect(parseHash("#garbage")).toEqual({ type: "empty" });
  });
});

describe("formatRoute", () => {
  it("formats each route type", () => {
    expect(formatRoute({ type: "empty" })).toBe("#/");
    expect(formatRoute({ type: "graph" })).toBe("#/graph");
    expect(formatRoute({ type: "graph-local", id: "n1" })).toBe("#/graph/n1");
    expect(formatRoute({ type: "note", id: "n1" })).toBe("#/note/n1");
  });

  it("encodes ids that need it", () => {
    expect(formatRoute({ type: "note", id: "foo bar" })).toBe("#/note/foo%20bar");
  });

  it("round-trips through parseHash", () => {
    const routes: Route[] = [
      { type: "graph" },
      { type: "graph-local", id: "aaaa-1111" },
      { type: "note", id: "aaaa-1111" },
    ];
    for (const r of routes) {
      expect(parseHash(formatRoute(r))).toEqual(r);
    }
  });
});

describe("routesEqual", () => {
  it("compares by type and id", () => {
    expect(routesEqual({ type: "graph" }, { type: "graph" })).toBe(true);
    expect(routesEqual({ type: "note", id: "a" }, { type: "note", id: "a" })).toBe(true);
    expect(routesEqual({ type: "note", id: "a" }, { type: "note", id: "b" })).toBe(false);
    expect(routesEqual({ type: "note", id: "a" }, { type: "graph" })).toBe(false);
    expect(
      routesEqual({ type: "graph-local", id: "a" }, { type: "graph-local", id: "a" }),
    ).toBe(true);
    expect(
      routesEqual({ type: "graph-local", id: "a" }, { type: "graph-local", id: "b" }),
    ).toBe(false);
  });
});

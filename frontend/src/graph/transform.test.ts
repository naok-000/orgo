import { describe, expect, it } from "vitest";
import {
  localNeighborhood,
  neighborIds,
  nodeHasTags,
  nodeRadius,
  toForceGraphData,
} from "./transform.ts";
import type { GraphNode, GraphResponse } from "../types.ts";

function node(id: string, opts: Partial<GraphNode> = {}): GraphNode {
  return {
    id,
    title: id,
    file: `${id}.org`,
    level: 0,
    tags: [],
    nlinks: 0,
    ...opts,
  };
}

// Fixture: a chain A - B - C - D, plus an isolated node E.
// (A)---(B)---(C)---(D)   (E)
function chainGraph(): GraphResponse {
  return {
    nodes: [node("A"), node("B"), node("C"), node("D"), node("E")],
    edges: [
      { source: "A", target: "B" },
      { source: "B", target: "C" },
      { source: "C", target: "D" },
    ],
  };
}

describe("toForceGraphData", () => {
  it("maps edges to links, keeping nodes as-is", () => {
    const graph: GraphResponse = {
      nodes: [node("A"), node("B")],
      edges: [{ source: "A", target: "B" }],
    };
    expect(toForceGraphData(graph)).toEqual({
      nodes: [node("A"), node("B")],
      links: [{ source: "A", target: "B" }],
    });
  });
});

describe("nodeRadius", () => {
  it("grows with nlinks but never shrinks below the base size", () => {
    expect(nodeRadius(0)).toBeGreaterThan(0);
    expect(nodeRadius(4)).toBeGreaterThan(nodeRadius(0));
    expect(nodeRadius(16)).toBeGreaterThan(nodeRadius(4));
  });

  it("is stable for negative input (defensive)", () => {
    expect(nodeRadius(-5)).toBe(nodeRadius(0));
  });
});

describe("nodeHasTags", () => {
  it("reflects whether the node has any tags", () => {
    expect(nodeHasTags(node("A", { tags: [] }))).toBe(false);
    expect(nodeHasTags(node("A", { tags: ["dev"] }))).toBe(true);
  });
});

describe("localNeighborhood", () => {
  it("depth 1 includes only direct neighbors", () => {
    const result = localNeighborhood(chainGraph(), "B", 1);
    expect(result.nodes.map((n) => n.id).sort()).toEqual(["A", "B", "C"]);
    expect(result.edges).toEqual([
      { source: "A", target: "B" },
      { source: "B", target: "C" },
    ]);
  });

  it("depth 2 expands to two hops", () => {
    const result = localNeighborhood(chainGraph(), "B", 2);
    expect(result.nodes.map((n) => n.id).sort()).toEqual(["A", "B", "C", "D"]);
  });

  it("treats edges as undirected for neighborhood purposes", () => {
    // D only appears as an edge *target*; centering on D should still reach C.
    const result = localNeighborhood(chainGraph(), "D", 1);
    expect(result.nodes.map((n) => n.id).sort()).toEqual(["C", "D"]);
  });

  it("returns just the node itself when it has no links", () => {
    const result = localNeighborhood(chainGraph(), "E", 2);
    expect(result.nodes.map((n) => n.id)).toEqual(["E"]);
    expect(result.edges).toEqual([]);
  });

  it("returns an empty graph for an id that doesn't exist", () => {
    const result = localNeighborhood(chainGraph(), "ghost", 2);
    expect(result).toEqual({ nodes: [], edges: [] });
  });

  it("only keeps edges whose both endpoints are within the neighborhood", () => {
    const result = localNeighborhood(chainGraph(), "A", 1);
    expect(result.nodes.map((n) => n.id).sort()).toEqual(["A", "B"]);
    expect(result.edges).toEqual([{ source: "A", target: "B" }]);
  });
});

describe("neighborIds", () => {
  it("collects direct neighbors from both edge directions", () => {
    expect([...neighborIds(chainGraph(), "B")].sort()).toEqual(["A", "C"]);
    expect([...neighborIds(chainGraph(), "A")].sort()).toEqual(["B"]);
    expect([...neighborIds(chainGraph(), "E")].sort()).toEqual([]);
  });
});

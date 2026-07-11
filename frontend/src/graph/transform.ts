// Pure transforms from the /api/graph response shape into what the
// `force-graph` package + our graph view need. No DOM, no force-graph
// import here, so this is trivially unit-testable.

import type { GraphEdge, GraphNode, GraphResponse } from "../types.ts";

export interface ForceGraphLink {
  source: string;
  target: string;
}

export interface ForceGraphData {
  nodes: GraphNode[];
  links: ForceGraphLink[];
}

/** Adapt the API's {nodes, edges} shape to force-graph's {nodes, links}. */
export function toForceGraphData(graph: GraphResponse): ForceGraphData {
  return {
    nodes: graph.nodes,
    links: graph.edges.map((e) => ({ source: e.source, target: e.target })),
  };
}

/**
 * Drawn node radius (px), gently scaled by link count and capped so hub
 * nodes never dwarf the graph. This is the actual radius we want on screen.
 */
export function baseNodeRadius(nlinks: number): number {
  return Math.min(8, 2 + Math.sqrt(Math.max(0, nlinks)) * 0.9);
}

/**
 * force-graph's `nodeVal` is AREA-proportional: drawn radius =
 * `Math.sqrt(nodeVal) * nodeRelSize`. Squaring baseNodeRadius here means
 * that with `nodeRelSize(scale)` the drawn radius comes out to exactly
 * `baseNodeRadius(nlinks) * scale`.
 */
export function nodeVal(nlinks: number): number {
  return baseNodeRadius(nlinks) ** 2;
}

export function nodeHasTags(node: GraphNode): boolean {
  return node.tags.length > 0;
}

function buildUndirectedAdjacency(
  edges: GraphEdge[],
): Map<string, Set<string>> {
  const adj = new Map<string, Set<string>>();
  const link = (a: string, b: string) => {
    let set = adj.get(a);
    if (!set) {
      set = new Set();
      adj.set(a, set);
    }
    set.add(b);
  };
  for (const e of edges) {
    link(e.source, e.target);
    link(e.target, e.source);
  }
  return adj;
}

/**
 * Extract the local neighborhood of `centerId` up to `depth` hops
 * (undirected — a node linked in either direction counts as a neighbor).
 * Returns an empty graph if centerId isn't present among graph.nodes.
 */
export function localNeighborhood(
  graph: GraphResponse,
  centerId: string,
  depth: 1 | 2,
): GraphResponse {
  if (!graph.nodes.some((n) => n.id === centerId)) {
    return { nodes: [], edges: [] };
  }

  const adjacency = buildUndirectedAdjacency(graph.edges);
  const visited = new Set<string>([centerId]);
  let frontier = new Set<string>([centerId]);

  for (let d = 0; d < depth; d++) {
    const next = new Set<string>();
    for (const id of frontier) {
      for (const nb of adjacency.get(id) ?? []) {
        if (!visited.has(nb)) {
          visited.add(nb);
          next.add(nb);
        }
      }
    }
    frontier = next;
  }

  const nodes = graph.nodes.filter((n) => visited.has(n.id));
  const edges = graph.edges.filter(
    (e) => visited.has(e.source) && visited.has(e.target),
  );
  return { nodes, edges };
}

/** ids of a node's direct neighbors (undirected), used for select+highlight. */
export function neighborIds(graph: GraphResponse, id: string): Set<string> {
  const result = new Set<string>();
  for (const e of graph.edges) {
    if (e.source === id) result.add(e.target);
    else if (e.target === id) result.add(e.source);
  }
  return result;
}

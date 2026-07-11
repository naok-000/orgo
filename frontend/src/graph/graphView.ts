// Thin wrapper around the `force-graph` canvas widget. Owns the ForceGraph
// instance and translates our GraphResponse data + selection state into its
// accessor functions. Not unit tested (it's all canvas/DOM); kept as thin as
// possible so the interesting logic (transform.ts) stays pure and tested.

import ForceGraph from "force-graph";
import type { LinkObject } from "force-graph";
import type { GraphNode, GraphResponse } from "../types.ts";
import { baseNodeRadius, neighborIds, nodeHasTags, nodeVal } from "./transform.ts";

export interface GraphColors {
  background: string;
  link: string;
  linkDim: string;
  nodeTagged: string;
  nodeUntagged: string;
  nodeDim: string;
  nodeSelected: string;
  label: string;
}

export interface GraphViewCallbacks {
  /** Single click: select/highlight only, never mutates tabs. */
  onSelect: (id: string | null) => void;
  /** Double click: open (or focus) that note's tab. */
  onOpen: (id: string) => void;
}

export interface GraphStyle {
  /** Multiplier on the drawn node radius (force-graph nodeRelSize). */
  nodeScale: number;
  /** Link stroke width in px. */
  linkWidth: number;
}

const DOUBLE_CLICK_MS = 350;
/** Nodes with at least this many links get a persistent text label. */
const LABEL_NLINKS_THRESHOLD = 5;

export class GraphView {
  private readonly fg: ForceGraph<GraphNode>;
  private data: GraphResponse = { nodes: [], edges: [] };
  private selectedId: string | null = null;
  private neighbors: Set<string> = new Set();
  private lastClick: { id: string; time: number } | null = null;
  private nodeScale = 1;

  constructor(
    container: HTMLElement,
    private callbacks: GraphViewCallbacks,
    private colors: GraphColors,
  ) {
    this.fg = new ForceGraph<GraphNode>(container)
      .nodeId("id")
      .nodeRelSize(this.nodeScale)
      .nodeVal((n) => nodeVal(n.nlinks))
      .linkWidth(1)
      .nodeLabel((n) => escapeHtml(n.title))
      .nodeColor((n) => this.colorForNode(n))
      .linkColor((l) => this.colorForLink(l))
      .backgroundColor(this.colors.background)
      .nodeCanvasObjectMode("after")
      .nodeCanvasObject((n, ctx, globalScale) => this.drawLabel(n, ctx, globalScale))
      .onNodeClick((n) => this.handleNodeClick(n))
      .onBackgroundClick(() => this.select(null))
      // The graph is a small, local, single-user visualization; always
      // redrawing keeps selection/theme updates responsive without reaching
      // into force-graph's private redraw-scheduling internals.
      .autoPauseRedraw(false);
  }

  setData(data: GraphResponse): void {
    this.data = data;
    if (this.selectedId && !data.nodes.some((n) => n.id === this.selectedId)) {
      this.selectedId = null;
      this.neighbors = new Set();
    }
    this.fg.graphData({
      nodes: data.nodes,
      links: data.edges.map((e) => ({ source: e.source, target: e.target })),
    });
  }

  select(id: string | null): void {
    this.selectedId = id;
    this.neighbors = id ? neighborIds(this.data, id) : new Set();
    this.callbacks.onSelect(id);
  }

  zoomToFit(durationMs = 400, padding = 40): void {
    this.fg.zoomToFit(durationMs, padding);
  }

  setColors(colors: GraphColors): void {
    this.colors = colors;
    this.fg.backgroundColor(colors.background);
  }

  /** Live-update node/link sizing from user preferences. */
  setStyle(style: GraphStyle): void {
    this.nodeScale = style.nodeScale;
    this.fg.nodeRelSize(style.nodeScale).linkWidth(style.linkWidth);
  }

  resize(width: number, height: number): void {
    this.fg.width(width).height(height);
  }

  private handleNodeClick(node: GraphNode): void {
    const now = Date.now();
    if (
      this.lastClick &&
      this.lastClick.id === node.id &&
      now - this.lastClick.time < DOUBLE_CLICK_MS
    ) {
      this.lastClick = null;
      this.callbacks.onOpen(node.id);
      return;
    }
    this.lastClick = { id: node.id, time: now };
    this.select(node.id);
  }

  private colorForNode(n: GraphNode): string {
    const base = nodeHasTags(n) ? this.colors.nodeTagged : this.colors.nodeUntagged;
    if (!this.selectedId) return base;
    if (n.id === this.selectedId) return this.colors.nodeSelected;
    if (this.neighbors.has(n.id)) return base;
    return this.colors.nodeDim;
  }

  private colorForLink(l: LinkObject<GraphNode>): string {
    if (!this.selectedId) return this.colors.link;
    const sid = idOf(l.source);
    const tid = idOf(l.target);
    if (sid === this.selectedId || tid === this.selectedId) return this.colors.link;
    return this.colors.linkDim;
  }

  private drawLabel(
    node: GraphNode & { x?: number; y?: number },
    ctx: CanvasRenderingContext2D,
    globalScale: number,
  ): void {
    const shouldLabel =
      node.id === this.selectedId ||
      this.neighbors.has(node.id) ||
      node.nlinks >= LABEL_NLINKS_THRESHOLD;
    if (!shouldLabel || node.x === undefined || node.y === undefined) return;

    const fontSize = Math.max(10 / globalScale, 3);
    ctx.font = `${fontSize}px system-ui, sans-serif`;
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    ctx.fillStyle = this.colors.label;
    const r = baseNodeRadius(node.nlinks) * this.nodeScale;
    ctx.fillText(node.title, node.x, node.y + r + 1);
  }
}

function idOf(v: unknown): string | undefined {
  if (typeof v === "string") return v;
  if (v && typeof v === "object" && "id" in v) {
    const id = (v as { id?: unknown }).id;
    return typeof id === "string" ? id : undefined;
  }
  return undefined;
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

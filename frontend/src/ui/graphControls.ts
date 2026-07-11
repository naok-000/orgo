// Graph controls overlay: depth buttons, node-size / link-width sliders,
// zoom-to-fit, the local-filter chip ("Filtered: <title> · depth N" +
// "Show full graph"), and the node-selection label. Pure DOM + callbacks;
// main.ts owns the state and calls render() on route/preference changes.

import {
  LINK_WIDTH_RANGE,
  NODE_SCALE_RANGE,
  type GraphDepth,
} from "../store/preferences.ts";
import { el } from "./dom.ts";

const DEPTH_VALUES: GraphDepth[] = [1, 2];

export interface GraphFilter {
  /** Center node id of the local neighborhood. */
  id: string;
  /** Resolved title from the full graph, or null when the id is unknown. */
  title: string | null;
}

export interface GraphControlsState {
  /** Non-null in graph-local mode. */
  filter: GraphFilter | null;
  depth: GraphDepth;
  nodeScale: number;
  linkWidth: number;
}

export interface GraphControlsCallbacks {
  onDepthChange: (depth: GraphDepth) => void;
  onStyleChange: (style: { nodeScale: number; linkWidth: number }) => void;
  onZoomFit: () => void;
  onShowFullGraph: () => void;
}

function clamp(
  v: number,
  range: { min: number; max: number },
  fallback: number,
): number {
  if (!Number.isFinite(v)) return fallback;
  return Math.min(range.max, Math.max(range.min, v));
}

function buildSlider(
  label: string,
  range: { min: number; max: number },
  step: number,
  onInput: () => void,
): { wrap: HTMLElement; input: HTMLInputElement } {
  const input = el("input", {
    className: "graph-slider-input",
    attrs: {
      type: "range",
      min: String(range.min),
      max: String(range.max),
      step: String(step),
      "aria-label": label,
    },
    on: { input: onInput },
  });
  const wrap = el(
    "label",
    { className: "graph-slider", attrs: { title: label } },
    [el("span", { className: "graph-slider-label", text: label }), input],
  );
  return { wrap, input };
}

export class GraphControls {
  readonly root: HTMLElement;
  private readonly depthGroupEl: HTMLElement;
  private readonly depthButtons = new Map<GraphDepth, HTMLButtonElement>();
  private readonly nodeScaleInput: HTMLInputElement;
  private readonly linkWidthInput: HTMLInputElement;
  private readonly filterChipEl: HTMLElement;
  private readonly filterTextEl: HTMLElement;
  private readonly selectionLabelEl: HTMLElement;

  constructor(private readonly callbacks: GraphControlsCallbacks) {
    this.depthGroupEl = el("div", { className: "btn-group graph-depth-group" });
    for (const d of DEPTH_VALUES) {
      const btn = el("button", {
        attrs: { type: "button" },
        text: `Depth ${d}`,
      });
      btn.addEventListener("click", () => this.callbacks.onDepthChange(d));
      this.depthButtons.set(d, btn);
      this.depthGroupEl.append(btn);
    }

    const nodeScale = buildSlider("Node size", NODE_SCALE_RANGE, 0.1, () =>
      this.emitStyle(),
    );
    const linkWidth = buildSlider("Link width", LINK_WIDTH_RANGE, 0.25, () =>
      this.emitStyle(),
    );
    this.nodeScaleInput = nodeScale.input;
    this.linkWidthInput = linkWidth.input;

    const zoomBtn = el("button", {
      className: "graph-zoom-fit",
      attrs: { type: "button", title: "Zoom to fit" },
      text: "Zoom to fit",
    });
    zoomBtn.addEventListener("click", () => this.callbacks.onZoomFit());

    const showFullBtn = el("button", {
      className: "graph-show-full",
      attrs: { type: "button" },
      text: "Show full graph",
    });
    showFullBtn.addEventListener("click", () => this.callbacks.onShowFullGraph());

    this.filterTextEl = el("span", { className: "graph-filter-text" });
    this.filterChipEl = el("div", { className: "graph-filter-chip" }, [
      this.filterTextEl,
      showFullBtn,
    ]);

    this.selectionLabelEl = el("div", { className: "graph-selection-label" });

    this.root = el("div", { className: "graph-controls" }, [
      this.filterChipEl,
      this.depthGroupEl,
      nodeScale.wrap,
      linkWidth.wrap,
      zoomBtn,
      this.selectionLabelEl,
    ]);
  }

  render(state: GraphControlsState): void {
    const local = state.filter !== null;
    this.depthGroupEl.classList.toggle("visible", local);
    for (const [d, btn] of this.depthButtons) {
      btn.classList.toggle("active", d === state.depth);
    }

    this.filterChipEl.classList.toggle("visible", local);
    if (state.filter) {
      const title = state.filter.title ?? state.filter.id;
      this.filterTextEl.textContent = `Filtered: ${title} · depth ${state.depth}`;
    } else {
      this.filterTextEl.textContent = "";
    }

    const ns = String(state.nodeScale);
    if (this.nodeScaleInput.value !== ns) this.nodeScaleInput.value = ns;
    const lw = String(state.linkWidth);
    if (this.linkWidthInput.value !== lw) this.linkWidthInput.value = lw;
  }

  /** Overwritten by node selection; independent of the filter chip. */
  setSelectionText(text: string): void {
    this.selectionLabelEl.textContent = text;
  }

  private emitStyle(): void {
    this.callbacks.onStyleChange({
      nodeScale: clamp(parseFloat(this.nodeScaleInput.value), NODE_SCALE_RANGE, 1),
      linkWidth: clamp(parseFloat(this.linkWidthInput.value), LINK_WIDTH_RANGE, 1),
    });
  }
}

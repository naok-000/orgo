// DOM interaction tests (jsdom) for the graph controls overlay: filter chip
// visibility/content, depth buttons, sliders, and callback wiring.

import { beforeEach, describe, expect, it, vi } from "vitest";
import { GraphControls, type GraphControlsState } from "./graphControls.ts";

function makeControls() {
  const callbacks = {
    onDepthChange: vi.fn(),
    onStyleChange: vi.fn(),
    onZoomFit: vi.fn(),
    onShowFullGraph: vi.fn(),
  };
  const controls = new GraphControls(callbacks);
  document.body.append(controls.root);
  return { controls, callbacks };
}

function state(overrides: Partial<GraphControlsState> = {}): GraphControlsState {
  return { filter: null, depth: 1, nodeScale: 1, linkWidth: 1, ...overrides };
}

function chip(root: HTMLElement): HTMLElement {
  const elm = root.querySelector<HTMLElement>(".graph-filter-chip");
  if (!elm) throw new Error("filter chip not found");
  return elm;
}

function slider(root: HTMLElement, label: string): HTMLInputElement {
  const input = root.querySelector<HTMLInputElement>(`input[aria-label="${label}"]`);
  if (!input) throw new Error(`slider "${label}" not found`);
  return input;
}

beforeEach(() => {
  document.body.replaceChildren();
});

describe("GraphControls filter chip", () => {
  it("is hidden in full-graph mode", () => {
    const { controls } = makeControls();
    controls.render(state({ filter: null }));
    expect(chip(controls.root).classList.contains("visible")).toBe(false);
    expect(controls.root.querySelector(".graph-depth-group")?.classList.contains("visible")).toBe(
      false,
    );
  });

  it("shows title, depth, and the Show-full-graph button in local mode", () => {
    const { controls } = makeControls();
    controls.render(state({ filter: { id: "n1", title: "My Note" }, depth: 2 }));

    const c = chip(controls.root);
    expect(c.classList.contains("visible")).toBe(true);
    expect(c.textContent).toContain("Filtered: My Note");
    expect(c.textContent).toContain("depth 2");
    const btn = c.querySelector("button.graph-show-full");
    expect(btn?.textContent).toBe("Show full graph");
    expect(controls.root.querySelector(".graph-depth-group")?.classList.contains("visible")).toBe(
      true,
    );
  });

  it("falls back to the raw id when the title is unknown", () => {
    const { controls } = makeControls();
    controls.render(state({ filter: { id: "raw-id-123", title: null } }));
    expect(chip(controls.root).textContent).toContain("Filtered: raw-id-123");
  });

  it("updates the displayed depth when re-rendered after a depth change", () => {
    const { controls, callbacks } = makeControls();
    controls.render(state({ filter: { id: "n1", title: "My Note" }, depth: 1 }));
    expect(chip(controls.root).textContent).toContain("depth 1");

    const depthBtns = controls.root.querySelectorAll<HTMLButtonElement>(
      ".graph-depth-group button",
    );
    depthBtns[1]!.click();
    expect(callbacks.onDepthChange).toHaveBeenCalledWith(2);

    // main.ts responds to the callback by re-rendering with the new depth.
    controls.render(state({ filter: { id: "n1", title: "My Note" }, depth: 2 }));
    expect(chip(controls.root).textContent).toContain("depth 2");
    expect(depthBtns[1]!.classList.contains("active")).toBe(true);
    expect(depthBtns[0]!.classList.contains("active")).toBe(false);
  });

  it("fires onShowFullGraph when the button is clicked", () => {
    const { controls, callbacks } = makeControls();
    controls.render(state({ filter: { id: "n1", title: "My Note" } }));
    chip(controls.root).querySelector<HTMLButtonElement>("button.graph-show-full")!.click();
    expect(callbacks.onShowFullGraph).toHaveBeenCalledTimes(1);
  });

  it("hides the chip again when navigating back to the full graph", () => {
    const { controls } = makeControls();
    controls.render(state({ filter: { id: "n1", title: "My Note" } }));
    controls.render(state({ filter: null }));
    expect(chip(controls.root).classList.contains("visible")).toBe(false);
  });
});

describe("GraphControls sliders", () => {
  it("reflects rendered pref values", () => {
    const { controls } = makeControls();
    controls.render(state({ nodeScale: 1.5, linkWidth: 2.25 }));
    expect(slider(controls.root, "Node size").value).toBe("1.5");
    expect(slider(controls.root, "Link width").value).toBe("2.25");
  });

  it("fires onStyleChange with both values when a slider moves", () => {
    const { controls, callbacks } = makeControls();
    controls.render(state());

    const nodeSize = slider(controls.root, "Node size");
    nodeSize.value = "1.5";
    nodeSize.dispatchEvent(new Event("input", { bubbles: true }));
    expect(callbacks.onStyleChange).toHaveBeenLastCalledWith({
      nodeScale: 1.5,
      linkWidth: 1,
    });

    const linkWidth = slider(controls.root, "Link width");
    linkWidth.value = "2.75";
    linkWidth.dispatchEvent(new Event("input", { bubbles: true }));
    expect(callbacks.onStyleChange).toHaveBeenLastCalledWith({
      nodeScale: 1.5,
      linkWidth: 2.75,
    });
  });

  it("clamps out-of-range slider values before firing the callback", () => {
    const { controls, callbacks } = makeControls();
    controls.render(state());

    const nodeSize = slider(controls.root, "Node size");
    nodeSize.value = "999";
    nodeSize.dispatchEvent(new Event("input", { bubbles: true }));
    expect(callbacks.onStyleChange).toHaveBeenLastCalledWith({
      nodeScale: 2,
      linkWidth: 1,
    });

    const linkWidth = slider(controls.root, "Link width");
    linkWidth.value = "0.001";
    linkWidth.dispatchEvent(new Event("input", { bubbles: true }));
    expect(callbacks.onStyleChange).toHaveBeenLastCalledWith({
      nodeScale: 2,
      linkWidth: 0.5,
    });
  });
});

describe("GraphControls zoom + selection", () => {
  it("fires onZoomFit when Zoom to fit is clicked", () => {
    const { controls, callbacks } = makeControls();
    controls.root.querySelector<HTMLButtonElement>("button.graph-zoom-fit")!.click();
    expect(callbacks.onZoomFit).toHaveBeenCalledTimes(1);
  });

  it("keeps the selection label separate from the filter chip", () => {
    const { controls } = makeControls();
    controls.render(state({ filter: { id: "n1", title: "My Note" } }));
    controls.setSelectionText("Another Node");

    expect(controls.root.querySelector(".graph-selection-label")?.textContent).toBe(
      "Another Node",
    );
    expect(chip(controls.root).textContent).toContain("Filtered: My Note");
  });
});

// Theme + reading + graph preferences. Small pure helpers plus a thin
// persisted wrapper, mirroring the tabs store pattern.

import type { NamespacedStorage } from "./storage.ts";

export type Theme = "light" | "dark";
export type ContentWidth = "narrow" | "wide";
export type FontSize = "s" | "m" | "l";
export type GraphDepth = 1 | 2;

export interface ReadingPrefs {
  width: ContentWidth;
  fontSize: FontSize;
}

export interface GraphPrefs {
  depth: GraphDepth;
  /** Multiplier on the drawn node radius. */
  nodeScale: number;
  /** Link stroke width in px. */
  linkWidth: number;
}

export const DEFAULT_READING_PREFS: ReadingPrefs = {
  width: "narrow",
  fontSize: "m",
};

export const DEFAULT_GRAPH_PREFS: GraphPrefs = {
  depth: 1,
  nodeScale: 1,
  linkWidth: 1,
};

/** Slider ranges — sanitizeGraphPrefs clamps to these, the UI mirrors them. */
export const NODE_SCALE_RANGE = { min: 0.5, max: 2 } as const;
export const LINK_WIDTH_RANGE = { min: 0.5, max: 3 } as const;

/** Pure: resolve the initial theme from persisted value + OS preference. */
export function resolveInitialTheme(
  stored: string | undefined,
  prefersDark: boolean,
): Theme {
  if (stored === "light" || stored === "dark") return stored;
  return prefersDark ? "dark" : "light";
}

function isFontSize(v: unknown): v is FontSize {
  return v === "s" || v === "m" || v === "l";
}

function isContentWidth(v: unknown): v is ContentWidth {
  return v === "narrow" || v === "wide";
}

export function sanitizeReadingPrefs(data: unknown): ReadingPrefs {
  if (!data || typeof data !== "object") return { ...DEFAULT_READING_PREFS };
  const d = data as Partial<ReadingPrefs>;
  return {
    width: isContentWidth(d.width) ? d.width : DEFAULT_READING_PREFS.width,
    fontSize: isFontSize(d.fontSize)
      ? d.fontSize
      : DEFAULT_READING_PREFS.fontSize,
  };
}

/** Clamp v into [min, max] when it is a finite number; else the default. */
function clampNumber(
  v: unknown,
  range: { min: number; max: number },
  fallback: number,
): number {
  if (typeof v !== "number" || !Number.isFinite(v)) return fallback;
  return Math.min(range.max, Math.max(range.min, v));
}

export function sanitizeGraphPrefs(data: unknown): GraphPrefs {
  if (!data || typeof data !== "object") return { ...DEFAULT_GRAPH_PREFS };
  const d = data as Partial<GraphPrefs>;
  return {
    depth: d.depth === 2 ? 2 : 1,
    nodeScale: clampNumber(
      d.nodeScale,
      NODE_SCALE_RANGE,
      DEFAULT_GRAPH_PREFS.nodeScale,
    ),
    linkWidth: clampNumber(
      d.linkWidth,
      LINK_WIDTH_RANGE,
      DEFAULT_GRAPH_PREFS.linkWidth,
    ),
  };
}

export class PreferencesStore {
  private theme: Theme;
  private reading: ReadingPrefs;
  private graph: GraphPrefs;

  constructor(
    private readonly storage: NamespacedStorage,
    prefersDark: boolean,
  ) {
    this.theme = resolveInitialTheme(storage.getString("theme"), prefersDark);
    this.reading = sanitizeReadingPrefs(storage.getJSON("reading"));
    this.graph = sanitizeGraphPrefs(storage.getJSON("graph"));
  }

  getTheme(): Theme {
    return this.theme;
  }

  setTheme(theme: Theme): void {
    this.theme = theme;
    this.storage.setString("theme", theme);
  }

  getReading(): ReadingPrefs {
    return this.reading;
  }

  setReading(prefs: Partial<ReadingPrefs>): void {
    this.reading = { ...this.reading, ...prefs };
    this.storage.setJSON("reading", this.reading);
  }

  getGraph(): GraphPrefs {
    return this.graph;
  }

  setGraph(prefs: Partial<GraphPrefs>): void {
    // Sanitize the merged result, not just what came from localStorage, so
    // the store invariant (clamped, finite numbers) holds for callers too.
    this.graph = sanitizeGraphPrefs({ ...this.graph, ...prefs });
    this.storage.setJSON("graph", this.graph);
  }
}

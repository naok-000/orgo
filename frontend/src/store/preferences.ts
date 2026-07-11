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
}

export const DEFAULT_READING_PREFS: ReadingPrefs = {
  width: "narrow",
  fontSize: "m",
};

export const DEFAULT_GRAPH_PREFS: GraphPrefs = { depth: 1 };

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

export function sanitizeGraphPrefs(data: unknown): GraphPrefs {
  if (!data || typeof data !== "object") return { ...DEFAULT_GRAPH_PREFS };
  const d = data as Partial<GraphPrefs>;
  return { depth: d.depth === 2 ? 2 : 1 };
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
    this.graph = { ...this.graph, ...prefs };
    this.storage.setJSON("graph", this.graph);
  }
}

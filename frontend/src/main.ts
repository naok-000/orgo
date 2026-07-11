import { ApiError, getGraph, getMeta, getNote, getNotes, search, subscribeEvents } from "./api.ts";
import type { GraphResponse, Meta, NoteSummary } from "./types.ts";
import { formatRoute, parseHash, type Route } from "./router.ts";
import { NamespacedStorage } from "./store/storage.ts";
import { NoteCache } from "./store/noteCache.ts";
import { TabStore } from "./store/tabs.ts";
import {
  PreferencesStore,
  type ContentWidth,
  type FontSize,
  type GraphDepth,
  type ReadingPrefs,
  type Theme,
} from "./store/preferences.ts";
import type { SortMode } from "./notesList.ts";
import { localNeighborhood } from "./graph/transform.ts";
import { GraphView, type GraphColors } from "./graph/graphView.ts";
import { Sidebar } from "./ui/sidebar.ts";
import { TabBar } from "./ui/tabbar.ts";
import { NoteView } from "./ui/noteView.ts";
import { CommandPalette } from "./ui/commandPalette.ts";
import { GraphControls, type GraphFilter } from "./ui/graphControls.ts";
import { el, clear } from "./ui/dom.ts";

const GRAPH_COLORS: Record<Theme, GraphColors> = {
  light: {
    background: "#ffffff",
    link: "rgba(70,75,90,0.35)",
    linkDim: "rgba(70,75,90,0.08)",
    nodeTagged: "#3b6fd6",
    nodeUntagged: "#9096a3",
    nodeDim: "rgba(140,144,155,0.25)",
    nodeSelected: "#e0592a",
    label: "#2b2f38",
  },
  dark: {
    background: "#14161c",
    link: "rgba(200,204,220,0.28)",
    linkDim: "rgba(200,204,220,0.07)",
    nodeTagged: "#6fa8ff",
    nodeUntagged: "#9298a5",
    nodeDim: "rgba(140,144,155,0.2)",
    nodeSelected: "#ff9d5c",
    label: "#dfe2ea",
  },
};

const WIDTH_VALUES: ContentWidth[] = ["narrow", "wide"];
const FONT_VALUES: FontSize[] = ["s", "m", "l"];

class App {
  private storage!: NamespacedStorage;
  private prefs!: PreferencesStore;
  private tabStore!: TabStore;

  private notes: NoteSummary[] = [];
  private notesById = new Map<string, NoteSummary>();
  private fullGraph: GraphResponse = { nodes: [], edges: [] };
  private readonly noteCache = new NoteCache((id) => getNote(id));

  private currentRoute: Route = { type: "empty" };
  private lastGraphRoute: Route = { type: "graph" };
  private filterText = "";
  private sortMode: SortMode = "title";
  private sidebarCollapsed = false;

  private sidebar!: Sidebar;
  private tabbar!: TabBar;
  private noteView!: NoteView;
  private graphView!: GraphView;
  private graphControls!: GraphControls;
  private palette!: CommandPalette;

  private graphPaneEl!: HTMLElement;
  private notePaneEl!: HTMLElement;
  private emptyPaneEl!: HTMLElement;
  private connectionDotEl!: HTMLElement;
  private themeToggleBtn!: HTMLButtonElement;
  private readonly widthButtons = new Map<ContentWidth, HTMLButtonElement>();
  private readonly fontButtons = new Map<FontSize, HTMLButtonElement>();

  async boot(): Promise<void> {
    const root = document.getElementById("app");
    if (!root) return;
    root.append(el("div", { className: "boot-loading", text: "Loading orgo…" }));

    let meta: Meta;
    try {
      meta = await getMeta();
    } catch {
      clear(root);
      root.append(
        el("div", { className: "state state-error boot-error" }, [
          el("h1", { text: "orgo" }),
          el("p", { text: "Could not reach the orgo server. Is it running?" }),
        ]),
      );
      return;
    }
    this.storage = new NamespacedStorage(meta.workspaceId);
    const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? false;
    this.prefs = new PreferencesStore(this.storage, prefersDark);
    this.tabStore = new TabStore(this.storage);
    this.sidebarCollapsed = this.storage.getString("sidebarCollapsed") === "1";

    clear(root);
    this.buildShell(root);
    this.applyTheme(this.prefs.getTheme());
    this.applyReadingPrefs(this.prefs.getReading());

    this.tabStore.subscribe(() => this.renderTabsAndSidebar());

    window.addEventListener("hashchange", () => this.handleRouteChange(parseHash(location.hash)));
    window.addEventListener("keydown", (e) => this.handleGlobalKeydown(e));
    window.addEventListener("resize", () => this.resizeGraph());

    const notesPromise = this.loadNotes();
    const graphPromise = this.loadGraph();

    this.resolveInitialRoute();

    await Promise.all([notesPromise, graphPromise]);
    this.reconcileMissingTabs();
    this.renderTabsAndSidebar();

    subscribeEvents(
      () => void this.handleReload(),
      (connected) => this.setConnectionStatus(connected),
    );

    if (typeof ResizeObserver !== "undefined") {
      new ResizeObserver(() => this.resizeGraph()).observe(this.graphPaneEl);
    }
  }

  // ---- shell construction -------------------------------------------------

  private buildShell(root: HTMLElement): void {
    this.sidebar = new Sidebar({
      onFilterChange: (v) => {
        this.filterText = v;
        this.renderSidebar();
      },
      onSortChange: (s) => {
        this.sortMode = s;
        this.renderSidebar();
      },
      onToggleCollapse: () => {
        this.sidebarCollapsed = !this.sidebarCollapsed;
        this.storage.setString("sidebarCollapsed", this.sidebarCollapsed ? "1" : "0");
        this.renderSidebar();
      },
      onOpenNote: (id) => this.navigate({ type: "note", id }),
    });

    this.tabbar = new TabBar(
      {
        onFocus: (key) => this.focusTabKey(key),
        onClose: (id) => this.closeTab(id),
      },
      (id) => this.titleForNote(id),
    );

    this.noteView = new NoteView({
      onShowInGraph: (id) => this.navigate({ type: "graph-local", id }),
    });

    this.palette = new CommandPalette(
      { onSelect: (id) => this.navigate({ type: "note", id }) },
      (q) => search(q),
    );

    const graphCanvas = el("div", { className: "graph-canvas" });
    this.graphControls = new GraphControls({
      onDepthChange: (d) => this.setGraphDepth(d),
      onStyleChange: (style) => this.setGraphStyle(style),
      onZoomFit: () => this.graphView.zoomToFit(),
      onShowFullGraph: () => this.navigate({ type: "graph" }),
    });
    this.graphPaneEl = el("div", { className: "pane graph-pane" }, [
      graphCanvas,
      this.graphControls.root,
    ]);

    this.graphView = new GraphView(
      graphCanvas,
      {
        onSelect: (id) => this.updateGraphSelectionLabel(id),
        onOpen: (id) => this.navigate({ type: "note", id }),
      },
      GRAPH_COLORS[this.prefs.getTheme()],
    );
    this.applyGraphStyle();
    this.renderGraphControls();

    this.notePaneEl = el("div", { className: "pane note-pane" }, [this.noteView.root]);

    this.emptyPaneEl = el("div", { className: "pane empty-pane" }, [
      el("div", { className: "state state-empty" }, [
        el("h2", { text: "Nothing open" }),
        el("p", { text: "Open a note from the sidebar, or explore the graph." }),
      ]),
    ]);

    const content = el("div", { className: "content" }, [
      this.graphPaneEl,
      this.notePaneEl,
      this.emptyPaneEl,
    ]);

    const topbar = this.buildTopbar();
    const main = el("div", { className: "main" }, [topbar, content]);

    root.append(this.sidebar.root, main, this.palette.root);
  }

  private buildTopbar(): HTMLElement {
    const searchBtn = el("button", {
      className: "topbar-search",
      attrs: { type: "button", title: "Search (Ctrl+K)" },
      text: "Search",
    });
    searchBtn.addEventListener("click", () => this.palette.open());

    this.themeToggleBtn = el("button", {
      className: "topbar-theme",
      attrs: { type: "button", title: "Toggle theme" },
    }) as HTMLButtonElement;
    this.themeToggleBtn.addEventListener("click", () => this.toggleTheme());

    const widthGroup = el("div", { className: "btn-group" });
    for (const w of WIDTH_VALUES) {
      const btn = el("button", { attrs: { type: "button", title: `${w} content` }, text: w[0]?.toUpperCase() ?? w }) as HTMLButtonElement;
      btn.addEventListener("click", () => {
        this.prefs.setReading({ width: w });
        this.applyReadingPrefs(this.prefs.getReading());
      });
      this.widthButtons.set(w, btn);
      widthGroup.append(btn);
    }

    const fontGroup = el("div", { className: "btn-group" });
    for (const f of FONT_VALUES) {
      const btn = el("button", { attrs: { type: "button", title: `font size ${f}` }, text: f.toUpperCase() }) as HTMLButtonElement;
      btn.addEventListener("click", () => {
        this.prefs.setReading({ fontSize: f });
        this.applyReadingPrefs(this.prefs.getReading());
      });
      this.fontButtons.set(f, btn);
      fontGroup.append(btn);
    }

    this.connectionDotEl = el("span", {
      className: "connection-dot connected",
      attrs: { title: "Live reload connected" },
    });

    const controls = el("div", { className: "topbar-controls" }, [
      widthGroup,
      fontGroup,
      searchBtn,
      this.themeToggleBtn,
      this.connectionDotEl,
    ]);

    return el("div", { className: "topbar" }, [
      el("div", { className: "brand", text: "orgo" }),
      this.tabbar.root,
      controls,
    ]);
  }

  // ---- routing --------------------------------------------------------

  private resolveInitialRoute(): void {
    const initial = parseHash(location.hash);
    if (initial.type !== "empty") {
      this.handleRouteChange(initial);
      return;
    }
    const state = this.tabStore.getState();
    const route: Route = state.active === "graph" ? { type: "graph" } : { type: "note", id: state.active };
    history.replaceState(null, "", formatRoute(route));
    this.handleRouteChange(route);
  }

  private navigate(route: Route, opts: { replace?: boolean } = {}): void {
    const hash = formatRoute(route);
    if (opts.replace) {
      history.replaceState(null, "", hash);
      this.handleRouteChange(route);
      return;
    }
    if (location.hash === hash) {
      // Same URL (e.g. re-clicking the active tab): still process so any
      // stale UI state (like a missing-note pane) gets refreshed.
      this.handleRouteChange(route);
      return;
    }
    location.hash = hash;
  }

  private handleRouteChange(route: Route): void {
    this.currentRoute = route;
    switch (route.type) {
      case "note":
        this.tabStore.openNote(route.id);
        this.showPane("note");
        this.loadAndShowNote(route.id);
        break;
      case "graph":
        this.tabStore.focusGraph();
        this.lastGraphRoute = route;
        this.showPane("graph");
        this.renderGraphPane();
        break;
      case "graph-local":
        this.tabStore.focusGraph();
        this.lastGraphRoute = route;
        this.showPane("graph");
        this.renderGraphPane();
        break;
      case "empty":
        this.showPane("empty");
        break;
    }
    this.renderTabsAndSidebar();
  }

  private showPane(which: "note" | "graph" | "empty"): void {
    this.graphPaneEl.classList.toggle("visible", which === "graph");
    this.notePaneEl.classList.toggle("visible", which === "note");
    this.emptyPaneEl.classList.toggle("visible", which === "empty");
    if (which === "graph") this.resizeGraph();
  }

  private focusTabKey(key: string): void {
    if (key === "graph") {
      this.navigate(this.lastGraphRoute);
    } else {
      this.navigate({ type: "note", id: key });
    }
  }

  private closeTab(id: string): void {
    const wasActive = this.tabStore.getState().active === id;
    this.tabStore.closeNote(id);
    if (wasActive) {
      const state = this.tabStore.getState();
      const route: Route = state.active === "graph" ? this.lastGraphRoute : { type: "note", id: state.active };
      this.navigate(route, { replace: true });
    } else {
      this.renderTabsAndSidebar();
    }
  }

  // ---- notes ------------------------------------------------------------

  private async loadNotes(): Promise<void> {
    try {
      const notes = await getNotes();
      this.notes = notes;
      this.notesById = new Map(notes.map((n) => [n.id, n]));
      this.reconcileMissingTabs();
      this.renderTabsAndSidebar();
    } catch (err) {
      console.error("orgo: failed to load notes", err);
    }
  }

  private loadAndShowNote(id: string): void {
    const cached = this.noteCache.peek(id);
    if (cached) {
      this.noteView.showNote(cached);
      return;
    }
    this.noteView.showLoading();
    this.noteCache.fetch(id).then(
      (note) => {
        this.tabStore.setMissing(id, false);
        if (this.currentRoute.type === "note" && this.currentRoute.id === id) {
          this.noteView.showNote(note);
        }
      },
      (err: unknown) => {
        if (err instanceof ApiError && err.status === 404) {
          this.tabStore.setMissing(id, true);
        }
        if (!(this.currentRoute.type === "note" && this.currentRoute.id === id)) return;
        if (err instanceof ApiError && err.status === 404) {
          this.noteView.showMissing(id);
        } else {
          this.noteView.showError(err instanceof Error ? err.message : String(err));
        }
      },
    );
  }

  private titleForNote(id: string): string {
    return this.noteCache.peek(id)?.title ?? this.notesById.get(id)?.title ?? id;
  }

  private reconcileMissingTabs(): void {
    const state = this.tabStore.getState();
    for (const tab of state.tabs) {
      if (tab.kind === "note") {
        this.tabStore.setMissing(tab.id, !this.notesById.has(tab.id));
      }
    }
  }

  // ---- graph --------------------------------------------------------------

  private async loadGraph(): Promise<void> {
    try {
      this.fullGraph = await getGraph();
      if (this.currentRoute.type === "graph" || this.currentRoute.type === "graph-local") {
        this.renderGraphPane();
      }
    } catch (err) {
      console.error("orgo: failed to load graph", err);
    }
  }

  private renderGraphPane(): void {
    const route = this.currentRoute;
    let data: GraphResponse;
    let missingCenter: string | null = null;

    if (route.type === "graph-local") {
      const depth = this.prefs.getGraph().depth;
      data = localNeighborhood(this.fullGraph, route.id, depth);
      if (!this.fullGraph.nodes.some((n) => n.id === route.id)) missingCenter = route.id;
    } else {
      data = this.fullGraph;
    }

    this.graphView.setData(data);
    this.renderGraphControls();

    this.graphControls.setSelectionText(
      missingCenter ? `Note "${missingCenter}" not found.` : "",
    );

    requestAnimationFrame(() => this.graphView.zoomToFit(300));
  }

  private renderGraphControls(): void {
    const route = this.currentRoute;
    const filter: GraphFilter | null =
      route.type === "graph-local"
        ? {
            id: route.id,
            title: this.fullGraph.nodes.find((n) => n.id === route.id)?.title ?? null,
          }
        : null;
    const prefs = this.prefs.getGraph();
    this.graphControls.render({
      filter,
      depth: prefs.depth,
      nodeScale: prefs.nodeScale,
      linkWidth: prefs.linkWidth,
    });
  }

  private setGraphDepth(depth: GraphDepth): void {
    this.prefs.setGraph({ depth });
    if (this.currentRoute.type === "graph-local") {
      this.renderGraphPane();
    } else {
      this.renderGraphControls();
    }
  }

  private setGraphStyle(style: { nodeScale: number; linkWidth: number }): void {
    this.prefs.setGraph(style);
    this.applyGraphStyle();
    this.renderGraphControls();
  }

  private applyGraphStyle(): void {
    const { nodeScale, linkWidth } = this.prefs.getGraph();
    this.graphView.setStyle({ nodeScale, linkWidth });
  }

  private updateGraphSelectionLabel(id: string | null): void {
    if (!id) {
      this.graphControls.setSelectionText("");
      return;
    }
    const node = this.fullGraph.nodes.find((n) => n.id === id);
    this.graphControls.setSelectionText(node ? node.title : id);
  }

  private resizeGraph(): void {
    const rect = this.graphPaneEl.getBoundingClientRect();
    if (rect.width > 0 && rect.height > 0) this.graphView.resize(rect.width, rect.height);
  }

  // ---- live reload ----------------------------------------------------

  private async handleReload(): Promise<void> {
    // Anything cached may be stale after a re-index: drop the whole note
    // cache so focusing ANY tab (not just the active one) re-fetches — a
    // note deleted on disk then 404s and shows the missing state.
    this.noteCache.invalidateAll();
    await Promise.all([this.loadNotes(), this.loadGraph()]);
    if (this.currentRoute.type === "note") {
      this.loadAndShowNote(this.currentRoute.id);
    }
  }

  private setConnectionStatus(connected: boolean): void {
    this.connectionDotEl.classList.toggle("connected", connected);
    this.connectionDotEl.classList.toggle("disconnected", !connected);
    this.connectionDotEl.title = connected
      ? "Live reload connected"
      : "Reconnecting to orgo server…";
  }

  // ---- theme & reading prefs --------------------------------------------

  private toggleTheme(): void {
    const next: Theme = this.prefs.getTheme() === "dark" ? "light" : "dark";
    this.prefs.setTheme(next);
    this.applyTheme(next);
  }

  private applyTheme(theme: Theme): void {
    document.documentElement.setAttribute("data-theme", theme);
    this.themeToggleBtn.textContent = theme === "dark" ? "Dark" : "Light";
    this.graphView?.setColors(GRAPH_COLORS[theme]);
  }

  private applyReadingPrefs(prefs: ReadingPrefs): void {
    this.notePaneEl.classList.remove("width-narrow", "width-wide");
    this.notePaneEl.classList.add(`width-${prefs.width}`);
    this.notePaneEl.classList.remove("font-s", "font-m", "font-l");
    this.notePaneEl.classList.add(`font-${prefs.fontSize}`);
    for (const [w, btn] of this.widthButtons) btn.classList.toggle("active", w === prefs.width);
    for (const [f, btn] of this.fontButtons) btn.classList.toggle("active", f === prefs.fontSize);
  }

  // ---- rendering helpers --------------------------------------------------

  private renderSidebar(): void {
    this.sidebar.render({
      notes: this.notes,
      filter: this.filterText,
      sort: this.sortMode,
      collapsed: this.sidebarCollapsed,
      activeNoteId: this.currentRoute.type === "note" ? this.currentRoute.id : null,
    });
  }

  private renderTabsAndSidebar(): void {
    const state = this.tabStore.getState();
    this.tabbar.render(state.tabs, state.active);
    this.renderSidebar();
  }

  private handleGlobalKeydown(e: KeyboardEvent): void {
    const mod = e.ctrlKey || e.metaKey;
    if (mod && e.key.toLowerCase() === "k") {
      e.preventDefault();
      this.palette.toggle();
    }
  }
}

void new App().boot();

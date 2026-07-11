// Pure hash-route parsing/formatting. No DOM access here so it is trivially
// unit-testable; src/main.ts wires this to window.location/hashchange.

export type Route =
  | { type: "empty" }
  | { type: "note"; id: string }
  | { type: "graph" }
  | { type: "graph-local"; id: string };

/**
 * Parse a location hash (with or without the leading "#") into a Route.
 * Unknown shapes fall back to "empty" rather than throwing, so a stray or
 * hand-edited URL never crashes the app.
 */
export function parseHash(hash: string): Route {
  let h = hash.startsWith("#") ? hash.slice(1) : hash;
  // Normalize leading slash: "/note/x" -> ["note", "x"]
  if (h.startsWith("/")) h = h.slice(1);
  if (h === "") return { type: "empty" };

  const parts = h.split("/").filter((p) => p.length > 0);

  if (parts[0] === "graph") {
    if (parts.length === 1) return { type: "graph" };
    if (parts.length >= 2 && parts[1]) {
      return { type: "graph-local", id: decodeURIComponent(parts[1]) };
    }
    return { type: "empty" };
  }

  if (parts[0] === "note" && parts.length >= 2 && parts[1]) {
    return { type: "note", id: decodeURIComponent(parts[1]) };
  }

  return { type: "empty" };
}

/** Format a Route back into a location hash, e.g. "#/note/<id>". */
export function formatRoute(route: Route): string {
  switch (route.type) {
    case "note":
      return `#/note/${encodeURIComponent(route.id)}`;
    case "graph":
      return "#/graph";
    case "graph-local":
      return `#/graph/${encodeURIComponent(route.id)}`;
    case "empty":
      return "#/";
  }
}

export function routesEqual(a: Route, b: Route): boolean {
  if (a.type !== b.type) return false;
  if (a.type === "note" && b.type === "note") return a.id === b.id;
  if (a.type === "graph-local" && b.type === "graph-local")
    return a.id === b.id;
  return true;
}

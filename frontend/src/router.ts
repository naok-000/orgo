// Pure hash-route parsing/formatting. No DOM access here so it is trivially
// unit-testable; src/main.ts wires this to window.location/hashchange.

export type Route =
  | { type: "empty" }
  | { type: "note"; id: string }
  | { type: "graph" }
  | { type: "graph-local"; id: string };

/** decodeURIComponent that returns null instead of throwing on bad input. */
function safeDecode(s: string): string | null {
  try {
    return decodeURIComponent(s);
  } catch {
    return null;
  }
}

/**
 * Parse a location hash (with or without the leading "#") into a Route.
 *
 * Everything after "#/note/" (or "#/graph/") is treated as ONE
 * percent-encoded segment: the full remainder of the hash is taken verbatim
 * and then decoded. This keeps ids containing "/" working — the renderer
 * emits them percent-encoded (e.g. "#/note/area%2Fproject") — and a raw,
 * unencoded "/" in the remainder simply folds into the id rather than
 * being split into extra path segments.
 *
 * Unknown shapes and malformed percent-encoding (e.g. "#/note/%") fall back
 * to "empty" rather than throwing, so a stray or hand-edited URL never
 * crashes init or hashchange handling.
 */
export function parseHash(hash: string): Route {
  let h = hash.startsWith("#") ? hash.slice(1) : hash;
  // Normalize leading slash: "/note/x" -> "note/x"
  if (h.startsWith("/")) h = h.slice(1);
  if (h === "") return { type: "empty" };

  if (h === "graph" || h === "graph/") return { type: "graph" };

  if (h.startsWith("graph/")) {
    const id = safeDecode(h.slice("graph/".length));
    return id ? { type: "graph-local", id } : { type: "empty" };
  }

  if (h.startsWith("note/")) {
    const id = safeDecode(h.slice("note/".length));
    return id ? { type: "note", id } : { type: "empty" };
  }

  return { type: "empty" };
}

/**
 * Format a Route back into a location hash, e.g. "#/note/<id>". The id is
 * percent-encoded as a single segment, symmetric with parseHash, so
 * parseHash(formatRoute(r)) round-trips for any id — including ones
 * containing "/", spaces, or "%".
 */
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

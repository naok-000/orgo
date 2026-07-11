// Types mirroring the HTTP API contract documented in docs/DESIGN.md.

export interface Meta {
  root: string;
  workspaceId: string;
  version: string;
  noteCount: number;
}

export interface GraphNode {
  id: string;
  title: string;
  file: string;
  level: number;
  tags: string[];
  nlinks: number;
}

export interface GraphEdge {
  source: string;
  target: string;
}

export interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface NoteSummary {
  id: string;
  title: string;
  file: string;
  tags: string[];
  mtime: string;
}

export interface Backlink {
  id: string;
  title: string;
}

export interface NoteDetail {
  id: string;
  title: string;
  file: string;
  level: number;
  tags: string[];
  aliases: string[];
  refs: string[];
  html: string;
  backlinks: Backlink[];
}

export interface SearchResult {
  id: string;
  title: string;
  snippet: string;
}

export interface ApiErrorBody {
  error: string;
}

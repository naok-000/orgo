package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"time"

	"github.com/naok-000/orgo/internal/render"
)

type diagnosticsResponse struct {
	DuplicateIDs  []string `json:"duplicateIds"`
	DeadLinkCount int      `json:"deadLinkCount"`
}

type metaResponse struct {
	Root        string              `json:"root"`
	WorkspaceID string              `json:"workspaceId"`
	Version     string              `json:"version"`
	NoteCount   int                 `json:"noteCount"`
	Diagnostics diagnosticsResponse `json:"diagnostics"`
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	idx := s.Index()
	diag := idx.Diagnostics()
	writeJSON(w, http.StatusOK, metaResponse{
		Root:        idx.Root(),
		WorkspaceID: workspaceID(idx.Root()),
		Version:     s.version,
		NoteCount:   idx.NoteCount(),
		Diagnostics: diagnosticsResponse{
			DuplicateIDs:  nonNil(diag.DuplicateIDs),
			DeadLinkCount: diag.DeadLinkCount,
		},
	})
}

// workspaceID derives a short, stable identifier from the absolute root
// path so the frontend can namespace localStorage state (tabs, theme, ...)
// per org-roam directory.
func workspaceID(absRoot string) string {
	sum := sha256.Sum256([]byte(absRoot))
	return hex.EncodeToString(sum[:])[:12]
}

type graphNode struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	File   string   `json:"file"`
	Level  int      `json:"level"`
	Tags   []string `json:"tags"`
	NLinks int      `json:"nlinks"`
}

type graphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type graphResponse struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	idx := s.Index()
	nodes := idx.Nodes()
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	gn := make([]graphNode, 0, len(nodes))
	for _, n := range nodes {
		gn = append(gn, graphNode{
			ID:     n.ID,
			Title:  n.Title,
			File:   n.File,
			Level:  n.Level,
			Tags:   nonNil(n.Tags),
			NLinks: idx.Degree(n.ID),
		})
	}

	edges := idx.Edges()
	ge := make([]graphEdge, 0, len(edges))
	for _, e := range edges {
		ge = append(ge, graphEdge{Source: e.Source, Target: e.Target})
	}

	writeJSON(w, http.StatusOK, graphResponse{Nodes: gn, Edges: ge})
}

type noteSummary struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	File  string    `json:"file"`
	Tags  []string  `json:"tags"`
	MTime time.Time `json:"mtime"`
}

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	idx := s.Index()
	nodes := idx.Nodes()
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Title != nodes[j].Title {
			return nodes[i].Title < nodes[j].Title
		}
		return nodes[i].ID < nodes[j].ID
	})

	out := make([]noteSummary, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, noteSummary{
			ID:    n.ID,
			Title: n.Title,
			File:  n.File,
			Tags:  nonNil(n.Tags),
			MTime: n.ModTime,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type backlinkRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type noteDetail struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	File      string        `json:"file"`
	Level     int           `json:"level"`
	Tags      []string      `json:"tags"`
	Aliases   []string      `json:"aliases"`
	Refs      []string      `json:"refs"`
	HTML      string        `json:"html"`
	Backlinks []backlinkRef `json:"backlinks"`
}

func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	idx := s.Index()

	n, ok := idx.Node(id)
	if !ok {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}

	html, err := render.Render(n.Source, func(target string) bool {
		_, ok := idx.Node(target)
		return ok
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to render note")
		return
	}

	bl := idx.Backlinks(id)
	blOut := make([]backlinkRef, 0, len(bl))
	for _, b := range bl {
		blOut = append(blOut, backlinkRef{ID: b.ID, Title: b.Title})
	}

	writeJSON(w, http.StatusOK, noteDetail{
		ID:        n.ID,
		Title:     n.Title,
		File:      n.File,
		Level:     n.Level,
		Tags:      nonNil(n.Tags),
		Aliases:   nonNil(n.Aliases),
		Refs:      nonNil(n.Refs),
		HTML:      html,
		Backlinks: blOut,
	})
}

type searchResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

const searchLimit = 50

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing required query parameter: q")
		return
	}

	idx := s.Index()
	hits := idx.Search(q, searchLimit)
	out := make([]searchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, searchResult{ID: h.ID, Title: h.Title, Snippet: h.Snippet})
	}
	writeJSON(w, http.StatusOK, out)
}

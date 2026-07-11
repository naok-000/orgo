package roam

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Index is an immutable, queryable snapshot of a scanned org-roam directory.
// A new Index is built on every rescan (see Scan); the server keeps an
// atomic reference to the current one so readers never observe a
// partially-built index.
type Index struct {
	root        string
	nodes       map[string]*Node
	edges       []Edge
	degree      map[string]int
	backlinks   map[string][]string // target id -> source ids, deduplicated
	diagnostics Diagnostics
}

// Scan walks root recursively for *.org files (skipping dotfiles and
// dot-directories), parses each one, and builds an Index. Files that can't
// be read or that fail to parse are logged and skipped; Scan itself never
// fails because of a single bad file.
func Scan(root string) (*Index, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	relFiles, err := walkOrgFiles(absRoot)
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", absRoot, err)
	}

	var all []*Node
	for _, rel := range relFiles {
		full := filepath.Join(absRoot, filepath.FromSlash(rel))
		content, err := os.ReadFile(full)
		if err != nil {
			log.Printf("roam: skipping %s: %v", rel, err)
			continue
		}
		modTime := time.Now()
		if info, err := os.Stat(full); err == nil {
			modTime = info.ModTime()
		}
		all = append(all, parseFileSafe(rel, content, modTime)...)
	}

	return NewIndex(absRoot, all), nil
}

// parseFileSafe wraps ParseFile with a recover so a single malformed file
// can never crash a scan.
func parseFileSafe(rel string, content []byte, modTime time.Time) (nodes []*Node) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("roam: failed to parse %s: %v", rel, r)
			nodes = nil
		}
	}()
	return ParseFile(rel, content, modTime)
}

func walkOrgFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("roam: skipping %s: %v", path, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if path != root && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(name), ".org") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

// NewIndex builds an Index from an already-parsed set of nodes (possibly
// spanning multiple files, possibly containing duplicate ids). It resolves
// duplicate ids deterministically, resolves outgoing links into a
// deduplicated edge set, and computes backlinks/degree/diagnostics. It does
// no I/O, which makes it easy to unit test independently of the filesystem.
func NewIndex(root string, all []*Node) *Index {
	byID := map[string][]*Node{}
	for _, nd := range all {
		byID[nd.ID] = append(byID[nd.ID], nd)
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	winners := make(map[string]*Node, len(ids))
	var dupIDs []string
	for _, id := range ids {
		group := byID[id]
		if len(group) > 1 {
			sort.Slice(group, func(i, j int) bool {
				if group[i].File != group[j].File {
					return group[i].File < group[j].File
				}
				return group[i].Pos < group[j].Pos
			})
			dupIDs = append(dupIDs, id)
		}
		winners[id] = group[0]
	}

	edgeSet := map[Edge]bool{}
	var edges []Edge
	degree := map[string]int{}
	backlinks := map[string][]string{}
	deadCount := 0

	// Sort winners by id before walking their links so edge discovery order
	// (and therefore append order into per-target backlink slices) is
	// deterministic.
	sortedWinners := make([]*Node, 0, len(winners))
	for _, nd := range winners {
		sortedWinners = append(sortedWinners, nd)
	}
	sort.Slice(sortedWinners, func(i, j int) bool { return sortedWinners[i].ID < sortedWinners[j].ID })

	for _, nd := range sortedWinners {
		for _, target := range nd.Links {
			if _, ok := winners[target]; !ok {
				deadCount++
				continue
			}
			e := Edge{Source: nd.ID, Target: target}
			if edgeSet[e] {
				continue
			}
			edgeSet[e] = true
			edges = append(edges, e)
			degree[e.Source]++
			degree[e.Target]++
			backlinks[e.Target] = append(backlinks[e.Target], e.Source)
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})

	return &Index{
		root:      root,
		nodes:     winners,
		edges:     edges,
		degree:    degree,
		backlinks: backlinks,
		diagnostics: Diagnostics{
			DuplicateIDs:  dupIDs,
			DeadLinkCount: deadCount,
		},
	}
}

// Root returns the absolute path of the scanned directory.
func (idx *Index) Root() string { return idx.root }

// NoteCount returns the number of nodes (file + heading) in the index.
func (idx *Index) NoteCount() int { return len(idx.nodes) }

// Diagnostics returns the diagnostics collected while building the index.
func (idx *Index) Diagnostics() Diagnostics { return idx.diagnostics }

// Node looks up a node by id.
func (idx *Index) Node(id string) (*Node, bool) {
	nd, ok := idx.nodes[id]
	return nd, ok
}

// Nodes returns all nodes in the index, in unspecified order.
func (idx *Index) Nodes() []*Node {
	out := make([]*Node, 0, len(idx.nodes))
	for _, nd := range idx.nodes {
		out = append(out, nd)
	}
	return out
}

// Edges returns the deduplicated, resolved link edges, sorted by
// (source, target).
func (idx *Index) Edges() []Edge {
	return idx.edges
}

// Degree returns nlinks for a node: the number of resolved edges it
// participates in, as either source or target.
func (idx *Index) Degree(id string) int { return idx.degree[id] }

// Backlinks returns the nodes that link to id, sorted by title (ties broken
// by id).
func (idx *Index) Backlinks(id string) []*Node {
	sources := idx.backlinks[id]
	out := make([]*Node, 0, len(sources))
	for _, srcID := range sources {
		if nd, ok := idx.nodes[srcID]; ok {
			out = append(out, nd)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title != out[j].Title {
			return out[i].Title < out[j].Title
		}
		return out[i].ID < out[j].ID
	})
	return out
}

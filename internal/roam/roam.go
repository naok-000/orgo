// Package roam scans a directory of org-roam files, parses the org-roam
// compatibility subset described in docs/DESIGN.md, and builds an in-memory
// index of nodes, links, and backlinks. It does not depend on Emacs or the
// org-roam SQLite database; it re-implements just enough of org-roam's file
// format to browse a note collection.
package roam

import "time"

// Node is a single org-roam node: either a whole file (Level == 0) or a
// heading within a file that carries its own :ID: property (Level == the
// heading's star count).
type Node struct {
	ID      string
	Title   string
	Level   int
	Tags    []string
	Aliases []string
	Refs    []string

	// File is the path to the file containing this node, relative to the
	// scanned root, using forward slashes regardless of OS.
	File string

	// Source is the raw org markup for this node: the whole file for a file
	// node, or the heading's subtree (from its headline to the next headline
	// of level <= its own) for a heading node. It is what gets rendered to
	// HTML for /api/note/{id}.
	Source string

	// Body is Source with property drawers and #+keyword: lines stripped,
	// used as the text /api/search matches against and extracts snippets
	// from. Keeping it separate from Source means search results and
	// snippets surface actual note prose instead of :ID:/:ROAM_REFS:/
	// #+title: metadata.
	Body string

	// Links holds the raw id: link targets found in this node's content, in
	// order of appearance. It may contain duplicates and ids that don't
	// resolve to any known node (dead links); both are resolved later at the
	// Index level.
	Links []string

	ModTime time.Time

	// Pos is the 1-based line number the node starts at within its file. It
	// is only used to break ties between nodes that declare the same :ID:
	// within a single file.
	Pos int
}

// Edge is a resolved, deduplicated directed link between two nodes that both
// exist in the index.
type Edge struct {
	Source string
	Target string
}

// Diagnostics summarizes problems found while indexing that are worth
// surfacing to the user but don't stop indexing.
type Diagnostics struct {
	// DuplicateIDs lists the ids (sorted, deduplicated) that were declared by
	// more than one node. For each, the node with the lexicographically
	// smallest file path wins (ties broken by the smaller line position);
	// the rest are dropped from the index.
	DuplicateIDs []string
	// DeadLinkCount is the number of [[id:...]] link occurrences (across all
	// nodes) whose target id does not resolve to any known node.
	DeadLinkCount int
}

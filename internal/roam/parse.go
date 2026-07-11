package roam

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	headlineRe     = regexp.MustCompile(`^(\*+)\s+(.*)$`)
	propStartRe    = regexp.MustCompile(`(?i)^\s*:PROPERTIES:\s*$`)
	propEndRe      = regexp.MustCompile(`(?i)^\s*:END:\s*$`)
	propLineRe     = regexp.MustCompile(`^\s*:([A-Za-z][A-Za-z0-9_+-]*):\s*(.*?)\s*$`)
	titleKeywRe    = regexp.MustCompile(`(?i)^#\+title:\s*(.*)$`)
	filetagsKeywRe = regexp.MustCompile(`(?i)^#\+filetags:\s*(.*)$`)
	headingTagsRe  = regexp.MustCompile(`^(.*?)\s+:([A-Za-z0-9_@#%:]+):\s*$`)
	idLinkRe       = regexp.MustCompile(`\[\[id:([^][]+)\]`)
	// planningRe matches org planning lines (SCHEDULED:/DEADLINE:/CLOSED:),
	// which org places between a headline and its property drawer. Node
	// detection must skip over them to find the drawer.
	planningRe = regexp.MustCompile(`(?i)^\s*(SCHEDULED|DEADLINE|CLOSED):`)
	// verbatimSpanRe matches org's single-line verbatim/code emphasis
	// (=literal= and ~literal~). Their contents are literal text, not
	// parsed markup, so an "id:" link-looking string inside one (e.g. an
	// example like "links look like =[[id:...]]=") must not be scanned for
	// links.
	verbatimSpanRe = regexp.MustCompile(`=[^=\t\n]+=|~[^~\t\n]+~`)
	blockBeginRe   = regexp.MustCompile(`(?i)^\s*#\+begin_(\S+)`)
	blockEndRe     = regexp.MustCompile(`(?i)^\s*#\+end_(\S+)`)
	// drawerStartAnyRe matches the start of any drawer, not just
	// :PROPERTIES: (e.g. :LOGBOOK:). It's only used by buildBody to keep
	// drawer contents out of search text; node parsing itself only ever
	// looks for :PROPERTIES: specifically.
	drawerStartAnyRe = regexp.MustCompile(`(?i)^\s*:[A-Za-z][A-Za-z0-9_+-]*:\s*$`)
	// keywordLineRe matches any "#+KEYWORD: value" line (#+title:,
	// #+filetags:, #+roam_key:, ...); block markers like "#+begin_src" have
	// no colon right after the keyword and so don't match.
	keywordLineRe = regexp.MustCompile(`(?i)^\s*#\+[A-Za-z_-]+:.*$`)
)

// ParseFile parses the org-roam nodes contained in a single file's content.
// relPath is the file's path relative to the scanned root (forward-slash
// separated); it is stored on every returned node. A file with no :ID:
// anywhere (no file-level property drawer with :ID:, and no heading with its
// own :ID:) yields no nodes at all, per org-roam's own semantics.
func ParseFile(relPath string, content []byte, modTime time.Time) []*Node {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	n := len(lines)

	// lineOff[i] is the byte offset of the start of lines[i] within text;
	// lineOff[n] is len(text)+1 (as if a final "\n" existed). Node sources
	// are produced by slicing text with these offsets rather than copying,
	// so every node of a file shares one backing array and retained memory
	// stays O(file) even for deeply nested ID subtrees.
	lineOff := make([]int, n+1)
	for i, l := range lines {
		lineOff[i+1] = lineOff[i] + len(l) + 1
	}
	sourceSlice := func(from, to int) string {
		start, end := lineOff[from], lineOff[to]-1
		if end > len(text) {
			end = len(text)
		}
		if end < start {
			end = start
		}
		return text[start:end]
	}

	literal := literalRegions(lines)

	firstHeading := n
	var headingIdxs, headingLevels []int
	for i, l := range lines {
		if literal[i] {
			continue
		}
		if m := headlineRe.FindStringSubmatch(l); m != nil {
			if i < firstHeading {
				firstHeading = i
			}
			headingIdxs = append(headingIdxs, i)
			headingLevels = append(headingLevels, len(m[1]))
		}
	}

	fileID, fileTitle, fileTags, fileAliases, fileRefs := parsePreamble(lines[:firstHeading])
	if fileTitle == "" {
		fileTitle = titleFromFilename(relPath)
	}

	bodyAll, bodyOff := buildBody(lines)
	bodySlice := func(from, to int) string {
		return strings.TrimSuffix(bodyAll[bodyOff[from]:bodyOff[to]], "\n")
	}

	var nodes []*Node
	nodeByID := map[string]*Node{}

	if fileID != "" {
		fileNode := &Node{
			ID:      fileID,
			Title:   fileTitle,
			Level:   0,
			Tags:    fileTags,
			Aliases: fileAliases,
			Refs:    fileRefs,
			File:    relPath,
			Source:  text,
			Body:    bodySlice(0, n),
			ModTime: modTime,
			Pos:     1,
		}
		nodes = append(nodes, fileNode)
		nodeByID[fileID] = fileNode
	}

	type frame struct {
		level  int
		nodeID string
		tags   []string
	}
	var stack []frame
	nearest := fileID
	hPos := 0

	for i := 0; i < n; i++ {
		if hPos < len(headingIdxs) && headingIdxs[hPos] == i {
			level := headingLevels[hPos]
			for len(stack) > 0 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}

			m := headlineRe.FindStringSubmatch(lines[i])
			title, tags := splitHeadingTags(m[2])

			// The property drawer may be separated from the headline by
			// planning lines (SCHEDULED:/DEADLINE:/CLOSED:); skip them.
			id := ""
			var props map[string]string
			j := i + 1
			for j < n && planningRe.MatchString(lines[j]) {
				j++
			}
			if j < n && propStartRe.MatchString(lines[j]) {
				props, _ = parsePropertyDrawer(lines, j)
				id = strings.TrimSpace(props["ID"])
			}

			if id != "" {
				end := n
				for q := hPos + 1; q < len(headingIdxs); q++ {
					if headingLevels[q] <= level {
						end = headingIdxs[q]
						break
					}
				}
				// Org tag inheritance: file tags, then the tags of every
				// ancestor headline (whether or not it is itself a node),
				// then the heading's own tags.
				var ancestorTags []string
				for _, f := range stack {
					ancestorTags = append(ancestorTags, f.tags...)
				}
				nd := &Node{
					ID:      id,
					Title:   strings.TrimSpace(title),
					Level:   level,
					Tags:    unionTags(fileTags, ancestorTags, tags),
					Aliases: splitOrgWords(props["ROAM_ALIASES"]),
					Refs:    strings.Fields(props["ROAM_REFS"]),
					File:    relPath,
					Source:  sourceSlice(i, end),
					Body:    bodySlice(i, end),
					ModTime: modTime,
					Pos:     i + 1,
				}
				nodes = append(nodes, nd)
				nodeByID[id] = nd
			}

			newNearest := id
			if newNearest == "" {
				if len(stack) > 0 {
					newNearest = stack[len(stack)-1].nodeID
				} else {
					newNearest = fileID
				}
			}
			stack = append(stack, frame{level: level, nodeID: newNearest, tags: tags})
			nearest = newNearest
			hPos++

			// Links in the headline text itself belong to this heading's
			// nearest node (the heading itself when it is a node).
			if nd, ok := nodeByID[nearest]; ok {
				for _, lm := range idLinkRe.FindAllStringSubmatch(stripVerbatim(m[2]), -1) {
					nd.Links = append(nd.Links, strings.TrimSpace(lm[1]))
				}
			}
			continue
		}

		if nearest == "" || literal[i] {
			continue
		}
		nd, ok := nodeByID[nearest]
		if !ok {
			continue
		}
		for _, lm := range idLinkRe.FindAllStringSubmatch(stripVerbatim(lines[i]), -1) {
			nd.Links = append(nd.Links, strings.TrimSpace(lm[1]))
		}
	}

	return nodes
}

// literalRegions reports, for every line, whether it lies inside a literal
// block — src, example, export, or comment — whose contents are raw text
// rather than org markup (inclusive of the begin/end marker lines). Only
// those regions are excluded from headline detection and link scanning:
// quote/center/verse and other blocks contain real org markup whose links
// must count as edges (go-org renders them as links too, keeping the graph
// and the rendered HTML in agreement).
func literalRegions(lines []string) []bool {
	lit := make([]bool, len(lines))
	open := "" // lowercased name of the currently open literal block, if any
	for i, l := range lines {
		if m := blockBeginRe.FindStringSubmatch(l); m != nil {
			if open != "" {
				lit[i] = true // a begin marker inside a literal block is content
				continue
			}
			if isLiteralBlockName(m[1]) {
				open = strings.ToLower(m[1])
				lit[i] = true
			}
			continue
		}
		if m := blockEndRe.FindStringSubmatch(l); m != nil {
			if open != "" {
				lit[i] = true
				if strings.EqualFold(m[1], open) {
					open = ""
				}
			}
			continue
		}
		lit[i] = open != ""
	}
	return lit
}

func isLiteralBlockName(name string) bool {
	switch strings.ToLower(name) {
	case "src", "example", "export", "comment":
		return true
	default:
		return false
	}
}

// buildBody builds the search text for a whole file — every line except
// drawers (:PROPERTIES:, :LOGBOOK:, ...) and #+keyword: lines, joined by
// "\n" — plus, for every line index, its byte offset into that string.
// Each node's Body is then sliced out of the shared string instead of
// copied, keeping retained memory O(file). Offsets of stripped lines point
// at the next kept content; the returned offsets slice has len(lines)+1
// entries, the last being the total length.
func buildBody(lines []string) (string, []int) {
	keep := make([]bool, len(lines))
	for i := 0; i < len(lines); {
		line := lines[i]
		if drawerStartAnyRe.MatchString(line) && !propEndRe.MatchString(line) {
			j := i + 1
			for j < len(lines) && !propEndRe.MatchString(lines[j]) {
				j++
			}
			if j < len(lines) {
				j++ // also drop the :END: line itself
			}
			i = j
			continue
		}
		if !keywordLineRe.MatchString(line) {
			keep[i] = true
		}
		i++
	}

	var sb strings.Builder
	off := make([]int, len(lines)+1)
	for i, l := range lines {
		off[i] = sb.Len()
		if keep[i] {
			sb.WriteString(l)
			sb.WriteByte('\n')
		}
	}
	off[len(lines)] = sb.Len()
	return sb.String(), off
}

// stripVerbatim blanks out org's single-line verbatim/code emphasis spans
// (=...= and ~...~) so their literal contents are never mistaken for real
// markup (in particular, real [[id:...]] links).
func stripVerbatim(line string) string {
	return verbatimSpanRe.ReplaceAllStringFunc(line, func(m string) string {
		return strings.Repeat(" ", len(m))
	})
}

// parsePreamble scans the lines before the first headline for the file-level
// property drawer (:ID:, :ROAM_ALIASES:, :ROAM_REFS:), #+title, and
// #+filetags.
func parsePreamble(lines []string) (id, title string, tags, aliases, refs []string) {
	i := 0
	for i < len(lines) {
		line := lines[i]
		switch {
		case propStartRe.MatchString(line):
			props, end := parsePropertyDrawer(lines, i)
			if v := strings.TrimSpace(props["ID"]); v != "" {
				id = v
			}
			aliases = append(aliases, splitOrgWords(props["ROAM_ALIASES"])...)
			refs = append(refs, strings.Fields(props["ROAM_REFS"])...)
			i = end
		case titleKeywRe.MatchString(line):
			m := titleKeywRe.FindStringSubmatch(line)
			title = strings.TrimSpace(m[1])
			i++
		case filetagsKeywRe.MatchString(line):
			m := filetagsKeywRe.FindStringSubmatch(line)
			tags = unionTags(tags, parseTagString(m[1]))
			i++
		default:
			i++
		}
	}
	return id, title, tags, aliases, refs
}

// parsePropertyDrawer parses a :PROPERTIES: ... :END: drawer starting at
// lines[start] (which must be the :PROPERTIES: line itself). It returns the
// parsed key/value pairs (keys upper-cased) and the index of the line
// following :END:. If the drawer is never closed, it returns whatever was
// collected and len(lines).
func parsePropertyDrawer(lines []string, start int) (map[string]string, int) {
	props := map[string]string{}
	i := start + 1
	for i < len(lines) {
		if propEndRe.MatchString(lines[i]) {
			return props, i + 1
		}
		if m := propLineRe.FindStringSubmatch(lines[i]); m != nil {
			key := strings.ToUpper(m[1])
			val := strings.TrimSpace(m[2])
			if existing, ok := props[key]; ok && existing != "" && val != "" {
				props[key] = existing + " " + val
			} else {
				props[key] = val
			}
		}
		i++
	}
	return props, i
}

// splitHeadingTags splits the text following the stars of a headline into
// its title and trailing org tags (`* Title here :tag1:tag2:`).
func splitHeadingTags(rest string) (title string, tags []string) {
	if m := headingTagsRe.FindStringSubmatch(rest); m != nil {
		return strings.TrimSpace(m[1]), splitColonTags(m[2])
	}
	return strings.TrimSpace(rest), nil
}

func splitColonTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ":") {
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// parseTagString parses a #+filetags value. Org's canonical form is
// colon-delimited with leading/trailing colons (":a:b:"); orgo is lenient
// and also accepts comma- or whitespace-separated forms.
func parseTagString(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, ":") && strings.HasSuffix(s, ":") && strings.Count(s, ":") >= 2 {
		return splitColonTags(strings.Trim(s, ":"))
	}
	if strings.Contains(s, ",") {
		var out []string
		for _, t := range strings.Split(s, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	return strings.Fields(s)
}

// splitOrgWords implements org's quoted-word splitting used for
// ROAM_ALIASES: "double quoted phrases" may contain spaces; everything else
// is split on whitespace.
func splitOrgWords(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	i, n := 0, len(s)
	for i < n {
		for i < n && isOrgSpace(s[i]) {
			i++
		}
		if i >= n {
			break
		}
		if s[i] == '"' {
			j := i + 1
			var sb strings.Builder
			for j < n && s[j] != '"' {
				if s[j] == '\\' && j+1 < n {
					sb.WriteByte(s[j+1])
					j += 2
					continue
				}
				sb.WriteByte(s[j])
				j++
			}
			out = append(out, sb.String())
			if j < n {
				j++ // skip closing quote
			}
			i = j
			continue
		}
		j := i
		for j < n && !isOrgSpace(s[j]) {
			j++
		}
		out = append(out, s[i:j])
		i = j
	}
	return out
}

func isOrgSpace(b byte) bool { return b == ' ' || b == '\t' }

// unionTags merges tag lists, preserving first-seen order and dropping
// duplicates/empties.
func unionTags(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range lists {
		for _, t := range list {
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func titleFromFilename(relPath string) string {
	base := filepath.Base(relPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

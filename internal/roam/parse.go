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
	// verbatimSpanRe matches org's single-line verbatim/code emphasis
	// (=literal= and ~literal~). Their contents are literal text, not
	// parsed markup, so an "id:" link-looking string inside one (e.g. an
	// example like "links look like =[[id:...]]=") must not be scanned for
	// links.
	verbatimSpanRe = regexp.MustCompile(`=[^=\t\n]+=|~[^~\t\n]+~`)
	blockBeginRe   = regexp.MustCompile(`(?i)^\s*#\+begin_\S+`)
	blockEndRe     = regexp.MustCompile(`(?i)^\s*#\+end_\S+`)
	// drawerStartAnyRe matches the start of any drawer, not just
	// :PROPERTIES: (e.g. :LOGBOOK:). It's only used by bodyText to keep
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
	lines := splitLines(string(content))
	n := len(lines)
	inBlock := blockRegions(lines)

	firstHeading := n
	headingIdxs := make([]int, 0)
	headingLevels := make([]int, 0)
	for i, l := range lines {
		if inBlock[i] {
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
			Source:  string(content),
			Body:    bodyText(lines),
			ModTime: modTime,
			Pos:     1,
		}
		nodes = append(nodes, fileNode)
		nodeByID[fileID] = fileNode
	}

	type frame struct {
		level  int
		nodeID string
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

			id := ""
			var props map[string]string
			if i+1 < n && propStartRe.MatchString(lines[i+1]) {
				props, _ = parsePropertyDrawer(lines, i+1)
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
				nd := &Node{
					ID:      id,
					Title:   strings.TrimSpace(title),
					Level:   level,
					Tags:    unionTags(fileTags, tags),
					Aliases: splitOrgWords(props["ROAM_ALIASES"]),
					Refs:    strings.Fields(props["ROAM_REFS"]),
					File:    relPath,
					Source:  strings.Join(lines[i:end], "\n"),
					Body:    bodyText(lines[i:end]),
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
			stack = append(stack, frame{level: level, nodeID: newNearest})
			nearest = newNearest
			hPos++
			continue
		}

		if nearest == "" || inBlock[i] {
			continue
		}
		nd, ok := nodeByID[nearest]
		if !ok {
			continue
		}
		for _, m := range idLinkRe.FindAllStringSubmatch(stripVerbatim(lines[i]), -1) {
			nd.Links = append(nd.Links, strings.TrimSpace(m[1]))
		}
	}

	return nodes
}

// blockRegions reports, for every line, whether it falls inside a
// #+begin_.../#+end_... block (SRC, EXAMPLE, QUOTE, ...), inclusive of the
// begin/end marker lines themselves. Block content is raw/literal in org
// (most obviously for SRC and EXAMPLE): it must not be scanned for
// headlines or links, since it isn't prose.
func blockRegions(lines []string) []bool {
	inBlock := make([]bool, len(lines))
	depth := 0
	for i, l := range lines {
		switch {
		case blockBeginRe.MatchString(l):
			inBlock[i] = true
			depth++
		case blockEndRe.MatchString(l):
			inBlock[i] = true
			if depth > 0 {
				depth--
			}
		default:
			inBlock[i] = depth > 0
		}
	}
	return inBlock
}

// bodyText joins lines back into text with drawers (:PROPERTIES:, :LOGBOOK:,
// ...) and #+keyword: lines removed, so it reads as prose rather than
// metadata. It's used for roam.Node.Body (search text), never for Source
// (which keeps everything so it renders faithfully).
func bodyText(lines []string) string {
	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		line := lines[i]
		if drawerStartAnyRe.MatchString(line) && !propEndRe.MatchString(line) {
			j := i + 1
			for j < len(lines) && !propEndRe.MatchString(lines[j]) {
				j++
			}
			i = j + 1 // also skip the :END: line itself
			continue
		}
		if keywordLineRe.MatchString(line) {
			i++
			continue
		}
		out = append(out, line)
		i++
	}
	return strings.Join(out, "\n")
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

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

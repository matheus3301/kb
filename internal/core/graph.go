package core

import (
	"context"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

var referenceDef = regexp.MustCompile(`^ {0,3}\[([^\]]+)\]:\s*(<[^>]+>|\S+)`)
var referenceUse = regexp.MustCompile(`\[([^\]\n]+)\]\[([^\]\n]*)\]`)

type heading struct {
	text, id          string
	start, end, level int
}

func headings(raw string) []heading {
	lines := strings.Split(raw, "\n")
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var out []heading
	counts := map[string]int{}
	var fence fenceState
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if fence.skip(line) {
			continue
		}
		level := 0
		for level < len(trim) && level < 6 && trim[level] == '#' {
			level++
		}
		if level == 0 || level >= len(trim) || trim[level] != ' ' {
			continue
		}
		text := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(trim[level:]), "#"))
		slug := headingSlug(text)
		id := slug
		if counts[slug] > 0 {
			id = slug + "-" + itoa(counts[slug])
		}
		counts[slug]++
		out = append(out, heading{text: text, id: id, start: i + 1, end: len(lines), level: level})
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j].level <= out[i].level {
				out[i].end = out[j].start - 1
				break
			}
		}
	}
	return out
}
func headingSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if r == ' ' || r == '-' {
			b.WriteRune('-')
		} else if r == '_' || isWord(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
func resolveLink(from, target string, files map[string][]byte) api.Link {
	link := api.Link{From: from, To: target}
	u, err := url.Parse(strings.Trim(target, "<>"))
	if err != nil {
		return link
	}
	if u.IsAbs() || u.Host != "" {
		link.External = true
		return link
	}
	link.Anchor = u.Fragment
	p := u.Path
	if p == "" {
		p = from
	} else if strings.HasPrefix(p, "/") {
		p = strings.TrimPrefix(p, "/")
	} else {
		p = path.Join(path.Dir(from), p)
	}
	p = path.Clean(p)
	if validPath(p) != nil {
		return link
	}
	if strings.HasSuffix(u.Path, "/") {
		p = path.Join(p, "index.md")
	}
	link.To = p
	raw, ok := files[p]
	link.Resolved = ok
	if ok && u.Fragment != "" {
		link.Resolved = false
		for _, h := range headings(string(raw)) {
			if h.id == u.Fragment {
				link.Resolved = true
				break
			}
		}
	}
	return link
}
func buildLinks(files map[string][]byte, notes []api.Note) []api.Link {
	var links []api.Link
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		bodyBytes, start := rawBody(files[p])
		body := string(bodyBytes)
		defs := map[string]string{}
		lines := visibleLines(body)
		for _, line := range lines {
			if match := referenceDef.FindStringSubmatch(line); match != nil {
				defs[strings.ToLower(match[1])] = match[2]
			}
		}
		for i, line := range lines {

			for _, match := range inlineTargets(line) {
				link := resolveLink(p, match.target, files)
				link.Text = match.text
				link.Kind = "markdown"
				link.Line = start + i + 1
				links = append(links, link)
			}
			for _, match := range referenceUse.FindAllStringSubmatch(line, -1) {
				key := match[2]
				if key == "" {
					key = match[1]
				}
				if target, ok := defs[strings.ToLower(key)]; ok {
					link := resolveLink(p, target, files)
					link.Text = match[1]
					link.Kind = "markdown"
					link.Line = start + i + 1
					links = append(links, link)
				}
			}
		}
	}
	for _, note := range notes {
		resources := []string{}
		if v, ok := note.Metadata["resource"].(string); ok {
			resources = append(resources, v)
		}
		if sources, ok := note.Metadata["sources"].([]any); ok {
			for _, v := range sources {
				if source, ok := v.(map[string]any); ok {
					if resource, ok := source["resource"].(string); ok {
						resources = append(resources, resource)
					}
				}
			}
		}
		if v, ok := note.Metadata["computation"].(string); ok {
			resources = append(resources, v)
		}
		for _, key := range []string{"executor", "attester"} {
			if obj, ok := note.Metadata[key].(map[string]any); ok {
				if v, ok := obj["resource"].(string); ok {
					resources = append(resources, v)
				}
			}
		}
		for _, resource := range resources {
			if strings.Contains(resource, " ") && !strings.HasSuffix(resource, ".md") && !strings.HasPrefix(resource, "/") && !strings.Contains(resource, "://") {
				continue
			}
			link := resolveLink(note.Path, resource, files)
			link.Kind = "provenance"
			links = append(links, link)
		}
	}
	sort.SliceStable(links, func(i, j int) bool {
		a, b := links[i], links[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.To < b.To
	})
	return links
}
func basisChanged(n api.Note, files map[string][]byte) bool {
	basis, ok := n.Metadata["basis_revisions"].(map[string]any)
	if !ok {
		return false
	}
	for p, v := range basis {
		if !strings.HasSuffix(p, ".md") {
			p += ".md"
		}
		target := strings.TrimPrefix(p, "/")
		raw, ok := files[target]
		if !ok || v != api.Revision(raw) {
			return true
		}
	}
	return false
}

// Links returns the bounded on-disk graph neighborhood of a scoped document.
func (s *Store) Links(ctx context.Context, p, direction string) ([]api.Link, error) {
	if err := validPath(p); err != nil {
		return nil, err
	}
	if direction == "" {
		direction = "both"
	}
	if direction != "both" && direction != "in" && direction != "out" && direction != "incoming" && direction != "outgoing" {
		return nil, fault("INVALID_INPUT", "direction must be in, out, or both")
	}
	snap, err := s.readSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := snap.Files[p]; !ok {
		return nil, fault("NOT_FOUND", "document not in scope")
	}
	return neighborhood(snap.Links, p, direction), nil
}
func neighborhood(links []api.Link, p, direction string) []api.Link {
	out := []api.Link{}
	for _, link := range links {
		if (direction == "both" || direction == "out" || direction == "outgoing") && link.From == p || (direction == "both" || direction == "in" || direction == "incoming") && link.To == p && !link.External {
			out = append(out, link)
		}
	}
	return out
}

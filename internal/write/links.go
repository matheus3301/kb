package write

import (
	"bytes"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

type destination struct {
	start, end int
	value      string
}

// markdownDestinations preserves surrounding Markdown and skips fenced/inline code.
func markdownDestinations(text string) []destination {
	var spans []destination
	var fence codeFence
	var containers listContext
	offset := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		content, contentStart := containers.content(line, fence.marker != 0)
		if fence.consume(content) {
			offset += len(line)
			continue
		}
		for i := contentStart; i < len(line); i++ {
			if line[i] == '\\' {
				i++
				continue
			}
			if line[i] == '`' {
				n := 1
				for i+n < len(line) && line[i+n] == '`' {
					n++
				}
				close := strings.Index(line[i+n:], strings.Repeat("`", n))
				if close < 0 {
					break
				}
				i += n + close + n - 1
				continue
			}
			if line[i] != ']' || i+1 >= len(line) {
				continue
			}
			start := i + 2
			switch line[i+1] {
			case '(':
			case ':':
				if strings.TrimSpace(line[contentStart:i]) == "" || !strings.HasPrefix(strings.TrimSpace(line[contentStart:i]), "[") {
					continue
				}
			default:
				continue
			}
			for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
				start++
			}
			if start >= len(line) {
				continue
			}
			angle := line[start] == '<'
			if angle {
				start++
			}
			end := start
			depth := 0
			for end < len(line) {
				c := line[end]
				if c == '\\' && end+1 < len(line) {
					end += 2
					continue
				}
				if angle {
					if c == '>' {
						break
					}
				} else {
					if c == '(' {
						depth++
					}
					if c == ')' {
						if depth == 0 {
							break
						}
						depth--
					}
					if (c == ' ' || c == '\t' || c == '\n' || c == '\r') && depth == 0 {
						break
					}
				}
				end++
			}
			if end > start {
				spans = append(spans, destination{offset + start, offset + end, line[start:end]})
			}
			i = end
		}
		offset += len(line)
	}
	return spans
}

func resolveReference(from, value string) (string, string, bool) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "" || u.Host != "" || u.RawQuery != "" || u.Path == "" || strings.Contains(u.Path, "\\") {
		return "", "", false
	}
	anchor := ""
	if at := strings.Index(value, "#"); at >= 0 {
		anchor = value[at:]
	}
	var target string
	if strings.HasPrefix(u.Path, "/") {
		target = strings.TrimPrefix(u.Path, "/")
	} else {
		target = path.Join(path.Dir(from), u.Path)
	}
	if cleanPath(target) != nil {
		return "", "", false
	}
	return target, anchor, true
}

func resolveKnown(from, value string, files map[string][]byte) (string, string, bool) {
	target, anchor, ok := resolveReference(from, value)
	if !ok {
		return "", "", false
	}
	candidates := []string{target}
	if path.Ext(target) == "" {
		candidates = append(candidates, target+".md", path.Join(target, "index.md"))
	}
	found := ""
	for _, candidate := range candidates {
		if _, exists := files[candidate]; exists {
			if found != "" {
				return "", "", false
			}
			found = candidate
		}
	}
	return found, anchor, found != ""
}

func relativeReference(from, target, original, anchor string) string {
	if strings.HasPrefix(original, "/") {
		return "/" + linkURL(target) + anchor
	}
	rel, err := filepath.Rel(filepath.FromSlash(path.Dir(from)), filepath.FromSlash(target))
	if err != nil {
		return original
	}
	return linkURL(filepath.ToSlash(rel)) + anchor
}

func (s *Service) relocate(snapshot api.Snapshot, before, after map[string][]byte, moves map[string]string, actor string, warnings []api.Diagnostic) ([]api.Diagnostic, error) {
	origins := map[string]string{}
	for old, fresh := range moves {
		origins[fresh] = old
	}
	known := map[string]map[string]bool{}
	for _, link := range snapshot.Links {
		if !link.Resolved || link.External {
			continue
		}
		if known[link.From] == nil {
			known[link.From] = map[string]bool{}
		}
		known[link.From][link.To] = true
	}
	for name, raw := range after {
		original := name
		if old, ok := origins[name]; ok {
			original = old
		}
		note, _ := s.provider.Engine().Parse(name, raw)
		if reserved(name) {
			note.Body = string(raw)
		}
		body := note.Body
		spans := markdownDestinations(body)
		for i := len(spans) - 1; i >= 0; i-- {
			span := spans[i]
			target, anchor, ok := resolveKnown(original, span.value, before)
			if !ok || !known[original][target] {
				continue
			}
			newTarget := target
			if moved, ok := moves[target]; ok {
				newTarget = moved
			}
			if newTarget == target && original == name {
				continue
			}
			replacement := relativeReference(name, newTarget, span.value, anchor)
			body = body[:span.start] + replacement + body[span.end:]
		}
		metadata, err := metadataJSON(note.Metadata)
		if err != nil {
			return nil, err
		}
		values, _ := metadata.(map[string]any)
		patch := map[string]any{}
		if sources, ok := values["sources"].([]any); ok {
			changed := false
			for _, item := range sources {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				resource, _ := entry["resource"].(string)
				target, anchor, ok := resolveKnown(original, resource, before)
				if !ok {
					continue
				}
				newTarget := target
				if moved, ok := moves[target]; ok {
					newTarget = moved
				}
				if target == newTarget && original == name {
					continue
				}
				entry["resource"] = relativeReference(name, newTarget, resource, anchor)
				changed = true
			}
			if changed {
				patch["sources"] = sources
			}
		}
		if basis, ok := values["basis_revisions"].([]any); ok {
			changed := false
			for _, item := range basis {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				dependency, _ := entry["path"].(string)
				if moved, ok := moves[dependency]; ok {
					entry["path"] = moved
					changed = true
				}
			}
			if changed {
				patch["basis_revisions"] = basis
			}
		}
		if body == note.Body && len(patch) == 0 {
			continue
		}
		if reserved(name) {
			after[name] = []byte(body)
			continue
		}
		op := api.Operation{Kind: "patch", MetadataPatch: patch}
		if body != note.Body {
			op.Body = &body
		}
		if _, verified := note.Metadata["verified"]; verified {
			op.RemoveMetadataKeys = []string{"verified"}
			warnings = append(warnings, api.Diagnostic{Path: name, Code: "VERIFICATION_CLEARED", Layer: "advisories", Message: "references changed; prior human verification was cleared"})
		}
		fresh, err := s.provider.Engine().Patch(name, raw, op, actor, s.now().UTC())
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(fresh, raw) {
			after[name] = fresh
		}
	}
	return warnings, nil
}

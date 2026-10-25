package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

// Read returns source lines from a scoped document, including invalid documents.
func (s *Store) Read(ctx context.Context, request api.ReadRequest) (api.ReadResult, error) {
	if err := validPath(request.Path); err != nil {
		return api.ReadResult{}, err
	}
	if request.Section != "" && (request.LineStart != 0 || request.LineEnd != 0) {
		return api.ReadResult{}, fault("INVALID_INPUT", "section and line range are mutually exclusive")
	}
	snap, err := s.readSnapshot(ctx)
	if err != nil {
		return api.ReadResult{}, err
	}
	raw, ok := snap.Files[request.Path]
	if !ok {
		return api.ReadResult{}, fault("NOT_FOUND", "document not in scope")
	}
	n, ds := (codec{}).Parse(request.Path, raw)
	n.PARA = s.category(n.Path)
	for _, note := range snap.Notes {
		if note.Path == request.Path {
			n = note
			break
		}
	}
	lines := strings.SplitAfter(string(raw), "\n")
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	start, end := 1, len(lines)
	if request.Section != "" {
		hs := headings(string(raw))
		matches := []heading{}
		explicit := strings.HasPrefix(request.Section, "#")
		key := strings.TrimPrefix(request.Section, "#")
		for _, h := range hs {
			if explicit && h.id == key || !explicit && (h.text == key || h.id == key) {
				matches = append(matches, h)
			}
		}
		if len(matches) == 0 {
			return api.ReadResult{}, fault("NOT_FOUND", "section not found")
		}
		if !explicit {
			count := 0
			for _, h := range hs {
				if strings.EqualFold(h.text, key) || headingSlug(h.text) == key {
					count++
				}
			}
			if count > 1 {
				return api.ReadResult{}, &api.DomainError{Code: "AMBIGUOUS_SECTION", Message: "use an explicit #section-id", Details: sectionIDs(matches)}
			}
		}
		start, end = matches[0].start, matches[0].end
	}
	if request.LineStart < 0 || request.LineEnd < 0 {
		return api.ReadResult{}, fault("INVALID_INPUT", "line numbers must be positive")
	}
	if request.LineStart > 0 {
		start = request.LineStart
	}
	if request.LineEnd > 0 {
		end = request.LineEnd
	}
	if start < 1 || start > len(lines) || end < start || end > len(lines) {
		return api.ReadResult{}, fault("INVALID_INPUT", "line range is outside the document")
	}
	selected := strings.Join(lines[start-1:end], "")
	truncated := len(selected) > api.MaxReadBytes
	if truncated {
		selected = truncateUTF8(selected, api.MaxReadBytes)
		end = start + strings.Count(selected, "\n")
		if strings.HasSuffix(selected, "\n") {
			end--
		}
		ds = append(ds, diagnostic(n.Path, "TRUNCATED", "read reached the byte ceiling", "advisories"))
	}
	n.Body = ""
	n.Raw = nil
	n.Metadata = cloneMap(n.Metadata)
	n.Tags = append([]string(nil), n.Tags...)
	out := api.ReadResult{Generation: snap.Generation, Note: n, Markdown: selected, LineStart: start, LineEnd: end, Diagnostics: ds}
	if request.IncludeLinks {
		snap = s.withLinks(snap)
		out.Links = neighborhood(snap.Links, request.Path, "both")
		if len(out.Links) > api.MaxHits {
			out.Links = out.Links[:api.MaxHits]
			out.Diagnostics = append(out.Diagnostics, diagnostic(n.Path, "TRUNCATED", "relationship limit reached", "advisories"))
		}
	}
	// Bound the entire response, including metadata rather than just Markdown.
	encoded, err := json.Marshal(out)
	if err != nil {
		return api.ReadResult{}, err
	}
	if len(encoded) > api.MaxReadBytes {
		excess := len(encoded) - api.MaxReadBytes
		budget := len(out.Markdown) - excess - 256
		if budget < 0 {
			return api.ReadResult{}, fault("LIMIT_EXCEEDED", "metadata exceeds read response ceiling")
		}
		out.Markdown = truncateUTF8(out.Markdown, budget)
		out.LineEnd = start + strings.Count(out.Markdown, "\n")
		if strings.HasSuffix(out.Markdown, "\n") {
			out.LineEnd--
		}
		out.Diagnostics = append(out.Diagnostics, diagnostic(n.Path, "TRUNCATED", "response reached the byte ceiling", "advisories"))
		encoded, err = json.Marshal(out)
		if err != nil {
			return api.ReadResult{}, err
		}
		if len(encoded) > api.MaxReadBytes {
			return api.ReadResult{}, fault("LIMIT_EXCEEDED", "encoded source exceeds response ceiling")
		}
	}
	return out, nil
}
func sectionIDs(hs []heading) []string {
	ids := make([]string, len(hs))
	for i, h := range hs {
		ids[i] = "#" + h.id
	}
	return ids
}

// Context composes bounded, citable original excerpts; it generates no answers.
func (s *Store) Context(ctx context.Context, request api.ContextRequest) (api.ContextResult, error) {
	if request.MaxTokens == 0 {
		request.MaxTokens = api.DefaultContextTokens
	}
	if request.MaxBytes == 0 {
		request.MaxBytes = api.MaxReadBytes
	}
	if request.MaxTokens < 1 || request.MaxTokens > api.MaxContextTokens || request.MaxBytes < 1 || request.MaxBytes > api.MaxReadBytes {
		return api.ContextResult{}, fault("LIMIT_EXCEEDED", "context budget exceeds allowed limits")
	}
	searchRequest := api.SearchRequest{Query: request.Query, Filters: request.Filters, Limit: api.MaxHits}
	if len([]rune(request.Query)) > api.MaxQueryRunes {
		return api.ContextResult{}, fault("LIMIT_EXCEEDED", "query too long")
	}
	if request.Filters.Prefix != "" {
		if err := validPath(strings.TrimSuffix(request.Filters.Prefix, "/")); err != nil {
			return api.ContextResult{}, err
		}
	}
	snap, err := s.readSnapshot(ctx)
	if err != nil {
		return api.ContextResult{}, err
	}
	result, err := s.searchSnapshot(ctx, snap, searchRequest)
	if err != nil {
		return api.ContextResult{}, err
	}
	out := api.ContextResult{Generation: snap.Generation, Estimator: "utf8-bytes/4-ceiling", Sources: []api.ContextSource{}}
	selected := make([]string, 0, len(result.Hits))
	seen := map[string]bool{}
	notes := map[string]api.Note{}
	for _, note := range snap.Notes {
		notes[note.Path] = note
	}
	for _, hit := range result.Hits {
		selected = append(selected, hit.Path)
		seen[hit.Path] = true
	}
	if request.IncludeLinks {
		snap = s.withLinks(snap)
		for _, p := range append([]string(nil), selected...) {
			for _, link := range neighborhood(snap.Links, p, "both") {
				target := link.To
				if target == p {
					target = link.From
				}
				if link.Resolved && !link.External && !reserved(target) && !seen[target] {
					if note, ok := notes[target]; ok && matches(note, request.Filters, s.config) {
						selected = append(selected, target)
						seen[target] = true
					}
				}
			}
		}
	}
	budget := min(request.MaxBytes, request.MaxTokens*4)
	var b strings.Builder
	for i, p := range selected {
		if err := ctx.Err(); err != nil {
			return api.ContextResult{}, err
		}
		n := notes[p]
		if reserved(p) {
			continue
		}
		ss := snippets(n, tokens(request.Query, s.config.Search.AccentFolding), s.config.Search.AccentFolding)
		start := strings.Count(string(n.Raw[:len(n.Raw)-len(n.Body)]), "\n") + 1
		heading := ""
		if len(ss) > 0 {
			start = ss[0].LineStart
			heading = ss[0].Heading
		}
		lines := strings.SplitAfter(string(n.Raw), "\n")
		end := min(len(lines), start+15)
		excerpt := strings.Join(lines[start-1:end], "")
		prefix := fmt.Sprintf("[%s:%d-%d %s]\n", p, start, end, n.Revision)
		remaining := budget - b.Len()
		if len(prefix) >= remaining {
			out.Truncated = true
			out.Omitted = len(selected) - i
			break
		}
		take := min(len(excerpt), remaining-len(prefix))
		text := truncateUTF8(excerpt, take)
		if len(text) < len(excerpt) {
			out.Truncated = true
			out.Omitted = len(selected) - i - 1
			end = start + strings.Count(text, "\n")
			if strings.HasSuffix(text, "\n") {
				end--
			}
			prefix = fmt.Sprintf("[%s:%d-%d %s]\n", p, start, end, n.Revision)
		}
		b.WriteString(prefix)
		b.WriteString(text)
		out.Sources = append(out.Sources, api.ContextSource{Path: p, Revision: n.Revision, Heading: heading, LineStart: start, LineEnd: end})
		if out.Truncated {
			break
		}
	}
	out.Text = b.String()
	out.EstimatedTokens = (len(out.Text) + 3) / 4
	if result.Total > len(result.Hits) {
		out.Truncated = true
		out.Omitted += result.Total - len(result.Hits)
	}
	return out, nil
}

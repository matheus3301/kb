package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) }
func fold(s string, accents bool) string {
	s = cases.Fold().String(norm.NFC.String(s))
	if !accents {
		return s
	}
	s = norm.NFD.String(s)
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, s)
}
func tokens(s string, accents bool) []string {
	return strings.FieldsFunc(fold(s, accents), func(r rune) bool { return !isWord(r) })
}
func contains(set []string, v string) bool {
	for _, x := range set {
		if x == v {
			return true
		}
	}
	return false
}
func matches(n api.Note, f api.Filters, c api.Config) bool {
	if len(f.PARA) > 0 && !contains(f.PARA, n.PARA) || len(f.Types) > 0 && !contains(f.Types, n.Type) || len(f.Statuses) > 0 && !contains(f.Statuses, n.Status) || len(f.Trust) > 0 && !contains(f.Trust, n.Trust) {
		return false
	}
	if f.Prefix != "" && n.Path != strings.TrimSuffix(f.Prefix, "/") && !strings.HasPrefix(n.Path, strings.TrimSuffix(f.Prefix, "/")+"/") {
		return false
	}
	if n.PARA == "archive" && !f.IncludeArchives && !c.Search.IncludeArchives {
		return false
	}
	if n.Status == "deprecated" && !f.IncludeDeprecated && !c.Search.IncludeDeprecated {
		return false
	}
	if len(f.Tags) > 0 {
		anyTag := false
		for _, tag := range f.Tags {
			if contains(n.Tags, tag) {
				anyTag = true
				break
			}
		}
		if !anyTag {
			return false
		}
	}

	if f.Stale != nil && *f.Stale != n.Stale {
		return false
	}
	if f.HasSources != nil {
		sources, ok := n.Metadata["sources"].([]any)
		if *f.HasSources != (ok && len(sources) > 0) {
			return false
		}
	}
	return true
}

type pageCursor struct {
	Generation string `json:"generation"`
	Query      string `json:"query"`
	Offset     int    `json:"offset"`
}

// List provides the same filtered inventory as search, with an empty query.
func (s *Store) List(ctx context.Context, request api.SearchRequest) (api.SearchResult, error) {
	request.Query = ""
	return s.Search(ctx, request)
}

// Search ranks exact Unicode tokens using weighted BM25 term frequencies.
func (s *Store) Search(ctx context.Context, request api.SearchRequest) (api.SearchResult, error) {
	if !utf8.ValidString(request.Query) || utf8.RuneCountInString(request.Query) > api.MaxQueryRunes {
		return api.SearchResult{}, fault("LIMIT_EXCEEDED", "query exceeds the character ceiling")
	}
	if request.Limit == 0 {
		request.Limit = api.DefaultHits
	}
	if request.Limit < 1 || request.Limit > api.MaxHits {
		return api.SearchResult{}, fault("LIMIT_EXCEEDED", "hit limit is outside the allowed range")
	}
	if request.Filters.Prefix != "" {
		if err := validPath(strings.TrimSuffix(request.Filters.Prefix, "/")); err != nil {
			return api.SearchResult{}, err
		}
	}
	snap, err := s.readSnapshot(ctx)
	if err != nil {
		return api.SearchResult{}, err
	}
	return s.searchSnapshot(ctx, snap, request)
}
func (s *Store) searchSnapshot(ctx context.Context, snap api.Snapshot, request api.SearchRequest) (api.SearchResult, error) {
	result := api.SearchResult{Generation: snap.Generation, Hits: []api.Hit{}, Warnings: append([]api.Diagnostic(nil), snap.Diagnostics...)}
	effective := request
	effective.Cursor = ""
	queryBytes, err := json.Marshal(struct {
		Request api.SearchRequest
		Config  api.Config
	}{effective, s.Config()})
	if err != nil {
		return result, err
	}
	queryHash := api.Revision(queryBytes)
	offset := 0
	if request.Cursor != "" {
		if len(request.Cursor) > 2048 {
			return result, fault("INVALID_CURSOR", "cursor is too long")
		}
		raw, err := base64.RawURLEncoding.DecodeString(request.Cursor)
		if err != nil {
			return result, fault("INVALID_CURSOR", "malformed cursor")
		}
		var cursor pageCursor
		if err := json.Unmarshal(raw, &cursor); err != nil || cursor.Offset < 0 {
			return result, fault("INVALID_CURSOR", "malformed cursor")
		}
		if cursor.Generation != snap.Generation {
			return result, fault("STALE_CURSOR", "inventory changed; restart pagination")
		}
		if cursor.Query != queryHash {
			return result, fault("INVALID_CURSOR", "cursor belongs to a different effective query")
		}
		offset = cursor.Offset
	}
	query := tokens(request.Query, s.config.Search.AccentFolding)
	querySet := map[string]bool{}
	for _, q := range query {
		querySet[q] = true
	}

	scores := make([]float64, len(snap.Notes))
	counts := make([]int, len(snap.Notes))
	nonempty := strings.TrimSpace(request.Query) != ""
	if nonempty && len(query) > 0 {
		index, err := s.lexical(ctx, snap)
		if err != nil {
			return result, err
		}
		for term := range querySet {
			postings := index.terms[term]
			idf := math.Log(1 + (float64(len(snap.Notes)-len(postings))+0.5)/(float64(len(postings))+0.5))
			for _, p := range postings {
				counts[p.doc]++
				scores[p.doc] += idf * (p.tf * 2.2) / (p.tf + 1.2*(0.25+0.75*index.lengths[p.doc]/index.average))
			}
		}
	}
	type candidate struct {
		doc   int
		score float64
	}
	candidates := []candidate{}
	for i, note := range snap.Notes {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if !matches(note, request.Filters, s.config) {
			continue
		}
		if nonempty {
			if len(query) == 0 || counts[i] != len(querySet) {
				continue
			}
			if request.Literal && !strings.Contains(fold(note.Title+"\n"+note.Path+"\n"+strings.Join(note.Tags, " ")+"\n"+note.Description+"\n"+note.Body, s.config.Search.AccentFolding), fold(request.Query, s.config.Search.AccentFolding)) {
				continue
			}
		}
		candidates = append(candidates, candidate{doc: i, score: scores[i]})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return snap.Notes[candidates[i].doc].Path < snap.Notes[candidates[j].doc].Path
	})
	hits := make([]api.Hit, 0, min(request.Limit, len(candidates)))
	result.Total = len(candidates)
	if offset > len(candidates) {
		return result, fault("INVALID_CURSOR", "cursor offset exceeds results")
	}
	end := min(len(candidates), offset+request.Limit)
	for _, candidate := range candidates[offset:end] {
		original := snap.Notes[candidate.doc]
		note := original
		note.Body = ""
		note.Raw = nil
		note.Metadata = cloneMap(note.Metadata)
		note.Tags = append([]string(nil), note.Tags...)
		hits = append(hits, api.Hit{Note: note, Score: candidate.score, ParseStatus: "valid", OKF: true, Snippets: snippets(original, query, s.config.Search.AccentFolding)})
	}
	result.Hits = hits
	if end < len(candidates) {
		raw, err := json.Marshal(pageCursor{Generation: snap.Generation, Query: queryHash, Offset: end})
		if err != nil {
			return result, err
		}
		result.Cursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return result, nil
}

type posting struct {
	doc int
	tf  float64
}
type lexicalIndex struct {
	generation string
	terms      map[string][]posting
	lengths    []float64
	average    float64
}

func (s *Store) lexical(ctx context.Context, snap api.Snapshot) (*lexicalIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.searchIndex != nil && s.searchIndex.generation == snap.Generation {
		return s.searchIndex, nil
	}
	index := &lexicalIndex{generation: snap.Generation, terms: map[string][]posting{}, lengths: make([]float64, len(snap.Notes))}
	for i, note := range snap.Notes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tf := map[string]float64{}
		for _, field := range []struct {
			text   string
			weight float64
		}{{note.Title, 6}, {note.Path, 4}, {strings.Join(note.Tags, " "), 5}, {note.Description, 3}, {note.Body, 1}} {
			ts := tokens(field.text, s.config.Search.AccentFolding)
			index.lengths[i] += float64(len(ts))
			for _, token := range ts {
				tf[token] += field.weight
			}
		}
		index.average += index.lengths[i]
		for term, freq := range tf {
			index.terms[term] = append(index.terms[term], posting{doc: i, tf: freq})
		}
	}
	if len(snap.Notes) > 0 {
		index.average /= float64(len(snap.Notes))
	}
	if index.average == 0 {
		index.average = 1
	}
	s.searchIndex = index
	return index, nil
}

func snippets(n api.Note, query []string, accents bool) []api.Snippet {
	raw := string(n.Raw)
	bodyStart := strings.Count(raw[:len(raw)-len(n.Body)], "\n")
	lines := strings.Split(n.Body, "\n")
	heading := ""
	var out []api.Snippet
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			heading = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
		}
		matched := len(query) == 0 && strings.TrimSpace(line) != ""
		lineTokens := tokens(line, accents)
		for _, q := range query {
			if contains(lineTokens, q) {
				matched = true
			}
		}
		if !matched {
			continue
		}
		text := truncateUTF8(line, 512)
		out = append(out, api.Snippet{Heading: heading, LineStart: bodyStart + i + 1, LineEnd: bodyStart + i + 1, Text: text})
		if len(out) == 2 {
			break
		}
	}
	return out
}
func truncateUTF8(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return s[:limit]
}

package write

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

const indexStart = "<!-- kb:index:start -->"
const indexEnd = "<!-- kb:index:end -->"

func escapeText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	value = html.EscapeString(value)
	return strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`", "|", "\\|").Replace(value)
}

func linkURL(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func (s *Service) indexes(files map[string][]byte) error {
	dirs := map[string]bool{".": true}
	for name := range files {
		if name == "log.md" {
			continue
		}
		for dir := path.Dir(name); ; dir = path.Dir(dir) {
			dirs[dir] = true
			if dir == "." {
				break
			}
		}
	}
	names := make([]string, 0, len(dirs))
	for dir := range dirs {
		names = append(names, dir)
	}
	sort.Strings(names)
	for _, dir := range names {
		index := path.Join(dir, "index.md")
		if !inScope(index, s.provider.Config()) {
			continue
		}
		raw, exists := files[index]
		if !exists {
			raw = []byte("# Index\n\n" + indexStart + "\n" + indexEnd + "\n")
		}
		text := string(raw)
		start, end, appendSafe, err := indexRegion(text)
		if err != nil {
			return err
		}
		if start < 0 {
			if !appendSafe {
				return problem("INVALID_INDEX", "an unclosed code fence prevents safely appending an index region")
			}
			if len(s.provider.Engine().ValidateReserved(index, raw)) > 0 {
				return problem("INVALID_INDEX", "unmanaged index requires explicit repair")
			}
			prefix := strings.TrimRight(text, "\r\n") + "\n\n"
			start = len(prefix)
			end = start + len(indexStart) + 1
			text = prefix + indexStart + "\n" + indexEnd + "\n"
		}
		manual := text[:start] + text[end+len(indexEnd):]
		targets := map[string]string{}
		for name, data := range files {
			if name == index || name == "log.md" || path.Dir(name) != dir {
				continue
			}
			title := strings.TrimSuffix(path.Base(name), ".md")
			note, _ := s.provider.Engine().Parse(name, data)
			if note.Title != "" {
				title = note.Title
			}
			line := "- [" + escapeText(title) + "](" + linkURL(path.Base(name)) + ")"
			if note.Description != "" {
				line += " — " + escapeText(note.Description)
			}
			targets[path.Base(name)] = line
		}
		for child := range dirs {
			if child == "." || child == dir || path.Dir(child) != dir {
				continue
			}
			childIndex := path.Join(child, "index.md")
			if !inScope(childIndex, s.provider.Config()) {
				continue
			}
			target := path.Base(child) + "/index.md"
			targets[target] = "- [" + escapeText(path.Base(child)) + "](" + linkURL(target) + ")"
		}
		keys := make([]string, 0, len(targets))
		for target := range targets {
			keys = append(keys, target)
		}
		sort.Strings(keys)
		var generated strings.Builder
		generated.WriteByte('\n')
		for _, target := range keys {
			if manualContains(manual, dir, path.Join(dir, target)) {
				continue
			}
			generated.WriteString(targets[target])
			generated.WriteByte('\n')
		}
		files[index] = []byte(text[:start+len(indexStart)] + generated.String() + text[end:])
	}
	return nil
}

func manualContains(manual, dir, target string) bool {
	for _, span := range markdownDestinations(manual) {
		resolved, _, ok := resolveReference(path.Join(dir, "index.md"), span.value)
		if ok && resolved == target {
			return true
		}
	}
	return false
}

func (s *Service) log(files map[string][]byte, summary, actor string, warnings []api.Diagnostic, before map[string][]byte) error {
	raw, exists := files["log.md"]
	if exists && len(s.provider.Engine().ValidateReserved("log.md", raw)) > 0 {
		return problem("INVALID_LOG", "existing log requires explicit repair")
	}
	if !exists {
		raw = []byte("# Log\n")
	}
	var links []string
	for name, data := range files {
		if name == "log.md" || reserved(name) || string(data) == string(before[name]) {
			continue
		}
		links = append(links, "["+escapeText(strings.TrimSuffix(name, ".md"))+"]("+linkURL(name)+")")
	}
	sort.Strings(links)
	line := "- " + escapeText(summary) + " (" + escapeText(actor) + ")"
	if len(links) > 0 {
		line += ": " + strings.Join(links, ", ")
	}
	for _, warning := range warnings {
		if warning.Code == "VERIFICATION_CLEARED" {
			line += "; verification cleared because content changed"
			break
		}
	}
	line += "\n"
	date := s.now().UTC().Format("2006-01-02")
	text := string(raw)
	heading := "## " + date + "\n"
	firstDate := strings.Index(text, "\n## ")
	if firstDate >= 0 && strings.HasPrefix(text[firstDate+1:], heading) {
		at := firstDate + 1 + len(heading)
		text = text[:at] + "\n" + line + text[at:]
	} else if firstDate >= 0 {
		at := firstDate + 1
		text = text[:at] + heading + "\n" + line + "\n" + text[at:]
	} else {
		text = strings.TrimRight(text, "\r\n") + "\n\n" + heading + "\n" + line
	}
	files["log.md"] = []byte(text)
	return nil
}

func (s *Service) Maintain(ctx context.Context, dryRun bool) (api.Receipt, error) {
	return s.locked(ctx, func(root *os.Root) (api.Receipt, error) {
		if dryRun {
			pending, err := readImage(root, pendingPath, maxJournalBytes)
			if err != nil {
				return api.Receipt{}, err
			}
			if pending.Exists {
				return api.Receipt{}, problem("RECOVERY_REQUIRED", "pending transaction requires recovery")
			}
		} else if _, err := s.recover(ctx, root, false); err != nil {
			return api.Receipt{}, err
		}
		snapshot, err := s.provider.SnapshotUnlocked(ctx)
		if err != nil {
			return api.Receipt{}, err
		}
		before := cloneFiles(snapshot.Files)
		if err := s.conform(before); err != nil {
			return api.Receipt{}, err
		}
		for name := range before {
			if !inScope(name, s.provider.Config()) {
				return api.Receipt{}, problem("PATH_OUTSIDE_ROOT", "provider inventory is outside scope")
			}
		}
		after := cloneFiles(before)
		if s.provider.Config().Write.MaintainIndexes {
			if err := s.indexes(after); err != nil {
				return api.Receipt{}, err
			}
		}
		if changed(before, after) && s.provider.Config().Write.MaintainLog {
			if err := s.log(after, "Maintain navigation indexes", "kb", nil, before); err != nil {
				return api.Receipt{}, err
			}
		}
		if err := s.conform(after); err != nil {
			return api.Receipt{}, err
		}
		key := fmt.Sprintf("maintain:%s:%s", api.Generation(before), api.Generation(after))
		return s.publish(ctx, root, before, after, key, api.Revision([]byte(key)), dryRun, nil)
	})
}

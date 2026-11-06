package setup

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

func newline(raw []byte) string {
	if bytes.Contains(raw, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}
func blockText(a Artifact, raw []byte) ([]byte, error) {
	if !validKey(a.Key) {
		return nil, invalid("unsafe managed block key")
	}
	nl := newline(raw)
	text := strings.ReplaceAll(string(a.Content), "\r\n", "\n")
	text = strings.ReplaceAll(strings.TrimRight(text, "\n"), "\n", nl)
	if strings.Contains(text, "<!-- kb:") || strings.Contains(text, "# kb:") {
		return nil, invalid("managed block content contains a marker")
	}
	if a.Kind == "gitignore" {
		return []byte("# kb:" + a.Key + ":v1 begin" + nl + text + nl + "# kb:" + a.Key + ":v1 end"), nil
	}
	return []byte("<!-- kb:" + a.Key + ":v1 begin -->" + nl + text + nl + "<!-- kb:" + a.Key + ":v1 end -->"), nil
}
func blockSpan(raw []byte, key string, kinds ...string) (int, int, error) {
	prefix, suffix := "<!-- kb:", " -->"
	if len(kinds) > 0 && kinds[0] == "gitignore" {
		prefix, suffix = "# kb:", ""
	}
	startMarker := []byte(prefix + key + ":v1 begin" + suffix)
	endMarker := []byte(prefix + key + ":v1 end" + suffix)
	starts, ends := bytes.Count(raw, startMarker), bytes.Count(raw, endMarker)
	if starts == 0 && ends == 0 {
		if bytes.Contains(raw, []byte(prefix+key+":")) {
			return 0, 0, conflict("unsupported or malformed managed block: " + key)
		}
		return -1, -1, nil
	}
	if starts != 1 || ends != 1 {
		return 0, 0, conflict("duplicate or orphaned managed block: " + key)
	}
	start, end := bytes.Index(raw, startMarker), bytes.Index(raw, endMarker)+len(endMarker)
	if end <= start {
		return 0, 0, conflict("reversed managed block: " + key)
	}
	if (start > 0 && raw[start-1] != '\n' && !(start == 3 && bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}))) || (end < len(raw) && raw[end] != '\r' && raw[end] != '\n') {
		return 0, 0, conflict("managed markers must occupy complete lines")
	}
	return start, end, nil
}
func editBlock(raw []byte, a Artifact, old *asset, remove bool) ([]byte, []byte, bool, error) {
	if !utf8.Valid(raw) {
		return nil, nil, false, conflict("instruction file is not UTF-8")
	}
	text, err := blockText(a, raw)
	if err != nil {
		return nil, nil, false, err
	}
	start, end, err := blockSpan(raw, a.Key, a.Kind)
	if err != nil {
		return nil, nil, false, err
	}
	if old != nil && old.Borrowed {
		return raw, old.Managed, true, nil
	}
	if start >= 0 {
		existing := raw[start:end]
		if old == nil {
			if bytes.Equal(existing, text) {
				return raw, existing, true, nil
			}
			return nil, nil, false, conflict("foreign managed block: " + a.Key)
		}
		if !bytes.Equal(existing, old.Managed) {
			return nil, nil, false, conflict("modified owned managed block: " + a.Key)
		}
		if remove {
			return splice(raw, start, end, nil), nil, false, nil
		}
		return splice(raw, start, end, text), text, false, nil
	}
	if old != nil {
		return nil, nil, false, conflict("missing owned managed block: " + a.Key)
	}
	if remove {
		return raw, nil, false, nil
	}
	addition := append([]byte(nil), text...)
	if len(raw) > 0 && !bytes.HasSuffix(raw, []byte("\n")) {
		addition = append([]byte(newline(raw)), addition...)
	}
	addition = append(addition, []byte(newline(raw))...)
	return append(append([]byte(nil), raw...), addition...), text, false, nil
}

// tomlTables identifies physical table boundaries while respecting multiline
// strings. Decoding validates the full document before this conservative scan.
func tomlTables(raw []byte) ([]int, []string, error) {
	var decoded map[string]any
	if _, err := toml.Decode(string(raw), &decoded); err != nil {
		return nil, nil, conflict("invalid TOML configuration")
	}
	var starts []int
	var names []string
	offset := 0
	multi := ""
	for _, line := range bytes.SplitAfter(raw, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if multi == "" && strings.HasPrefix(s, "[") {
			if strings.HasPrefix(s, "[[") {
				starts = append(starts, offset)
				names = append(names, "[array]")
			} else {
				end := strings.IndexByte(s, ']')
				if end < 0 {
					return nil, nil, conflict("unsupported TOML table header")
				}
				name := strings.TrimSpace(s[1:end])
				starts = append(starts, offset)
				names = append(names, name)
			}
		}
		// Triple-quoted strings may contain apparent [headers]. A stateful lexical
		// scan also ignores quotes inside comments and ordinary quoted strings.
		quote := byte(0)
		for i := 0; i < len(line); i++ {
			if multi != "" {
				if multi == "\"\"\"" && line[i] == '\\' {
					i++
					continue
				}
				if i+3 <= len(line) && string(line[i:i+3]) == multi {
					multi = ""
					i += 2
				}
				continue
			}
			if quote != 0 {
				if line[i] == '\\' && quote == '"' {
					i++
					continue
				}
				if line[i] == quote {
					quote = 0
				}
				continue
			}
			if line[i] == '#' {
				break
			}
			if line[i] == '\'' || line[i] == '"' {
				if i+3 <= len(line) && line[i] == line[i+1] && line[i] == line[i+2] {
					multi = string(line[i : i+3])
					i += 2
				} else {
					quote = line[i]
				}
			}
		}
		offset += len(line)
	}
	return starts, names, nil
}
func tomlSpan(raw []byte, key string) (int, int, error) {
	starts, names, err := tomlTables(raw)
	if err != nil {
		return 0, 0, err
	}
	found := -1
	for i, name := range names {
		if name == key {
			if found >= 0 {
				return 0, 0, conflict("duplicate TOML section")
			}
			found = i
		}
		if name != key && strings.HasPrefix(name, key+".") {
			return 0, 0, conflict("nested TOML sections at owned table")
		}
	}
	if found < 0 {
		return -1, -1, nil
	}
	end := len(raw)
	if found+1 < len(starts) {
		end = starts[found+1]
	}
	return starts[found], end, nil
}
func editTOML(raw []byte, a Artifact, old *asset, remove bool) ([]byte, []byte, bool, error) {
	if !validKey(a.Key) {
		return nil, nil, false, invalid("unsupported TOML managed table key")
	}
	start, end, err := tomlSpan(raw, a.Key)
	if err != nil {
		return nil, nil, false, err
	}
	text := a.Content
	if !strings.HasPrefix(strings.TrimSpace(string(text)), "["+a.Key+"]") {
		text = append([]byte("["+a.Key+"]\n"), text...)
	}
	text = []byte(strings.ReplaceAll(strings.ReplaceAll(string(text), "\r\n", "\n"), "\n", newline(raw)))
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, []byte(newline(raw))...)
	}
	if _, _, err := tomlTables(text); err != nil {
		return nil, nil, false, invalid("invalid managed TOML section")
	}
	if old != nil && old.Borrowed {
		return raw, old.Managed, true, nil
	}
	if start >= 0 {
		existing := raw[start:end]
		if old == nil {
			if bytes.Equal(existing, text) {
				return raw, existing, true, nil
			}
			return nil, nil, false, conflict("foreign TOML table: " + a.Key)
		}
		if !bytes.Equal(existing, old.Managed) {
			return nil, nil, false, conflict("modified owned TOML table: " + a.Key)
		}
		out := splice(raw, start, end, text)
		if remove {
			out = splice(raw, start, end, nil)
		}
		if _, _, err := tomlTables(out); err != nil {
			return nil, nil, false, err
		}
		if remove {
			return out, nil, false, nil
		}
		return out, text, false, nil
	}
	if old != nil {
		return nil, nil, false, conflict("missing owned TOML table: " + a.Key)
	}
	if remove {
		return raw, nil, false, nil
	}
	addition := text
	if len(raw) > 0 && !bytes.HasSuffix(raw, []byte("\n")) {
		addition = append([]byte(newline(raw)), text...)
	}
	out := append(append([]byte(nil), raw...), addition...)
	if _, _, err := tomlTables(out); err != nil {
		return nil, nil, false, err
	}
	return out, text, false, nil
}
func editArtifact(raw []byte, a Artifact, old *asset, remove bool) ([]byte, []byte, bool, error) {
	if old != nil && old.Borrowed {
		if remove {
			return raw, old.Managed, true, nil
		}
		_, managed, borrowed, err := editArtifact(raw, a, nil, false)
		if err != nil {
			return nil, nil, false, err
		}
		if !borrowed {
			return nil, nil, false, conflict("borrowed artifact is missing or no longer matches; preserve it and reconcile explicitly")
		}
		return raw, managed, true, nil
	}
	switch a.Kind {
	case "file":
		if old != nil && old.Borrowed {
			return raw, old.Managed, true, nil
		}
		if old != nil && !bytes.Equal(raw, old.Managed) {
			return nil, nil, false, conflict("modified owned file")
		}
		if old == nil && raw != nil {
			if bytes.Equal(raw, a.Content) {
				return raw, raw, true, nil
			}
			return nil, nil, false, conflict("custom file already exists")
		}
		if remove {
			return nil, nil, false, nil
		}
		return a.Content, a.Content, false, nil
	case "block", "gitignore":
		return editBlock(raw, a, old, remove)
	case "json", "json-array", "jsonc", "jsonc-array":
		return editJSON(raw, a, old, remove)
	case "toml":
		return editTOML(raw, a, old, remove)
	default:
		return nil, nil, false, fmt.Errorf("%w", invalid("unknown artifact kind"))
	}
}

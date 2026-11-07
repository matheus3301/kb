package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type jsonMember struct {
	key   string
	start int
	value *jsonNode
}
type jsonNode struct {
	start, end int
	kind       byte
	members    []jsonMember
	items      []*jsonNode
}
type jsonParser struct {
	raw   []byte
	pos   int
	depth int
}

func parseJSON(raw []byte) (*jsonNode, error) {
	if !json.Valid(raw) {
		return nil, conflict("invalid JSON configuration")
	}
	p := jsonParser{raw: raw}
	n, err := p.value()
	if err != nil {
		return nil, err
	}
	p.space()
	if p.pos != len(raw) {
		return nil, conflict("trailing JSON document")
	}
	return n, nil
}
func (p *jsonParser) space() {
	for p.pos < len(p.raw) && bytes.ContainsRune([]byte(" \t\r\n"), rune(p.raw[p.pos])) {
		p.pos++
	}
}
func (p *jsonParser) quoted() (string, error) {
	start := p.pos
	p.pos++
	for p.pos < len(p.raw) {
		c := p.raw[p.pos]
		p.pos++
		if c == '\\' {
			p.pos++
			continue
		}
		if c == '"' {
			var s string
			err := json.Unmarshal(p.raw[start:p.pos], &s)
			return s, err
		}
	}
	return "", conflict("unterminated JSON string")
}
func (p *jsonParser) value() (*jsonNode, error) {
	p.space()
	if p.pos >= len(p.raw) {
		return nil, conflict("missing JSON value")
	}
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 128 {
		return nil, conflict("JSON nesting exceeds 128")
	}
	n := &jsonNode{start: p.pos, kind: p.raw[p.pos]}
	switch n.kind {
	case '{':
		p.pos++
		p.space()
		seen := map[string]bool{}
		for p.raw[p.pos] != '}' {
			start := p.pos
			key, err := p.quoted()
			if err != nil {
				return nil, err
			}
			if seen[key] {
				return nil, conflict("duplicate JSON object key")
			}
			seen[key] = true
			p.space()
			p.pos++
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			n.members = append(n.members, jsonMember{key: key, start: start, value: v})
			p.space()
			if p.raw[p.pos] == ',' {
				p.pos++
				p.space()
				continue
			}
			break
		}
		p.pos++
	case '[':
		p.pos++
		p.space()
		for p.raw[p.pos] != ']' {
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			n.items = append(n.items, v)
			p.space()
			if p.raw[p.pos] == ',' {
				p.pos++
				p.space()
				continue
			}
			break
		}
		p.pos++
	case '"':
		if _, err := p.quoted(); err != nil {
			return nil, err
		}
	default:
		for p.pos < len(p.raw) && !bytes.ContainsRune([]byte(",]} \t\r\n"), rune(p.raw[p.pos])) {
			p.pos++
		}
	}
	n.end = p.pos
	return n, nil
}
func member(n *jsonNode, key string) (*jsonNode, int) {
	for i, m := range n.members {
		if m.key == key {
			return m.value, i
		}
	}
	return nil, -1
}
func jsonAt(n *jsonNode, parts []string) (*jsonNode, error) {
	for _, part := range parts {
		if n.kind != '{' {
			return nil, conflict("JSON path traverses a nonobject")
		}
		n, _ = member(n, part)
		if n == nil {
			return nil, nil
		}
	}
	return n, nil
}
func jsonParts(key string) ([]string, error) {
	parts := strings.Split(key, ".")
	for _, p := range parts {
		if p == "" {
			return nil, invalid("empty JSON key segment")
		}
	}
	return parts, nil
}
func splice(raw []byte, start, end int, replacement []byte) []byte {
	result := make([]byte, 0, len(raw)-(end-start)+len(replacement))
	result = append(result, raw[:start]...)
	result = append(result, replacement...)
	return append(result, raw[end:]...)
}
func jsonEqual(a, b []byte) bool {
	var ac, bc bytes.Buffer
	if json.Compact(&ac, a) != nil || json.Compact(&bc, b) != nil {
		return false
	}
	return bytes.Equal(ac.Bytes(), bc.Bytes())
}

// setJSON changes exactly one value/member; untouched whitespace and numbers are
// never decoded and reserialized. Empty inserted parents are restored from the
// per-file baseline on clean uninstall, not inferred as user-owned or disposable.
func setJSON(raw []byte, key string, value []byte, remove bool, format ...string) ([]byte, []byte, bool, error) {
	if raw == nil {
		raw = []byte("{}\n")
	}
	kind := "json"
	if len(format) > 0 {
		kind = format[0]
	}
	n, err := parseJSONFormat(raw, kind)
	if err != nil {
		return nil, nil, false, err
	}
	parts, err := jsonParts(key)
	if err != nil {
		return nil, nil, false, err
	}
	return setJSONNode(raw, n, parts, value, remove)
}
func setJSONNode(raw []byte, n *jsonNode, parts []string, value []byte, remove bool) ([]byte, []byte, bool, error) {
	if n.kind != '{' {
		return nil, nil, false, conflict("JSON configuration path is not an object")
	}
	child, index := member(n, parts[0])
	if len(parts) > 1 {
		if child != nil {
			return setJSONNode(raw, child, parts[1:], value, remove)
		}
		if remove {
			return raw, nil, false, nil
		}
		wrapped := value
		for i := len(parts) - 1; i >= 1; i-- {
			k, _ := json.Marshal(parts[i])
			wrapped = append(append(append([]byte{'{'}, k...), ':'), append(wrapped, '}')...)
		}
		return setJSONNode(raw, n, parts[:1], wrapped, false)
	}
	if child != nil {
		existing := raw[child.start:child.end]
		if remove {
			start, end := n.members[index].start, child.end
			if index < len(n.members)-1 {
				comma := commaOffset(raw, end, n.members[index+1].start)
				end += comma + 1
			} else if index > 0 {
				start = n.members[index-1].value.end + commaOffset(raw, n.members[index-1].value.end, start)
			} else if trailingComma(raw, n) {
				end += commaOffset(raw, end, n.end-1) + 1
			}
			return splice(raw, start, end, nil), existing, true, nil
		}
		return splice(raw, child.start, child.end, value), existing, true, nil
	}
	if remove {
		return raw, nil, false, nil
	}
	k, _ := json.Marshal(parts[0])
	addition := append(append(append([]byte{}, k...), ':'), value...)
	if len(n.members) > 0 && !trailingComma(raw, n) {
		addition = append([]byte{','}, addition...)
	}
	return splice(raw, n.end-1, n.end-1, addition), nil, false, nil
}

func editJSON(raw []byte, a Artifact, old *asset, remove bool) ([]byte, []byte, bool, error) {
	out, managed, borrowed, err := editJSONEntry(raw, a, old, remove)
	if err != nil {
		return nil, nil, false, err
	}
	if _, err := parseJSONFormat(out, a.Kind); err != nil {
		return nil, nil, false, conflict("managed JSON edit would produce invalid configuration")
	}
	return out, managed, borrowed, nil
}

func editJSONEntry(raw []byte, a Artifact, old *asset, remove bool) ([]byte, []byte, bool, error) {
	if _, err := parseJSON(a.Content); err != nil {
		return nil, nil, false, invalid("artifact JSON value: " + err.Error())
	}
	source := raw
	if source == nil {
		source = []byte("{}\n")
	}
	n, err := parseJSONFormat(source, a.Kind)
	if err != nil {
		return nil, nil, false, err
	}
	parts, err := jsonParts(a.Key)
	if err != nil {
		return nil, nil, false, err
	}
	node, err := jsonAt(n, parts)
	if err != nil {
		return nil, nil, false, err
	}
	if a.Kind == "json-array" || a.Kind == "jsonc-array" {
		return editJSONArray(source, node, a, old, remove)
	}
	var existing []byte
	if node != nil {
		existing = source[node.start:node.end]
	}
	if old != nil && !old.Borrowed && !bytes.Equal(existing, old.Managed) {
		return nil, nil, false, conflict("modified owned JSON entry " + a.Key)
	}
	if old == nil && node != nil {
		if !jsonEqual(existing, a.Content) {
			return nil, nil, false, conflict("foreign JSON entry " + a.Key)
		}
		return raw, existing, true, nil
	}
	if old != nil && old.Borrowed {
		return raw, existing, true, nil
	}
	if remove && node == nil {
		return nil, nil, false, conflict("missing owned JSON entry " + a.Key)
	}
	out, _, _, err := setJSON(source, a.Key, a.Content, remove, a.Kind)
	if err != nil {
		return nil, nil, false, err
	}
	if remove {
		return out, nil, false, nil
	}
	// The inserted value remains byte-identical to Content, even when parent
	// objects were synthesized to reach the requested entry.
	return out, append([]byte(nil), a.Content...), false, nil
}
func editJSONArray(raw []byte, node *jsonNode, a Artifact, old *asset, remove bool) ([]byte, []byte, bool, error) {
	if node == nil {
		if old != nil {
			return nil, nil, false, conflict("missing owned JSON hook array " + a.Key)
		}
		if remove {
			return raw, nil, false, nil
		}
		out, _, _, err := setJSON(raw, a.Key, append(append([]byte{'['}, a.Content...), ']'), false, a.Kind)
		return out, a.Content, false, err
	}
	if node.kind != '[' {
		return nil, nil, false, conflict("hook destination is not an array: " + a.Key)
	}
	index := -1
	expected := a.Content
	if old != nil {
		expected = old.Managed
	}
	for i, v := range node.items {
		if jsonEqual(raw[v.start:v.end], expected) {
			if index != -1 {
				return nil, nil, false, conflict("duplicate hook entry: " + a.Key)
			}
			index = i
		}
	}
	if old == nil && index >= 0 {
		v := node.items[index]
		return raw, raw[v.start:v.end], true, nil
	}
	if old != nil && old.Borrowed {
		return raw, old.Managed, true, nil
	}
	if old != nil && index < 0 {
		return nil, nil, false, conflict("modified or missing owned hook entry: " + a.Key)
	}
	if index >= 0 {
		v := node.items[index]
		if old != nil && !bytes.Equal(raw[v.start:v.end], old.Managed) {
			return nil, nil, false, conflict("modified owned hook bytes: " + a.Key)
		}
		if !remove {
			for i, other := range node.items {
				if i != index && jsonEqual(raw[other.start:other.end], a.Content) {
					return nil, nil, false, conflict("new hook registration duplicates a foreign handler: " + a.Key)
				}
			}
			return splice(raw, v.start, v.end, a.Content), a.Content, false, nil
		}
		start, end := v.start, v.end
		if index < len(node.items)-1 {
			end += commaOffset(raw, end, node.items[index+1].start) + 1
		} else if index > 0 {
			start = node.items[index-1].end + commaOffset(raw, node.items[index-1].end, start)
		} else if trailingComma(raw, node) {
			end += commaOffset(raw, end, node.end-1) + 1
		}
		return splice(raw, start, end, nil), nil, false, nil
	}
	if remove {
		return nil, nil, false, fmt.Errorf("%w", conflict("missing hook entry"))
	}
	addition := a.Content
	if len(node.items) > 0 && !trailingComma(raw, node) {
		addition = append([]byte{','}, addition...)
	}
	return splice(raw, node.end-1, node.end-1, addition), a.Content, false, nil
}

package core

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// decimalNumber normalizes finite YAML decimals to JSON without binary floats.
// Nonfinite YAML scalars retain their existing interoperability behavior.
func decimalNumber(value string) (json.Number, bool) {
	value = strings.TrimPrefix(strings.ReplaceAll(value, "_", ""), "+")
	mantissa, exponent := value, ""
	if i := strings.IndexAny(value, "eE"); i >= 0 {
		mantissa, exponent = value[:i], value[i:]
	}
	sign := ""
	if strings.HasPrefix(mantissa, "-") {
		sign, mantissa = "-", mantissa[1:]
	}
	if strings.HasPrefix(mantissa, ".") {
		mantissa = "0" + mantissa
	}
	if strings.HasSuffix(mantissa, ".") {
		mantissa += "0"
	}
	for len(mantissa) > 1 && mantissa[0] == '0' && mantissa[1] >= '0' && mantissa[1] <= '9' {
		mantissa = mantissa[1:]
	}
	value = sign + mantissa + exponent
	if len(mantissa) == 0 || mantissa[0] < '0' || mantissa[0] > '9' || !json.Valid([]byte(value)) {
		return "", false
	}
	return json.Number(value), true
}

// updatedNode retains unchanged children, comments, styles and anchors when a
// structural metadata edit cannot be represented by scalar span replacements.
func updatedNode(old *yaml.Node, value any) (*yaml.Node, error) {
	if sameValue(nodeValue(old), value) {
		return old, nil
	}
	if old.Kind == yaml.AliasNode {
		node, err := updatedNode(old.Alias, value)
		if err != nil {
			return nil, err
		}
		// Editing an alias must not change its original template or duplicate its
		// anchors. Unchanged aliases inside the expanded value still refer to it.
		node = withoutAnchors(node)
		node.HeadComment = strings.TrimSpace(node.HeadComment + "\n" + old.HeadComment)
		node.LineComment = strings.TrimSpace(node.LineComment + "\n" + old.LineComment)
		node.FootComment = strings.TrimSpace(node.FootComment + "\n" + old.FootComment)
		if node.Kind != yaml.ScalarNode && node.Style&yaml.FlowStyle == 0 {
			node.HeadComment = strings.TrimSpace(node.HeadComment + "\n" + node.LineComment)
			node.LineComment = ""
		}
		return node, nil
	}
	node := *old
	node.Content = nil
	if values, ok := value.(map[string]any); ok && old.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(old.Content); i += 2 {
			key := old.Content[i]
			if next, exists := values[key.Value]; exists {
				child, err := updatedNode(old.Content[i+1], next)
				if err != nil {
					return nil, err
				}
				node.Content = append(node.Content, key, child)
				seen[key.Value] = true
			}
		}
		for _, key := range sortedKeys(values) {
			if seen[key] {
				continue
			}
			var child yaml.Node
			if err := encodeValue(&child, values[key]); err != nil {
				return nil, err
			}
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &child)
		}
		return &node, nil
	}
	if values, ok := value.([]any); ok && old.Kind == yaml.SequenceNode {
		for i, next := range values {
			var child *yaml.Node
			var err error
			if i < len(old.Content) {
				child, err = updatedNode(old.Content[i], next)
			} else {
				child = &yaml.Node{}
				err = encodeValue(child, next)
			}
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return &node, nil
	}
	var replacement yaml.Node
	if err := encodeValue(&replacement, value); err != nil {
		return nil, err
	}
	if replacement.Kind == old.Kind && replacement.Tag == old.Tag {
		replacement.Style = old.Style
	}
	replacement.Anchor = old.Anchor
	replacement.HeadComment, replacement.LineComment, replacement.FootComment = old.HeadComment, old.LineComment, old.FootComment
	return &replacement, nil
}

func withoutAnchors(old *yaml.Node) *yaml.Node {
	node := *old
	node.Anchor = ""
	node.Content = nil
	for _, child := range old.Content {
		node.Content = append(node.Content, withoutAnchors(child))
	}
	return &node
}

type scalarEdit struct {
	start, end int
	value      []byte
}

// patchHeaderScalars changes only existing scalar tokens when collection shape
// is unchanged. Nested annotations, indentation and numeric lexemes stay exact.
func patchHeaderScalars(f frontmatter, changes map[string]any) (frontmatter, error) {
	var edits []scalarEdit
	for i := 0; i < len(f.node.Content); i += 2 {
		key, node := f.node.Content[i], f.node.Content[i+1]
		value, exists := changes[key.Value]
		if !exists {
			continue
		}
		part, ok := scalarEdits(f.header, node, value, f.node.Style&yaml.FlowStyle != 0)
		if ok {
			edits = append(edits, part...)
			delete(changes, key.Value)
		}
	}
	if len(edits) == 0 {
		return f, nil
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	header := bytes.Clone(f.header)
	for _, edit := range edits {
		header = append(append(append([]byte{}, header[:edit.start]...), edit.value...), header[edit.end:]...)
	}
	raw := append([]byte("---"+f.newline), header...)
	raw = append(raw, []byte("---"+f.newline)...)
	raw = append(raw, f.body...)
	return splitFront(raw)
}

func scalarEdits(header []byte, old *yaml.Node, value any, flow bool) ([]scalarEdit, bool) {
	if sameValue(nodeValue(old), value) {
		return nil, true
	}
	flow = flow || old.Style&yaml.FlowStyle != 0
	var edits []scalarEdit
	switch old.Kind {
	case yaml.MappingNode:
		values, ok := value.(map[string]any)
		if !ok || len(values)*2 != len(old.Content) {
			return nil, false
		}
		for i := 0; i < len(old.Content); i += 2 {
			next, exists := values[old.Content[i].Value]
			part, ok := scalarEdits(header, old.Content[i+1], next, flow)
			if !exists || !ok {
				return nil, false
			}
			edits = append(edits, part...)
		}
	case yaml.SequenceNode:
		values, ok := value.([]any)
		if !ok || len(values) != len(old.Content) {
			return nil, false
		}
		for i, next := range values {
			part, ok := scalarEdits(header, old.Content[i], next, flow)
			if !ok {
				return nil, false
			}
			edits = append(edits, part...)
		}
	case yaml.ScalarNode:
		start, end, ok := scalarSpan(header, old, flow)
		if !ok {
			return nil, false
		}
		node, err := updatedNode(old, value)
		if err != nil || node.Kind != yaml.ScalarNode {
			return nil, false
		}
		node = withoutAnchors(node)
		node.HeadComment, node.LineComment, node.FootComment = "", "", ""
		if flow && node.Tag == "!!str" && node.Style == 0 {
			node.Style = yaml.DoubleQuotedStyle
		}
		encoded, err := encodeNode(node)
		encoded = bytes.TrimSuffix(encoded, []byte("\n"))
		if err != nil || bytes.ContainsAny(encoded, "\r\n") {
			return nil, false
		}
		edits = append(edits, scalarEdit{start, end, encoded})
	default:
		return nil, false
	}
	return edits, true
}

func scalarSpan(header []byte, node *yaml.Node, flow bool) (int, int, bool) {
	if node.Anchor != "" || node.Style&(yaml.TaggedStyle|yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		return 0, 0, false
	}
	start := 0
	for line := 1; line < node.Line; line++ {
		i := bytes.IndexByte(header[start:], '\n')
		if i < 0 {
			return 0, 0, false
		}
		start += i + 1
	}
	for column := 1; column < node.Column && start < len(header); column++ {
		_, size := utf8.DecodeRune(header[start:])
		start += size
	}
	if start >= len(header) {
		return 0, 0, false
	}
	end := start
	quote := header[start]
	if quote == '\'' || quote == '"' {
		for end = start + 1; end < len(header) && header[end] != '\n' && header[end] != '\r'; end++ {
			if quote == '"' && header[end] == '\\' {
				end++
				continue
			}
			if header[end] != quote {
				continue
			}
			if quote == '\'' && end+1 < len(header) && header[end+1] == '\'' {
				end++
				continue
			}
			end++
			break
		}
	} else {
		for end < len(header) && header[end] != '\r' && header[end] != '\n' {
			if flow && strings.ContainsRune(",[]{}", rune(header[end])) || header[end] == '#' && end > start && (header[end-1] == ' ' || header[end-1] == '\t') {
				break
			}
			end++
		}
	}
	if end > len(header) {
		return 0, 0, false
	}
	end = start + len(bytes.TrimRight(header[start:end], " \t"))
	var parsed yaml.Node
	if err := yaml.Unmarshal(header[start:end], &parsed); err != nil || len(parsed.Content) != 1 || parsed.Content[0].Kind != yaml.ScalarNode || parsed.Content[0].Value != node.Value || parsed.Content[0].Tag != node.Tag {
		return 0, 0, false
	}
	return start, end, true
}

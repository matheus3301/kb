package okf

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

func encodeNode(n *yaml.Node) ([]byte, error) {
	var b bytes.Buffer
	e := yaml.NewEncoder(&b)
	e.SetIndent(2)
	err := e.Encode(n)
	return b.Bytes(), errorsJoin(err, e.Close())
}
func errorsJoin(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// patchHeader replaces only changed top-level entries in block mappings.
// Untouched key spans retain their original bytes, including comments and style.
func patchHeader(f frontmatter, changes map[string]any, removes map[string]bool) ([]byte, error) {
	var err error
	f, err = patchHeaderScalars(f, changes)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 && len(removes) == 0 {
		return f.header, nil
	}
	if f.node.Style&yaml.FlowStyle != 0 {
		var content []*yaml.Node
		for i := 0; i < len(f.node.Content); i += 2 {
			k, v := f.node.Content[i], f.node.Content[i+1]
			if removes[k.Value] {
				continue
			}
			if x, ok := changes[k.Value]; ok {
				node, err := updatedNode(v, x)
				if err != nil {
					return nil, err
				}
				v = node
				delete(changes, k.Value)
			}
			content = append(content, k, v)
		}
		for _, k := range sortedKeys(changes) {
			var v yaml.Node
			if err := encodeValue(&v, changes[k]); err != nil {
				return nil, err
			}
			content = append(content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, &v)
		}
		f.node.Content = content
		node := f.node
		if f.document != nil {
			node = f.document
		}
		encoded, err := encodeNode(node)
		if err != nil {
			return nil, err
		}
		return []byte(strings.ReplaceAll(string(encoded), "\n", f.newline)), nil
	}
	lines := strings.SplitAfter(string(f.header), "\n")
	footer := len(lines)
	if f.document != nil {
		footer = beforeHeaderComment(lines, footer, f.document.FootComment)
		if !endsInBlockScalar(f.node) {
			for footer > 0 && strings.TrimSpace(lines[footer-1]) == "" {
				footer--
			}
		}
	}
	var out strings.Builder
	cursor := 0
	for i := 0; i < len(f.node.Content); i += 2 {
		key, value := f.node.Content[i], f.node.Content[i+1]
		start := key.Line - 1
		end := footer
		if i+2 < len(f.node.Content) {
			next := f.node.Content[i+2]
			end = beforeHeaderComment(lines, next.Line-1, next.HeadComment)
		}
		if !endsInBlockScalar(value) {
			for end > start+1 && strings.TrimSpace(lines[end-1]) == "" {
				end--
			}
		}
		out.WriteString(strings.Join(lines[cursor:start], ""))
		if x, ok := changes[key.Value]; ok && !removes[key.Value] {
			v, err := updatedNode(value, x)
			if err != nil {
				return nil, err
			}
			// Leading comments are already in the retained prefix above this key.
			encodedKey := *key
			encodedKey.HeadComment = ""
			mapping := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{&encodedKey, v}}
			encoded, err := encodeNode(mapping)
			if err != nil {
				return nil, err
			}
			out.WriteString(strings.ReplaceAll(string(encoded), "\n", f.newline))
			delete(changes, key.Value)
		} else if !removes[key.Value] {
			out.WriteString(strings.Join(lines[start:end], ""))
		} else if key.FootComment != "" {
			// Clearing verification removes the field, not the standalone
			// annotations following it. Keep their original bytes and newlines.
			commentStart := beforeHeaderComment(lines, end, key.FootComment)
			out.WriteString(strings.Join(lines[commentStart:end], ""))
		}
		cursor = end
	}
	out.WriteString(strings.Join(lines[cursor:footer], ""))
	for _, k := range sortedKeys(changes) {
		if removes[k] {
			continue
		}
		var v yaml.Node
		if err := encodeValue(&v, changes[k]); err != nil {
			return nil, err
		}
		mapping := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, &v}}
		encoded, err := encodeNode(mapping)
		if err != nil {
			return nil, err
		}
		out.WriteString(strings.ReplaceAll(string(encoded), "\n", f.newline))
	}
	out.WriteString(strings.Join(lines[footer:], ""))
	return []byte(out.String()), nil
}

// Whitespace-only trailing lines can be block-scalar content, not separators.
// Follow the final structural child; an alias token has no block content here.
func endsInBlockScalar(node *yaml.Node) bool {
	for node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode {
		if len(node.Content) == 0 {
			return false
		}
		node = node.Content[len(node.Content)-1]
	}
	return node.Kind == yaml.ScalarNode && node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0
}

// Comments owned by the next key or the document remain outside this entry's
// replacement. Its own foot comments are emitted once with the retained nodes.
func beforeHeaderComment(lines []string, end int, comment string) int {
	comment = strings.TrimRight(comment, "\r\n")
	if comment == "" {
		return end
	}
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return max(0, end-strings.Count(comment, "\n")-1)
}
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sameValue(a, b any) bool {
	x, ex := json.Marshal(a)
	y, ey := json.Marshal(b)
	if ex == nil && ey == nil {
		return bytes.Equal(x, y)
	}
	return reflect.DeepEqual(a, b)
}

// encodeValue keeps JSON numbers exact instead of treating their string-backed
// representation as a YAML string.
func encodeValue(n *yaml.Node, value any) error {
	switch v := value.(type) {
	case json.Number:
		tag := "!!int"
		if strings.ContainsAny(v.String(), ".eE") {
			tag = "!!float"
		}
		*n = yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: v.String()}
		return nil
	case map[string]any:
		*n = yaml.Node{Kind: yaml.MappingNode}
		for _, k := range sortedKeys(v) {
			var child yaml.Node
			if err := encodeValue(&child, v[k]); err != nil {
				return err
			}
			n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, &child)
		}
		return nil
	case []any:
		*n = yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range v {
			var child yaml.Node
			if err := encodeValue(&child, item); err != nil {
				return err
			}
			n.Content = append(n.Content, &child)
		}
		return nil
	default:
		return n.Encode(value)
	}
}

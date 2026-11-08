package okf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

func fault(code, message string) error { return &Error{Code: code, Message: message} }

type frontmatter struct {
	header, body []byte
	node         *yaml.Node
	document     *yaml.Node
	newline      string
}

func splitFront(raw []byte) (frontmatter, error) {
	f := frontmatter{body: raw, newline: "\n"}
	if len(raw) > MaxDocumentBytes {
		return f, fault("LIMIT_EXCEEDED", "note exceeds 1 MiB")
	}
	if !utf8.Valid(raw) {
		return f, fault("INVALID_UTF8", "note is not valid UTF-8")
	}
	if bytes.HasPrefix(raw, []byte("---\r\n")) {
		f.newline = "\r\n"
	}
	opening := []byte("---" + f.newline)
	if !bytes.HasPrefix(raw, opening) {
		return f, fault("MISSING_FRONTMATTER", "a YAML frontmatter block is required")
	}
	offset := len(opening)
	for offset <= len(raw) {
		end := bytes.IndexByte(raw[offset:], '\n')
		next := len(raw)
		if end >= 0 {
			next = offset + end + 1
		}
		line := bytes.TrimSuffix(bytes.TrimSuffix(raw[offset:next], []byte("\n")), []byte("\r"))
		if bytes.Equal(line, []byte("---")) {
			f.header = raw[len(opening):offset]
			f.body = raw[next:]
			var doc yaml.Node
			d := yaml.NewDecoder(bytes.NewReader(f.header))
			if err := d.Decode(&doc); err != nil {
				return f, fault("INVALID_YAML", "frontmatter must be a YAML mapping")
			}
			var extra yaml.Node
			if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
				return f, fault("INVALID_YAML", "only one YAML document is permitted")
			}
			if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
				return f, fault("INVALID_YAML", "frontmatter must be a mapping")
			}
			f.node = doc.Content[0]
			f.document = &doc
			count, aliases := 0, 0
			if err := checkNode(f.node, 0, &count, &aliases, map[*yaml.Node]bool{}); err != nil {
				return f, err
			}
			return f, nil
		}
		if next == len(raw) {
			break
		}
		offset = next
	}
	return f, fault("INVALID_YAML", "frontmatter closing delimiter is missing")
}

func checkNode(n *yaml.Node, depth int, count, aliases *int, active map[*yaml.Node]bool) error {
	*count++
	if depth > 32 || *count > 50000 {
		return fault("YAML_LIMIT", "YAML nesting or node limit exceeded")
	}
	if n.Kind == yaml.AliasNode {
		*aliases++
		if *aliases > 16 || active[n.Alias] {
			return fault("YAML_LIMIT", "YAML alias limit or cycle")
		}
		active[n.Alias] = true
		err := checkNode(n.Alias, depth+1, count, aliases, active)
		delete(active, n.Alias)
		return err
	}
	if n.Kind == yaml.MappingNode {
		keys := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" {
				return fault("METADATA_KEY_POLICY", "metadata mapping keys must be strings for the JSON API")
			}
			if keys[k.Value] {
				return fault("DUPLICATE_KEY", "duplicate YAML mapping key")
			}
			keys[k.Value] = true
		}
	}
	for _, child := range n.Content {
		if err := checkNode(child, depth+1, count, aliases, active); err != nil {
			return err
		}
	}
	return nil
}

func nodeValue(n *yaml.Node) any {
	if n.Kind == yaml.AliasNode {
		return nodeValue(n.Alias)
	}
	if n.Kind == yaml.MappingNode {
		m := map[string]any{}
		for i := 0; i < len(n.Content); i += 2 {
			m[n.Content[i].Value] = nodeValue(n.Content[i+1])
		}
		return m
	}
	if n.Kind == yaml.SequenceNode {
		a := make([]any, 0, len(n.Content))
		for _, v := range n.Content {
			a = append(a, nodeValue(v))
		}
		return a
	}
	if n.Tag == "!!int" {
		value := strings.ReplaceAll(n.Value, "_", "")
		if integer, ok := new(big.Int).SetString(value, 0); ok {
			return json.Number(integer.String())
		}
	}
	if n.Tag == "!!timestamp" {
		return n.Value
	}
	if n.Tag == "!!float" {
		if number, ok := decimalNumber(n.Value); ok {
			return number
		}
	}
	var value any
	if err := n.Decode(&value); err != nil {
		return n.Value
	}
	return value
}

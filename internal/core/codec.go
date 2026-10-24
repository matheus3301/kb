// Package core implements confined, local knowledge-base reads and note codecs.
package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"path"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
	"go.yaml.in/yaml/v3"
)

type codec struct{}

type frontmatter struct {
	header, body []byte
	node         *yaml.Node
	newline      string
}

func fault(code, message string) error { return &api.DomainError{Code: code, Message: message} }
func diagnostic(p, code, message, layer string) api.Diagnostic {
	return api.Diagnostic{Path: p, Code: code, Message: message, Layer: layer}
}
func reserved(p string) bool { return path.Base(p) == "index.md" || path.Base(p) == "log.md" }

func splitFront(raw []byte) (frontmatter, error) {
	f := frontmatter{body: raw, newline: "\n"}
	if len(raw) > api.MaxNoteBytes {
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
	var value any
	if err := n.Decode(&value); err != nil {
		return n.Value
	}
	return value
}

func (codec) Parse(p string, raw []byte) (api.Note, []api.Diagnostic) {
	n := api.Note{ID: strings.TrimSuffix(p, ".md"), Path: p, Title: strings.TrimSuffix(path.Base(p), ".md"), Revision: api.Revision(raw), Raw: raw, Trust: "unverified", Status: "", Metadata: map[string]any{}}
	if reserved(p) {
		n.Body = string(raw)
		return n, (codec{}).ValidateReserved(p, raw)
	}
	f, err := splitFront(raw)
	if err != nil {
		n.Body = string(raw)
		layer := "conformance_errors"
		if strings.Contains(err.Error(), "METADATA_KEY_POLICY") {
			layer = "policy_findings"
		}
		return n, []api.Diagnostic{diagnostic(p, "INVALID_NOTE", err.Error(), layer)}
	}
	n.Body = string(f.body)
	n.Metadata = nodeValue(f.node).(map[string]any)
	var ds []api.Diagnostic
	add := func(code, msg, layer string) {
		if len(ds) < 64 {
			ds = append(ds, diagnostic(p, code, msg, layer))
		}
	}
	m := n.Metadata
	n.Type, _ = m["type"].(string)
	if strings.TrimSpace(n.Type) == "" {
		add("MISSING_TYPE", "type must be a nonempty string", "conformance_errors")
	}
	for _, key := range []string{"title", "description", "resource", "status", "stale_after", "runtime", "computation"} {
		if v, ok := m[key]; ok {
			if _, ok := v.(string); !ok {
				add("INVALID_METADATA", key+" must be a string", "policy_findings")
			}
		}
	}
	if v, ok := m["title"].(string); ok {
		n.Title = v
	}
	n.Description, _ = m["description"].(string)
	if v, ok := m["status"].(string); ok {
		n.Status = v
		if v != "draft" && v != "stable" && v != "deprecated" {
			add("INVALID_STATUS", "status must be draft, stable, or deprecated", "policy_findings")
		}
	}
	if tags, ok := m["tags"]; ok {
		a, ok := tags.([]any)
		if !ok {
			add("INVALID_TAGS", "tags must be a list of strings", "policy_findings")
		} else {
			for _, v := range a {
				s, ok := v.(string)
				if !ok {
					add("INVALID_TAGS", "tags must be strings", "policy_findings")
				} else {
					n.Tags = append(n.Tags, s)
				}
			}
		}
	}
	validateTime := func(key string, v any) {
		s, ok := v.(string)
		if !ok {
			add("INVALID_TIME", key+" must be a timezone-bearing ISO 8601 instant", "policy_findings")
			return
		}
		if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
			add("INVALID_TIME", key+" must be a timezone-bearing ISO 8601 instant", "policy_findings")
		}
	}
	validateActor := func(key string, v any, requireAt bool) {
		event, ok := v.(map[string]any)
		if !ok {
			add("INVALID_ACTOR", key+" must be a mapping", "policy_findings")
			return
		}
		by, ok := event["by"].(string)
		if !ok || strings.TrimSpace(by) == "" {
			add("INVALID_ACTOR", key+".by must be nonempty", "policy_findings")
		}
		if at, ok := event["at"]; ok {
			validateTime(key+".at", at)
		} else if requireAt {
			add("INVALID_TIME", key+".at is required", "policy_findings")
		}
	}
	if v, ok := m["generated"]; ok {
		validateActor("generated", v, false)
	}
	if v, ok := m["verified"]; ok {
		events, ok := v.([]any)
		if !ok {
			events = []any{v}
		}
		for _, event := range events {
			validateActor("verified", event, true)
			if obj, ok := event.(map[string]any); ok {
				if by, ok := obj["by"].(string); ok {
					if strings.HasPrefix(by, "human:") {
						n.Trust = "human-reviewed"
					} else if n.Trust != "human-reviewed" {
						n.Trust = "machine-confirmed"
					}
				}
			}
		}
	}
	if v, ok := m["stale_after"]; ok {
		validateTime("stale_after", v)
	}
	if v, ok := m["sources"]; ok {
		sources, ok := v.([]any)
		if !ok {
			add("INVALID_SOURCES", "sources must be a list of mappings with resource", "policy_findings")
		} else {
			for _, source := range sources {
				obj, ok := source.(map[string]any)
				if !ok {
					add("INVALID_SOURCE", "source must be a mapping", "policy_findings")
					continue
				}
				resource, ok := obj["resource"].(string)
				if !ok || strings.TrimSpace(resource) == "" {
					add("INVALID_SOURCE", "source.resource is required", "policy_findings")
				}
				if v, ok := obj["last_modified"]; ok {
					validateTime("source.last_modified", v)
				}
			}
		}
	}
	if n.Type == "Attested Computation" {
		if v, ok := m["runtime"].(string); !ok || strings.TrimSpace(v) == "" {
			add("INVALID_COMPUTATION", "Attested Computation requires runtime", "policy_findings")
		}
		for _, key := range []string{"executor", "attester"} {
			if v, ok := m[key]; ok {
				obj, ok := v.(map[string]any)
				if !ok {
					add("INVALID_COMPUTATION", key+" must be a mapping", "policy_findings")
				} else if s, ok := obj["resource"].(string); !ok || s == "" {
					add("INVALID_COMPUTATION", key+".resource is required", "policy_findings")
				}
			}
		}
	}
	return n, ds
}

func (codec) ValidateReserved(p string, raw []byte) []api.Diagnostic {
	var ds []api.Diagnostic
	add := func(msg string) { ds = append(ds, diagnostic(p, "INVALID_RESERVED", msg, "conformance_errors")) }
	if len(raw) > api.MaxNoteBytes || !utf8.Valid(raw) {
		add("reserved file exceeds the size limit or is not UTF-8")
		return ds
	}
	body := string(raw)
	if strings.HasPrefix(body, "---\n") || strings.HasPrefix(body, "---\r\n") {
		f, err := splitFront(raw)
		if err != nil {
			add(err.Error())
			return ds
		}
		if p != "index.md" {
			add("only the root index permits frontmatter")
		} else {
			m := nodeValue(f.node).(map[string]any)
			for k, v := range m {
				if k != "okf_version" {
					add("root index metadata permits only okf_version")
				} else if s, ok := v.(string); !ok || s == "" {
					add("okf_version must be a string")
				}
			}
		}
		body = string(f.body)
	}
	if path.Base(p) == "log.md" {
		previous := "9999-99-99"
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "## ") {
				date := strings.TrimSpace(strings.TrimPrefix(line, "## "))
				if _, err := time.Parse("2006-01-02", date); err != nil {
					add("log date headings must use YYYY-MM-DD")
				} else if date > previous {
					add("log dates must be newest first")
				}
				previous = date
			}
		}
	}
	if path.Base(p) == "index.md" {
		if _, _, _, err := indexRegion(body); err != nil {
			add("index managed markers must form one standalone ordered pair outside code")
		}
	}

	return ds
}

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
				var node yaml.Node
				if err := encodeValue(&node, x); err != nil {
					return nil, err
				}
				node.HeadComment = v.HeadComment
				node.LineComment = v.LineComment
				node.FootComment = v.FootComment
				v = &node
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
		encoded, err := encodeNode(f.node)
		if err != nil {
			return nil, err
		}
		return []byte(strings.ReplaceAll(string(encoded), "\n", f.newline)), nil
	}
	lines := strings.SplitAfter(string(f.header), "\n")
	var out strings.Builder
	cursor := 0
	for i := 0; i < len(f.node.Content); i += 2 {
		key, value := f.node.Content[i], f.node.Content[i+1]
		start := key.Line - 1
		end := len(lines)
		if i+2 < len(f.node.Content) {
			end = f.node.Content[i+2].Line - 1
		}
		for end > start+1 {
			t := strings.TrimSpace(lines[end-1])
			if t == "" || strings.HasPrefix(t, "#") {
				end--
			} else {
				break
			}
		}
		out.WriteString(strings.Join(lines[cursor:start], ""))
		if x, ok := changes[key.Value]; ok && !removes[key.Value] {
			var v yaml.Node
			if err := encodeValue(&v, x); err != nil {
				return nil, err
			}
			v.LineComment = value.LineComment
			mapping := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: key.Value}, &v}}
			encoded, err := encodeNode(mapping)
			if err != nil {
				return nil, err
			}
			out.WriteString(strings.ReplaceAll(string(encoded), "\n", f.newline))
			delete(changes, key.Value)
		} else if !removes[key.Value] {
			out.WriteString(strings.Join(lines[start:end], ""))
		}
		cursor = end
	}
	out.WriteString(strings.Join(lines[cursor:], ""))
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
	return []byte(out.String()), nil
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

func (c codec) Patch(p string, raw []byte, op api.Operation, actor string, at time.Time) ([]byte, error) {
	if err := validPath(p); err != nil {
		return nil, err
	}
	if op.Body != nil && len(op.BodyEdits) > 0 {
		return nil, fault("INVALID_INPUT", "body and body_edits are mutually exclusive")
	}
	if len(op.BodyEdits) > api.MaxOperations {
		return nil, fault("LIMIT_EXCEEDED", "too many body edits")
	}
	if op.Document != nil {
		out := []byte(*op.Document)
		_, ds := c.Parse(p, out)
		if hasErrors(ds) {
			return nil, fault("INVALID_NOTE", "replacement document failed validation")
		}
		before, _ := c.Parse(p, raw)
		after, _ := c.Parse(p, out)
		if !sameValue(before.Metadata["verified"], after.Metadata["verified"]) && after.Metadata["verified"] != nil {
			return nil, fault("POLICY_VIOLATION", "document replacement cannot add verification")
		}
		if bytes.Equal(raw, out) {
			return raw, nil
		}
		if reserved(p) {
			return out, nil
		}
		return c.Patch(p, out, api.Operation{Kind: "patch", RemoveMetadataKeys: []string{"verified"}, MetadataPatch: map[string]any{"generated": map[string]any{"by": actor, "at": at.UTC().Format(time.RFC3339Nano)}}}, actor, at)
	}
	if reserved(p) {
		if len(op.MetadataPatch) > 0 || len(op.Metadata) > 0 {
			return nil, fault("INVALID_INPUT", "reserved files do not use concept metadata")
		}
		out, err := editBody(string(raw), op)
		if err != nil {
			return nil, err
		}
		if ds := c.ValidateReserved(p, []byte(out)); len(ds) > 0 {
			return nil, fault("INVALID_NOTE", "invalid reserved file")
		}
		return []byte(out), nil
	}
	creating := op.Kind == "create"
	if creating && len(raw) > 0 {
		return nil, fault("ALREADY_EXISTS", "create requires an empty document")
	}
	var f frontmatter
	var err error
	if creating {
		if strings.TrimSpace(op.Type) == "" {
			return nil, fault("INVALID_INPUT", "create requires type")
		}
		f = frontmatter{newline: "\n", node: &yaml.Node{Kind: yaml.MappingNode}}
	} else {
		f, err = splitFront(raw)
		if err != nil {
			return nil, err
		}
	}
	current := map[string]any{}
	if !creating {
		current = nodeValue(f.node).(map[string]any)
	}
	changes := map[string]any{}
	removes := map[string]bool{}
	for k, v := range op.MetadataPatch {
		if !sameValue(current[k], v) {
			changes[k] = v
		}
	}
	if creating {
		for k, v := range op.Metadata {
			changes[k] = v
		}
		changes["type"] = op.Type
		if op.Title != "" {
			changes["title"] = op.Title
		}
	}
	for _, key := range op.RemoveMetadataKeys {
		if _, ok := current[key]; ok {
			removes[key] = true
		}
		delete(changes, key)
	}
	if _, ok := changes["verified"]; ok {
		return nil, fault("POLICY_VIOLATION", "agent-origin edits cannot add verification")
	}
	body, err := editBody(string(f.body), op)
	if err != nil {
		return nil, err
	}
	meaningful := creating || body != string(f.body) || len(changes) > 0 || len(removes) > 0
	if !meaningful {
		return raw, nil
	}
	if strings.TrimSpace(actor) == "" || strings.HasPrefix(actor, "human:") {
		return nil, fault("POLICY_VIOLATION", "an agent-origin edit requires a nonhuman actor")
	}
	if _, ok := current["verified"]; ok {
		removes["verified"] = true
	}
	generated := map[string]any{}
	if old, ok := current["generated"].(map[string]any); ok {
		for k, v := range old {
			generated[k] = v
		}
	}
	delete(removes, "generated")
	generated["by"] = actor
	generated["at"] = at.UTC().Format(time.RFC3339Nano)
	changes["generated"] = generated
	header, err := patchHeader(f, changes, removes)
	if err != nil {
		return nil, err
	}
	out := append([]byte("---"+f.newline), header...)
	out = append(out, []byte("---"+f.newline)...)
	out = append(out, body...)
	_, ds := c.Parse(p, out)
	if hasErrors(ds) {
		return nil, &api.DomainError{Code: "INVALID_NOTE", Message: "patched note failed validation", Details: ds}
	}
	return out, nil
}
func editBody(body string, op api.Operation) (string, error) {
	if op.Body != nil {
		return *op.Body, nil
	}
	for _, edit := range op.BodyEdits {
		if edit.OldText == "" || strings.Count(body, edit.OldText) != 1 {
			return "", fault("EDIT_CONFLICT", "old_text must occur exactly once")
		}
		body = strings.Replace(body, edit.OldText, edit.NewText, 1)
	}
	return body, nil
}
func hasErrors(ds []api.Diagnostic) bool {
	for _, d := range ds {
		if d.Layer == "conformance_errors" || d.Layer == "policy_findings" {
			return true
		}
	}
	return false
}

var _ api.Codec = codec{}

// encodeValue keeps JSON numbers exact instead of treating their string-backed
// representation as a YAML string.
func encodeValue(n *yaml.Node, value any) error {
	switch v := value.(type) {
	case json.Number:
		if !json.Valid([]byte(v.String())) {
			return fault("INVALID_INPUT", "invalid JSON number")
		}
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

// Package core implements confined, local knowledge-base reads and note codecs.
package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"path"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/pkg/okf"
)

type codec struct{}

func fault(code, message string) error { return &api.DomainError{Code: code, Message: message} }
func diagnostic(p, code, message, layer string) api.Diagnostic {
	return enrichDiagnostic(api.Diagnostic{Path: p, Code: code, Message: message, Layer: layer})
}
func reserved(p string) bool { return path.Base(p) == "index.md" || path.Base(p) == "log.md" }

func (codec) Parse(p string, raw []byte) (api.Note, []api.Diagnostic) {
	n := api.Note{ID: strings.TrimSuffix(p, ".md"), Path: p, Title: strings.TrimSuffix(path.Base(p), ".md"), Revision: api.Revision(raw), Raw: raw, Trust: "unverified", Status: "", Metadata: map[string]any{}}
	if reserved(p) {
		n.Body = string(raw)
		return n, (codec{}).ValidateReserved(p, raw)
	}
	document, err := okf.Parse(raw)
	if err != nil {
		n.Body = string(raw)
		layer := "conformance_errors"
		if strings.Contains(err.Error(), "METADATA_KEY_POLICY") {
			layer = "policy_findings"
		}
		d := diagnostic(p, "INVALID_NOTE", err.Error(), layer)
		if d.Rule == "MISSING_FRONTMATTER" {
			d.Line = 1
		}
		return n, []api.Diagnostic{d}
	}
	n.Body = document.Body()
	n.Metadata = document.Metadata()
	m := n.Metadata
	n.Type, _ = m["type"].(string)
	if title, ok := m["title"].(string); ok {
		n.Title = title
	}
	n.Description, _ = m["description"].(string)
	n.Status, _ = m["status"].(string)
	if tags, ok := m["tags"].([]any); ok {
		for _, value := range tags {
			if tag, ok := value.(string); ok {
				n.Tags = append(n.Tags, tag)
			}
		}
	}
	if verified, exists := m["verified"]; exists {
		events, ok := verified.([]any)
		if !ok {
			events = []any{verified}
		}
		for _, event := range events {
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
	var ds []api.Diagnostic
	for _, finding := range document.Validate() {
		layer := "policy_findings"
		if finding.Kind == okf.Requirement {
			layer = "conformance_errors"
		}
		d := diagnostic(p, finding.Code, finding.Message, layer)
		d.Field, d.Line = finding.Field, finding.Line
		ds = append(ds, d)
	}
	return n, ds
}

func (codec) ValidateReserved(p string, raw []byte) []api.Diagnostic {
	role := okf.Index
	if p == "index.md" {
		role = okf.RootIndex
	} else if path.Base(p) == "log.md" {
		role = okf.Log
	}
	var ds []api.Diagnostic
	add := func(msg string) { ds = append(ds, diagnostic(p, "INVALID_RESERVED", msg, "conformance_errors")) }
	for _, finding := range okf.ValidateReserved(role, raw) {
		add(finding.Message)
	}
	if len(raw) > api.MaxNoteBytes || !utf8.Valid(raw) {
		return ds
	}
	body := string(raw)
	if strings.HasPrefix(body, "---\n") || strings.HasPrefix(body, "---\r\n") {
		document, err := okf.Parse(raw)
		if err != nil {
			return ds
		}
		body = document.Body()
	}
	if path.Base(p) == "index.md" {
		if _, _, _, err := indexRegion(body); err != nil {
			add("index managed markers must form one standalone ordered pair outside code")
		}
	}

	return ds
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
	var document *okf.Document
	current := map[string]any{}
	originalBody := ""
	if creating {
		if strings.TrimSpace(op.Type) == "" {
			return nil, fault("INVALID_INPUT", "create requires type")
		}
	} else {
		var err error
		document, err = okf.Parse(raw)
		if err != nil {
			return nil, formatError(err)
		}
		current = document.Metadata()
		originalBody = document.Body()
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
	body, err := editBody(originalBody, op)
	if err != nil {
		return nil, err
	}
	meaningful := creating || body != originalBody || len(changes) > 0 || len(removes) > 0
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
	var out []byte
	if creating {
		out, err = okf.Marshal(changes, body)
	} else {
		keys := make([]string, 0, len(removes))
		for key := range removes {
			keys = append(keys, key)
		}
		out, err = document.Edit(okf.Edit{Set: changes, Remove: keys, Body: &body})
	}
	if err != nil {
		// Public editing checks assembled syntax; preserve the codec's legacy
		// final-validation envelope rather than leaking a public error type.
		if out != nil {
			_, ds := c.Parse(p, out)
			return nil, &api.DomainError{Code: "INVALID_NOTE", Message: "patched note failed validation", Details: ds}
		}
		if len(body) > api.MaxNoteBytes {
			d := diagnostic(p, "INVALID_NOTE", "LIMIT_EXCEEDED: note exceeds 1 MiB", "conformance_errors")
			return nil, &api.DomainError{Code: "INVALID_NOTE", Message: "patched note failed validation", Details: []api.Diagnostic{d}}
		}
		var failure *okf.Error
		if errors.As(err, &failure) && failure.Code == "YAML_LIMIT" {
			// Caller-tree checks reject excess depth/nodes before rendering.
			// These structural changes previously failed final document validation,
			// unlike oversized scalar span patches, whose direct error stays intact.
			d := diagnostic(p, "INVALID_NOTE", err.Error(), "conformance_errors")
			return nil, &api.DomainError{Code: "INVALID_NOTE", Message: "patched note failed validation", Details: []api.Diagnostic{d}}
		}
		return nil, formatError(err)
	}
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

// Keep public format errors independent of the application's transport DTOs.
func formatError(err error) error {
	var failure *okf.Error
	if errors.As(err, &failure) {
		return fault(failure.Code, failure.Message)
	}
	return err
}

func errorsJoin(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

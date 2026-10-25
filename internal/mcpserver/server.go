package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strings"

	"github.com/matheus3301/kb/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New uses the official SDK for negotiation, framing, cancellation and shutdown.
// All requests retain the root captured by the supplied adapter.
func New(a *Adapter, version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "kb", Version: version}, &mcp.ServerOptions{Instructions: instructions(a.ReadOnly), PageSize: api.DefaultHits, Capabilities: &mcp.ServerCapabilities{}})
	inputs := map[string]any{"status": struct{}{}, "list": api.SearchRequest{}, "search": api.SearchRequest{}, "read": api.ReadRequest{}, "context": api.ContextRequest{}, "create": SingleInput{}, "patch": SingleInput{}, "move": SingleInput{}, "apply": api.Plan{}, "validate": ValidateInput{}, "backup": api.BackupInput{}, "restore": api.RestoreInput{}}
	for _, name := range []string{"status", "list", "search", "read", "context", "create", "patch", "move", "apply", "validate", "backup", "restore"} {
		if a.ReadOnly && IsWrite(name) {
			continue
		}
		closed := false
		s.AddTool(&mcp.Tool{Name: "kb_" + name, Description: description(name), InputSchema: inputSchema(name, inputs[name]), OutputSchema: outputSchema(), Annotations: &mcp.ToolAnnotations{ReadOnlyHint: !IsWrite(name), IdempotentHint: name != "backup" && name != "restore", OpenWorldHint: &closed}}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return toolResult(a.Call(ctx, name, req.Params.Arguments)), nil
		})
	}
	s.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "kb://local/{+path}", Name: "Knowledge note", MIMEType: "text/markdown"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		p, err := ResourcePath(req.Params.URI)
		if err != nil {
			return nil, err
		}
		e := a.Call(ctx, "read", mustJSON(api.ReadRequest{Path: p, Raw: true}))
		if !e.OK {
			return nil, e.Error
		}
		r := e.Data.(api.ReadResult)
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: ResourceURI(p), MIMEType: "text/markdown", Text: r.Markdown, Meta: mcp.Meta{"revision": r.Note.Revision}}}}, nil
	})
	// Resource inventory is requested afresh so externally added or removed
	// notes do not leave startup-only resource registrations behind.
	s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "tools/call" && a.ReadOnly {
				if p, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok && IsWrite(strings.TrimPrefix(p.Name, "kb_")) {
					return toolResult(Failure(a.Reader.Root(), &api.DomainError{Code: "READ_ONLY", Message: "writes are disabled"})), nil
				}
			}
			if method == "resources/list" {
				cursor := ""
				if p, ok := req.GetParams().(*mcp.ListResourcesParams); ok && p != nil {
					cursor = p.Cursor
				}
				r, err := a.Reader.List(ctx, api.SearchRequest{Limit: api.DefaultHits, Cursor: cursor, Filters: api.Filters{IncludeArchives: true, IncludeDeprecated: true}})
				if err != nil {
					return nil, err
				}
				resources := make([]*mcp.Resource, 0, len(r.Hits))
				for _, hit := range r.Hits {
					resources = append(resources, &mcp.Resource{URI: ResourceURI(hit.Path), Name: hit.Path, Title: hit.Title, MIMEType: "text/markdown"})
				}
				return &mcp.ListResourcesResult{Resources: resources, NextCursor: r.Cursor}, nil
			}
			return next(ctx, method, req)
		}
	})
	return s
}

func toolResult(e Envelope) *mcp.CallToolResult {
	text := "kb.v1 result"
	if !e.OK {
		text = e.Error.Code + ": " + e.Error.Message
	}
	return &mcp.CallToolResult{StructuredContent: e, Content: []mcp.Content{&mcp.TextContent{Text: text}}, IsError: !e.OK}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ResourceURI encodes each path segment while retaining hierarchy separators.
func ResourceURI(p string) string {
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return "kb://local/" + strings.Join(parts, "/")
}

// ResourcePath accepts only canonical local resource paths. The core then
// enforces scoped inventory membership and filesystem confinement.
func ResourcePath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "kb" || u.Host != "local" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", Invalid("invalid knowledge resource URI")
	}
	p := strings.TrimPrefix(u.Path, "/")
	if p == "" || strings.ContainsAny(p, "\\\x00") || strings.HasPrefix(p, "/") || path.Clean(p) != p {
		return "", Invalid("invalid knowledge resource path")
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." || part == "." || part == "" {
			return "", Invalid("invalid knowledge resource path")
		}
	}
	if !strings.HasSuffix(strings.ToLower(p), ".md") {
		return "", Invalid("resource must be a Markdown note")
	}
	return p, nil
}

// Schema objects are closed at every DTO boundary. Metadata remains extensible
// because unknown knowledge metadata is part of the public domain contract.
func schema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return schema(t.Elem())
	}
	switch t.Kind() {
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous {
				embedded := schema(f.Type)
				for k, v := range embedded["properties"].(map[string]any) {
					props[k] = v
				}
				required = append(required, embedded["required"].([]string)...)
				continue
			}
			tag := f.Tag.Get("json")
			name := strings.Split(tag, ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			props[name] = schema(f.Type)
			if !strings.Contains(tag, "omitempty") {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": true}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": schema(t.Elem()), "maxItems": api.MaxOperations}
	case reflect.String:
		return map[string]any{"type": "string", "maxLength": api.MaxNoteBytes}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int:
		return map[string]any{"type": "integer", "minimum": 0}
	default:
		panic(fmt.Sprintf("unsupported input type %s", t))
	}
}

func inputSchema(action string, input any) map[string]any {
	s := schema(reflect.TypeOf(input))
	describeFields(s)
	p := s["properties"].(map[string]any)
	for name, max := range map[string]int{"query": api.MaxQueryRunes, "limit": api.MaxHits, "max_tokens": api.MaxContextTokens, "max_bytes": api.MaxReadBytes} {
		if prop, ok := p[name].(map[string]any); ok {
			if name == "query" {
				prop["maxLength"] = max
			} else {
				prop["maximum"] = max
			}
		}
	}
	if action == "create" || action == "patch" || action == "move" {
		required := []string{"path", "summary", "actor", "idempotency_key"}
		if action == "create" {
			required = append(required, "type")
		}
		if action != "create" {
			required = append(required, "expected_revision")
		}
		if action == "move" {
			required = append(required, "to")
		}
		s["required"] = required
		p["kind"].(map[string]any)["enum"] = []string{action}
	}
	if action == "list" {
		s["required"] = []string{}
	}
	return s
}

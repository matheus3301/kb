package core

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
)

func TestCodecMinimalUnknownAndKnownMetadata(t *testing.T) {
	c := codec{}
	n, ds := c.Parse("notes/Example.md", []byte("---\ntype: Unregistered Type\ncustom: {nested: [yes, 42]}\n---\nHello\n"))
	if len(ds) != 0 || n.Type != "Unregistered Type" || n.ID != "notes/Example" || n.Title != "Example" || n.Status != "" {
		t.Fatalf("note=%+v diagnostics=%+v", n, ds)
	}
	for _, raw := range []string{"no YAML", "---\ntitle: Missing type\n---\n", "---\ntype: 42\n---\n", "---\ntype: A\ntype: B\n---\n", "---\ntype: [invalid\n---\n", "---\n[]\n---\n", "---\ntype: X\n", "---\ntype: X\n---\n\xff"} {
		if _, ds := c.Parse("bad.md", []byte(raw)); !hasErrors(ds) {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, metadata := range []string{"tags: wrong", "status: wrong", "sources: [{}]", "sources: nope", "verified: {by: process:test, at: 2026-10-01}", "stale_after: 2026-10-01", "generated: {}", "title: [bad]", "tags: [12]", "verified: [nope]", "sources: [not-a-mapping]", "type: Attested Computation"} {
		_, ds := c.Parse("bad.md", []byte("---\ntype: Example\n"+metadata+"\n---\n"))
		if !hasErrors(ds) {
			t.Errorf("accepted %s", metadata)
		}
	}
	_, ds = c.Parse("extension.md", []byte("---\ntype: Example\ncustom: {42: value}\n---\n"))
	if len(ds) != 1 || ds[0].Layer != "policy_findings" {
		t.Fatalf("nonstring key policy: %+v", ds)
	}
}
func TestCodecVerificationSingletonListAndTimezones(t *testing.T) {
	for _, verified := range []string{"{by: human:reviewer, at: 2026-10-01T02:00:00+02:00}", "[{by: process:test, at: 2026-10-01T00:00:00Z}, {by: human:reviewer, at: 2026-10-01T01:00:00Z}]"} {
		n, ds := (codec{}).Parse("trust.md", []byte("---\ntype: Reference\nverified: "+verified+"\nstale_after: 2026-11-01T00:00:00Z\n---\n"))
		if len(ds) > 0 || n.Trust != "human-reviewed" {
			t.Fatalf("%+v %+v", n, ds)
		}
	}
	n, ds := (codec{}).Parse("machine.md", []byte("---\ntype: Reference\nverified: {by: process:test, at: 2026-10-01T00:00:00Z}\n---\n"))
	if len(ds) > 0 || n.Trust != "machine-confirmed" {
		t.Fatal(ds, n.Trust)
	}
}
func TestCodecPatchPreservesSpansNoOpAndTrust(t *testing.T) {
	c := codec{}
	at := time.Date(2026, 10, 24, 9, 0, 0, 0, time.FixedZone("offset", 3600))
	raw := []byte("---\r\n# file comment\r\ntype: 'Example' # keep\r\ncustom: {value: 'literal'} # untouched\r\n# description note\r\ndescription: Original # inline\r\nverified: {by: human:reviewer, at: 2026-10-01T00:00:00Z}\r\n---\r\n# Body\r\nOriginal body\r\n")
	noOp, err := c.Patch("sample.md", raw, api.Operation{Kind: "patch", MetadataPatch: map[string]any{"description": "Original"}}, "process:test", at)
	if err != nil || !bytes.Equal(noOp, raw) {
		t.Fatal(err, string(noOp))
	}
	updated, err := c.Patch("sample.md", raw, api.Operation{Kind: "patch", BodyEdits: []api.TextEdit{{OldText: "Original body", NewText: "Corrected body"}}}, "process:real-actor", at)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# file comment\r\n", "type: 'Example' # keep\r\n", "custom: {value: 'literal'} # untouched\r\n", "# description note\r\ndescription: Original # inline\r\n", "Corrected body\r\n", "process:real-actor", "2026-10-24T08:00:00Z"} {
		if !strings.Contains(string(updated), want) {
			t.Errorf("missing %q in %s", want, updated)
		}
	}
	n, ds := c.Parse("sample.md", updated)
	if hasErrors(ds) || n.Trust != "unverified" {
		t.Fatal(ds, n.Trust)
	}
	metadata, err := c.Patch("sample.md", updated, api.Operation{Kind: "patch", MetadataPatch: map[string]any{"description": "Changed"}}, "process:test", at)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metadata), "custom: {value: 'literal'} # untouched\r\n") || !strings.Contains(string(metadata), "description: Changed # inline") {
		t.Fatal(string(metadata))
	}
	for _, op := range []api.Operation{{MetadataPatch: map[string]any{"verified": map[string]any{"by": "human:fake"}}}, {BodyEdits: []api.TextEdit{{OldText: "missing", NewText: "x"}}}, {BodyEdits: []api.TextEdit{{OldText: "", NewText: "x"}}}} {
		if _, err := c.Patch("sample.md", raw, op, "process:test", at); err == nil {
			t.Fatal("invalid edit accepted")
		}
	}
	if _, err := c.Patch("sample.md", raw, api.Operation{Body: ptr("new")}, "human:spoof", at); err == nil {
		t.Fatal("human actor accepted")
	}
}
func TestCodecCreateNumericMetadataAndFlowStyle(t *testing.T) {
	c := codec{}
	at := time.Now()
	op := api.Operation{Kind: "create", Type: "Number Test", Body: ptr("Synthetic numeric example.\n"), Metadata: map[string]any{"ttl": json.Number("600"), "large": json.Number("900719925474099312345"), "nested": map[string]any{"values": []any{json.Number("1.25"), true}}}}
	raw, err := c.Patch("number.md", nil, op, "process:test", at)
	if err != nil {
		t.Fatal(err)
	}
	n, ds := c.Parse("number.md", raw)
	if hasErrors(ds) || n.Metadata["ttl"] != json.Number("600") || n.Metadata["large"] != json.Number("900719925474099312345") {
		t.Fatalf("%s %+v %+v", raw, n.Metadata, ds)
	}
	flow := []byte("---\n{type: 'Example', custom: [one, two], title: Original}\n---\nBody\n")
	out, err := c.Patch("flow.md", flow, api.Operation{MetadataPatch: map[string]any{"title": "Changed"}}, "process:test", at)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "type: 'Example'") || !strings.Contains(string(out), "custom: [one, two]") {
		t.Fatal(string(out))
	}
}
func TestReservedValidation(t *testing.T) {
	c := codec{}
	for _, test := range []struct {
		p, raw string
		bad    bool
	}{{"index.md", "---\nokf_version: \"0.2\"\n---\n# Index\n", false}, {"topic/index.md", "---\nokf_version: \"0.2\"\n---\n", true}, {"index.md", "---\ntype: Wrong\n---\n", true}, {"index.md", indexEnd + indexStart, true}, {"index.md", indexStart, true}, {"log.md", "# Log\n## 2026-10-24\n- Entry\n## 2026-10-23\n", false}, {"log.md", "## October 24\n", true}, {"log.md", "## 2026-10-23\n## 2026-10-24\n", true}, {"log.md", "---\ntype: Wrong\n---\n", true}} {
		if ds := c.ValidateReserved(test.p, []byte(test.raw)); (len(ds) > 0) != test.bad {
			t.Errorf("%s %q: %+v", test.p, test.raw, ds)
		}
	}
}
func TestCodecLimitsAndUniqueEdits(t *testing.T) {
	c := codec{}
	for _, raw := range []string{strings.Repeat("x", api.MaxNoteBytes+1), "---\ntype: X\na: " + strings.Repeat("[", 40) + "0" + strings.Repeat("]", 40) + "\n---\n", "---\ntype: X\na: &cycle [*cycle]\n---\n"} {
		if _, ds := c.Parse("bad.md", []byte(raw)); !hasErrors(ds) {
			t.Fatal("limit accepted")
		}
	}
	raw := []byte("---\ntype: X\n---\nx x\n")
	if _, err := c.Patch("a.md", raw, api.Operation{BodyEdits: []api.TextEdit{{OldText: "x", NewText: "y"}}}, "process:test", time.Now()); err == nil {
		t.Fatal("ambiguous text edit")
	}
	if _, err := c.Patch("../bad.md", raw, api.Operation{}, "process:test", time.Now()); err == nil {
		t.Fatal("traversal accepted")
	}
	if _, err := c.Patch("a.md", nil, api.Operation{Kind: "create"}, "process:test", time.Now()); err == nil {
		t.Fatal("missing type")
	}
}
func ptr(s string) *string { return &s }
func FuzzParse(f *testing.F) {
	for _, s := range []string{"---\ntype: Example\n---\nBody", "plain", "---\na: &a [*a]\n---\n", "---\ntype: X\nvalue: {42: example}\n---\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > api.MaxNoteBytes+1 {
			return
		}
		_, _ = (codec{}).Parse("synthetic.md", []byte(raw))
	})
}
func FuzzPath(f *testing.F) {
	for _, p := range []string{"note.md", "../escape.md", "a/../b.md", "/absolute.md", "Unicode/ação.md", "a\\b.md"} {
		f.Add(p)
	}
	f.Fuzz(func(t *testing.T, p string) {
		if validPath(p) == nil {
			for _, part := range strings.Split(p, "/") {
				if part == ".." || part == "." || part == "" {
					t.Fatalf("accepted %q", p)
				}
			}
		}
	})
}

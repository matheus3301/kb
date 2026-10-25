package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

// readRepairFile confines fixture reads to the test-owned temporary root.
func readRepairFile(t *testing.T, root, name string) ([]byte, error) {
	t.Helper()
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := dir.Close(); err != nil {
			t.Error(err)
		}
	}()
	return dir.ReadFile(name)
}

func repairRequest(t *testing.T, root, name string) map[string]any {
	t.Helper()
	raw, err := readRepairFile(t, root, name)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"repair": true, "incremental": true, "actor": "agent:repair-test", "summary": "Prepare a synthetic note for OKF use", "idempotency_key": "repair-" + name,
		"operations": []map[string]any{{"operation_id": "fix", "kind": "patch", "path": name, "expected_revision": api.Revision(raw), "document": "---\ntype: Reference\n---\n" + string(raw)}},
	}
}

func requestJSON(t *testing.T, input any) string {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestAgentRepairLoopAcrossMoreThanFiftyInvalidNotes(t *testing.T) {
	root := realFixture(t)
	const count = 53
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("legacy-%02d.md", i)
		if err := os.WriteFile(filepath.Join(root, name), []byte("# Synthetic source\nPreserve this exact body.\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cs, cleanup := realSession(t, root, false, nil)
	defer cleanup()
	code, first := invoke(t, []string{"--root", root, "validate"}, "")
	report := first["data"].(map[string]any)
	if code != 1 || report["summary"].(map[string]any)["errors"] != float64(count) {
		t.Fatalf("%d %#v", code, report)
	}
	diagnostic := report["diagnostics"].([]any)[0].(map[string]any)
	if diagnostic["rule"] != "MISSING_FRONTMATTER" || diagnostic["severity"] != "error" || diagnostic["line"] != float64(1) || diagnostic["hint"] == "" || diagnostic["revision"] == "" {
		t.Fatalf("unusable diagnostic %#v", diagnostic)
	}
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("legacy-%02d.md", i)
		request := repairRequest(t, root, name)
		request["dry_run"] = true
		raw := requestJSON(t, request)
		_, preview := invoke(t, []string{"--root", root, "apply", "--input", "-"}, raw)
		remote := protocolCall(t, cs, "apply", raw)
		if preview["ok"] != true || !reflect.DeepEqual(preview, remote) {
			t.Fatalf("preview mismatch %#v %#v", preview, remote)
		}
		progress := preview["data"].(map[string]any)["repair_progress"].(map[string]any)
		if progress["errors_before"] != float64(count-i) || progress["errors_remaining"] != float64(count-i-1) {
			t.Fatal(progress)
		}
		untouched, err := readRepairFile(t, root, name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.HasPrefix(untouched, []byte("---")) {
			t.Fatal("preview wrote note")
		}
		request["dry_run"] = false
		raw = requestJSON(t, request)
		code, applied := invoke(t, []string{"--root", root, "apply", "--repair", "--incremental", "--input", "-"}, raw)
		if code != 0 || applied["ok"] != true {
			t.Fatalf("apply %d %#v", code, applied)
		}
		replay := protocolCall(t, cs, "apply", raw)
		if replay["ok"] != true || replay["data"].(map[string]any)["replayed"] != true || !reflect.DeepEqual(applied["data"].(map[string]any)["repair_progress"], replay["data"].(map[string]any)["repair_progress"]) {
			t.Fatal(replay)
		}
		after, err := readRepairFile(t, root, name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(after, append([]byte("---\ntype: Reference\n---\n"), untouched...)) {
			t.Fatal("repair changed body")
		}
	}
	code, final := invoke(t, []string{"--root", root, "validate"}, "")
	if code != 0 || final["data"].(map[string]any)["clean"] != true {
		t.Fatalf("%d %#v", code, final)
	}
	code, hits := invoke(t, []string{"--root", root, "search", "synthetic", "--limit", "100"}, "")
	if code != 0 || hits["data"].(map[string]any)["total"].(float64) < count {
		t.Fatal("repaired notes not searchable", hits)
	}
}

func TestRepairRejectsUnsafeOrNonprogressingPlans(t *testing.T) {
	for _, tc := range []string{"missing repair", "strict repair", "move", "create", "invalid replacement", "stale revision", "unrelated edit"} {
		t.Run(tc, func(t *testing.T) {
			root := realFixture(t)
			for _, name := range []string{"legacy-a.md", "legacy-b.md"} {
				if err := os.WriteFile(filepath.Join(root, name), []byte("Legacy body\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			request := repairRequest(t, root, "legacy-a.md")
			op := request["operations"].([]map[string]any)[0]
			switch tc {
			case "missing repair":
				request["repair"] = false
			case "strict repair":
				request["incremental"] = false
			case "move":
				op["kind"] = "move"
				op["to"] = "moved.md"
			case "create":
				op["kind"] = "create"
				op["path"] = "new.md"
				delete(op, "expected_revision")
			case "invalid replacement":
				op["document"] = "Still invalid\n"
			case "stale revision":
				op["expected_revision"] = api.Revision([]byte("obsolete"))
			case "unrelated edit":
				name := "03_Resources/alpha note.md"
				raw, err := readRepairFile(t, root, name)
				if err != nil {
					t.Fatal(err)
				}
				op["path"] = name
				op["expected_revision"] = api.Revision(raw)
				op["document"] = string(raw) + "Unrelated change\n"
			}
			before, err := readRepairFile(t, root, "legacy-a.md")
			if err != nil {
				t.Fatal(err)
			}
			code, result := invoke(t, []string{"--root", root, "apply", "--input", "-"}, requestJSON(t, request))
			if code == 0 || result["ok"] != false {
				t.Fatal("unsafe repair accepted", result)
			}
			after, err := readRepairFile(t, root, "legacy-a.md")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("rejected repair wrote note")
			}
		})
	}
}

func TestStrictHealthAndFieldDiagnosticsHaveCLIAndMCPParity(t *testing.T) {
	root := realFixture(t)
	raw := []byte("---\ntype: Reference\nstatus: active\nstale_after: yesterday\n---\nBody\n")
	if err := os.WriteFile(filepath.Join(root, "bad-fields.md"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	_, result := invoke(t, []string{"--root", root, "validate"}, "")
	found := map[string]bool{}
	for _, value := range result["data"].(map[string]any)["diagnostics"].([]any) {
		d := value.(map[string]any)
		if d["path"] == "bad-fields.md" {
			if d["field"] == nil || d["line"] == nil || d["hint"] == "" {
				t.Fatal(d)
			}
			found[d["rule"].(string)] = true
		}
	}
	if !found["INVALID_STATUS"] || !found["INVALID_TIME"] {
		t.Fatal(found)
	}
	if err := os.Remove(filepath.Join(root, "bad-fields.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "orphan.md"), []byte("---\ntype: Reference\n---\nUnlinked\n"), 0600); err != nil {
		t.Fatal(err)
	}
	code, result := invoke(t, []string{"--root", root, "validate", "--profile", "health"}, "")
	if code != 0 || result["data"].(map[string]any)["conformant"] != true || result["data"].(map[string]any)["clean"] != false {
		t.Fatal(code, result)
	}
	code, strict := invoke(t, []string{"--root", root, "validate", "--profile", "health", "--strict"}, "")
	if code != 1 {
		t.Fatal(code, strict)
	}
	cs, cleanup := realSession(t, root, true, nil)
	defer cleanup()
	remote := protocolCall(t, cs, "validate", `{"profile":"health","strict":true}`)
	if !reflect.DeepEqual(strict, remote) {
		t.Fatalf("%#v != %#v", strict, remote)
	}
}

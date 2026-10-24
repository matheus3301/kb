package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func invoke(t *testing.T, args []string, input string) (int, map[string]any) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(context.Background(), args, strings.NewReader(input), &out, &errOut)
	var e map[string]any
	if err := json.Unmarshal(out.Bytes(), &e); err != nil {
		t.Fatalf("%s: %v (stderr %s)", out.String(), err, errOut.String())
	}
	return code, e
}

func TestMainWiresCoreAndWriter(t *testing.T) {
	root := filepath.Join(t.TempDir(), "knowledge")
	code, e := invoke(t, []string{"--root", root, "init"}, "")
	if code != 0 || e["ok"] != true {
		t.Fatalf("init %d %#v", code, e)
	}
	input := `{"path":"memory.md","type":"Concept","body":"# Reusable memory\nSynthetic knowledge.","actor":"agent:test","summary":"Record a synthetic finding","idempotency_key":"main-test"}`
	code, e = invoke(t, []string{"--root", root, "create", "--input", "-", "--dry-run"}, input)
	if code != 0 || e["ok"] != true {
		t.Fatalf("dry-run %d %#v", code, e)
	}
	if _, err := os.Stat(filepath.Join(root, "memory.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created a note: %v", err)
	}
	code, e = invoke(t, []string{"--root", root, "create", "--input", "-"}, input)
	if code != 0 || e["ok"] != true {
		t.Fatalf("create %d %#v", code, e)
	}
	code, e = invoke(t, []string{"--root", root, "read", "memory.md"}, "")
	if code != 0 {
		t.Fatalf("read %d %#v", code, e)
	}
	data := e["data"].(map[string]any)
	note := data["note"].(map[string]any)
	if note["revision"] == "" || !strings.Contains(data["markdown"].(string), "Synthetic knowledge") {
		t.Fatalf("bad read %#v", data)
	}
	code, e = invoke(t, []string{"--root", root, "--read-only", "create", "--input", "-"}, input)
	if code != 5 || e["ok"] != false {
		t.Fatalf("readonly %d %#v", code, e)
	}
}

func helperCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	argv := append([]string{"-test.run=^TestMainHelperProcess$", "--"}, args...)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, argv...) // #nosec G204 -- Re-executes only this Go test binary with synthetic test arguments.
	cmd.Env = append(os.Environ(), "KB_TEST_HELPER=1")
	return cmd
}

func TestMainHelperProcess(t *testing.T) {
	if os.Getenv("KB_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"kb"}, os.Args[i+1:]...)
			main()
			return
		}
	}
	t.Fatal("missing helper arguments")
}

func TestMainProcessExitAndVersion(t *testing.T) {
	cmd := helperCommand(t, "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var e map[string]any
	if err = json.Unmarshal(out, &e); err != nil {
		t.Fatal(err)
	}
	data := e["data"].(map[string]any)
	if data["version"] != "dev" || data["commit"] != "unknown" {
		t.Fatalf("%#v", data)
	}
	cmd = helperCommand(t, "unknown")
	out, err = cmd.Output()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 {
		t.Fatalf("%v %s", err, out)
	}
	if err = json.Unmarshal(out, &e); err != nil || e["ok"] != false {
		t.Fatalf("%v %#v", err, e)
	}
}

func TestMainStdioMCPHasNoOutputPollutionAndShutsDown(t *testing.T) {
	root := filepath.Join(t.TempDir(), "knowledge")
	if code, e := invoke(t, []string{"--root", root, "init"}, ""); code != 0 {
		t.Fatalf("init %d %#v", code, e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := helperCommand(t, "--root", root, "mcp", "--read-only")
	var errOut bytes.Buffer
	cmd.Stderr = &errOut
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "integration-test", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("protocol connection: %v stderr %s", err, errOut.String())
	}
	tools, err := cs.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 6 {
		t.Fatalf("%v %#v", err, tools)
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_status", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("%v %#v", err, res)
	}
	if err := cs.Close(); err != nil {
		t.Fatal(err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

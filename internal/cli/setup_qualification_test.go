package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSetupCheckDoesNotQualifyUnknownNativeBuild(t *testing.T) {
	f := newSetupFixture(t)
	nativeFixture(t, &f, "omp")
	marker := filepath.Join(f.home, "host-executed")
	// A discoverable executable is not a supported-version or activation probe.
	host := filepath.Join(f.home, "bin", "omp")
	putSetup(t, host, "#!/bin/sh\nprintf 'omp/999.0.0-unknown\\n'\nprintf 'executed' > '"+marker+"'\n")
	//nolint:gosec // Executable is an isolated synthetic host fixture, never a native agent.
	if err := os.Chmod(host, 0700); err != nil {
		t.Fatal(err)
	}
	code, e, _, stderr := runSetup(t, f, f.install("omp", "--hooks", "required"), "")
	if code != 0 {
		t.Fatal(e, stderr)
	}
	before := treeBytes(t, f.root)
	code, e, _, _ = runSetup(t, f, []string{"setup", "check", "--agent", "omp", "--project", f.project}, "")
	data := e["data"].(map[string]any)
	if code != 0 || data["configured"] != true || data["ready"] != true || data["activation_observed"] != false || data["status"] != "activation_pending" {
		t.Fatal("local executable presence falsely qualified native activation", code, data)
	}
	diagnostics := data["diagnostics"].([]any)
	text := ""
	for _, d := range diagnostics {
		text += d.(string) + "\n"
	}
	if !strings.Contains(text, "version") || !strings.Contains(text, "platform") || !strings.Contains(text, "unobserved") {
		t.Fatal("inspection failed to disclose unqualified native capability", text)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("inspection executed an untrusted host executable", err)
	}
	if !reflect.DeepEqual(before, treeBytes(t, f.root)) {
		t.Fatal("capability inspection mutated collection")
	}
}

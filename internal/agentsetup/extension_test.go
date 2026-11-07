package agentsetup

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exercise the generated wrapper with real subprocesses, not mocked execFile.
func TestNativeExtensionProcessFailures(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("Node is required for the generated-wrapper regression; run mise install, then mise exec -- go test ./internal/agentsetup: %v", err)
	}
	for _, scenario := range []string{"timeout", "failed", "empty", "malformed", "oversized", "no-hit", "disabled"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			pidFile := filepath.Join(dir, "child.pid")
			pidJSON, _ := json.Marshal(pidFile)
			helper := "#!" + node + "\nimport { writeFileSync } from 'node:fs';\nwriteFileSync(" + string(pidJSON) + ", String(process.pid));\n"
			switch scenario {
			case "timeout":
				helper += "process.on('SIGTERM', () => {}); process.stdin.resume(); setInterval(() => {}, 1000);\n"
			case "failed":
				helper += "console.log(JSON.stringify({context:'FALSE_SUCCESS'})); process.exit(7);\n"
			case "malformed":
				helper += "console.log('not JSON');\n"
			case "oversized":
				helper += "console.log(JSON.stringify({context:'x'.repeat(20000)}));\n"
			case "no-hit":
				helper += "console.log(JSON.stringify({context:'No lexical matches. This does not establish absence.'}));\n"
			case "disabled":
				helper += "console.log(JSON.stringify({context:''}));\n"
			}
			write := func(name, text string, mode os.FileMode) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, name), []byte(text), mode); err != nil {
					t.Fatal(err)
				}
			}
			write("helper.mjs", helper, 0700)
			r := fixtureRequest()
			r.Agent, r.Executable = "omp", filepath.Join(dir, "helper.mjs")
			write("extension.mjs", nativeExtension(r), 0600)
			write("runner.mjs", `import { readFileSync } from 'node:fs';
import extension from './extension.mjs';
const handlers = {};
let sent = 0;
extension({on(name, fn) { handlers[name] = fn; }, sendMessage() { sent++; }});
const started = Date.now();
const result = await handlers.before_agent_start({prompt:'absent-token'}, {cwd:process.cwd()});
if (result === undefined) await handlers.session_start({}, {cwd:process.cwd()});
const pid = Number(readFileSync('child.pid', 'utf8'));
try {
  process.kill(pid, 0);
  throw new Error('hook child survived completion');
} catch (error) {
  if (error.code !== 'ESRCH') throw error;
}
console.log(JSON.stringify({content:result?.message?.content ?? '', suppressed:result === undefined, sent, elapsed:Date.now()-started}));
`, 0600)
			// Kill the deliberately uncooperative fixture even when the regression fails.
			t.Cleanup(func() {
				// #nosec G304 -- Test-owned PID file under t.TempDir.
				data, readErr := os.ReadFile(pidFile)
				if readErr != nil {
					return
				}
				pid, parseErr := strconv.Atoi(string(data))
				if parseErr == nil {
					if process, findErr := os.FindProcess(pid); findErr == nil {
						_ = process.Kill()
					}
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			// #nosec G204 -- Executes the generated wrapper and synthetic helper in a disposable directory.
			cmd := exec.CommandContext(ctx, node, filepath.Join(dir, "runner.mjs"))
			cmd.Dir = dir
			output, runErr := cmd.CombinedOutput()
			if runErr != nil {
				t.Fatalf("generated wrapper did not settle: %v (%v)\n%s", runErr, ctx.Err(), output)
			}
			var result struct {
				Content    string `json:"content"`
				Elapsed    int    `json:"elapsed"`
				Suppressed bool   `json:"suppressed"`
				Sent       int    `json:"sent"`
			}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatal(err, string(output))
			}
			if scenario == "disabled" {
				if !result.Suppressed || result.Sent != 0 || result.Content != "" {
					t.Fatal("opted-out wrapper emitted a prompt or session message", result)
				}
			} else if scenario == "no-hit" {
				if !strings.Contains(result.Content, "No lexical matches") || strings.Contains(result.Content, "unavailable") {
					t.Fatal("empty retrieval was misreported as infrastructure failure", result)
				}
			} else if !strings.Contains(result.Content, "KB hook unavailable") || strings.Contains(result.Content, "FALSE_SUCCESS") {
				t.Fatal("failure claimed successful consultation", result)
			}
			if scenario == "timeout" && (result.Elapsed < 3000 || result.Elapsed > 6500) {
				t.Fatal("timeout outside generous scheduling bound", result)
			}
			t.Logf("%s: %d ms, %s", scenario, result.Elapsed, result.Content)
		})
	}
}

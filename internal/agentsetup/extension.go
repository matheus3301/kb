package agentsetup

import "strings"

// The wrapper uses only Node-compatible builtins, also available in OMP's Bun
// runtime. It has no runtime package dependency and never invokes a shell.
func nativeExtension(r Request) string {
	return strings.NewReplacer("__EXECUTABLE__", string(mustJSON(invocation(r))), "__AGENT__", string(mustJSON(r.Agent))).Replace(nativeExtensionSource)
}

const nativeExtensionSource = `import { execFile } from "node:child_process";

const executable = __EXECUTABLE__;
const agent = __AGENT__;
const unavailable = "KB hook unavailable; resolve the current binding and consult manually. Do not claim consultation succeeded.";

function retrieve(event, ctx, prompt = "") {
  const payload = JSON.stringify({ hook_event_name: event, cwd: ctx.cwd, prompt });
  if (Buffer.byteLength(payload, "utf8") > 65536) return Promise.resolve(unavailable);
  return new Promise((resolve) => {
    const child = execFile(executable, ["setup", "hook", "--agent", agent, "--event", event],
      { cwd: ctx.cwd, timeout: 3500, killSignal: "SIGKILL", maxBuffer: 16384, encoding: "utf8", shell: false },
      (error, stdout) => {
        if (error) { resolve(unavailable); return; }
        try {
          const result = JSON.parse(stdout);
          resolve(typeof result.context === "string" ? result.context : unavailable);
        } catch { resolve(unavailable); }
      });
    child.stdin.on("error", () => {});
    child.stdin.end(payload);
  });
}

export default function knowledgeBase(pi) {
  pi.on("session_start", async (_event, ctx) => {
    const content = await retrieve("SessionStart", ctx);
    pi.sendMessage({ customType: "kb-session", content, display: false });
  });
  pi.on("before_agent_start", async (event, ctx) => {
    // OMP revivals and Pi custom-message turns are not submitted user prompts.
    if (event.reason === "revive" || event.messageType === "custom") return;
    if (typeof event.prompt !== "string") return;
    const content = await retrieve("UserPromptSubmit", ctx, event.prompt);
    return { message: { customType: "kb-context", content, display: false } };
  });
}
`

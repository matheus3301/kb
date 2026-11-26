#!/usr/bin/env python3
"""Check the additive OKF v0.3 root-index slice through a real CLI and stdio MCP."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

from compatibility import CLI, ConsumerMCP, require


def run(binary):
    started = time.perf_counter()
    counts = {"fixtures": 0, "validation_parity": 0, "write_transports": 0,
              "exact_exports": 0, "escape_rejections": 0}
    with tempfile.TemporaryDirectory(prefix="kb-okf-v03-") as temporary:
        directory = Path(temporary).resolve()
        home = directory / "home"
        home.mkdir()
        environment = dict(os.environ, HOME=str(home), XDG_CONFIG_HOME=str(home / ".config"))
        environment.pop("KB_ROOT", None)
        for transport in ("cli", "mcp"):
            for newline in ("\n", "\r\n"):
                counts["fixtures"] += 1
                root = directory / f"knowledge-{counts['fixtures']}"
                cli = CLI(binary, root, environment)
                cli.run("init")
                sentinel = b"Synthetic outside sentinel.\n"
                (directory / "outside.txt").write_bytes(sentinel)
                header = ("---\n# Synthetic bundle metadata\nokf_version: '0.3'\n"
                          "title: '[untrusted]<script>'\ndescription: |\n"
                          "  <!-- kb:index:start -->\n  <!-- kb:index:end -->\n"
                          "license: ../../outside.txt\n"
                          "x-fixture: &fixture {decimal: 1234567890.123456789, text: 'café'}\n"
                          "owner: *fixture # Keep unknown keys and alias\n"
                          "verified: {by: 'human:synthetic', at: '2026-01-01T00:00:00Z'}\n"
                          "---\n").replace("\n", newline).encode()
                original = header + b"# Bundle\n\nManual prose.\n"
                (root / "index.md").write_bytes(original)
                note = b"---\ntype: Reference\nx-precision: 1234567890.123456789 # Keep\n---\nUnchanged.\n"
                (root / "untouched.md").write_bytes(note)
                (root / "LICENSE.txt").write_bytes(b"Synthetic license asset.\n")
                mcp = ConsumerMCP(cli)
                try:
                    valid = cli.run("validate", "--profile", "okf")
                    require(valid == mcp.result("kb_validate", {"profile": "okf"}), "validation parity")
                    require(valid["data"]["conformant"], "declared v0.3 metadata rejected")
                    counts["validation_parity"] += 1
                    cli.run("maintain", "--dry-run")
                    require((root / "index.md").read_bytes() == original, "preview changed index")
                    cli.run("maintain", "--apply")
                    request = {"path": "new note.md", "type": "Reference", "body": "Synthetic evidence.\n",
                               "metadata": {"title": "[unsafe]<script>", "description": "[outside](../../outside.md)"},
                               "actor": "process:okf-v03", "summary": "Record synthetic evidence",
                               "idempotency_key": "v03-create"}
                    if transport == "cli":
                        cli.run("create", payload=request)
                    else:
                        mcp.result("kb_create", request)
                    counts["write_transports"] += 1
                    after = (root / "index.md").read_bytes()
                    require(after.startswith(header) and b"Manual prose." in after, "header/manual fidelity")
                    require(b"- [\\[unsafe\\]&lt;script&gt;](new%20note.md)" in after, "navigation escaping")
                    require((root / "untouched.md").read_bytes() == note, "unchanged note fidelity")
                    require(mcp.result("kb_read", {"path": "new note.md"})["data"]["note"]["trust"] == "unverified",
                            "bundle metadata granted concept trust")
                    destination = directory / f"export-{counts['fixtures']}"
                    exported = cli.run("export", str(destination))["data"]
                    require(exported["conformant"], "export unexpectedly degraded")
                    require((destination / "index.md").read_bytes() == after, "export index fidelity")
                    require((destination / "untouched.md").read_bytes() == note, "export note fidelity")
                    require("LICENSE.txt" in exported["excluded"] and not (destination / "LICENSE.txt").exists(),
                            "export asset scope changed")
                    manifest = json.loads((destination / "manifest.json").read_text())
                    require(manifest["conformant"] and "index.md" in manifest["files"], "export manifest")
                    counts["exact_exports"] += 1
                    # A declared v0.3 index never changes v0.2/unknown-version policy.
                    for declaration, valid in (("'0.2'", True), ("'99.0'", True), ("0.3", False)):
                        (root / "index.md").write_text(f"---\nokf_version: {declaration}\n---\n# Bundle\n")
                        result = cli.run("validate", "--profile", "okf", exit_code=0 if valid else 1)
                        require(result == mcp.result("kb_validate", {"profile": "okf"}), "version diagnostics parity")
                        require(result["data"]["conformant"] == valid, "version policy changed")
                        counts["validation_parity"] += 1
                    for declaration, field in (("'0.2'", "title: Synthetic"), ("'99.0'", "title: Synthetic"),
                                               ("'0.3'", "title: 42"), ("'0.3'", "license: {}"),
                                               ("'0.3'", "x-extra: {duplicate: one, duplicate: two}")):
                        invalid = f"---\nokf_version: {declaration}\n{field}\n---\n# Bundle\n".encode()
                        (root / "index.md").write_bytes(invalid)
                        result = cli.run("validate", "--profile", "okf", exit_code=1)
                        require(result == mcp.result("kb_validate", {"profile": "okf"}), "invalid diagnostics parity")
                        require(not result["data"]["conformant"], "invalid metadata accepted")
                        counts["validation_parity"] += 1
                    rejected = dict(request, path="blocked.md", idempotency_key="blocked")
                    cli.run("create", payload=rejected, exit_code=1, code="INVALID_FRONTMATTER")
                    mcp.result("kb_create", rejected, code="INVALID_FRONTMATTER")
                    require(not (root / "blocked.md").exists(), "invalid-base write published")
                    (root / "index.md").write_bytes(after)
                    (root / "escape.md").symlink_to(directory / "outside.txt")
                    for path in ("../outside.txt", "escape.md"):
                        # The exact existing confinement error may differ by path; both interfaces must agree.
                        process = subprocess.run([str(binary), "--root", str(root), "read", path],
                                                 env=environment, capture_output=True, text=True, timeout=10)
                        value = json.loads(process.stdout)
                        require(process.returncode != 0 and not value["ok"], "escape read succeeded")
                        require(value == mcp.result("kb_read", {"path": path}, code=value["error"]["code"]),
                                "escape diagnostics parity")
                        counts["escape_rejections"] += 1
                    require((directory / "outside.txt").read_bytes() == sentinel, "outside sentinel changed")
                finally:
                    mcp.close()
    return {"ok": True, "scope": "synthetic source-built CLI/stdio MCP; not upstream conformance",
            "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
            "counts": counts, "elapsed_ms": round((time.perf_counter() - started) * 1000, 3)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    args = parser.parse_args()
    print(json.dumps(run(args.binary.resolve()), indent=2))


if __name__ == "__main__":
    main()

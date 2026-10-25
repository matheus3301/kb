#!/usr/bin/env python3
"""Check installed kb move fidelity using disposable, synthetic Markdown.

Run with Python 3.12+ and kb on PATH, or set KB_BIN to an installed executable.
Exit 1 means a preservation defect was observed; exit 2 means setup failed.
Temporary knowledge bases are removed on exit. No source build is performed.
"""

import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


def invoke(binary, root, *args, request=None):
    result = subprocess.run(
        [binary, "--root", str(root), *args],
        input=None if request is None else json.dumps(request),
        capture_output=True, text=True, timeout=30,
    )
    response = json.loads(result.stdout)
    if result.returncode != 0 or not response.get("ok"):
        raise RuntimeError(f"kb {args[0]} failed: {response.get('error')}")
    return response


def main():
    binary = os.environ.get("KB_BIN") or shutil.which("kb")
    if not binary:
        raise RuntimeError("Install kb or set KB_BIN before running this probe")
    version = subprocess.run(
        [binary, "version"], capture_output=True, text=True, timeout=10, check=True,
    )
    version_data = json.loads(version.stdout)["data"]
    with tempfile.TemporaryDirectory(prefix="kb-move-fidelity-") as directory:
        root = Path(directory) / "knowledge"
        invoke(binary, root, "init")
        source = b"---\ntype: Reference\ntitle: Source\n---\n# Source\n\nSynthetic evidence.\n"
        (root / "source.md").write_bytes(source)
        revision = "sha256:" + hashlib.sha256(source).hexdigest()
        before = f"""---
type: Synthesis
sources:
  # Source annotation must survive
  - resource: source.md
    unknown_decimal: 1234567890.123456789 # Preserve numeric precision
    unknown_tiny: 0.1234567890123456789
basis_revisions:
  # Basis annotation must survive
  - path: source.md
    revision: {revision}
    unknown_decimal: 1234567890.123456789
    custom: keep # Custom basis comment
verified:
  - by: human:synthetic-reviewer
    at: '2025-01-02T00:00:00Z'
# Trailing annotation independent of verification
---
# Synthetic synthesis
"""
        (root / "synthesis.md").write_text(before, encoding="utf-8")
        invoke(
            binary, root, "move", "source.md", "nested/source.md", "--input", "-",
            request={
                "expected_revision": revision,
                "summary": "Relocate a synthetic source",
                "actor": "process:qa",
                "idempotency_key": "move-fidelity/probe",
            },
        )
        after = (root / "synthesis.md").read_text(encoding="utf-8")
        validation = invoke(binary, root, "validate", "--profile", "okf")
        required = [
            "unknown_decimal: 1234567890.123456789",
            "unknown_tiny: 0.1234567890123456789",
            "# Source annotation must survive",
            "# Preserve numeric precision",
            "# Basis annotation must survive",
            "# Custom basis comment",
            "# Trailing annotation independent of verification",
        ]
        checks = {text: before.count(text) == after.count(text) for text in required}
        checks["source path updated"] = "resource: nested/source.md" in after
        checks["basis path updated"] = "path: nested/source.md" in after
        checks["verification cleared"] = "verified:" not in after
        checks["OKF conformant"] = validation["data"]["conformant"]
        # Do not print operational roots or receipt metadata into public evidence.
        print(json.dumps({
            "build": version_data, "checks": checks,
            "before": before, "after": after,
        }, indent=2))
        return 0 if all(checks.values()) else 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, RuntimeError, subprocess.SubprocessError) as error:
        print(f"QA setup failed: {error}", file=sys.stderr)
        sys.exit(2)

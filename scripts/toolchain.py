"""Read authoritative mise pins and reject toolchain/configuration drift."""

import json
import os
from pathlib import Path
import re
import sys

try:
    import tomllib
except ModuleNotFoundError:
    raise SystemExit("Toolchain bootstrap requires Python 3.11+ with tomllib; use the Python selector in mise.toml.") from None


ROOT = Path(__file__).resolve().parents[1]
TOOL_KEYS = {
    name: name for name in
    ("python", "go", "node", "goreleaser", "golangci-lint", "actionlint", "gitleaks")
}
TOOL_KEYS["pnpm"] = "npm:pnpm"
TOOL_KEYS["govulncheck"] = "go:golang.org/x/vuln/cmd/govulncheck"


def versions(root=ROOT):
    tools = tomllib.loads((root / "mise.toml").read_text())["tools"]
    if set(tools) != set(TOOL_KEYS.values()):
        raise ValueError("mise.toml must declare exactly the supported development tools")
    for name, value in tools.items():
        pattern = r"3\.(\d+)" if name == "python" else r"\d+\.\d+\.\d+"
        match = re.fullmatch(pattern, value) if isinstance(value, str) else None
        if not match or (name == "python" and int(match[1]) < 11):
            raise ValueError("mise.toml requires exact tool pins and a Python 3.11+ minor selector")
    return {name: tools[key] for name, key in TOOL_KEYS.items()}


def pin_findings(root=ROOT):
    try:
        pins = versions(root)
    except (OSError, KeyError, ValueError) as error:
        return [f"invalid mise toolchain: {error}"]
    findings = []
    for name in (".tool-versions", ".golangci-lint-version", "site/package-lock.json"):
        if (root / name).exists():
            findings.append(f"obsolete split toolchain file: {name}")
    go_mod = (root / "go.mod").read_text()
    if re.findall(r"^go (\S+)$", go_mod, re.M) != [pins["go"]]:
        findings.append("go.mod Go version must match mise.toml")
    toolchain = re.findall(r"^toolchain (\S+)$", go_mod, re.M)
    if toolchain and toolchain != ["go" + pins["go"]]:
        findings.append("go.mod toolchain must match mise.toml")
    package = json.loads((root / "site/package.json").read_text())
    if package.get("packageManager") != "pnpm@" + pins["pnpm"]:
        findings.append("site packageManager must match mise.toml pnpm")
    mappings = json.loads((root / "scripts/tools.json").read_text())
    for name, tool in mappings.items():
        if name not in pins or set(tool) != {"module"}:
            findings.append("scripts/tools.json must contain module mappings only, without versions")
    if not (root / "site/pnpm-lock.yaml").is_file():
        findings.append("site must commit pnpm-lock.yaml")
    return findings


def workflow_findings(text):
    findings = []
    jobs = re.findall(r"^  [\w-]+:\s*\n(.*?)(?=^  [\w-]+:|\Z)", text, re.M | re.S)
    for job in jobs:
        steps = list(re.finditer(r"^      - [^\n]+\n(.*?)(?=^      - |\Z)", job, re.M | re.S))
        readers = {"actions/setup-go": ("go-version", "go"),
                   "actions/setup-node": ("node-version", "node"),
                   "actions/setup-python": ("python-version", "python"),
                   "pnpm/action-setup": ("version", "pnpm"),
                   "golangci/golangci-lint-action": ("version", "golangci-lint"),
                   "goreleaser/goreleaser-action": ("version", "goreleaser")}
        for step in steps:
            action = re.search(r"^(?:      - |        )uses: ([\w/-]+)@", step[0], re.M)
            if not action or action[1] not in readers:
                continue
            field, name = readers[action[1]]
            expected = "${{ steps.toolchain.outputs." + name + " }}"
            values = re.findall(r"^          " + field + r": (.+)$", step[0], re.M)
            if values != [expected] or re.search(r"^          (?:go-|node-|python-)?version-file:", step[0], re.M):
                findings.append(f"{action[1]} must read {name} from mise toolchain outputs")
            output = re.search(r"^      - id: toolchain\n(?:        working-directory: \.\n)?"
                               r"        run: python3 scripts/toolchain.py github-outputs\n", job, re.M)
            if not output or output.end() > step.start():
                findings.append(f"{action[1]} needs the preceding checked mise output step")
    return findings


def site_findings(root=ROOT):
    makefile = (root / "Makefile").read_text()
    match = re.search(r"^site-check:\n((?:\t[^\n]*\n)+)", makefile, re.M)
    required = [
        "pnpm --dir site install --frozen-lockfile",
        "pnpm --dir site audit --audit-level=high",
        "pnpm --dir site run check",
        "KB_SITE_BASE=/kb/ pnpm --dir site run build",
        "pnpm --dir site exec playwright install chromium",
        "KB_SITE_BASE=/kb/ pnpm --dir site test",
    ]
    if not match or match[1].splitlines() != ["\t" + command for command in required]:
        return ["make site-check must retain frozen install, full audit, source/build/browser gates"]
    return []


def findings(root=ROOT):
    result = pin_findings(root)
    result.extend(site_findings(root))
    for path in sorted((root / ".github/workflows").glob("*.y*ml")):
        result.extend(f"{path.name}: {message}" for message in workflow_findings(path.read_text()))
    return result


def main():
    errors = pin_findings()
    if errors:
        raise SystemExit("\n".join(errors))
    pins = versions()
    if sys.argv[1:] == ["github-outputs"]:
        with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
            for name, version in pins.items():
                output.write(f"{name}={version}\n")
    elif sys.argv[1:] == ["check"]:
        errors = findings()
        if errors:
            raise SystemExit("\n".join(errors))
        print("Toolchain pins and development gates are consistent.")
    else:
        raise SystemExit("usage: toolchain.py {check|github-outputs}")


if __name__ == "__main__":
    main()

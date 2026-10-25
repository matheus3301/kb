"""Repository and Go quality gates; no application behavior lives here."""

import json
import os
from pathlib import Path
import re
import subprocess
import sys
from urllib.parse import unquote


ROOT = Path(__file__).resolve().parents[1]
IGNORED = {".git", ".tools", ".build", "__pycache__"}
MINIMUM_COVERAGE = 80.0


def tracked_content(root):
    return sorted(
        p for p in root.rglob("*")
        if p.is_file() and not any(part in IGNORED for part in p.relative_to(root).parts)
    )


def run(*args, env=None):
    print("+", " ".join(str(a) for a in args), flush=True)
    subprocess.run(args, cwd=ROOT, env=env, check=True)


def release_write_allowed(text, name, position):
    # Publishing a tag is the sole write-token exception. Keep its event and
    # verification dependency explicit; PR workflows remain read-only.
    header = 'name: Release\n\non:\n  push:\n    tags: ["v*"]\n\npermissions:\n  contents: read\n'
    if name != "release.yml" or not text.startswith(header):
        return False
    jobs = list(re.finditer(r"^  ([\w-]+):\s*\n(.*?)(?=^  [\w-]+:|\Z)", text, re.M | re.S))
    publish = next((job for job in jobs if job[1] == "publish"), None)
    verify = next((job for job in jobs if job[1] == "verify"), None)
    if not publish or not verify or not publish.start() <= position < publish.end():
        return False
    guard = "    if: ${{ github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v') }}\n"
    return ("    needs: verify\n" in publish[0] and guard in publish[0]
            and "    permissions:\n      contents: write\n" in publish[0]
            and "run: python3 scripts/release.py verify-tag\n" in verify[0]
            and "run: make tools check\n" in verify[0])


def workflow_findings(text, name=""):
    findings = []
    for target in re.findall(r"^\s*-?\s*uses:\s*([^\s#]+)", text, re.M):
        if not re.fullmatch(r"[\w.-]+/[\w./-]+@[a-f0-9]{40}", target):
            findings.append(f"action must be pinned to a full SHA: {target}")
    for forbidden in ("pull_request_target", "continue-on-error:", "write-all"):
        if forbidden in text:
            findings.append(f"forbidden workflow setting: {forbidden}")
    for match in re.finditer(r"^[ \t]+([\w-]+):[ \t]*['\"]?write['\"]?[ \t]*(?:#.*)?$", text, re.M):
        if not (match[1] == "contents" and release_write_allowed(text, name, match.start())):
            findings.append("workflow requests a write token")
    if re.search(r"^\s*permissions:[ \t]*\S", text, re.M):
        findings.append("permissions must use explicit block mappings")
    if not re.search(r"^permissions:\s*\n\s+contents: read\s*$", text, re.M):
        findings.append("workflow must declare contents: read")
    if "persist-credentials: false" not in text:
        findings.append("checkout must not persist credentials")
    return findings


def local_link_findings(root, path):
    # Code examples are not rendered links. Remote URLs are deliberately not
    # fetched: repository validation must not depend on third-party availability.
    body = re.sub(r"```.*?```", "", path.read_text(), flags=re.S)
    findings = []
    targets = re.findall(r"\]\(([^)\s]+)\)", body)
    targets += re.findall(r"^\s*\[[^\]]+\]:\s*<?([^\s>]+)>?", body, re.M)
    targets += re.findall(r"^\s*resource:\s*(\S+)", body, re.M)
    targets += re.findall(r'<(?:img|a|source)\b[^>]*\b(?:src|href)=["\']([^"\']+)', body)
    for target in targets:
        for prefix in ("https://raw.githubusercontent.com/matheus3301/kb/main/",
                       "https://github.com/matheus3301/kb/blob/main/"):
            if target.startswith(prefix):
                target = str(root / unquote(target[len(prefix):].split("#")[0]))
                break
        if re.match(r"[a-zA-Z][a-zA-Z0-9+.-]*:", target) or target.startswith("#"):
            continue
        resolved = (path.parent / unquote(target.split("#")[0])).resolve()
        if not resolved.is_relative_to(root.resolve()) or not resolved.exists():
            findings.append(f"{path.relative_to(root)}: missing/outside link {target}")
    return findings


def skill_findings(directory):
    # Validate the distributed folder, independently of the product checkout.
    entry = directory / "SKILL.md"
    if not entry.is_file():
        return [f"{directory.name}: missing SKILL.md"]
    header = re.match(r"\A---\n(.*?)\n---\n", entry.read_text(), re.S)
    if not header or not re.search(r"^name: " + re.escape(directory.name) + r"$", header[1], re.M):
        return [f"{directory.name}: frontmatter name must match the installed directory"]
    findings = []
    for path in directory.rglob("*.md"):
        findings.extend(local_link_findings(directory, path))
    return findings


def public_content_findings(path, content):
    findings = []
    workstation = r"/(?:Users|home)/[\w.-]+|[A-Za-z]:\\+Users\\+[\w.-]+"
    if re.search(workstation, content):
        findings.append(f"{path}: personal workstation path in public content")
    if re.search(r"(?:file|vscode|codex)://", content):
        findings.append(f"{path}: local-only URI in public content")
    if path.suffix == ".md" or path.name == "llms.txt":
        prose = re.sub(r"```.*?```", "", content, flags=re.S)
        portuguese = r"\b(?:n[aã]o|especifica[çc][aã]o|reposit[oó]rio|aprendizado|pr[oó]ximos|vis[aã]o|proposta|conhecimento)\b"
        if re.search(portuguese, prose, re.I):
            findings.append(f"{path}: repository Markdown must be English")
    return findings


def acceptance_findings(root, spec, criteria):
    expected = set(re.findall(r"^\| (A\d+) \|", spec, re.M))
    ids = [item["id"] for item in criteria]
    findings = []
    if len(ids) != len(set(ids)) or set(ids) != expected:
        findings.append("acceptance inventory must match every spec ID exactly once")
    for item in criteria:
        if item["status"] not in {"planned", "in_progress", "verified"}:
            findings.append(f"{item['id']}: unknown acceptance status")
        if item["status"] == "verified" and not (item["tests"] and item["evidence"]):
            findings.append(f"{item['id']}: verification needs tests and evidence")
        for target in item["tests"] + item["evidence"]:
            path = (root / target).resolve()
            if not path.is_relative_to(root.resolve()) or not path.is_file():
                findings.append(f"{item['id']}: missing/outside evidence {target}")
    return findings


def gate_findings(needs, event):
    findings = []
    for job in ("repository", "static", "tests", "security", "build", "release-config"):
        if needs.get(job, {}).get("result") != "success":
            findings.append(f"required job did not succeed: {job}")
    expected = "success" if event == "pull_request" else "skipped"
    if needs.get("dependency-review", {}).get("result") != expected:
        findings.append("dependency review did not reach its expected result")
    return findings


def require_clean(findings):
    if findings:
        raise SystemExit("\n".join(findings))


def repository():
    findings = []
    for path in tracked_content(ROOT):
        if path.suffix in {".md", ".json", ".yaml", ".yml", ".go", ".py", ".toml", ".svg"} or path.name == "llms.txt":
            findings.extend(public_content_findings(path.relative_to(ROOT), path.read_text()))
        if path.suffix == ".md" or path.name == "llms.txt":
            findings.extend(local_link_findings(ROOT, path))
    for directory in (ROOT / "skills").iterdir():
        if directory.is_dir():
            findings.extend(skill_findings(directory))
    workflows = ROOT / ".github/workflows"
    for path in sorted(list(workflows.glob("*.yml")) + list(workflows.glob("*.yaml"))):
        findings.extend(f"{path.name}: {f}" for f in workflow_findings(path.read_text(), path.name))
    criteria = json.loads((ROOT / "docs/acceptance.json").read_text())
    findings.extend(acceptance_findings(ROOT, (ROOT / "docs/spec.md").read_text(), criteria))
    require_clean(findings)
    print(f"Repository checks passed; {len(criteria)} acceptance criteria tracked.")


def has_go():
    if any(p.suffix == ".go" for p in tracked_content(ROOT)):
        return True
    print("NOT IMPLEMENTED: no Go source; application validation is not applicable yet.")
    return False


def lint():
    if not has_go():
        return
    files = [str(p.relative_to(ROOT)) for p in tracked_content(ROOT) if p.suffix == ".go"]
    result = subprocess.check_output(["gofmt", "-l", *files], cwd=ROOT, text=True)
    if result.strip():
        raise SystemExit("gofmt changes required:\n" + result)
    run("go", "mod", "tidy", "-diff")
    run("go", "mod", "verify")
    run("go", "vet", "./...")
    run("golangci-lint", "run")


def coverage_findings(output, minimum=MINIMUM_COVERAGE):
    match = re.search(r"^total:\s+\(statements\)\s+([\d.]+)%", output, re.M)
    if not match:
        return ["coverage total is missing"]
    actual = float(match[1])
    if actual < minimum:
        return [f"statement coverage {actual}% is below {minimum}%"]
    return []


def test():
    if not has_go():
        return
    build_dir = ROOT / ".build"
    build_dir.mkdir(exist_ok=True)
    profile = build_dir / "coverage.out"
    result = subprocess.run(
        ["go", "test", "-json", "-race", "-shuffle=on", "-count=1", "-timeout=5m",
         "-covermode=atomic", "-coverpkg=./...", f"-coverprofile={profile}", "./..."],
        cwd=ROOT, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
    )
    (build_dir / "tests.ndjson").write_text(result.stdout)
    print(result.stdout)
    result.check_returncode()
    events = []
    for line in result.stdout.splitlines():
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            pass
    if not any(e.get("Action") == "pass" and e.get("Test", "").startswith("Test") for e in events):
        raise SystemExit("Go source exists but no named Go test passed")
    output = subprocess.check_output(["go", "tool", "cover", "-func", str(profile)], cwd=ROOT, text=True)
    print(output)
    require_clean(coverage_findings(output))


def build():
    if not has_go():
        return
    for goos, goarch in (("linux", "amd64"), ("linux", "arm64"),
                         ("darwin", "amd64"), ("darwin", "arm64")):
        env = dict(os.environ, GOOS=goos, GOARCH=goarch, CGO_ENABLED="0")
        run("go", "build", "./...", env=env)


def fuzz():
    if not has_go():
        return
    stream = subprocess.check_output(["go", "list", "-json", "./..."], cwd=ROOT, text=True)
    decoder = json.JSONDecoder()
    targets = 0
    while stream.strip():
        package, end = decoder.raw_decode(stream.lstrip())
        stream = stream.lstrip()[end:]
        for name in package.get("TestGoFiles", []) + package.get("XTestGoFiles", []):
            content = (Path(package["Dir"]) / name).read_text()
            for target in re.findall(r"^func (Fuzz\w+)\(", content, re.M):
                targets += 1
                run("go", "test", package["ImportPath"], "-run=^$",
                    f"-fuzz=^{target}$", "-fuzztime=5s", "-timeout=1m")
    print(f"Fuzz smoke targets executed: {targets}. Parser/path work must introduce fuzz targets.")


def install_tools(names):
    tools = json.loads((ROOT / "scripts/tools.json").read_text())
    if not names:
        names = list(tools) + ["golangci-lint"]
    target = ROOT / ".tools/bin"
    target.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, GOBIN=str(target))
    for name in names:
        if name == "golangci-lint":
            module = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
            version = (ROOT / ".golangci-lint-version").read_text().strip()
        else:
            module, version = tools[name]["module"], tools[name]["version"]
        run("go", "install", f"{module}@{version}", env=env)


def main():
    os.environ["PATH"] = str(ROOT / ".tools/bin") + os.pathsep + os.environ["PATH"]
    command = sys.argv[1]
    if command == "install-tools":
        install_tools(sys.argv[2:])
    elif command == "static":
        run("actionlint")
        run("golangci-lint", "config", "verify")
        lint()
    elif command == "security":
        run("gitleaks", "git", "--redact", "--no-banner")
        if has_go():
            run("govulncheck", "./...")
    elif command == "gate":
        require_clean(gate_findings(json.loads(os.environ["NEEDS_JSON"]), os.environ["CI_EVENT_NAME"]))
        print("All required jobs reached their expected results.")
    elif command == "vuln":
        if has_go():
            run("govulncheck", "./...")
    else:
        {"repo": repository, "lint": lint, "test": test, "build": build, "fuzz": fuzz}[command]()


if __name__ == "__main__":
    main()

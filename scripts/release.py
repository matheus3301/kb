"""Release guardrails and source-formula publication; GoReleaser builds archives."""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess


ROOT = Path(__file__).resolve().parents[1]
REPOSITORY = "matheus3301/kb"
TAP = "matheus3301/homebrew-tap"
FORMULA_PATH = "Formula/kb.rb"
BOT = {"name": "github-actions[bot]", "email": "41898282+github-actions[bot]@users.noreply.github.com"}
SEMVER = re.compile(r"v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?")


def command(*args):
    return subprocess.check_output(args, cwd=ROOT, text=True)


def go_license(goroot):
    path = goroot / "LICENSE"
    # Homebrew moves the distribution license out of libexec into the keg root.
    if not path.is_file() and goroot.name == "libexec":
        path = goroot.parent / "LICENSE"
    if not path.is_file():
        raise ValueError("missing Go standard library license notice")
    return path.read_text()


def dependency_notices():
    environment = dict(os.environ, GOWORK="off", GOTOOLCHAIN="local")
    raw = subprocess.check_output(["go", "list", "-deps", "-json", "./cmd/kb"],
                                  cwd=ROOT, env=environment, text=True)
    decoder = json.JSONDecoder()
    modules = {}
    while raw.strip():
        package, end = decoder.raw_decode(raw.lstrip())
        raw = raw.lstrip()[end:]
        module = package.get("Module")
        if not module or module.get("Main"):
            continue
        if module.get("Replace"):
            raise SystemExit("release builds must not use replaced modules")
        modules[module["Path"]] = module
    goroot = Path(subprocess.check_output(["go", "env", "GOROOT"], env=environment, text=True).strip())
    sections = ["# Third-party notices\n\nProject code is MIT licensed. Dependency notices follow.\n",
                "## Go standard library\n\n```text\n" + go_license(goroot) + "\n```\n"]
    for name, module in sorted(modules.items()):
        directory = Path(module["Dir"])
        found = False
        for path in sorted(directory.iterdir()):
            if path.is_file() and path.name.upper().startswith(("LICENSE", "COPYING", "NOTICE")):
                found = True
                sections.append(f"## {name}@{module['Version']} — {path.name}\n\n```text\n" +
                                path.read_text(errors="replace") + "\n```\n")
        if not found:
            raise SystemExit(f"missing dependency license notice: {name}")
    return "\n".join(sections).encode()



def release_tag(value):
    match = SEMVER.fullmatch(value)
    if not match or any(not part or (part.isdigit() and len(part) > 1 and part[0] == "0")
                        for part in (match.group(4) or "prerelease").split(".")):
        raise ValueError("tag must be a v-prefixed semantic version")
    if value == "v0.1.0":
        raise ValueError("v0.1.0 was withdrawn after Go module publication; use a new version")
    return value


def require_ci(checks, commit):
    gates = [check for check in checks if check.get("name") == "CI gate"
             and check.get("app", {}).get("id") == 15368 and check.get("head_sha") == commit]
    latest = max(gates, key=lambda check: check["id"], default={})
    if latest.get("status") != "completed" or latest.get("conclusion") != "success":
        raise ValueError("the latest GitHub Actions CI gate must succeed on the exact tagged commit")


def verify_tag():
    tag = release_tag(os.environ["GITHUB_REF_NAME"])
    if os.environ.get("GITHUB_REF_TYPE") != "tag":
        raise ValueError("release publication requires a tag push")
    commit = command("git", "rev-parse", f"refs/tags/{tag}^{{commit}}").strip()
    if commit != os.environ["GITHUB_SHA"]:
        raise ValueError("tag does not resolve to the workflow source commit")
    subprocess.run(["git", "merge-base", "--is-ancestor", commit, "origin/main"], cwd=ROOT, check=True)
    pages = json.loads(command("gh", "api", "--paginate", "--slurp",
                               f"repos/{REPOSITORY}/commits/{commit}/check-runs?per_page=100"))
    require_ci([check for page in pages for check in page["check_runs"]], commit)
    print(f"Verified {tag}: protected-main ancestry and exact-commit CI gate.")


def formula_text(tag, commit, filename, checksum):
    release_tag(tag)
    if not re.fullmatch(r"[a-f0-9]{40}", commit):
        raise ValueError("commit must be a full Git SHA")
    if filename != f"kb_{tag[1:]}_source.tar.gz" or not re.fullmatch(r"[a-f0-9]{64}", checksum):
        raise ValueError("invalid source archive identity")
    return f'''# frozen_string_literal: true

# Install kb from its checksummed release source.
class Kb < Formula
  desc "Local Markdown Knowledge Base for humans and AI agents"
  homepage "https://github.com/{REPOSITORY}"
  url "https://github.com/{REPOSITORY}/releases/download/{tag}/{filename}"
  sha256 "{checksum}"
  license "MIT"

  depends_on "go" => :build

  def install
    ENV["CGO_ENABLED"] = "0"
    ENV["GOWORK"] = "off"
    ENV["GOTOOLCHAIN"] = "local"
    ldflags = "-s -w -X main.version=v#{{version}} -X main.commit={commit}"
    system "go", "build", *std_go_args(ldflags:), "./cmd/kb"
    pkgshare.install "docs", "skills", "assets", "schemas", "llms.txt", "THIRD_PARTY_NOTICES.md"
  end

  test do
    assert_match version.to_s, shell_output("#{{bin}}/kb version --json")
    system bin/"kb", "--root", testpath/"knowledge", "init", "--para", "--json"
    system bin/"kb", "--root", testpath/"knowledge", "validate", "--profile", "okf", "--json"
  end
end
'''


def prepare_formula(dist, output):
    dist = dist.resolve()
    metadata = json.loads((dist / "metadata.json").read_text())
    tag = release_tag("v" + metadata["version"])
    commit = metadata["commit"]
    filename = f"kb_{metadata['version']}_source.tar.gz"
    archive = dist / filename
    if archive.is_symlink() or not archive.is_file():
        raise ValueError("source archive is missing or is a symlink")
    checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
    listed = (dist / "checksums.txt").read_text().splitlines()
    if f"{checksum}  {filename}" not in listed:
        raise ValueError("source archive does not match GoReleaser checksums")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(formula_text(tag, commit, filename, checksum))
    print(f"Prepared {FORMULA_PATH} for {tag} from the checksummed source archive.")


def publish_tap(path):
    tag = release_tag(os.environ["GITHUB_REF_NAME"])
    if "-" in tag:
        raise ValueError("prereleases must not update the stable tap")
    if not os.environ.get("GH_TOKEN"):
        raise ValueError("GH_TOKEN must provide Contents: write access to the tap")
    text = path.read_text()
    # The download job must receive the formula produced for this exact tag.
    expected_url = f"https://github.com/{REPOSITORY}/releases/download/{tag}/kb_{tag[1:]}_source.tar.gz"
    checksum = re.search(r'^  sha256 "([a-f0-9]{64})"$', text, re.M)
    if (f'  url "{expected_url}"' not in text or not checksum or
            text != formula_text(tag, os.environ["GITHUB_SHA"], f"kb_{tag[1:]}_source.tar.gz", checksum[1])):
        raise ValueError("formula artifact does not match this release")
    command("gh", "api", f"repos/{TAP}")
    endpoint = f"repos/{TAP}/contents/{FORMULA_PATH}"
    lookup = subprocess.run(["gh", "api", endpoint], cwd=ROOT, text=True, capture_output=True)
    previous = None
    if lookup.returncode == 0:
        previous = json.loads(lookup.stdout)
        if base64.b64decode(previous["content"]).decode() == text:
            print("Tap already contains this exact formula; no duplicate commit.")
            return
        old = base64.b64decode(previous["content"]).decode()
        published = re.search(r"/releases/download/v(\d+)\.(\d+)\.(\d+)/", old)
        current = SEMVER.fullmatch(tag)
        if not published or tuple(map(int, published.groups())) > tuple(map(int, current.groups()[:3])):
            raise ValueError("refusing to overwrite an unknown formula or downgrade a newer release")
    elif "HTTP 404" not in lookup.stderr:
        raise ValueError("cannot read existing tap formula")
    payload = {"message": f"build(kb): update to {tag}", "content": base64.b64encode(text.encode()).decode(),
               "author": BOT, "committer": BOT}
    if previous:
        payload["sha"] = previous["sha"]
    subprocess.run(["gh", "api", "--method", "PUT", endpoint, "--input", "-"], cwd=ROOT,
                   input=json.dumps(payload), text=True, check=True, stdout=subprocess.DEVNULL)
    print(f"Published {FORMULA_PATH} to {TAP} for {tag}.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="operation", required=True)
    commands.add_parser("notices")
    commands.add_parser("verify-tag")
    formula = commands.add_parser("formula")
    formula.add_argument("--dist", type=Path, default=ROOT / ".build/dist")
    formula.add_argument("--output", type=Path, default=ROOT / ".build/homebrew/kb.rb")
    tap = commands.add_parser("publish-tap")
    tap.add_argument("--formula", type=Path, default=ROOT / ".build/homebrew/kb.rb")
    options = parser.parse_args()
    try:
        if options.operation == "notices":
            destination = ROOT / ".build/THIRD_PARTY_NOTICES.md"
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(dependency_notices())
        elif options.operation == "verify-tag":
            verify_tag()
        elif options.operation == "formula":
            prepare_formula(options.dist, options.output)
        else:
            publish_tap(options.formula)
    except (ValueError, KeyError) as error:
        parser.exit(1, f"release: {error}\n")


if __name__ == "__main__":
    main()

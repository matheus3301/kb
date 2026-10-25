"""Exercise actual GoReleaser artifacts without publishing or installing globally."""

import hashlib
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import tarfile
import tempfile

import acceptance
import release


def inspect_archive(path):
    with tarfile.open(path) as archive:
        for member in archive.getmembers():
            relative = Path(member.name)
            if (relative.is_absolute() or ".." in relative.parts or "{{" in member.name
                    or member.issym() or member.islnk()):
                raise ValueError("unsafe release archive member")
            if (member.uid, member.gid) != (0, 0) or member.uname not in {"", "root"} or member.gname not in {"", "root"}:
                raise ValueError("workstation identity in release archive metadata")
            if member.isfile() and relative.suffix in {".md", ".json", ".svg", ".yaml", ".yml", ".toml"}:
                data = archive.extractfile(member).read()
                if re.search(rb"/(?:Users|home)/[\w.-]+|(?:file|vscode|codex)://", data):
                    raise ValueError("workstation path in release content")
        return {member.name for member in archive.getmembers()}


def main():
    root = Path(__file__).resolve().parents[1]
    dist = root / ".build/dist"
    metadata = json.loads((dist / "metadata.json").read_text())
    version, commit = metadata["version"], metadata["commit"]
    expected = {f"kb_{version}_{system}_{arch}.tar.gz" for system in ("linux", "darwin")
                for arch in ("amd64", "arm64")}
    source_name = f"kb_{version}_source.tar.gz"
    expected.add(source_name)
    actual = {path.name for path in dist.glob("*.tar.gz")}
    if actual != expected:
        raise ValueError("release must contain exactly four binary archives and one source archive")
    listed = {}
    for line in (dist / "checksums.txt").read_text().splitlines():
        checksum, name = line.split("  ")
        if name in listed or name not in expected:
            raise ValueError("unexpected or duplicate checksum entry")
        listed[name] = checksum
    if set(listed) != expected:
        raise ValueError("missing release checksum")
    for name in expected:
        if hashlib.sha256((dist / name).read_bytes()).hexdigest() != listed[name]:
            raise ValueError("release checksum mismatch")
        members = inspect_archive(dist / name)
        if name != source_name:
            required = {"kb", "LICENSE", "THIRD_PARTY_NOTICES.md", "README.md", "docs/quickstart.md",
                        "assets/logo.svg", "skills/knowledge-memory/SKILL.md", "schemas/kb.v1.tools.json"}
            if not required <= members:
                raise ValueError("binary archive is missing public documentation or licenses")
        else:
            required = {f"kb-{version}/{entry}" for entry in ("go.mod", "cmd/kb/main.go", "LICENSE",
                        "THIRD_PARTY_NOTICES.md", "docs/quickstart.md", "assets/logo.svg",
                        "skills/knowledge-memory/SKILL.md", "schemas/kb.v1.tools.json")}
            if not required <= members:
                raise ValueError("source archive is missing source, docs, or license notices")
    expected_formula = release.formula_text("v" + version, commit, source_name, listed[source_name])
    if (root / ".build/homebrew/kb.rb").read_text() != expected_formula:
        raise ValueError("Homebrew formula does not match the checked source archive")
    system = "darwin" if platform.system() == "Darwin" else "linux"
    arch = {"arm64": "arm64", "aarch64": "arm64", "x86_64": "amd64"}[platform.machine()]
    with tempfile.TemporaryDirectory(prefix="kb-release-check-") as directory:
        workspace = Path(directory)
        native = workspace / "native"
        with tarfile.open(dist / f"kb_{version}_{system}_{arch}.tar.gz") as archive:
            archive.extractall(native, filter="data")
        binary = native / "kb"
        info = json.loads(subprocess.check_output([str(binary), "version", "--json"]))["data"]
        if info != {"version": "v" + version, "commit": commit}:
            raise ValueError("release binary source identity is incorrect")
        acceptance.exercise(binary)
        source = workspace / "source"
        with tarfile.open(dist / source_name) as archive:
            archive.extractall(source, filter="data")
        env = dict(os.environ, CGO_ENABLED="0", GOWORK="off", GOTOOLCHAIN="local")
        installed = workspace / "source-kb"
        subprocess.run(["go", "build", "-trimpath", "-ldflags",
                        f"-s -w -X main.version=v{version} -X main.commit={commit}",
                        "-o", str(installed), "./cmd/kb"],
                       cwd=source / f"kb-{version}", env=env, check=True)
        acceptance.exercise(installed)
    print("Release snapshot passed: four platforms, source archive, checksums, privacy, docs/licenses, "
          "formula identity, extracted binary workflow, and source build workflow.")


if __name__ == "__main__":
    main()

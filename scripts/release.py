"""Build CGO-free release archives without leaking workstation metadata."""

import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile


ROOT = Path(__file__).resolve().parents[1]
TARGETS = (("linux", "amd64"), ("linux", "arm64"), ("darwin", "amd64"), ("darwin", "arm64"))


def version_value(value):
    if not re.fullmatch(r"v\d+\.\d+\.\d+(?:-[a-z0-9.-]+)?", value):
        raise argparse.ArgumentTypeError("version must be a v-prefixed semantic version")
    return value


def write_archive(path, files, epoch):
    with path.open("wb") as output:
        with gzip.GzipFile(filename="", mode="wb", fileobj=output, mtime=epoch) as compressed:
            with tarfile.open(fileobj=compressed, mode="w") as archive:
                for name, data in sorted(files.items()):
                    relative = Path(name)
                    if relative.is_absolute() or ".." in relative.parts:
                        raise ValueError("archive paths must remain relative")
                    info = tarfile.TarInfo(name)
                    info.size = len(data)
                    info.mode = 0o755 if name == "kb" else 0o644
                    info.mtime = epoch
                    archive.addfile(info, io.BytesIO(data))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True, type=version_value)
    parser.add_argument("--output", type=Path, default=ROOT / ".build/release")
    options = parser.parse_args()
    dirty = subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True)
    if dirty.strip():
        raise SystemExit("release builds require a clean committed source tree")
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    epoch = int(subprocess.check_output(["git", "show", "-s", "--format=%ct", "HEAD"], cwd=ROOT, text=True))
    output = options.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    documents = {"LICENSE": (ROOT / "LICENSE").read_bytes(), "README.md": (ROOT / "README.md").read_bytes(),
                 "CONTRIBUTING.md": (ROOT / "CONTRIBUTING.md").read_bytes()}
    for parent in ("docs", "skills"):
        for path in (ROOT / parent).rglob("*"):
            if path.is_file() and path.suffix in {".md", ".json"}:
                if path.is_symlink() or not path.resolve().is_relative_to(ROOT):
                    raise SystemExit("release documentation must not contain symlinks or external paths")
                documents[str(path.relative_to(ROOT))] = path.read_bytes()
    assets = []
    for goos, goarch in TARGETS:
        binary = output / f"kb-{goos}-{goarch}"
        env = dict(os.environ, GOOS=goos, GOARCH=goarch, CGO_ENABLED="0", GOTOOLCHAIN="local")
        subprocess.run(
            ["go", "build", "-trimpath", "-ldflags",
             f"-s -w -X main.version={options.version} -X main.commit={commit}",
             "-o", str(binary), "./cmd/kb"], cwd=ROOT, env=env, check=True,
        )
        name = f"kb_{options.version}_{goos}_{goarch}.tar.gz"
        target = output / name
        write_archive(target, dict(documents, kb=binary.read_bytes()), epoch)
        binary.unlink()
        assets.append({"name": name, "sha256": hashlib.sha256(target.read_bytes()).hexdigest(),
                       "bytes": target.stat().st_size, "os": goos, "arch": goarch})
    manifest = {"version": options.version, "commit": commit,
                "go": subprocess.check_output(["go", "version"], text=True).strip(), "assets": assets}
    manifest_path = output / "release.json"
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n")
    hashes = [f"{asset['sha256']}  {asset['name']}" for asset in assets]
    hashes.append(f"{hashlib.sha256(manifest_path.read_bytes()).hexdigest()}  release.json")
    (output / "checksums.txt").write_text("\n".join(hashes) + "\n")
    print(json.dumps(manifest, indent=2))


if __name__ == "__main__":
    main()

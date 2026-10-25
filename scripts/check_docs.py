"""Run the public documentation's JSON requests against a disposable real CLI."""

import argparse
import copy
import json
from pathlib import Path
import re
import shlex
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[1]


def json_examples(path):
    return [json.loads(block) for block in re.findall(r"^```json\n(.*?)^```", path.read_text(), re.M | re.S)]


def fill_revisions(value, revision):
    if isinstance(value, dict):
        return {key: fill_revisions(item, revision) for key, item in value.items()}
    if isinstance(value, list):
        return [fill_revisions(item, revision) for item in value]
    if value in ("REVISION_FROM_READ", "REVISION_FROM_DEPENDENCY_READ"):
        return revision
    return value


def exercise(binary, docs=ROOT):
    binary = Path(binary).resolve()
    docs = Path(docs)
    count = 0
    with tempfile.TemporaryDirectory(prefix="kb-docs-") as directory:
        workspace = Path(directory)

        def run(root, *args, request=None, exit_code=0):
            nonlocal count
            command = [str(binary), "--root", str(root), *args]
            if request is not None:
                command += ["--input", "-"]
            result = subprocess.run(command, input=json.dumps(request) if request is not None else None,
                                    text=True, capture_output=True, check=False)
            if result.returncode != exit_code:
                raise AssertionError(f"{args}: expected exit {exit_code}, got {result.returncode}: {result.stderr}")
            envelope = json.loads(result.stdout)
            if envelope.get("schema_version") != "kb.v1" or envelope["ok"] != (exit_code == 0):
                raise AssertionError(f"{args}: unexpected response envelope")
            count += 1
            return envelope.get("data", envelope.get("error"))

        root = workspace / "readme"
        run(root, "init", "--para")
        readme = (docs / "README.md").read_text()
        request = json.loads(re.search(r"kb create --input - <<'JSON'\n(.*?)\nJSON", readme, re.S)[1])
        run(root, "create", request=request)
        for block in re.findall(r"^```sh\n(.*?)^```", readme, re.M | re.S):
            for line in block.splitlines():
                args = shlex.split(line)
                if args[:2] in (["kb", "search"], ["kb", "read"], ["kb", "context"]):
                    result = run(root, *args[1:])
                    if args[1] == "search" and not any(hit["path"] == request["path"] for hit in result["hits"]):
                        raise AssertionError("README search did not find its created note")
                    if args[1] == "context" and not result["sources"]:
                        raise AssertionError("README context has no citable evidence")

        for page in ("docs/cli.md", "skills/knowledge-base/references/writing.md"):
            root = workspace / Path(page).stem
            run(root, "init", "--para")
            examples = json_examples(docs / page)
            create = next(item for item in examples if "type" in item)
            patch = next(item for item in examples if "body_edits" in item)
            move = next(item for item in examples if "to" in item)
            before = {p.relative_to(root): p.read_bytes() for p in root.rglob("*.md")}
            preview = run(root, "create", "--dry-run", request=create)
            if not preview["dry_run"] or before != {p.relative_to(root): p.read_bytes() for p in root.rglob("*.md")}:
                raise AssertionError("Documented preview published Markdown")
            run(root, "create", request=create)
            replay = run(root, "create", request=create)
            if not replay["replayed"]:
                raise AssertionError("Documented create replay was not idempotent")
            revision = run(root, "read", create["path"])["note"]["revision"]
            patch = fill_revisions(patch, revision)
            run(root, "patch", "--dry-run", request=patch)
            run(root, "patch", request=patch)
            current = run(root, "read", create["path"])
            if not all(edit["new_text"] in current["markdown"] for edit in patch["body_edits"]):
                raise AssertionError("Documented patch did not change the intended text")
            stale = copy.deepcopy(patch)
            stale["idempotency_key"] += "/stale"
            error = run(root, "patch", request=stale, exit_code=4)
            if error["code"] != "REVISION_CONFLICT":
                raise AssertionError("A stale documented patch did not reject the revision")
            move = fill_revisions(move, current["note"]["revision"])
            run(root, "move", "--dry-run", request=move)
            run(root, "move", request=move)
            run(root, "read", move["to"])
            if (root / move["path"]).exists():
                raise AssertionError("Documented move left its source behind")

            if page.startswith("skills/"):
                plan = next(item for item in examples if "operations" in item)
                run(root, "apply", "--dry-run", request=plan)
                run(root, "apply", request=plan)
                provenance = next(item for item in examples if "basis_revisions" in item)
                source = {"path": "sources/policy.md", "type": "Source", "body": "# Evidence\n\nSynthetic policy.\n",
                          "summary": "Add synthetic evidence", "actor": "process:docs-check",
                          "idempotency_key": "docs-check/source"}
                run(root, "create", request=source)
                source_revision = run(root, "read", source["path"])["note"]["revision"]
                synthesis = {"path": "03_Resources/synthesis.md", "type": "Synthesis", "body": "# Synthesis\n",
                             "metadata": fill_revisions(provenance, source_revision),
                             "summary": "Link synthetic evidence", "actor": "process:docs-check",
                             "idempotency_key": "docs-check/synthesis"}
                run(root, "create", request=synthesis)
                archived = run(root, "read", move["to"])
                archive = {"path": move["to"], "expected_revision": archived["note"]["revision"],
                           "summary": "Archive the example note", "actor": "process:docs-check",
                           "idempotency_key": "docs-check/archive"}
                run(root, "archive", "--dry-run", request=archive)
                run(root, "archive", request=archive)

            report = run(root, "validate", "--profile", "health")
            if not report["conformant"]:
                raise AssertionError("Documented workflow did not produce a structurally healthy base")
            before = {p.relative_to(root): p.read_bytes() for p in root.rglob("*.md")}
            run(root, "search", "retention")
            run(root, "context", "retention", "--max-tokens", "1000")
            if before != {p.relative_to(root): p.read_bytes() for p in root.rglob("*.md")}:
                raise AssertionError("Answer-only retrieval changed Markdown or the log")
    print(f"Documentation workflows passed: {count} real CLI requests from README, CLI guide, and skill reference examples.")
    return count


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    args = parser.parse_args()
    exercise(args.binary)


if __name__ == "__main__":
    main()

"""Run the public CLI workflow against synthetic, disposable knowledge only."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile


def revision(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def exercise(binary):
    checks = []
    with tempfile.TemporaryDirectory(prefix="kb-acceptance-") as directory:
        workspace = Path(directory).resolve()
        root = workspace / "knowledge"
        home = workspace / "home"
        home.mkdir()
        sources = workspace / "sources"
        sources.mkdir()
        original = sources / "original.txt"
        original.write_text("Synthetic evidence: the retention period is 600 seconds.\n")
        original_hash = revision(original.read_bytes())
        environment = {"PATH": "", "HOME": str(home), "TMPDIR": str(workspace), "LANG": "C.UTF-8"}

        def invoke(*args, payload=None, expected=0):
            command = [str(binary), "--root", str(root), "--json", *args]
            result = subprocess.run(command, input=None if payload is None else json.dumps(payload),
                                    text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                    env=environment, timeout=30)
            if expected is None:
                assert result.returncode != 0, "invalid operation unexpectedly succeeded"
            else:
                assert result.returncode == expected, f"{args[0]} failed: {result.stderr} {result.stdout}"
            value = json.loads(result.stdout)
            assert value["schema_version"] == "kb.v1", "missing versioned envelope"
            assert value["ok"] == (result.returncode == 0), "inconsistent execution envelope"
            return value

        def write(kind, path=None, **fields):
            request = dict(fields, summary="Synthetic acceptance workflow", actor="process:acceptance")
            request.setdefault("idempotency_key", f"acceptance/{len(checks)}/{kind}/{path or 'plan'}")
            if kind == "create":
                request["path"] = path
                return invoke(kind, "--input", "-", payload=request)
            return invoke(kind, path, "--input", "-", payload=request)

        def read(path):
            return invoke("read", path)["data"]

        invoke("init", "--para")
        initial = {str(p.relative_to(root)): p.read_bytes() for p in root.rglob("*.md")}
        invoke("init", "--para")
        assert all((root / name).read_bytes() == data for name, data in initial.items())
        assert not (root / "AGENTS.md").exists()
        assert not (root / ".git").exists()
        assert not (home / ".agents").exists()
        checks.append({"name": "isolated-idempotent-init", "criteria": ["A01", "A19", "A26"]})

        path = "03_Resources/retention.md"
        created_request = {"path": path, "type": "Reference", "title": "Retention decision",
                           "body": "# Retention\n\nThe synthetic retention period is 300 seconds.\n",
                           "metadata": {"status": "stable", "tags": ["storage"], "x-counter": 9007199254740993},
                           "idempotency_key": "acceptance/create-retention", "actor": "process:acceptance",
                           "summary": "Record the synthetic retention decision"}
        first = invoke("create", "--input", "-", payload=created_request)
        before_log = (root / "log.md").read_bytes()
        replay = invoke("create", "--input", "-", payload=created_request)
        assert replay["data"]["replayed"] is True
        assert (root / "log.md").read_bytes() == before_log
        assert first["data"]["changes"]
        # Simulate a public editor change; no private collection is involved.
        (root / path).write_text((root / path).read_text().replace("type: Reference", "type: Reference # synthetic annotation"))
        before = read(path)
        assert before["note"]["metadata"]["x-counter"] == 9007199254740993
        expected_revision = before["note"]["revision"]
        assert expected_revision == revision((root / path).read_bytes())
        checks.append({"name": "create-read-and-durable-retry", "criteria": ["A03", "A11", "A12"]})

        patch = {"expected_revision": expected_revision, "idempotency_key": "acceptance/correct-retention",
                 "summary": "Apply the supplied synthetic evidence", "actor": "process:acceptance",
                 "body_edits": [{"old_text": "300 seconds", "new_text": "600 seconds"}]}
        preview = invoke("patch", path, "--input", "-", "--dry-run", payload=patch)
        assert preview["data"]["dry_run"] is True
        assert revision((root / path).read_bytes()) == expected_revision
        invoke("patch", path, "--input", "-", payload=patch)
        updated = read(path)
        assert "600 seconds" in updated["markdown"]
        assert "# synthetic annotation" in (root / path).read_text()
        assert updated["note"]["revision"] != expected_revision
        conflict = dict(patch, idempotency_key="acceptance/conflict")
        rejected = invoke("patch", path, "--input", "-", payload=conflict, expected=None)
        assert rejected["error"]["code"] == "REVISION_CONFLICT"
        checks.append({"name": "preview-cas-and-body-edit", "criteria": ["A09", "A10"]})

        unchanged = (root / path).read_bytes()
        unchanged_log = (root / "log.md").read_bytes()
        write("patch", path, expected_revision=updated["note"]["revision"],
              metadata_patch={"status": "stable"}, idempotency_key="acceptance/noop")
        assert (root / path).read_bytes() == unchanged
        assert (root / "log.md").read_bytes() == unchanged_log
        checks.append({"name": "noop-preserves-file-bytes-and-history", "criteria": ["A09"]})

        search = invoke("search", "retention")["data"]
        assert any(hit["path"] == path for hit in search["hits"])
        context = invoke("context", "retention", "--max-tokens", "64")["data"]
        assert context["estimated_tokens"] <= 64
        assert len(context["text"].encode()) <= 256 * 1024
        assert context["sources"] and all(source["revision"] for source in context["sources"])
        assert (root / "log.md").read_bytes() != before_log
        query_log = (root / "log.md").read_bytes()
        invoke("search", "retention")
        invoke("context", "retention")
        assert (root / "log.md").read_bytes() == query_log
        checks.append({"name": "bounded-citable-context-and-pure-queries", "criteria": ["A16", "A25"]})

        reference = "03_Resources/reference.md"
        write("create", reference, type="Reference", title="Related retention evidence",
              body="# Related evidence\n\nSee [retention](retention.md#retention).\n",
              idempotency_key="acceptance/create-reference")
        note_before = (root / path).read_bytes()
        reference_before = (root / reference).read_bytes()
        invalid_plan = {"idempotency_key": "acceptance/reject-batch", "summary": "Synthetic conflict check",
                        "actor": "process:acceptance", "operations": [
                            {"operation_id": "one", "kind": "patch", "path": path,
                             "expected_revision": read(path)["note"]["revision"], "body": "Changed\n"},
                            {"operation_id": "two", "kind": "patch", "path": reference,
                             "expected_revision": "sha256:" + "0" * 64, "body": "Changed\n"}]}
        invoke("apply", "--input", "-", payload=invalid_plan, expected=None)
        assert (root / path).read_bytes() == note_before
        assert (root / reference).read_bytes() == reference_before
        checks.append({"name": "batch-conflict-publishes-nothing", "criteria": ["A21"]})

        source = "03_Resources/source.md"
        valid_plan = {"idempotency_key": "acceptance/integrate-source", "summary": "Integrate synthetic evidence across concepts",
                      "actor": "process:acceptance", "operations": [
                          {"operation_id": "source", "kind": "create", "path": source, "type": "Source",
                           "body": "# Synthetic source\n\nThe supplied evidence establishes the retention period.\n",
                           "metadata": {"sources": [{"resource": "https://example.org/synthetic-policy"}]}},
                          {"operation_id": "decision", "kind": "patch", "path": path,
                           "expected_revision": read(path)["note"]["revision"],
                           "body_edits": [{"old_text": "600 seconds.", "new_text": "600 seconds; see [source](source.md)."}],
                           "metadata_patch": {"sources": [{"resource": "source.md"}]}},
                          {"operation_id": "reference", "kind": "patch", "path": reference,
                           "expected_revision": read(reference)["note"]["revision"],
                           "body": "# Related evidence\n\nSee [retention](retention.md#retention) and [source](source.md).\n",
                           "metadata_patch": {"sources": [{"resource": "source.md"}]}}]}
        event_count = sum(line.startswith("- ") for line in (root / "log.md").read_text().splitlines())
        invoke("apply", "--input", "-", payload=valid_plan)
        assert (root / source).is_file()
        assert "source.md" in (root / path).read_text() and "source.md" in (root / reference).read_text()
        assert sum(line.startswith("- ") for line in (root / "log.md").read_text().splitlines()) == event_count + 1
        checks.append({"name": "one-source-integrates-two-existing-concepts", "criteria": ["A21", "A25"]})

        synthesis = "03_Resources/synthesis.md"
        write("create", synthesis, type="Synthesis", title="Synthetic retention synthesis",
              body="# Synthesis\n\nA reusable synthetic conclusion.\n",
              metadata={"basis_revisions": [{"path": path, "revision": read(path)["note"]["revision"]},
                                             {"path": source, "revision": read(source)["note"]["revision"]}]},
              idempotency_key="acceptance/create-synthesis")
        assert read(synthesis)["note"]["basis_outdated"] is False
        write("patch", path, expected_revision=read(path)["note"]["revision"],
              metadata_patch={"status": "draft"}, idempotency_key="acceptance/change-dependency")
        assert read(synthesis)["note"]["basis_outdated"] is True
        checks.append({"name": "changed-basis-signals-review-without-rewriting-synthesis", "criteria": ["A23", "A24"]})

        move = {"expected_revision": read(path)["note"]["revision"], "summary": "Archive synthetic decision",
                "actor": "process:acceptance", "idempotency_key": "acceptance/archive-retention"}
        destination = "04_Archive/retention.md"
        invoke("move", path, destination, "--input", "-", payload=move)
        assert not (root / path).exists() and (root / destination).exists()
        assert "../04_Archive/retention.md#retention" in (root / reference).read_text()
        assert destination in (root / synthesis).read_text()
        checks.append({"name": "move-maintains-incoming-link-and-anchor", "criteria": ["A13"]})

        for unsafe in ("../sources/original.txt", str(original)):
            invoke("read", unsafe, "--raw", expected=None)
        link = root / "03_Resources/escape.md"
        link.symlink_to(original)
        try:
            invoke("read", "03_Resources/escape.md", "--raw", expected=None)
        finally:
            link.unlink()
        checks.append({"name": "traversal-absolute-and-symlink-denied", "criteria": ["A14"]})

        invalid = root / "03_Resources/invalid.md"
        invalid.write_text("# Invalid concept\n\nThis synthetic note has no type.\n")
        report = invoke("status")["data"]
        assert report["degraded"] is True and report["invalid"] >= 1
        raw = invoke("read", "03_Resources/invalid.md", "--raw")["data"]
        assert "Invalid concept" in raw["markdown"]
        blocked = dict(created_request, path="03_Resources/blocked.md", idempotency_key="acceptance/blocked")
        invoke("create", "--input", "-", payload=blocked, expected=None)
        assert not (root / "03_Resources/blocked.md").exists()
        repair = {"idempotency_key": "acceptance/repair", "summary": "Repair explicit synthetic document",
                  "actor": "process:acceptance", "repair": True, "operations": [
                      {"operation_id": "repair-one", "kind": "patch", "path": "03_Resources/invalid.md",
                       "expected_revision": revision(invalid.read_bytes()),
                       "document": "---\ntype: Reference\n---\n# Repaired concept\n"}]}
        invoke("apply", "--input", "-", payload=repair)
        validation = invoke("validate", "--profile", "okf")["data"]
        assert validation["conformant"] is True
        checks.append({"name": "degraded-base-raw-diagnostics-and-explicit-repair", "criteria": ["A04", "A05"]})

        export = workspace / "export"
        result = invoke("export", str(export))["data"]
        assert result["conformant"] is True
        assert (export / destination).is_file()
        assert not (export / ".kb").exists()
        assert revision(original.read_bytes()) == original_hash
        checks.append({"name": "validated-export-and-original-source-immutability", "criteria": ["A18", "A22"]})
        return {"schema_version": "kb.acceptance.v1", "passed": True, "synthetic_only": True,
                "runtime_path_empty": True, "checks": checks}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--output", type=Path)
    options = parser.parse_args()
    result = exercise(options.binary.resolve())
    text = json.dumps(result, indent=2) + "\n"
    if options.output:
        options.output.parent.mkdir(parents=True, exist_ok=True)
        options.output.write_text(text)
    print(text, end="")


if __name__ == "__main__":
    main()

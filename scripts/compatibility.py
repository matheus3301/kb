#!/usr/bin/env python3
"""Qualify a pinned prior version -> candidate through real CLI/MCP processes; synthetic roots only."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import signal
import stat
import subprocess
import tempfile
import time

from acceptance import revision
from benchmark import MCP

BASELINE_COMMIT = "2ca104f48cc4a6551f4bbc4fae12d51432e42dff"
READ_TOOLS = {"kb_status", "kb_list", "kb_search", "kb_read", "kb_context", "kb_validate"}
WRITE_TOOLS = {"kb_create", "kb_patch", "kb_move", "kb_apply"}
OPERATIONAL_TOOLS = {"kb_backup", "kb_restore"}


class CompatibilityFailure(Exception):
    """A public compatibility invariant was not observed."""


def require(condition, message):
    if not condition:
        raise CompatibilityFailure(message)


def envelope(value):
    """Check the published closed kb.v1 envelope, including failure/report distinction."""
    require(isinstance(value, dict), "envelope must be an object")
    require(set(value) in ({"schema_version", "ok", "data", "warnings", "meta"},
                           {"schema_version", "ok", "error", "warnings", "meta"}), "envelope fields")
    require(value["schema_version"] == "kb.v1" and type(value["ok"]) is bool, "wire version or ok type")
    require(("data" in value) == value["ok"], "success/error union")
    meta = value["meta"]
    require(isinstance(meta, dict) and set(meta) == {"root", "generation", "index_stale"}, "meta fields")
    require(isinstance(meta["root"], str) and isinstance(meta["generation"], str)
            and type(meta["index_stale"]) is bool, "meta types")
    require(isinstance(value["warnings"], list), "warnings type")
    warning_types = {"code": str, "message": str, "layer": str, "path": str, "line": int,
                     "rule": str, "field": str, "hint": str, "revision": str, "severity": str}
    for warning in value["warnings"]:
        require(isinstance(warning, dict) and {"code", "message", "layer"} <= warning.keys()
                and warning.keys() <= warning_types.keys(), "warning fields")
        require(all(type(v) is warning_types[k] for k, v in warning.items()), "warning types")
        require("line" not in warning or warning["line"] >= 1, "warning line")
        require("severity" not in warning or warning["severity"] in {"error", "warning"}, "warning severity")
    if not value["ok"]:
        error = value["error"]
        require(isinstance(error, dict) and {"code", "message", "retryable"} <= error.keys()
                and error.keys() <= {"code", "message", "retryable", "details"}, "error fields")
        require(isinstance(error["code"], str) and bool(error["code"])
                and isinstance(error["message"], str) and type(error["retryable"]) is bool, "error types")
    return value


def authoritative(root):
    """Exact content/config/receipt snapshot; intentionally exclude disposable operational outputs."""
    return {p.relative_to(root).as_posix(): p.read_bytes() for p in root.rglob("*") if p.is_file()
            and (".kb" not in p.relative_to(root).parts
                 or p.relative_to(root).as_posix() == ".kb/config.toml"
                 or p.relative_to(root).as_posix().startswith(".kb/state/receipts/"))}


def storage_snapshot(root):
    """Capture all synthetic paths, modes, links and bytes, including operational outputs."""
    result = {}
    for path in root.rglob("*"):
        mode = path.lstat().st_mode
        data = path.read_bytes() if stat.S_ISREG(mode) else (os.readlink(path) if stat.S_ISLNK(mode) else None)
        result[path.relative_to(root).as_posix()] = (mode, data)
    return result


class CLI:
    def __init__(self, binary, root, environment):
        self.binary, self.root, self.environment = str(binary), root, environment

    def run(self, *args, payload=None, raw=None, exit_code=0, code=None):
        command = [self.binary, "--root", str(self.root), "--json", *args]
        if payload is not None or raw is not None:
            command += ["--input", "-"]
        result = subprocess.run(command, input=json.dumps(payload) if payload is not None else raw,
                                text=True, capture_output=True, env=self.environment, timeout=30)
        require(result.returncode == exit_code, "CLI exit code: " + args[0])
        value = envelope(json.loads(result.stdout))
        require(value["ok"] == (code is None), "CLI success/error: " + args[0])
        if code is not None:
            require(value["error"]["code"] == code, "CLI domain code: " + args[0])
        return value


class ConsumerMCP(MCP):
    """Reuse the bounded JSON-RPC helper, but isolate the environment and retain discovery guidance."""
    def __init__(self, cli, read_only=False):
        args = [cli.binary, "--root", str(cli.root)]
        if read_only:
            args += ["--read-only"]
        self.process = subprocess.Popen(args + ["mcp"], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        stderr=subprocess.PIPE, env=cli.environment)
        self.sequence = 0
        try:
            self.initialization = self.rpc("initialize", {"protocolVersion": "2025-11-25", "capabilities": {},
                                           "clientInfo": {"name": "kb-compatibility", "version": "1"}})
            self.process.stdin.write(b'{"jsonrpc":"2.0","method":"notifications/initialized"}\n')
            self.process.stdin.flush()
        except BaseException:
            self.close()
            raise

    def result(self, name, request, code=None):
        result = self.rpc("tools/call", {"name": name, "arguments": request})
        value = envelope(result["structuredContent"])
        require(value["ok"] == (code is None), "MCP success/error: " + name)
        require(bool(result.get("isError", False)) == (code is not None), "MCP isError: " + name)
        if code is not None:
            require(value["error"]["code"] == code, "MCP domain code: " + name)
        return value

    def close(self):
        if self.process.stdin and not self.process.stdin.closed:
            self.process.stdin.close()
        try:
            self.process.wait(timeout=10)
            require(self.process.stdout.read() == b"", "MCP emitted unexpected trailing stdout")
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait(timeout=5)
            raise CompatibilityFailure("MCP shutdown timed out") from None
        finally:
            self.process.stdout.close()
            self.process.stderr.close()
        require(self.process.returncode == 0, "MCP process exit")


def check_catalog(names, required, allowed=None):
    actual = set(names)
    allowed = required if allowed is None else allowed
    require(required <= actual <= allowed and len(actual) == len(names), "MCP tool catalog")
    require(not actual & OPERATIONAL_TOOLS or OPERATIONAL_TOOLS <= actual, "MCP operational tool pair")


def discover(client, expected, allowed=None):
    initialization = client.initialization
    require(initialization.get("protocolVersion") == "2025-11-25", "MCP negotiation")
    require(bool(initialization.get("instructions")), "MCP independent initialization guidance")
    tools, cursors, params = [], set(), {}
    for _ in range(10):
        page = client.rpc("tools/list", params)
        tools.extend(page.get("tools", []))
        cursor = page.get("nextCursor")
        if not cursor:
            break
        require(cursor not in cursors, "MCP discovery cursor loop")
        cursors.add(cursor)
        params = {"cursor": cursor}
    else:
        raise CompatibilityFailure("MCP discovery page limit")
    check_catalog([t["name"] for t in tools], expected, allowed)
    for tool in tools:
        require(bool(tool.get("description")), "MCP tool description")
        schema = tool["inputSchema"]
        require(schema.get("type") == "object" and schema.get("additionalProperties") is False,
                "MCP closed input schema")
        require(tool.get("outputSchema", {}).get("type") == "object", "MCP output schema")
        require(all(bool(field.get("description")) for field in schema.get("properties", {}).values()),
                "MCP independent field guidance")
    return {t["name"]: t for t in tools}


def request(key, **fields):
    return dict(fields, actor="process:compatibility", summary="Qualify synthetic version compatibility",
                idempotency_key="compatibility/" + key)


def baseline_expectation(version, commit):
    require((version is None) == (commit is None), "baseline version and commit must be supplied together")
    if version is None:
        return "v0.3.1", BASELINE_COMMIT
    require(re.fullmatch(r"v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?", version) is not None,
            "baseline version must be an exact release version")
    require(re.fullmatch(r"[0-9a-f]{40}", commit) is not None, "baseline commit must be full 40-hex source identity")
    return version, commit


def check_identity(data, version, commit, label):
    require(isinstance(data, dict) and set(data) == {"version", "commit"}, "binary identity fields")
    require(isinstance(data["commit"], str) and re.fullmatch(r"[0-9a-f]{40}", data["commit"]) is not None,
            "exact source commit required")
    require(isinstance(data["version"], str) and data["version"] not in {"", "dev", "unknown"}, "identified binary required")
    require((version is None or data["version"] == version) and (commit is None or data["commit"] == commit),
            label + " source identity")


def identity(binary, version, commit, label):
    require(version is None or version not in {"", "dev", "unknown"}, "identified candidate version required")
    result = subprocess.run([str(binary), "version", "--json"], capture_output=True, text=True, timeout=10)
    require(result.returncode == 0, "version exit")
    data = envelope(json.loads(result.stdout))["data"]
    check_identity(data, version, commit, label)
    return {**data, "sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
            "binary_path": "$" + label.upper() + "/kb",
            "source": "https://github.com/matheus3301/kb/tree/" + data["commit"]}


def replay(cli, args, payload):
    before = authoritative(cli.root)
    require(cli.run(*args, payload=payload)["data"].get("replayed") is True, "durable receipt replay")
    require(authoritative(cli.root) == before, "replay changed authoritative bytes")


def common_contract(cli):
    before = authoritative(cli.root)
    note = cli.run("read", "fidelity.md")["data"]
    require(note["note"]["revision"] == revision((cli.root / "fidelity.md").read_bytes()), "read exact revision")
    require(note["note"]["type"] == "SyntheticExtension", "unknown type fidelity")
    require(note["note"]["metadata"]["x-counter"] == 9007199254740993, "integer fidelity")
    listed = cli.run("list", payload={"limit": 10})["data"]
    require(any(n["path"] == "fidelity.md" for n in listed["hits"]), "JSON list request")
    hits = cli.run("search", payload={"query": "quartzcompat", "limit": 5})["data"]["hits"]
    require([h["path"] for h in hits] == ["fidelity.md"], "JSON search request")
    context = cli.run("context", payload={"query": "quartzcompat", "max_tokens": 256})["data"]
    require(context["sources"][0]["revision"] == note["note"]["revision"]
            and "quartzcompat" in context["text"], "context citation fidelity")
    cli.run("read", "absent.md", exit_code=3, code="NOT_FOUND")
    cli.run("read", "../outside.md", exit_code=2, code="INVALID_PATH")
    cli.run("search", raw='{"query":"quartzcompat","unexpected":true}', exit_code=2, code="INVALID_INPUT")
    cli.run("search", raw='{"query":"quartzcompat"} {}', exit_code=2, code="INVALID_INPUT")
    cli.run("search", "absenttoken", payload={"query": "quartzcompat"}, exit_code=2, code="INVALID_INPUT")
    require(cli.run("search", "--limit", "1", payload={"query": "quartzcompat", "limit": 101})["data"]["total"] == 1,
            "explicit CLI flag precedence")
    cli.run("patch", "fidelity.md", payload=request("conflict", expected_revision="sha256:" + "0" * 64,
            body="Must not publish\n"), exit_code=4, code="REVISION_CONFLICT")
    cli.run("--read-only", "create", payload=request("denied", path="denied.md", type="Reference", body="No\n"),
            exit_code=5, code="READ_ONLY")
    require(authoritative(cli.root) == before, "queries or rejected requests changed authoritative state")
    invalid = cli.root / "invalid.md"
    invalid.write_bytes(b"# Synthetic invalid note\n")
    try:
        result = cli.run("validate", "--profile", "okf", exit_code=1)
        require(result["data"]["conformant"] is False, "validation report exit 1 remains successful envelope")
    finally:
        invalid.unlink()
    return note


def mcp_workflow(cli, candidate, baseline_additions=False):
    before = authoritative(cli.root)
    client = ConsumerMCP(cli)
    try:
        common = READ_TOOLS | WRITE_TOOLS
        tools = discover(client, common | OPERATIONAL_TOOLS if candidate else common,
                         common | OPERATIONAL_TOOLS if candidate or baseline_additions else common)
        cli_read = cli.run("read", "fidelity.md")["data"]
        mcp_read = client.result("kb_read", {"path": "fidelity.md"})["data"]
        require(mcp_read == cli_read, "CLI/MCP read parity")
        cli_search = cli.run("search", payload={"query": "quartzcompat"})["data"]
        require(client.result("kb_search", {"query": "quartzcompat"})["data"] == cli_search, "CLI/MCP search parity")
        cli_context = cli.run("context", payload={"query": "quartzcompat", "max_tokens": 256})["data"]
        require(client.result("kb_context", {"query": "quartzcompat", "max_tokens": 256})["data"] == cli_context,
                "CLI/MCP context parity")
        client.result("kb_read", {"path": "absent.md"}, code="NOT_FOUND")
        require(authoritative(cli.root) == before, "MCP queries changed authoritative state")
        path = "mcp-candidate.md" if candidate else "mcp-baseline.md"
        create = request("create-" + path, path=path, type="Reference", body="# Evidence\nSynthetic MCP workflow.\n")
        client.result("kb_create", dict(create, dry_run=True))
        require(authoritative(cli.root) == before, "MCP preview published")
        client.result("kb_create", create)
        after = authoritative(cli.root)
        require(client.result("kb_create", create)["data"].get("replayed") is True, "MCP replay")
        require(authoritative(cli.root) == after, "MCP retry duplicated effect")
        current = client.result("kb_read", {"path": path})["data"]["note"]["revision"]
        patch = request("patch-" + path, path=path, expected_revision=current,
                        body_edits=[{"old_text": "Synthetic MCP workflow.", "new_text": "Updated MCP evidence."}])
        client.result("kb_patch", patch)
        after_patch = authoritative(cli.root)
        client.result("kb_patch", dict(patch, idempotency_key=patch["idempotency_key"] + "-stale"), code="REVISION_CONFLICT")
        require(authoritative(cli.root) == after_patch, "MCP conflict published")
        if candidate:
            require(set(tools["kb_backup"]["inputSchema"]["properties"]) == {"archive"}, "backup schema")
            require(set(tools["kb_restore"]["inputSchema"]["properties"]) == {"archive", "destination"}, "restore schema")
    finally:
        client.close()
    read_only = ConsumerMCP(cli, read_only=True)
    try:
        discover(read_only, READ_TOOLS)
        for name in set(tools) - READ_TOOLS:
            denied = read_only.rpc("tools/call", {"name": name, "arguments": {}})
            require(denied.get("isError") is True, "hidden tool direct-call enforcement")
    finally:
        read_only.close()
    return create, set(tools)


def restore_source_policy(cli, archive):
    storage = cli.root / ".kb/backups"
    storage.chmod(0o755)
    try:
        before = storage_snapshot(cli.root)
        restore = {"archive": archive, "destination": ".kb/restores/rejected"}
        cli.run("restore", payload=restore, exit_code=5, code="ACCESS_DENIED")
        require(storage_snapshot(cli.root) == before, "CLI rejected restore changed paths, modes or bytes")
        client = ConsumerMCP(cli)
        try:
            client.result("kb_restore", restore, code="ACCESS_DENIED")
        finally:
            client.close()
        require(storage_snapshot(cli.root) == before, "MCP rejected restore changed paths, modes or bytes")
        require(stat.S_IMODE(storage.stat().st_mode) == 0o755, "restore silently changed source permissions")
        require(not (cli.root / ".kb/restores").exists(), "rejected restore created destination storage")
    finally:
        # Explicit synthetic fixture repair; the product must never do this implicitly.
        storage.chmod(0o700)


def produce_prior_archive(cli, edge):
    """Capture an archive from the prior executable before any candidate touches its fixture."""
    require(not (cli.root / ".kb/state/pending.json").exists(), "prior archive requires settled source")
    expected = authoritative(cli.root)
    archive = ".kb/backups/prior.tar"
    result = cli.run("backup", payload={"archive": archive})["data"]
    require(authoritative(cli.root) == expected, "prior backup changed authoritative source bytes")
    archive_path = cli.root / archive
    digest = hashlib.sha256(archive_path.read_bytes()).hexdigest()
    edge.update(status="produced", archive_path=archive, archive_sha256=digest,
                manifest_revision=result["manifest_revision"], files=result["files"], bytes=result["bytes"])
    return {"source": cli, "expected": expected, "source_snapshot": storage_snapshot(cli.root),
            "archive": archive_path, "archive_sha256": digest, "manifest_revision": result["manifest_revision"]}


def restore_prior_archive(cli, prior, edge, creates, patch):
    """Restore exact prior-produced bytes into separate candidate staging, then replay and write."""
    edge["status"] = "restoring"
    archive_bytes = prior["archive"].read_bytes()
    require(hashlib.sha256(archive_bytes).hexdigest() == prior["archive_sha256"], "prior archive bytes changed")
    cli.run("init")
    storage = cli.root / ".kb/backups"
    storage.mkdir(mode=0o700)
    staged = storage / "prior.tar"
    staged.write_bytes(archive_bytes)
    require(hashlib.sha256(staged.read_bytes()).hexdigest() == prior["archive_sha256"], "staged prior archive bytes changed")
    destination = ".kb/restores/prior"
    result = cli.run("restore", payload={"archive": ".kb/backups/prior.tar", "destination": destination})["data"]
    require(result["manifest_revision"] == prior["manifest_revision"], "prior archive manifest identity changed")
    restored = CLI(cli.binary, cli.root / destination, cli.environment)
    require(authoritative(restored.root) == prior["expected"], "prior archive restored bytes differ")
    replay(restored, ("patch", "fidelity.md"), patch)
    for create in creates:
        replay(restored, ("create",), create)
    require(authoritative(restored.root) == prior["expected"], "prior receipt replay changed restored bytes")
    read = restored.run("read", "fidelity.md")["data"]
    require(read["note"]["revision"] == revision(prior["expected"]["fidelity.md"]), "prior restored revision changed")
    restored.run("patch", "fidelity.md", payload=request("after-prior-restore", expected_revision=read["note"]["revision"],
                 body_edits=[{"old_text": "600 seconds", "new_text": "900 seconds"}]))
    require("900 seconds" in restored.run("read", "fidelity.md")["data"]["markdown"], "prior restored patch did not persist")
    restored.run("create", payload=request("create-after-prior-restore", path="after-prior-restore.md",
                 type="Reference", body="# Continued evidence\nCandidate writes after restoring a prior archive.\n"))
    require("Candidate writes after restoring a prior archive." in
            restored.run("read", "after-prior-restore.md")["data"]["markdown"], "prior restored create did not persist")
    require(storage_snapshot(prior["source"].root) == prior["source_snapshot"], "prior archive consumer changed original root")
    require(hashlib.sha256(staged.read_bytes()).hexdigest() == prior["archive_sha256"], "prior archive changed during consumption")
    edge.update(status="passed", restored_manifest_revision=result["manifest_revision"],
                replayed_requests=len(creates) + 1, continued_writes=["patch", "create"], source_unchanged=True,
                authoritative_bytes_exact=True, restore_destination=destination)


def backup_restore(cli, creates, patch):
    before = authoritative(cli.root)
    backup = cli.run("backup", payload={"archive": ".kb/backups/cli.tar"})["data"]
    restore_source_policy(cli, ".kb/backups/cli.tar")
    client = ConsumerMCP(cli)
    try:
        # Compare the same inventory, before either restore adds an excluded storage directory.
        mcp_backup = client.result("kb_backup", {"archive": ".kb/backups/mcp.tar"})["data"]
        require(mcp_backup["manifest_revision"] == backup["manifest_revision"], "CLI/MCP snapshot parity")
    finally:
        client.close()
    restored = cli.run("restore", payload={"archive": ".kb/backups/cli.tar", "destination": ".kb/restores/cli"})["data"]
    require(backup["manifest_revision"] == restored["manifest_revision"], "CLI archive integrity identity")
    restored_cli = CLI(cli.binary, cli.root / ".kb/restores/cli", cli.environment)
    require(authoritative(restored_cli.root) == before, "CLI restored content/config/receipt bytes")
    require(authoritative(cli.root) == before, "backup/restore modified source")
    replay(restored_cli, ("patch", "fidelity.md"), patch)
    for create in creates:
        replay(restored_cli, ("create",), create)
    restored_cli.run("status")
    require(restored_cli.run("validate", "--profile", "okf")["data"]["conformant"], "restored conformance")
    restored_cli.run("index", "rebuild")
    require(restored_cli.run("read", "fidelity.md")["data"]["note"]["revision"] == revision(before["fidelity.md"]),
            "restored revision")
    restored_cli.run("create", payload=request("after-restore", path="after-restore.md", type="Reference", body="Continued write.\n"))
    require((restored_cli.root / "after-restore.md").is_file(), "continued CLI writes after restore")
    cli.run("restore", payload={"archive": ".kb/backups/cli.tar", "destination": ".kb/restores/cli"},
            exit_code=4, code="DESTINATION_EXISTS")
    client = ConsumerMCP(cli)
    try:
        mcp_restore = client.result("kb_restore", {"archive": ".kb/backups/mcp.tar", "destination": ".kb/restores/mcp"})["data"]
        require(mcp_restore["manifest_revision"] == mcp_backup["manifest_revision"], "MCP restore integrity identity")
    finally:
        client.close()
    mcp_root = CLI(cli.binary, cli.root / ".kb/restores/mcp", cli.environment)
    require(authoritative(mcp_root.root) == before, "MCP restored exact bytes")
    restored_client = ConsumerMCP(mcp_root)
    try:
        require(restored_client.result("kb_create", creates[0])["data"].get("replayed") is True, "MCP restored old receipt")
        updated = restored_client.result("kb_read", {"path": "fidelity.md"})["data"]["note"]["revision"]
        restored_client.result("kb_patch", request("after-mcp-restore", path="fidelity.md", expected_revision=updated,
                               body_edits=[{"old_text": "600 seconds", "new_text": "900 seconds"}]))
        require("900 seconds" in restored_client.result("kb_read", {"path": "fidelity.md"})["data"]["markdown"],
                "continued MCP writes after restore")
        presentation = (mcp_root.root / "fidelity.md").read_bytes()
        require(b"x-decimal: 0.1234567890123456789 # Exact decimal\r\n" in presentation
                and b"# Trailing YAML annotation\r\n" in presentation,
                "candidate targeted write lost restored presentation")
    finally:
        restored_client.close()
    require(authoritative(cli.root) == before, "restored writes escaped source root")
    return {"manifest_revision": backup["manifest_revision"], "files": backup["files"], "bytes": backup["bytes"],
            "private_source_rejection": "CLI/MCP ACCESS_DENIED; exact path/mode/byte purity"}


def interrupted_upgrade(baseline, candidate, workspace, environment, baseline_version):
    """Kill the unmodified baseline only after its own pending journal becomes visible."""
    for attempt in range(3):
        root = workspace / ("interrupted-" + str(attempt))
        old = CLI(baseline, root, environment)
        old.run("init")
        plan = request("interrupted", operations=[{"kind": "create", "operation_id": str(i),
                       "path": "batch/note-%02d.md" % i, "type": "Reference",
                       "body": "# Synthetic boundary\n" + "Bounded evidence. " * 3000} for i in range(50)])
        before = authoritative(root)
        pending = root / ".kb/state/pending.json"
        with tempfile.TemporaryFile() as output, tempfile.TemporaryFile() as errors, tempfile.TemporaryFile() as input_file:
            input_file.write(json.dumps(plan).encode())
            input_file.seek(0)
            process = subprocess.Popen([str(baseline), "--root", str(root), "apply", "--input", "-"],
                                       stdin=input_file, stdout=output, stderr=errors, env=environment)
            stopped = False
            deadline = time.monotonic() + 30
            try:
                while time.monotonic() < deadline and process.poll() is None:
                    if pending.exists():
                        process.send_signal(signal.SIGSTOP)
                        os.waitpid(process.pid, os.WUNTRACED)
                        stopped = pending.exists()
                        break
                    time.sleep(0.0001)
                if stopped:
                    journal = json.loads(pending.read_bytes())
            finally:
                if process.poll() is None:
                    process.kill()
                process.wait(timeout=5)
        if not stopped:
            continue
        state = journal["state"]
        require(state in {"prepared", "committed"}, "observed old journal state")
        require(process.returncode == -signal.SIGKILL, "old writer was actually interrupted")
        pending_bytes = pending.read_bytes()
        old.run("read", "batch/note-00.md", exit_code=6, code="RECOVERY_REQUIRED")
        preview_before = authoritative(root)
        old.run("recover", "--dry-run")
        require(pending.read_bytes() == pending_bytes and authoritative(root) == preview_before, "recovery preview published")
        new = CLI(candidate, root, environment)
        new.run("backup", payload={"archive": ".kb/backups/pending.tar"}, exit_code=6, code="RECOVERY_REQUIRED")
        require(pending.read_bytes() == pending_bytes and authoritative(root) == preview_before, "backup silently recovered old pending")
        old.run("recover")
        require(not pending.exists(), "old recovery did not settle journal")
        if state == "prepared":
            require(authoritative(root) == before, "old prepared recovery did not restore exact pre-state")
        else:
            require(all((root / op["path"]).is_file() for op in plan["operations"]), "old committed recovery lost files")
        # Upgrade only after the old binary explicitly settles recovery.
        new.run("status")
        result = new.run("apply", payload=plan)["data"]
        require(bool(result.get("replayed", False)) == (state == "committed"), "recovered upgrade receipt semantics")
        replay(new, ("apply",), plan)
        new.run("backup", payload={"archive": ".kb/backups/settled.tar"})
        return {"observed_state": state, "attempts": attempt + 1, "operations": 50,
                "recovery_binary": baseline_version, "signal": "SIGKILL", "upgrade_after_old_recovery": True}
    raise CompatibilityFailure("could not observe old pending journal; no interruption evidence")


def seed(cli):
    cli.run("init")
    create = request("original-create", path="original.md", type="Reference", body="# Original\nSynthetic original.\n")
    cli.run("create", payload=create)
    fidelity = (b"---\r\ntype: SyntheticExtension # Preserve type annotation\r\n"
                b"x-counter: 9007199254740993\r\nx-decimal: 0.1234567890123456789 # Exact decimal\r\n"
                b"x-extension: {custom: 'unchanged'} # Preserve flow\r\n# Trailing YAML annotation\r\n"
                b"---\r\n# Evidence\r\nquartzcompat retention is 300 seconds.\r\n")
    (cli.root / "fidelity.md").write_bytes(fidelity)
    (cli.root / "source.bin").write_bytes(b"\x00\x01\x80\xffSynthetic immutable source\n")
    patch = request("fidelity-patch", expected_revision=revision(fidelity),
                    body_edits=[{"old_text": "300 seconds", "new_text": "600 seconds"}])
    return create, patch


def exercise_candidate(binary):
    """Shared native-prefix/source-package consumer smoke; no baseline installation required."""
    with tempfile.TemporaryDirectory(prefix="kb-package-consumer-") as directory:
        workspace = Path(directory).resolve()
        home = workspace / "home"
        home.mkdir()
        environment = {"PATH": "", "HOME": str(home), "TMPDIR": str(workspace), "LANG": "C.UTF-8", "GOMAXPROCS": "2"}
        cli = CLI(binary, workspace / "knowledge", environment)
        create, patch = seed(cli)
        common_contract(cli)
        cli.run("patch", "fidelity.md", payload=patch)
        mcp_create, _ = mcp_workflow(cli, True)
        result = backup_restore(cli, [create, mcp_create], patch)
        require(not list(home.iterdir()), "package workflow created global integration files")
        return result


def qualify(baseline, candidate, candidate_version, candidate_commit, report,
            baseline_version="v0.3.1", baseline_commit=BASELINE_COMMIT):
    report["identities"] = {"baseline": identity(baseline, baseline_version, baseline_commit, "baseline"),
                            "candidate": identity(candidate, candidate_version, candidate_commit, "candidate")}
    with tempfile.TemporaryDirectory(prefix="kb-compatibility-") as directory:
        workspace = Path(directory).resolve()
        home = workspace / "home"
        home.mkdir()
        environment = {"PATH": "", "HOME": str(home), "TMPDIR": str(workspace), "LANG": "C.UTF-8", "GOMAXPROCS": "2"}
        root = workspace / "knowledge"
        old, new = CLI(baseline, root, environment), CLI(candidate, root, environment)
        create, patch = seed(old)
        report["active_check"] = "baseline-cli-contract"
        common_contract(old)
        report["checks"].append("baseline-cli-contract")
        report["active_check"] = "baseline-independent-mcp"
        old_mcp_create, old_tools = mcp_workflow(old, False, baseline_additions=baseline_version != "v0.3.1")
        report["checks"].append("baseline-independent-mcp")
        old.run("patch", "fidelity.md", payload=patch)
        preserved = (root / "fidelity.md").read_bytes()
        if OPERATIONAL_TOOLS <= old_tools:
            report["active_check"] = "prior-produced-archive"
            edge = {"status": "producing", "producer": report["identities"]["baseline"],
                    "consumer": report["identities"]["candidate"]}
            report["prior_archive"] = edge
            prior = produce_prior_archive(old, edge)
            staging = CLI(candidate, workspace / "prior-archive-consumer", environment)
            restore_prior_archive(staging, prior, edge, [create, old_mcp_create], patch)
            report["checks"].append("prior-produced-archive")
        else:
            report["prior_archive"] = {"status": "not_applicable", "reason": "baseline has no operational archive tools",
                                       "producer": report["identities"]["baseline"],
                                       "consumer": report["identities"]["candidate"]}
        report["active_check"] = "upgrade-cli-and-receipts"
        common_contract(new)
        require((root / "fidelity.md").read_bytes() == preserved, "upgrade changed note bytes")
        for span in [b"type: SyntheticExtension # Preserve type annotation\r\n",
                     b"x-decimal: 0.1234567890123456789 # Exact decimal\r\n",
                     b"x-extension: {custom: 'unchanged'} # Preserve flow\r\n", b"# Trailing YAML annotation\r\n"]:
            require(span in preserved, "targeted edit metadata/comment/CRLF fidelity")
        replay(new, ("create",), create)
        replay(new, ("create",), old_mcp_create)
        replay(new, ("patch", "fidelity.md"), patch)
        before_conflict = authoritative(root)
        new.run("create", payload=dict(create, body="Changed request.\n"), exit_code=4, code="IDEMPOTENCY_CONFLICT")
        require(authoritative(root) == before_conflict, "key conflict modified files")
        report["checks"].append("upgrade-cli-and-receipts")
        report["active_check"] = "candidate-independent-mcp"
        candidate_create, candidate_tools = mcp_workflow(new, True)
        report["mcp_tools"] = {"baseline": len(old_tools), "candidate": len(candidate_tools)}
        report["checks"].append("candidate-independent-mcp")
        report["active_check"] = "operational-backup-restore"
        report["backup"] = backup_restore(new, [create, old_mcp_create, candidate_create], patch)
        report["checks"].append("operational-backup-restore")
        report["active_check"] = "rollback-boundary"
        if OPERATIONAL_TOOLS <= old_tools:
            old_operational = "advertised by declared baseline; no state downgrade attempted"
        else:
            settled = authoritative(root)
            old.run("backup", ".kb/backups/unsupported.tar", exit_code=2, code="INVALID_INPUT")
            old.run("restore", ".kb/backups/cli.tar", ".kb/restores/unsupported", exit_code=2, code="INVALID_INPUT")
            require(authoritative(root) == settled, "unsupported old command modified state")
            old_operational = "unsupported (INVALID_INPUT, exit 2)"
        report["rollback"] = {"old_backup_restore": old_operational,
                              "automatic_state_downgrade": "not supported or inferred"}
        report["checks"].append("rollback-boundary")
        report["active_check"] = "old-pending-recovery-then-upgrade"
        report["pending"] = interrupted_upgrade(baseline, candidate, workspace, environment, baseline_version)
        report["checks"].append("old-pending-recovery-then-upgrade")
        require(not list(home.iterdir()), "workflow created global integration files")
    report["active_check"] = "binary-immutability"
    for label, binary in [("baseline", baseline), ("candidate", candidate)]:
        require(hashlib.sha256(binary.read_bytes()).hexdigest() == report["identities"][label]["sha256"], "binary changed during run")
    report["checks"].append("binary-immutability")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline", required=True, type=Path, help="Preserved prior executable, not a moving package-manager symlink")
    parser.add_argument("--baseline-version", help="Declared additional baseline release; requires --baseline-commit")
    parser.add_argument("--baseline-commit", help="Declared full 40-hex baseline source; requires --baseline-version")
    parser.add_argument("--candidate", "--binary", required=True, type=Path)
    parser.add_argument("--candidate-version", help="Optional independently selected expected version")
    parser.add_argument("--candidate-commit", help="Optional independently selected expected 40-hex source commit")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        baseline_version, baseline_commit = baseline_expectation(args.baseline_version, args.baseline_commit)
    except CompatibilityFailure as error:
        parser.error(str(error))
    report = {"report_version": 1, "ok": False, "platform": {"system": platform.system(), "machine": platform.machine()},
              "checks": [], "active_check": "binary-identity", "limits": [
                  "Synthetic local process evidence, not universal crash or power-loss durability.",
                  "No installed-candidate qualification unless the supplied executable is that installation.",
                  "No automatic downgrade of newer operational state; recover pending state with the old binary before upgrade.",
                  "Physical executable paths are redacted as $BASELINE/kb and $CANDIDATE/kb; SHA-256 identifies exact bytes."]}
    try:
        qualify(args.baseline.resolve(), args.candidate.resolve(), args.candidate_version, args.candidate_commit, report,
                baseline_version, baseline_commit)
        report["ok"] = True
        del report["active_check"]
    except (CompatibilityFailure, OSError, ValueError, KeyError, TypeError, RuntimeError, subprocess.SubprocessError) as error:
        # Do not copy subprocess output, private executable paths or temporary roots into public evidence.
        report["failure"] = {"check": report["active_check"], "type": type(error).__name__}
        if report.get("prior_archive", {}).get("status") in {"producing", "produced", "restoring"}:
            report["prior_archive"]["status"] = "failed"
        if isinstance(error, CompatibilityFailure):
            report["failure"]["invariant"] = str(error)
    encoded = json.dumps(report, indent=2, sort_keys=True) + "\n"
    require(len(encoded.encode()) <= 32 * 1024, "bounded report")
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(encoded)
    print(encoded, end="")
    return 0 if report["ok"] else 1


if __name__ == "__main__":
    raise SystemExit(main())

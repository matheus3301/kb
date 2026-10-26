"""Reject false-positive compatibility evidence and preserve authoritative snapshot boundaries."""
import copy
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import compatibility as qualification


def success(data=None):
    return {"schema_version": "kb.v1", "ok": True, "data": data,
            "warnings": [], "meta": {"root": "example-kb", "generation": "", "index_stale": False}}


class EnvelopeContractTests(unittest.TestCase):
    def test_nonconformance_is_successful_report_not_execution_error(self):
        value = success({"conformant": False, "error_count": 1})
        self.assertIs(qualification.envelope(value), value)

    def test_domain_failure_requires_machine_code_and_retryability(self):
        value = success()
        del value["data"]
        value.update(ok=False, error={"code": "REVISION_CONFLICT", "message": "Synthetic conflict", "retryable": False})
        qualification.envelope(value)
        for field, replacement in [("code", ""), ("retryable", "false"), ("message", None)]:
            with self.subTest(field=field), self.assertRaises(qualification.CompatibilityFailure):
                broken = copy.deepcopy(value)
                broken["error"][field] = replacement
                qualification.envelope(broken)
        with self.assertRaises(qualification.CompatibilityFailure):
            qualification.envelope(dict(value, data={}))

    def test_rejects_wire_drift_missing_fields_and_numeric_booleans(self):
        mutations = [lambda v: v.update(schema_version="kb.v2"), lambda v: v.update(ok=1),
                     lambda v: v.pop("warnings"), lambda v: v.update(extra="unsupported"),
                     lambda v: v["meta"].update(index_stale=0), lambda v: v["meta"].pop("generation"),
                     lambda v: v.update(error={"code": "IO", "message": "Synthetic", "retryable": True}),
                     lambda v: v.pop("data")]
        for mutate in mutations:
            with self.subTest(mutation=mutations.index(mutate)), self.assertRaises(qualification.CompatibilityFailure):
                value = success()
                mutate(value)
                qualification.envelope(value)

    def test_warning_locations_cannot_silently_change_wire_types(self):
        warning = {"code": "INVALID_FRONTMATTER", "message": "Synthetic invalid note", "layer": "conformance",
                   "path": "invalid.md", "line": 1, "severity": "error", "rule": "frontmatter.required"}
        value = success()
        value["warnings"] = [warning]
        qualification.envelope(value)
        for field, replacement in [("line", 0), ("line", True), ("severity", "fatal"), ("rule", 7), ("extra", "drift")]:
            with self.subTest(field=field, replacement=replacement), self.assertRaises(qualification.CompatibilityFailure):
                broken = copy.deepcopy(value)
                broken["warnings"][0][field] = replacement
                qualification.envelope(broken)


class AuthoritativeSnapshotTests(unittest.TestCase):
    def test_receipts_config_and_unscoped_binary_sources_are_not_cache(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            included = {"note.md": b"---\r\ntype: Extension # annotation\r\nx-decimal: 0.1234567890123456789\r\n---\r\nEvidence\r\n",
                        "source.bin": bytes([0, 255, 128]), ".kb/config.toml": b"version = 1\n",
                        ".kb/state/receipts/request.json": b'{"synthetic":"receipt"}\n'}
            excluded = {".kb/cache/generation": b"derived", ".kb/state/lock": b"",
                        ".kb/backups/snapshot.tar": b"archive", ".kb/restores/new/note.md": b"restored"}
            for name, content in {**included, **excluded}.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(content)
            self.assertEqual(qualification.authoritative(root), included)
            # Corruption of an old receipt must be visible even when notes still match.
            (root / ".kb/state/receipts/request.json").write_bytes(b"corrupt")
            self.assertNotEqual(qualification.authoritative(root), included)
            (root / ".kb/state/receipts/request.json").unlink()
            self.assertNotIn(".kb/state/receipts/request.json", qualification.authoritative(root))

    def test_rejection_snapshot_detects_empty_storage_creation_and_chmod(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            storage = root / ".kb/backups"
            storage.mkdir(parents=True, mode=0o700)
            before = qualification.storage_snapshot(root)
            (root / ".kb/restores").mkdir(mode=0o700)
            self.assertNotEqual(qualification.storage_snapshot(root), before)
            (root / ".kb/restores").rmdir()
            storage.chmod(0o755)
            self.assertNotEqual(qualification.storage_snapshot(root), before)


class BaselineSelectionTests(unittest.TestCase):
    def test_default_origin_remains_immutable_and_additional_origin_is_explicit(self):
        self.assertEqual(qualification.baseline_expectation(None, None),
                         ("v0.3.1", "2ca104f48cc4a6551f4bbc4fae12d51432e42dff"))
        # This is a synthetic identity fixture, not a claimed v0.4.0 source.
        self.assertEqual(qualification.baseline_expectation("v0.4.0", "a" * 40), ("v0.4.0", "a" * 40))
        for version, commit in [("v0.4.0", None), (None, "a" * 40), ("dev", "a" * 40),
                                ("v0.4.0", "abc123"), ("v0.4.0", "A" * 40), ("v0.4", "a" * 40)]:
            with self.subTest(version=version, commit=commit), self.assertRaises(qualification.CompatibilityFailure):
                qualification.baseline_expectation(version, commit)

    def test_declared_identity_must_match_both_actual_version_and_full_commit(self):
        data = {"version": "v0.4.0", "commit": "a" * 40}
        qualification.check_identity(data, "v0.4.0", "a" * 40, "baseline")
        for actual in [dict(data, version="v0.3.1"), dict(data, commit="b" * 40),
                       dict(data, commit="a" * 39), dict(data, commit=None), dict(data, extra=True)]:
            with self.subTest(actual=actual), self.assertRaises(qualification.CompatibilityFailure):
                qualification.check_identity(actual, "v0.4.0", "a" * 40, "baseline")

    def test_new_baseline_may_add_only_known_tools_without_losing_common_contract(self):
        common = qualification.READ_TOOLS | qualification.WRITE_TOOLS
        candidate = common | qualification.OPERATIONAL_TOOLS
        qualification.check_catalog(list(common), common)
        qualification.check_catalog(list(candidate), common, candidate)
        qualification.check_catalog(list(candidate), candidate)
        bad_catalogs = [list(candidate - {"kb_read"}), list(candidate | {"kb_shell"}),
                        list(common | {"kb_backup"}), list(candidate) + ["kb_read"]]
        for names in bad_catalogs:
            with self.subTest(names=names), self.assertRaises(qualification.CompatibilityFailure):
                qualification.check_catalog(names, common, candidate)
        with self.assertRaises(qualification.CompatibilityFailure):
            qualification.check_catalog(list(common), candidate)


class ReportFailureTests(unittest.TestCase):
    def test_unavailable_baseline_is_failed_bounded_report_without_local_path(self):
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory)
            output = workspace / "report.json"
            private_name = workspace / "synthetic-private-executable"
            result = subprocess.run([sys.executable, str(Path(qualification.__file__)),
                                     "--baseline", str(private_name), "--binary", str(private_name),
                                     "--output", str(output)], text=True, capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 1)
            report = json.loads(result.stdout)
            self.assertEqual(report, json.loads(output.read_text()))
            self.assertFalse(report["ok"])
            self.assertEqual(report["failure"]["check"], "binary-identity")
            self.assertEqual(report["checks"], [])
            self.assertLessEqual(len(result.stdout.encode()), 32 * 1024)
            self.assertNotIn(str(workspace), result.stdout + result.stderr)
            self.assertNotIn(private_name.name, result.stdout + result.stderr)

    def test_additional_origin_flags_are_required_together_before_executable_access(self):
        for flags in [["--baseline-version", "v0.4.0"], ["--baseline-commit", "a" * 40]]:
            with self.subTest(flags=flags), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / "report.json"
                result = subprocess.run([sys.executable, str(Path(qualification.__file__)),
                                         "--baseline", "missing-kb", "--candidate", "missing-kb",
                                         "--output", str(output), *flags],
                                        text=True, capture_output=True, timeout=10)
                self.assertEqual(result.returncode, 2)
                self.assertIn("must be supplied together", result.stderr)
                self.assertFalse(output.exists())


if __name__ == "__main__":
    unittest.main()

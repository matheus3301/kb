"""Reject setup documentation checks that hide broken bindings or fake readiness."""

import copy
from pathlib import Path
import tempfile
import unittest

import check_docs


class SetupDocumentationTests(unittest.TestCase):
    def setUp(self):
        workspace = tempfile.TemporaryDirectory(prefix="kb-docs-contract-")
        self.addCleanup(workspace.cleanup)
        self.root = Path(workspace.name) / "knowledge"
        self.project = Path(workspace.name) / "project"
        self.envelope = {
            "schema_version": "kb.v1", "ok": True,
            "data": {
                "agent": "pi", "root": str(self.root.resolve()),
                "scope": "project", "project": str(self.project.resolve()),
                "mode": "cli", "status": "degraded", "configured": True,
                "ready": False, "activation_observed": False,
                "artifacts": [{"path": str(self.project / "AGENTS.md"), "status": "configured"}],
                "diagnostics": ["native agent executable is unavailable on PATH: pi"],
            },
        }

    def inspect(self, envelope, exit_code=1):
        check_docs.validate_unavailable_host_check(exit_code, envelope, self.root, self.project, "pi")

    def test_missing_host_must_have_the_exact_nonzero_inspection_status(self):
        for exit_code in (0, 2, 4, -1):
            with self.subTest(exit_code=exit_code), self.assertRaises(AssertionError):
                self.inspect(self.envelope, exit_code)

    def test_degradation_cannot_hide_an_incorrect_binding(self):
        for field, value in (("root", str(self.root.parent / "other")),
                             ("project", str(self.project.parent / "other")),
                             ("agent", "codex"), ("scope", "global"),
                             ("mode", "mcp"), ("status", "activation_pending")):
            envelope = copy.deepcopy(self.envelope)
            envelope["data"][field] = value
            with self.subTest(field=field), self.assertRaises(AssertionError):
                self.inspect(envelope)

    def test_artifact_failure_cannot_be_accepted_as_missing_native_host_only(self):
        for status in ("missing", "conflict"):
            envelope = copy.deepcopy(self.envelope)
            envelope["data"]["artifacts"][0]["status"] = status
            with self.subTest(status=status), self.assertRaises(AssertionError):
                self.inspect(envelope)
        envelope = copy.deepcopy(self.envelope)
        envelope["data"]["artifacts"] = []
        with self.assertRaises(AssertionError):
            self.inspect(envelope)

    def test_configured_does_not_authorize_native_readiness_or_activation_claims(self):
        for field, value in (("configured", False), ("configured", 1),
                             ("ready", True), ("ready", 0),
                             ("activation_observed", True)):
            envelope = copy.deepcopy(self.envelope)
            envelope["data"][field] = value
            with self.subTest(field=field, value=value), self.assertRaises(AssertionError):
                self.inspect(envelope)

    def test_generic_degraded_diagnostic_does_not_prove_expected_host_absence(self):
        for diagnostics in ([], ["selected collection is unavailable"],
                            ["native agent executable is unavailable on PATH: codex"]):
            envelope = copy.deepcopy(self.envelope)
            envelope["data"]["diagnostics"] = diagnostics
            with self.subTest(diagnostics=diagnostics), self.assertRaises(AssertionError):
                self.inspect(envelope)

    def test_failed_envelope_cannot_be_counted_as_successful_inspection(self):
        for field, value in (("ok", False), ("ok", 1), ("schema_version", "other")):
            envelope = copy.deepcopy(self.envelope)
            envelope[field] = value
            with self.subTest(field=field), self.assertRaises(AssertionError):
                self.inspect(envelope)


if __name__ == "__main__":
    unittest.main()

"""Regression cases for checks that must reject misleading CI success."""

import json
from pathlib import Path
import tempfile
import unittest

import quality


class QualityTests(unittest.TestCase):
    def workflow(self, action):
        return f"""permissions:
  contents: read
jobs:
  check:
    steps:
      - uses: {action}
        with:
          persist-credentials: false
"""

    def results(self):
        return {name: {"result": "success"} for name in
                ("repository", "static", "tests", "security", "build", "release-config", "dependency-review")}

    def test_unpinned_action_is_rejected(self):
        self.assertTrue(quality.workflow_findings(self.workflow("actions/checkout@v7")))

    def test_sha_pinned_action_is_accepted(self):
        self.assertEqual([], quality.workflow_findings(self.workflow("actions/checkout@" + "a" * 40)))

    def test_write_token_is_rejected(self):
        workflow = self.workflow("actions/checkout@" + "a" * 40) + "  contents: write\n"
        self.assertIn("workflow requests a write token", quality.workflow_findings(workflow))

    def test_only_verified_tag_publisher_can_have_contents_write(self):
        workflow = (quality.ROOT / ".github/workflows/release.yml").read_text()
        self.assertEqual([], quality.workflow_findings(workflow, "release.yml"))
        for altered, name in (
            (workflow, "ci.yml"),
            (workflow.replace('tags: ["v*"]', 'branches: ["main"]'), "release.yml"),
            (workflow.replace("    needs: verify\n", "    needs: []\n"), "release.yml"),
            (workflow.replace("run: python3 scripts/release.py verify-tag", "run: true"), "release.yml"),
            (workflow.replace("run: make tools check", "run: true"), "release.yml"),
            (workflow.replace("contents: write", "packages: write"), "release.yml"),
            (workflow.replace("      checks: read", "      checks: write"), "release.yml"),
            (workflow.replace("  contents: read\n", "  contents: write\n", 1), "release.yml"),
        ):
            with self.subTest(name=name, altered=altered[:80]):
                self.assertTrue(quality.workflow_findings(altered, name))

    def test_inline_permission_mapping_is_rejected(self):
        self.assertTrue(quality.workflow_findings("permissions: {contents: write}"))

    def test_failure_bypass_is_rejected(self):
        workflow = self.workflow("actions/checkout@" + "a" * 40) + "continue-on-error: true\n"
        self.assertTrue(quality.workflow_findings(workflow))

    def test_missing_job_is_rejected(self):
        self.assertTrue(quality.gate_findings({}, "pull_request"))

    def test_failed_cancelled_skipped_jobs_are_rejected(self):
        for status in ("failure", "cancelled", "skipped"):
            with self.subTest(status=status):
                jobs = self.results()
                jobs["tests"]["result"] = status
                self.assertTrue(quality.gate_findings(jobs, "pull_request"))

    def test_release_snapshot_is_required_by_the_gate(self):
        for status in ("failure", "cancelled", "skipped"):
            jobs = self.results()
            jobs["release-config"]["result"] = status
            with self.subTest(status=status):
                self.assertTrue(quality.gate_findings(jobs, "pull_request"))

    def test_successful_pr_is_accepted(self):
        self.assertEqual([], quality.gate_findings(self.results(), "pull_request"))

    def test_push_expects_dependency_review_to_be_skipped(self):
        jobs = self.results()
        jobs["dependency-review"]["result"] = "skipped"
        self.assertEqual([], quality.gate_findings(jobs, "push"))
        self.assertTrue(quality.gate_findings(jobs, "pull_request"))

    def test_coverage_threshold_and_missing_total(self):
        self.assertTrue(quality.coverage_findings("total: (statements) 79.9%"))
        self.assertTrue(quality.coverage_findings(""))
        self.assertEqual([], quality.coverage_findings("total: (statements) 80.0%"))

    def test_verified_claim_requires_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            item = {"id": "A01", "status": "verified", "tests": [], "evidence": []}
            self.assertTrue(quality.acceptance_findings(Path(directory), "| A01 | Test |", [item]))

    def test_missing_duplicate_and_outside_acceptance_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            item = {"id": "A01", "status": "planned", "tests": ["../outside"], "evidence": []}
            self.assertTrue(quality.acceptance_findings(Path(directory), "| A01 | Test |", [item, item]))

    def test_local_links_missing_and_external(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            page = root / "README.md"
            page.write_text("[external](https://example.com) [missing](absent.md)")
            self.assertEqual(1, len(quality.local_link_findings(root, page)))
            (root / "absent.md").write_text("exists\n")
            self.assertEqual([], quality.local_link_findings(root, page))

    def test_readme_html_images_and_links_must_resolve(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            page = root / "README.md"
            page.write_text('<img src="absent.svg" alt="Logo" /><a href="../outside.md">More</a>')
            self.assertEqual(2, len(quality.local_link_findings(root, page)))

    def test_reference_and_source_links_cannot_escape_checkout(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            page = root / "README.md"
            page.write_text("[private]: ../private.md\nresource: ../private.md\n")
            self.assertEqual(2, len(quality.local_link_findings(root, page)))

    def test_public_docs_reject_workstation_paths_and_local_uris(self):
        page = Path("README.md")
        personal_path = "/" + "Users/" + "synthetic-person" + "/private/note.md"
        local_uri = "file" + "://private/note.md"
        self.assertTrue(quality.public_content_findings(page, personal_path))
        self.assertTrue(quality.public_content_findings(page, local_uri))
        self.assertEqual([], quality.public_content_findings(page, "./example-knowledge"))

    def test_markdown_language_guard_allows_labeled_synthetic_code_examples(self):
        page = Path("README.md")
        word = "conhe" + "cimento"
        self.assertTrue(quality.public_content_findings(page, word))
        self.assertEqual([], quality.public_content_findings(page, "Synthetic search text:\n```text\n" + word + "\n```"))

    def test_verified_claim_with_existing_evidence_is_accepted(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "test.go").write_text("synthetic test reference")
            (root / "result.json").write_text(json.dumps({"passed": True}))
            item = {"id": "A01", "status": "verified", "tests": ["test.go"],
                    "evidence": ["result.json"]}
            self.assertEqual([], quality.acceptance_findings(root, "| A01 | Test |", [item]))


if __name__ == "__main__":
    unittest.main()

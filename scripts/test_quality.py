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
                ("repository", "static", "tests", "security", "build", "release-config", "compatibility", "dependency-review")}

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

    def test_published_archive_verification_cannot_be_removed_or_skipped(self):
        workflow = (quality.ROOT / ".github/workflows/release.yml").read_text()
        start, end = workflow.index("  verify-published:\n"), workflow.index("  tap:\n")
        verified = workflow[start:end]
        mutations = [
            workflow[:start] + workflow[end:],
            workflow.replace("run: python3 scripts/check_release.py", "run: true"),
            workflow.replace("    needs: publish\n", "    needs: []\n"),
            workflow.replace("        os: [ubuntu-24.04, macos-15]", "        os: [ubuntu-24.04]"),
            workflow.replace("      fail-fast: false", "      fail-fast: true"),
            workflow.replace("          ref: ${{ github.ref }}", "          ref: main"),
        ]
        for condition in ("false", "always()", "failure()", "cancelled()"):
            mutations.append(workflow.replace(verified, verified.replace(
                "    needs: publish\n", f"    needs: publish\n    if: ${{{{ {condition} }}}}\n")))
        for altered in mutations:
            with self.subTest(altered=altered[start:start + 150]):
                self.assertTrue(quality.workflow_findings(altered, "release.yml"))

    def test_published_archive_expected_identity_is_not_trusted_from_artifacts(self):
        workflow = (quality.ROOT / ".github/workflows/release.yml").read_text()
        for before, after in (
            ('release.release_tag(os.environ["GITHUB_REF_NAME"])', 'os.environ["GITHUB_REF_NAME"]'),
            ('["git", "rev-parse", "HEAD"]', '["git", "rev-parse", "main"]'),
            ('{"version": version, "commit": commit}', '{"version": "latest", "commit": "artifact-value"}'),
            ('command.extend(["--pattern", filename])', 'command.extend(["--pattern", "*"])'),
            ('subprocess.run(command, check=True)', 'subprocess.run(command, check=False)'),
        ):
            with self.subTest(before=before):
                self.assertTrue(quality.workflow_findings(workflow.replace(before, after), "release.yml"))

    def test_tap_requires_successful_complete_published_matrix(self):
        workflow = (quality.ROOT / ".github/workflows/release.yml").read_text()
        for before, after in (
            ("needs: [publish, verify-published]", "needs: publish"),
            ("needs: [publish, verify-published]", "needs: verify-published"),
            ("success() && !contains(github.ref_name, '-')", "always() && !contains(github.ref_name, '-')"),
            ("success() && !contains(github.ref_name, '-')", "success()"),
        ):
            with self.subTest(after=after):
                self.assertTrue(quality.workflow_findings(workflow.replace(before, after), "release.yml"))

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

    def test_prior_version_compatibility_is_required_by_the_gate(self):
        for status in ("failure", "cancelled", "skipped", None):
            jobs = self.results()
            if status is None:
                del jobs["compatibility"]
            else:
                jobs["compatibility"]["result"] = status
            with self.subTest(status=status):
                self.assertIn("required job did not succeed: compatibility",
                              quality.gate_findings(jobs, "pull_request"))

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

    def test_llms_raw_checkout_links_are_checked_without_network(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            page = root / "llms.txt"
            page.write_text("[Guide](https://raw.githubusercontent.com/matheus3301/kb/main/guide.md)\n")
            self.assertTrue(quality.local_link_findings(root, page))
            (root / "guide.md").write_text("# Guide\n")
            self.assertEqual([], quality.local_link_findings(root, page))
            page.write_text("[Escape](https://raw.githubusercontent.com/matheus3301/kb/main/%2e%2e/outside.md)")
            self.assertTrue(quality.local_link_findings(root, page))

    def test_installed_skill_rejects_references_outside_its_folder(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill = root / "example"
            skill.mkdir()
            entry = skill / "SKILL.md"
            entry.write_text("---\nname: example\ndescription: Synthetic skill\n---\n[Guide](../guide.md)\n")
            (root / "guide.md").write_text("# Guide\n")
            self.assertTrue(quality.skill_findings(skill))
            entry.write_text(entry.read_text().replace("../guide.md", "guide.md"))
            self.assertTrue(quality.skill_findings(skill))
            (skill / "guide.md").write_text("# Guide\n")
            self.assertEqual([], quality.skill_findings(skill))

    def test_renamed_skill_requires_matching_discovery_name(self):
        with tempfile.TemporaryDirectory() as directory:
            skill = Path(directory) / "new-name"
            skill.mkdir()
            self.assertTrue(quality.skill_findings(skill))
            (skill / "SKILL.md").write_text("---\nname: old-name\ndescription: Synthetic skill\n---\n")
            self.assertTrue(quality.skill_findings(skill))

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
        self.assertTrue(quality.public_content_findings(Path("llms.txt"), word))

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

"""Regression tests for release provenance and cross-repository publication."""

import base64
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import tarfile
import io
import unittest
from unittest.mock import patch

import check_release
import release


class ReleaseTests(unittest.TestCase):
    def test_go_license_supports_official_and_homebrew_layouts(self):
        with tempfile.TemporaryDirectory() as directory:
            keg = Path(directory)
            goroot = keg / "libexec"
            goroot.mkdir()
            (keg / "LICENSE").write_text("Synthetic keg license\n")
            self.assertEqual("Synthetic keg license\n", release.go_license(goroot))
            (goroot / "LICENSE").write_text("Synthetic distribution license\n")
            self.assertEqual("Synthetic distribution license\n", release.go_license(goroot))

    def test_missing_go_license_is_not_silently_replaced(self):
        with tempfile.TemporaryDirectory() as directory:
            keg = Path(directory)
            (keg / "LICENSE").write_text("Unrelated parent license\n")
            with self.assertRaises(ValueError):
                release.go_license(keg / "non-homebrew-root")
            with self.assertRaises(ValueError):
                release.go_license(keg / "nested" / "libexec")

    def test_archive_check_rejects_traversal_links_and_workstation_owner(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "synthetic.tar.gz"
            for name, uid, kind in (("../outside.md", 0, tarfile.REGTYPE),
                                    ("README.md", 501, tarfile.REGTYPE),
                                    ("linked.md", 0, tarfile.SYMTYPE)):
                with tarfile.open(path, "w:gz") as archive:
                    member = tarfile.TarInfo(name)
                    member.uid = uid
                    member.type = kind
                    member.linkname = "../outside"
                    member.size = 0
                    archive.addfile(member, io.BytesIO())
                with self.subTest(name=name, uid=uid), self.assertRaises(ValueError):
                    check_release.inspect_archive(path)

    def test_llms_archive_content_uses_the_same_privacy_guard(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "synthetic.tar.gz"
            content = ("/" + "Users/" + "synthetic-person" + "/note.md").encode()
            with tarfile.open(path, "w:gz") as archive:
                member = tarfile.TarInfo("llms.txt")
                member.size = len(content)
                archive.addfile(member, io.BytesIO(content))
            with self.assertRaises(ValueError):
                check_release.inspect_archive(path)

    def test_tag_validation_rejects_retired_or_injected_versions(self):
        for tag in ("v0.1.1", "v1.2.3-rc.1"):
            self.assertEqual(tag, release.release_tag(tag))
        for tag in ("latest", "0.1.1", "v0.1.0", "v01.2.3", "v1.2.3-01", "v1.2.3-a..b", 'v1.2.3";system("bad")'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                release.release_tag(tag)

    def test_only_latest_exact_commit_actions_gate_is_accepted(self):
        good = {"id": 1, "name": "CI gate", "app": {"id": 15368}, "head_sha": "synthetic",
                "status": "completed", "conclusion": "success"}
        release.require_ci([good], "synthetic")
        for change in ({"head_sha": "other"}, {"app": {"id": 1}}, {"conclusion": "failure"}, {"status": "in_progress"}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                release.require_ci([dict(good, **change)], "synthetic")
        with self.assertRaises(ValueError):
            release.require_ci([good, dict(good, id=2, conclusion="cancelled")], "synthetic")

    def test_formula_rejects_untrusted_identity_and_source_checksum(self):
        with tempfile.TemporaryDirectory() as directory:
            dist = Path(directory)
            (dist / "metadata.json").write_text(json.dumps({"version": "0.1.1", "commit": "a" * 40}))
            source = dist / "kb_0.1.1_source.tar.gz"
            source.write_bytes(b"synthetic source archive")
            (dist / "checksums.txt").write_text("0" * 64 + "  " + source.name + "\n")
            output = dist / "kb.rb"
            with self.assertRaises(ValueError):
                release.prepare_formula(dist, output)
            self.assertFalse(output.exists())
            checksum = hashlib.sha256(source.read_bytes()).hexdigest()
            (dist / "checksums.txt").write_text(checksum + "  " + source.name + "\n")
            release.prepare_formula(dist, output)
            self.assertIn(checksum, output.read_text())
            self.assertIn('main.commit=' + "a" * 40, output.read_text())
        for commit, name in (("invalid", "kb_0.1.1_source.tar.gz"), ("a" * 40, "../outside.tar.gz")):
            with self.subTest(commit=commit), self.assertRaises(ValueError):
                release.formula_text("v0.1.1", commit, name, "0" * 64)

    def test_tap_retry_is_noop_and_update_uses_expected_file_sha(self):
        text = release.formula_text("v0.1.1", "a" * 40, "kb_0.1.1_source.tar.gz", "0" * 64)
        env = {"GITHUB_REF_NAME": "v0.1.1", "GITHUB_SHA": "a" * 40, "GH_TOKEN": "synthetic-placeholder"}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "kb.rb"
            path.write_text(text)
            existing = {"sha": "synthetic-file-sha", "content": base64.b64encode(text.encode()).decode()}
            with patch.dict(os.environ, env), patch.object(release, "command"), patch.object(release.subprocess, "run") as run:
                run.return_value = subprocess.CompletedProcess([], 0, json.dumps(existing), "")
                release.publish_tap(path)
                self.assertEqual(1, run.call_count)
            existing["content"] = base64.b64encode(text.replace("0.1.1", "0.1.0").encode()).decode()
            with patch.dict(os.environ, env), patch.object(release, "command"), patch.object(release.subprocess, "run") as run:
                run.return_value = subprocess.CompletedProcess([], 0, json.dumps(existing), "")
                release.publish_tap(path)
                payload = json.loads(run.call_args.kwargs["input"])
                self.assertEqual("synthetic-file-sha", payload["sha"])
                self.assertEqual(release.BOT, payload["author"])
                self.assertEqual(release.BOT, payload["committer"])

    def test_tap_rejects_wrong_commit_extra_ruby_and_prereleases(self):
        text = release.formula_text("v0.1.1", "a" * 40, "kb_0.1.1_source.tar.gz", "0" * 64)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "kb.rb"
            for body, tag, commit in ((text, "v0.1.1", "b" * 40), (text + "system 'bad'\n", "v0.1.1", "a" * 40),
                                      (text, "v0.1.1-rc.1", "a" * 40)):
                path.write_text(body)
                with patch.dict(os.environ, {"GITHUB_REF_NAME": tag, "GITHUB_SHA": commit, "GH_TOKEN": "synthetic"}), \
                        self.assertRaises(ValueError), patch.object(release.subprocess, "run") as run:
                    release.publish_tap(path)
                run.assert_not_called()

    def test_old_release_retry_cannot_downgrade_the_tap(self):
        text = release.formula_text("v0.1.1", "a" * 40, "kb_0.1.1_source.tar.gz", "0" * 64)
        later = release.formula_text("v0.2.0", "b" * 40, "kb_0.2.0_source.tar.gz", "0" * 64)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "kb.rb"
            path.write_text(text)
            existing = {"sha": "synthetic-file-sha", "content": base64.b64encode(later.encode()).decode()}
            with patch.dict(os.environ, {"GITHUB_REF_NAME": "v0.1.1", "GITHUB_SHA": "a" * 40, "GH_TOKEN": "synthetic"}), \
                    patch.object(release, "command"), patch.object(release.subprocess, "run") as run:
                run.return_value = subprocess.CompletedProcess([], 0, json.dumps(existing), "")
                with self.assertRaises(ValueError):
                    release.publish_tap(path)
                self.assertEqual(1, run.call_count)


if __name__ == "__main__":
    unittest.main()

"""Exercise the future Go gate with temporary, synthetic packages only."""

import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import quality


@unittest.skipUnless(shutil.which("go"), "Go toolchain required for gate integration tests")
class GoGateTests(unittest.TestCase):
    def fixture(self, directory, test_body=None, fuzz=False):
        root = Path(directory)
        (root / "scripts").mkdir()
        shutil.copy2(quality.ROOT / "scripts/quality.py", root / "scripts/quality.py")
        (root / "go.mod").write_text("module example.com/gatefixture\n\ngo 1.27.3\n")
        (root / "choose.go").write_text(
            "package gatefixture\n\n"
            "func Choose(b bool) int { if b { return 1 }; return 2 }\n"
        )
        if test_body is not None:
            content = 'package gatefixture\n\nimport "testing"\n\n'
            content += "func TestChoose(t *testing.T) { " + test_body + " }\n"
            if fuzz:
                content += (
                    "func FuzzChoose(f *testing.F) { f.Add(true); "
                    "f.Fuzz(func(t *testing.T, b bool) { "
                    "got := Choose(b); if got < 1 || got > 2 { t.Fatal(got) } }) }\n"
                )
            (root / "choose_test.go").write_text(content)
        return root

    def invoke(self, root, command):
        return subprocess.run(
            [sys.executable, "scripts/quality.py", command], cwd=root,
            env=dict(os.environ, GOTOOLCHAIN="local"), text=True,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=180,
        )

    def test_go_source_without_named_tests_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            result = self.invoke(self.fixture(directory), "test")
            self.assertNotEqual(0, result.returncode, result.stdout)
            self.assertIn("no named Go test passed", result.stdout)

    def test_real_low_coverage_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = self.fixture(directory, 'if Choose(true) != 1 { t.Fatal("wrong result") }')
            result = self.invoke(root, "test")
            self.assertNotEqual(0, result.returncode, result.stdout)
            self.assertIn("below 80.0%", result.stdout)

    def test_real_tests_coverage_build_and_fuzz_are_executed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = self.fixture(
                directory,
                'if Choose(true) != 1 || Choose(false) != 2 { t.Fatal("wrong result") }',
                fuzz=True,
            )
            for command in ("test", "build", "fuzz"):
                with self.subTest(command=command):
                    result = self.invoke(root, command)
                    self.assertEqual(0, result.returncode, result.stdout)
                    self.assertNotIn("NOT IMPLEMENTED", result.stdout)
                    if command == "fuzz":
                        self.assertIn("Fuzz smoke targets executed: 1", result.stdout)

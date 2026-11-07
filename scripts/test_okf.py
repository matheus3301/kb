"""Regression checks for the public-format dependency gate."""

from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import check_okf


class OKFBoundaryTests(unittest.TestCase):
    def test_runtime_coupling_is_rejected(self):
        for dependency in ("github.com/matheus3301/kb/internal/api",
                           "github.com/matheus3301/kb/internal/core",
                           "github.com/spf13/cobra", "log/slog"):
            with self.subTest(dependency=dependency):
                packages = [{"ImportPath": dependency, "Standard": dependency == "log/slog"}]
                # Standard-library logging is not a runtime backend dependency.
                self.assertEqual(len(check_okf.dependency_findings(packages)),
                                 0 if dependency == "log/slog" else 1)

    def test_required_acceptance_stops_on_failed_external_consumer(self):
        with tempfile.TemporaryDirectory(prefix="okf-gate-") as temporary:
            root = Path(temporary)
            (root / "scripts").mkdir()
            (root / "scripts/check_okf.py").write_text(
                "raise SystemExit('OKF_BOUNDARY_REJECTED')\n")
            result = subprocess.run(
                ["make", "-f", str(check_okf.ROOT / "Makefile"), "acceptance",
                 f"PYTHON={sys.executable}"],
                cwd=root, text=True, stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT, timeout=30,
            )
            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn("OKF_BOUNDARY_REJECTED", result.stdout)
            self.assertNotIn("go build", result.stdout)

    def test_json_stream_checks_all_packages(self):
        packages = check_okf.decode_packages(
            '{"ImportPath":"bytes","Standard":true}\n'
            '{"ImportPath":"github.com/matheus3301/kb/okf"}\n'
            '{"ImportPath":"go.yaml.in/yaml/v3"}\n'
            '{"ImportPath":"github.com/modelcontextprotocol/go-sdk/mcp"}\n')
        findings = check_okf.dependency_findings(packages)
        self.assertEqual(findings, ["OKF runtime dependency is outside its format boundary: github.com/modelcontextprotocol/go-sdk/mcp"])


if __name__ == "__main__":
    unittest.main()

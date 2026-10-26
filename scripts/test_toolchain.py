"""Mutation regressions for authoritative pins and required development gates."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import toolchain


class ToolchainTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        for name in ("mise.toml", "go.mod", "scripts/tools.json", "site/package.json",
                     "site/pnpm-lock.yaml", "Makefile", ".github/workflows/ci.yml",
                     ".github/workflows/release.yml"):
            target = self.root / name
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(toolchain.ROOT / name, target)

    def mutate(self, name, before, after):
        path = self.root / name
        original = path.read_text()
        self.assertIn(before, original)
        path.write_text(original.replace(before, after))
        self.addCleanup(path.write_text, original)

    def test_consistent_toolchain_is_accepted(self):
        self.assertEqual([], toolchain.findings(self.root))

    def test_github_outputs_match_action_version_contract(self):
        output = self.root / "github-output"
        subprocess.run([sys.executable, str(toolchain.ROOT / "scripts/toolchain.py"), "github-outputs"],
                       env=dict(os.environ, GITHUB_OUTPUT=str(output)), check=True,
                       capture_output=True, text=True)
        values = dict(line.split("=", 1) for line in output.read_text().splitlines())
        pins = toolchain.versions()
        self.assertEqual("v" + pins["golangci-lint"], values.pop("golangci-lint"))
        self.assertEqual({name: pin for name, pin in pins.items() if name != "golangci-lint"}, values)

    def test_go_and_pnpm_ecosystem_pin_drift_is_rejected(self):
        pins = toolchain.versions(self.root)
        for name, before, after in (
            ("go.mod", "go " + pins["go"], "go 99.98.97"),
            ("go.mod", "go " + pins["go"], "go " + pins["go"] + "\ntoolchain go99.98.97"),
            ("site/package.json", "pnpm@" + pins["pnpm"], "pnpm@99.98.97"),
            ("site/package.json", '"packageManager"', '"removedPackageManager"'),
        ):
            with self.subTest(name=name, after=after):
                path = self.root / name
                original = path.read_text()
                self.mutate(name, before, after)
                self.assertTrue(toolchain.pin_findings(self.root))
                path.write_text(original)

    def test_unpinned_missing_and_additional_tools_are_rejected(self):
        pins = toolchain.versions(self.root)
        path = self.root / "mise.toml"
        original = path.read_text()
        for before, after in (
            ('node = "' + pins["node"] + '"', 'node = "22"'),
            ('"npm:pnpm" = "' + pins["pnpm"] + '"', '"npm:pnpm" = "latest"'),
            ('"npm:pnpm" = "', 'pnpm = "'),
            ('go = "' + pins["go"] + '"', ''),
            ('[tools]', '[tools]\nextra = "1.2.3"'),
            ('python = "' + pins["python"] + '"', 'python = "3"'),
            ('python = "' + pins["python"] + '"', 'python = "3.10"'),
        ):
            with self.subTest(after=after):
                self.mutate("mise.toml", before, after)
                self.assertTrue(toolchain.pin_findings(self.root))
                path.write_text(original)

    def test_obsolete_split_pin_and_lock_files_are_rejected(self):
        for name in (".tool-versions", ".golangci-lint-version", "site/package-lock.json"):
            with self.subTest(name=name):
                path = self.root / name
                path.write_text("synthetic obsolete configuration\n")
                self.assertIn(f"obsolete split toolchain file: {name}",
                              toolchain.pin_findings(self.root))
                path.unlink()
        (self.root / "site/pnpm-lock.yaml").unlink()
        self.assertIn("site must commit pnpm-lock.yaml", toolchain.pin_findings(self.root))

    def test_module_mapping_cannot_restore_an_independent_version(self):
        path = self.root / "scripts/tools.json"
        mappings = json.loads(path.read_text())
        mappings["actionlint"]["version"] = "v99.98.97"
        path.write_text(json.dumps(mappings))
        self.assertIn("scripts/tools.json must contain module mappings only, without versions",
                      toolchain.pin_findings(self.root))

    def test_every_ci_tool_reader_rejects_independent_pins(self):
        for filename in ("ci.yml", "release.yml"):
            workflow = (self.root / ".github/workflows" / filename).read_text()
            for name in ("python", "go", "node", "pnpm", "goreleaser", "golangci-lint"):
                output = "${{ steps.toolchain.outputs." + name + " }}"
                if output not in workflow:
                    continue
                with self.subTest(filename=filename, name=name):
                    altered = workflow.replace(output, '"99.98.97"')
                    self.assertTrue(toolchain.workflow_findings(altered))
            for before, after in (
                ("      - id: toolchain\n", "      - id: unrelated\n"),
                ("run: python3 scripts/toolchain.py github-outputs", "run: true"),
                ("go-version: ${{ steps.toolchain.outputs.go }}", "go-version-file: go.mod"),
            ):
                with self.subTest(filename=filename, after=after):
                    self.assertTrue(toolchain.workflow_findings(workflow.replace(before, after)))
            named = workflow.replace("- uses: actions/setup-go@", "- name: Install Go\n        uses: actions/setup-go@")
            self.assertEqual([], toolchain.workflow_findings(named))
            self.assertTrue(toolchain.workflow_findings(named.replace(
                "${{ steps.toolchain.outputs.go }}", '"99.98.97"')))

    def test_local_website_gate_removal_and_weakening_are_rejected(self):
        path = self.root / "Makefile"
        original = path.read_text()
        commands = (
            "pnpm --dir site install --frozen-lockfile",
            "pnpm --dir site audit --audit-level=high",
            "pnpm --dir site run check",
            "KB_SITE_BASE=/kb/ pnpm --dir site run build",
            "pnpm --dir site exec playwright install chromium",
            "KB_SITE_BASE=/kb/ pnpm --dir site test",
        )
        mutations = [("\t" + command + "\n", "") for command in commands]
        mutations += [("--frozen-lockfile", "--no-frozen-lockfile"),
                      ("audit --audit-level=high", "audit --audit-level=critical"),
                      ("audit --audit-level=high", "audit --prod --audit-level=high"),
                      ("audit --audit-level=high", "audit --audit-level=high --ignore-registry-errors"),
                      ("pnpm --dir site test", "pnpm --dir site test || true")]
        for before, after in mutations:
            with self.subTest(before=before, after=after):
                self.mutate("Makefile", before, after)
                self.assertTrue(toolchain.site_findings(self.root))
                path.write_text(original)


if __name__ == "__main__":
    unittest.main()

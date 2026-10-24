import argparse
from pathlib import Path
import tarfile
import tempfile
import unittest

import release


class ReleaseTests(unittest.TestCase):
    def test_archives_omit_local_identity_and_preserve_install_modes(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "release.tar.gz"
            release.write_archive(path, {"kb": b"synthetic binary", "README.md": b"Public docs"}, 1234)
            with tarfile.open(path) as archive:
                for item in archive.getmembers():
                    self.assertEqual((0, 0, "", ""), (item.uid, item.gid, item.uname, item.gname))
                    self.assertEqual(1234, item.mtime)
                self.assertEqual(0o755, archive.getmember("kb").mode)
                self.assertEqual(0o644, archive.getmember("README.md").mode)

    def test_archive_traversal_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(ValueError):
                release.write_archive(Path(directory) / "bad.tar.gz", {"../private": b"data"}, 0)

    def test_release_version_cannot_inject_build_arguments(self):
        self.assertEqual("v0.1.0", release.version_value("v0.1.0"))
        self.assertEqual("v0.1.0-rc.1", release.version_value("v0.1.0-rc.1"))
        for value in ("latest", "0.1.0", "v0.1.0 -X private=value"):
            with self.subTest(value=value), self.assertRaises(argparse.ArgumentTypeError):
                release.version_value(value)

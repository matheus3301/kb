"""Behavior checks for the standalone measurement harness."""
from pathlib import Path
import stat
import sys
import tempfile
import unittest
from unittest import mock

import benchmark


class BenchmarkTests(unittest.TestCase):
    def test_cli_spools_large_responses_without_pipe_deadlock(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            executable = root / 'synthetic-cli'
            executable.write_text('#!' + sys.executable + '\nimport json\nprint(json.dumps({"ok":True,"data":{"body":"x"*1048576}}))\n')
            executable.chmod(stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)
            result, elapsed, rss = benchmark.measured_cli(str(executable), root, ['read'])
            self.assertEqual(len(result['data']['body']), 1048576)
            self.assertGreaterEqual(elapsed, 0)
            self.assertGreater(rss, 0)

    def test_fixture_uses_exact_utf8_note_sizes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with mock.patch.object(benchmark, 'measured_cli'):
                benchmark.fixture('unused', root, 4)
            notes = sorted(root.glob('group-*/*.md'))
            self.assertEqual(len(notes), 4)
            for note in notes:
                self.assertEqual(len(note.read_bytes()), 2048)
                self.assertIn('ação', note.read_text())


    def test_rebuild_measures_real_command_and_checks_fresh_marker(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            executable = root / 'synthetic-rebuild'
            executable.write_text('#!' + sys.executable + "\nimport json,sys\nfrom pathlib import Path\nif sys.argv[-2:]!=['index','rebuild']: sys.exit(2)\nroot=Path(sys.argv[sys.argv.index('--root')+1])\nwith (root/'calls').open('a') as output: output.write('x')\nprint(json.dumps({'ok':True,'data':{'index_stale':False}}))\n")
            executable.chmod(stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)
            result = benchmark.rebuild(str(executable), root, 4, 3)
            self.assertEqual(result['cli']['index_rebuild']['samples'], 3)
            self.assertEqual((root / 'calls').read_text(), 'xxx')

    def test_percentile_uses_declared_nearest_rank(self):
        self.assertEqual(benchmark.percentile([5, 1, 4, 2, 3], .95), 5)
        self.assertEqual(benchmark.percentile(list(range(1, 21)), .95), 19)
        self.assertEqual(benchmark.summarize([1, 2, 3])['p50_ms'], 2)

    def test_mcp_runs_real_stdio_framing_and_shutdown(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            executable = root / 'synthetic-mcp'
            executable.write_text('#!' + sys.executable + '''
import json,sys
for line in sys.stdin:
    request=json.loads(line)
    if 'id' not in request: continue
    result={'structuredContent':{'ok':True,'data':{'total':1}}} if request['method']=='tools/call' else {}
    print(json.dumps({'jsonrpc':'2.0','id':request['id'],'result':result}),flush=True)
''')
            executable.chmod(stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)
            connection = benchmark.MCP(str(executable), root)
            try:
                result, elapsed = connection.call('search', {'query': 'synthetic'})
                self.assertEqual(result['data']['total'], 1)
                self.assertGreaterEqual(elapsed, 0)
            finally:
                self.assertGreater(connection.close(), 0)


if __name__ == '__main__':
    unittest.main()

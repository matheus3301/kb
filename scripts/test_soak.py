"""Consumer-visible accounting and fidelity failures must fail qualification."""
from collections import Counter
import copy
import json
import io
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import soak


class SoakTests(unittest.TestCase):
    def test_cas_rejects_zero_or_multiple_winners_and_missing_workers(self):
        for rows in ([{'won': False}] * 4, [{'won': True}] * 4,
                     [{'won': True}] + [{'won': False}] * 2):
            with self.subTest(rows=rows), self.assertRaisesRegex(RuntimeError, 'one winner'):
                soak.check_round(4, rows)
        winner = {'won': True, 'body': 'confirmed'}
        self.assertEqual(soak.check_round(4, [winner] + [{'won': False}] * 3), winner)

    def test_replay_requires_identical_receipt_not_only_replayed_flag(self):
        receipt = {'replayed': False, 'generation': 'one', 'idempotency_key': 'key',
                   'changes': [{'path': 'a.md', 'after_revision': 'confirmed'}]}
        soak.check_replay(receipt, dict(receipt, replayed=True))
        for changed in (dict(receipt), dict(receipt, replayed=True, generation='two'),
                        dict(receipt, replayed=True, changes=[])):
            with self.subTest(changed=changed), self.assertRaisesRegex(RuntimeError, 'replay'):
                soak.check_replay(receipt, changed)

    def test_missing_conflicts_or_inflated_replays_fail_accounting(self):
        counts = Counter(mutations=15, replays=15, conflicts=9, reads=30, searches=3)
        soak.check_counts(4, 3, counts)
        for field in counts:
            changed = counts.copy()
            changed[field] += 1
            with self.subTest(field=field), self.assertRaisesRegex(RuntimeError, field):
                soak.check_counts(4, 3, changed)

    def test_minimum_work_excludes_setup_backup_and_restore_counts(self):
        run = {'seed': 1729, 'ok': True, 'failures': [],
               'counts': {'mutations': 5, 'replays': 5, 'setup_mutations': 100,
                          'restored_mutations': 100, 'restored_replays': 100}}
        self.assertEqual(soak.qualification([run], [1729], 10), 10)
        with self.assertRaisesRegex(RuntimeError, 'below minimum'):
            soak.qualification([run], [1729], 11)
        with self.assertRaisesRegex(RuntimeError, 'missing or reordered'):
            soak.qualification([run], [1729, 2718], 10)
        failed = dict(run, ok=False, failures=['lost receipt'])
        with self.assertRaisesRegex(RuntimeError, 'failed'):
            soak.qualification([failed], [1729], 1)

    def test_inventory_detects_config_receipt_and_binary_source_changes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, data in {'note.md': b'---\ntype: Reference\n---\nconfirmed\n',
                               'raw/source.bin': b'\x00\xff', '.kb/config.toml': b'version = 1\n',
                               '.kb/state/receipts/one.json': b'confirmed receipt',
                               '.kb/cache/derived': b'ignored', '.kb/backups/a.tar': b'ignored'}.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
            before = soak.inventory(root)
            self.assertEqual(set(before), {'note.md', 'raw/source.bin', '.kb/config.toml', '.kb/state/receipts/one.json'})
            for name in before:
                path = root / name
                old = path.read_bytes()
                path.write_bytes(old + b'corruption')
                self.assertNotEqual(before, soak.inventory(root), name)
                path.write_bytes(old)

    def test_lost_receipts_log_changes_and_source_mutation_are_detected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            cli = soak.CLI('unused', root, {}, 1)
            workload = soak.Workload(cli, 1729, 4)
            (root / 'raw').mkdir()
            (root / 'raw/source.bin').write_bytes(workload.source)
            receipts = root / '.kb/state/receipts'
            receipts.mkdir(parents=True)
            receipt = {'idempotency_key': 'synthetic/1', 'replayed': False,
                       'dry_run': False, 'changes': [{'path': 'a.md'}]}
            workload.remember(receipt)
            saved = receipts / 'one.json'
            saved.write_text(json.dumps({'receipt': receipt}))
            log = root / 'log.md'
            event = '- Record synthetic soak effect synthetic/1 (process:reliability-soak)\n'
            log.write_text(event)
            workload.initial_log = event.encode()
            workload.verify_receipts()
            saved.unlink()
            with self.assertRaisesRegex(RuntimeError, 'durable receipt'):
                workload.verify_receipts()
            saved.write_text(json.dumps({'receipt': receipt}))
            changed = copy.deepcopy(receipt)
            changed['changes'] = []
            saved.write_text(json.dumps({'receipt': changed}))
            with self.assertRaisesRegex(RuntimeError, 'durable receipt'):
                workload.verify_receipts()
            saved.write_text(json.dumps({'receipt': receipt}))
            log.write_text(event * 2)
            with self.assertRaisesRegex(RuntimeError, 'immutable event log'):
                workload.verify_receipts()
            log.write_text(event)
            (root / 'raw/source.bin').write_bytes(b'changed')
            with self.assertRaisesRegex(RuntimeError, 'immutable source'):
                workload.verify_receipts()

    def test_only_revision_conflict_with_exit_four_is_a_safe_cas_loss(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'synthetic-cli'
            binary.write_text('#!' + sys.executable + '''
import json, sys
case = sys.argv[-1]
code, status = {
    'cas': ('REVISION_CONFLICT', 4),
    'wrong-exit': ('REVISION_CONFLICT', 6),
    'key': ('IDEMPOTENCY_CONFLICT', 4),
    'lock': ('LOCK_TIMEOUT', 6),
}[case]
print(json.dumps({'schema_version': 'kb.v1', 'ok': False, 'error': {'code': code}}))
sys.exit(status)
''')
            binary.chmod(0o700)
            cli = soak.CLI(str(binary), root, {}, 5)
            self.assertIsNone(cli.call('cas', conflict=True))
            for case in ('wrong-exit', 'key', 'lock'):
                with self.subTest(case=case), self.assertRaisesRegex(RuntimeError, 'unexpected exit'):
                    cli.call(case, conflict=True)
            with self.assertRaisesRegex(RuntimeError, 'unexpected exit'):
                cli.call('cas')
            self.assertEqual(cli.counts, Counter(conflict=1, error=4))

    def test_enabled_log_requires_one_effect_per_receipt_and_original_actor(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            log = root / 'log.md'
            events = [
                '- Record synthetic soak effect synthetic/1 (process:reliability-soak): [a](a.md)\n',
                '- Record synthetic soak effect synthetic/2 (process:reliability-soak): [b](b.md)\n']
            log.write_text(''.join(events))
            soak.check_log(root, ['synthetic/1', 'synthetic/2'], True, b'')
            for text in (events[0], events[0] * 2,
                         ''.join(events).replace('process:reliability-soak', 'incorrect-actor'),
                         ''.join(events) + events[0]):
                log.write_text(text)
                with self.subTest(text=text), self.assertRaisesRegex(RuntimeError, 'log'):
                    soak.check_log(root, ['synthetic/1', 'synthetic/2'], True, b'')

    def test_source_provenance_cannot_disappear_after_a_confirmed_write(self):
        note = {'note': {'metadata': {'sources': [{'resource': '/raw/source.bin'}]}}}
        soak.check_provenance(note)
        for sources in (None, [], [{'resource': '/raw/replacement.bin'}]):
            note['note']['metadata']['sources'] = sources
            with self.subTest(sources=sources), self.assertRaisesRegex(RuntimeError, 'provenance'):
                soak.check_provenance(note)

    def test_real_timeout_and_os_error_reports_exclude_private_paths(self):
        with tempfile.TemporaryDirectory(prefix='private-report-fixture-') as directory:
            root = Path(directory)
            executable = root / 'private-executable'
            executable.write_text('#!' + sys.executable + '\nimport time\ntime.sleep(20)\n')
            harness = Path(soak.__file__).resolve()
            for category in ('timeout', 'os-error'):
                with self.subTest(category=category):
                    executable.chmod(0o700 if category == 'timeout' else 0o600)
                    output = root / (category + '.json')
                    result = subprocess.run([
                        sys.executable, str(harness), '--binary', str(executable),
                        '--duration', '0.1', '--seeds', '1729', '--minimum-work', '1',
                        '--command-timeout', '0.000001' if category == 'timeout' else '1',
                        '--output', str(output)], capture_output=True, text=True, timeout=5)
                    self.assertEqual(result.returncode, 1)
                    report_text = output.read_text()
                    report = json.loads(report_text)
                    self.assertIs(report['ok'], False)
                    failure = report['runs'][0]['failures'][0]
                    self.assertEqual(failure['category'], category)
                    self.assertEqual(failure['operation'], 'init')
                    self.assertEqual(failure['error_type'],
                                     'TimeoutExpired' if category == 'timeout' else 'PermissionError')
                    for private in (str(executable), str(root), str(Path.home()),
                                    str(tempfile.gettempdir()), 'kb-soak-', 'private-report-fixture-'):
                        self.assertNotIn(private, report_text)
                        self.assertNotIn(private, result.stdout)
                    self.assertNotIn('time.sleep', report_text)

    def test_unexpected_error_messages_and_cli_response_codes_are_not_public(self):
        marker = '/synthetic-private-home/do-not-publish'
        result = soak.public_failure(OSError(13, marker, marker))
        self.assertEqual(result['category'], 'os-error')
        self.assertNotIn(marker, json.dumps(result))
        result = soak.public_failure(RuntimeError(marker))
        self.assertEqual(result['category'], 'unexpected-error')
        self.assertNotIn(marker, json.dumps(result))
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory) / 'untrusted-response'
            value = {'schema_version': 'kb.v1', 'ok': False, 'error': {'code': marker, 'message': marker}}
            binary.write_text('#!' + sys.executable + '\nimport sys\nprint(' + repr(json.dumps(value)) + ')\nsys.exit(4)\n')
            binary.chmod(0o700)
            with self.assertRaises(soak.CommandFailure) as caught:
                soak.CLI(str(binary), Path(directory), {}, 2).call('patch', conflict=True)
            report = soak.public_failure(caught.exception)
            self.assertEqual(report['check'], 'unexpected exit or error code')
            self.assertNotIn(marker, json.dumps(report))

    def test_binary_identity_preflight_failures_are_path_free_and_start_no_runs(self):
        with tempfile.TemporaryDirectory(prefix='private-preflight-fixture-') as directory:
            root = Path(directory)
            harness = Path(soak.__file__).resolve()
            cases = [('missing', 'FileNotFoundError'), ('directory', 'IsADirectoryError')]
            if os.geteuid() != 0:
                cases.append(('unreadable', 'PermissionError'))
            for case, error_type in cases:
                with self.subTest(case=case):
                    binary = root / case
                    if case == 'directory':
                        binary.mkdir()
                    elif case == 'unreadable':
                        binary.write_bytes(b'Synthetic unreadable executable')
                        binary.chmod(0)
                    output = root / (case + '.json')
                    result = subprocess.run([
                        sys.executable, str(harness), '--binary', str(binary),
                        '--output', str(output), '--duration', '0.01', '--seeds', '1729',
                        '--writers', '4', '--minimum-work', '1'],
                        capture_output=True, text=True, timeout=5)
                    self.assertEqual(result.returncode, 1)
                    self.assertEqual(result.stderr, '')
                    text = output.read_text()
                    report = json.loads(text)
                    self.assertIs(report['ok'], False)
                    self.assertIsNone(report['binary_revision'])
                    self.assertEqual(report['runs'], [])
                    self.assertEqual(report['acknowledged_work'], 0)
                    failure = report['failures'][0]
                    self.assertEqual(failure['category'], 'os-error')
                    self.assertEqual(failure['error_type'], error_type)
                    self.assertEqual(failure['operation'], 'binary-preflight')
                    self.assertEqual(failure['check'], 'binary-identity-read')
                    self.assertEqual(json.loads(result.stdout)['failures'], report['failures'])
                    for private in (str(binary), str(root), str(harness), str(harness.parent),
                                    str(Path.home()), str(tempfile.gettempdir()), 'Traceback'):
                        for surface in (text, result.stdout, result.stderr):
                            self.assertNotIn(private, surface)

    def test_unwritable_output_has_sanitized_nonzero_stdout_without_traceback(self):
        with tempfile.TemporaryDirectory(prefix='private-output-fixture-') as directory:
            root = Path(directory)
            blocker = root / 'not-a-directory'
            blocker.write_bytes(b'Preserve this synthetic file')
            harness = Path(soak.__file__).resolve()
            binary = root / 'missing-binary'
            output = blocker / 'report.json'
            result = subprocess.run([
                sys.executable, str(harness), '--binary', str(binary),
                '--output', str(output), '--duration', '0.01', '--seeds', '1729',
                '--minimum-work', '1'], capture_output=True, text=True, timeout=5)
            self.assertEqual(result.returncode, 1)
            self.assertEqual(result.stderr, '')
            report = json.loads(result.stdout)
            self.assertIs(report['ok'], False)
            self.assertEqual([failure['operation'] for failure in report['failures']],
                             ['binary-preflight', 'report-output'])
            self.assertEqual(report['failures'][1]['check'], 'report-output-write')
            self.assertEqual(blocker.read_bytes(), b'Preserve this synthetic file')
            for private in (str(binary), str(output), str(root), str(harness), str(Path.home()), 'Traceback'):
                self.assertNotIn(private, result.stdout + result.stderr)

    def test_failed_report_publication_cannot_report_a_successful_qualification(self):
        with tempfile.TemporaryDirectory() as directory:
            report = {'ok': True, 'runs': [], 'acknowledged_work': 10000, 'failures': []}
            stdout = io.StringIO()
            with mock.patch('sys.stdout', stdout):
                status = soak.publish_report(report, directory)
            self.assertEqual(status, 1)
            result = json.loads(stdout.getvalue())
            self.assertIs(result['ok'], False)
            self.assertEqual(result['acknowledged_work'], 10000)
            self.assertEqual(result['failures'][0]['operation'], 'report-output')
            self.assertEqual(result['failures'][0]['category'], 'os-error')
            self.assertNotIn(directory, stdout.getvalue())

    def _workspace_failure_report(self, root, factory):
        binary = root / 'private-executable'
        binary.write_bytes(b'Synthetic preflight bytes; no process is executed')
        output = root / 'private-report.json'
        stdout, stderr = io.StringIO(), io.StringIO()
        args = ['soak', '--binary', str(binary), '--output', str(output),
                '--duration', '0.1', '--seeds', '1729', '--minimum-work', '1']
        with mock.patch.object(soak.tempfile, 'TemporaryDirectory', factory), \
                mock.patch.object(sys, 'argv', args), \
                mock.patch.object(sys, 'stdout', stdout), mock.patch.object(sys, 'stderr', stderr):
            status = soak.main()
        text = output.read_text()
        report = json.loads(text)
        self.assertEqual(status, 1)
        self.assertIs(report['ok'], False)
        self.assertIs(json.loads(stdout.getvalue())['ok'], False)
        self.assertEqual(stderr.getvalue(), '')
        self.assertLess(len(text), 8192)
        for private in (str(root), str(binary), str(output), str(Path.home()), 'Traceback'):
            for surface in (text, stdout.getvalue(), stderr.getvalue()):
                self.assertNotIn(private, surface)
        return report

    def test_workspace_creation_and_entry_errors_publish_bounded_failures(self):
        for phase in ('creation', 'entry'):
            with self.subTest(phase=phase), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                failure = PermissionError(13, 'private workspace access denied', str(root / 'private-source'))
                factory = mock.Mock()
                if phase == 'creation':
                    factory.side_effect = failure
                else:
                    factory.return_value = mock.MagicMock()
                    factory.return_value.__enter__.side_effect = failure
                report = self._workspace_failure_report(root, factory)
                self.assertEqual(report['acknowledged_work'], 0)
                self.assertEqual(report['totals'], {})
                self.assertEqual(report['runs'][0]['counts'], {})
                self.assertEqual(report['runs'][0]['failures'], [{
                    'error_type': 'PermissionError', 'category': 'os-error', 'errno': 13,
                    'operation': 'workspace', 'check': 'workspace-create-or-enter'}])

    def test_workspace_cleanup_keeps_progress_and_prior_timeout(self):
        for prior_timeout in (False, True):
            with self.subTest(prior_timeout=prior_timeout), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                workspace = root / 'workspace'
                workspace.mkdir()
                manager = mock.MagicMock()
                manager.__enter__.return_value = str(workspace)
                manager.__exit__.side_effect = OSError(5, 'private cleanup failed', str(root / 'private-source'))
                clock = [0.0]
                progress = mock.Mock(spec=soak.Workload)
                progress.counts, progress.receipts = Counter(), {}

                # Unit progress fixture; the observed behavior is failure reporting,
                # not an application/soak pass manufactured from mocked operations.
                def construct(cli, *_):
                    def setup():
                        progress.counts['setup_mutations'] = 5
                        progress.receipts.update({str(i): {} for i in range(5)})
                        cli.acknowledgements['mutations'] = 5
                        cli.counts['success'] = 6
                        cli.timings.update(init={'calls': 1, 'total_seconds': .01, 'max_seconds': .01},
                                           create={'calls': 5, 'total_seconds': .25, 'max_seconds': .05})

                    def round_work(*_):
                        progress.counts.update(mutations=5, replays=5, conflicts=3, reads=10, searches=1)
                        progress.receipts.update({str(i): {} for i in range(5, 10)})
                        cli.acknowledgements.update(mutations=5, replays=5)
                        cli.counts.update(success=21, conflict=3)
                        cli.timings.update(patch={'calls': 13, 'total_seconds': .65, 'max_seconds': .05},
                                           read={'calls': 10, 'total_seconds': .2, 'max_seconds': .02},
                                           search={'calls': 1, 'total_seconds': .01, 'max_seconds': .01})
                        clock[0] = 1.0

                    def backup(number):
                        cli.counts['success'] += 1
                        cli.timings['backup'] = {'calls': 1, 'total_seconds': .01, 'max_seconds': .01}
                        if prior_timeout:
                            return cli.call('restore')
                        progress.counts.update(backup_checks=1, restored_mutations=1, restored_replays=2)
                        cli.acknowledgements.update(mutations=1, replays=2)
                        cli.counts['success'] += 8
                        cli.timings.update(restore={'calls': 1, 'total_seconds': .1, 'max_seconds': .1},
                                           index={'calls': 1, 'total_seconds': .01, 'max_seconds': .01})
                        cli.timings['patch']['calls'] += 3
                        cli.timings['patch']['total_seconds'] += .15
                        cli.timings['read']['calls'] += 3
                        cli.timings['read']['total_seconds'] += .06
                        return {'round': number, 'files': 19, 'bytes': 4096}

                    progress.setup.side_effect = setup
                    progress.round.side_effect = round_work
                    progress.backup.side_effect = backup
                    return progress

                def stored(_):
                    return {'receipt_files': len(progress.receipts), 'receipt_bytes': 4096, 'log_bytes': 23}

                timeout = subprocess.TimeoutExpired([str(root / 'private-executable')], 30,
                                                    output=str(root / 'private-stdout'),
                                                    stderr=str(root / 'private-stderr'))
                with mock.patch.object(soak, 'Workload', side_effect=construct), \
                        mock.patch.object(soak, 'storage', side_effect=stored), \
                        mock.patch.object(soak.time, 'monotonic', side_effect=lambda: clock[0]), \
                        mock.patch.object(soak.subprocess, 'run', side_effect=timeout):
                    report = self._workspace_failure_report(root, mock.Mock(return_value=manager))
                run = report['runs'][0]
                self.assertIs(run['ok'], False)
                self.assertEqual(report['acknowledged_work'], 10)
                self.assertEqual((report['totals']['mutations'], report['totals']['replays']), (5, 5))
                self.assertEqual(run['confirmed_receipts'], 10)
                self.assertEqual(run['rounds'], 1)
                self.assertEqual(run['work_seconds'], 1.0)
                self.assertEqual(run['timings']['create'], {'calls': 5, 'total_seconds': .25, 'max_seconds': .05})
                self.assertEqual(run['commands']['error'], int(prior_timeout))
                self.assertEqual(run['counts'].get('backup_checks', 0), 0 if prior_timeout else 1)
                self.assertEqual(len(run['backup_checks']), 0 if prior_timeout else 1)
                self.assertEqual(run['all_root_acknowledgements'],
                                 {'mutations': 10 if prior_timeout else 11, 'replays': 5 if prior_timeout else 7})
                if prior_timeout:
                    self.assertEqual(run['failures'][0], {
                        'error_type': 'TimeoutExpired', 'category': 'timeout',
                        'check': 'cli-command-timeout', 'operation': 'restore'})
                    self.assertEqual(run['timings']['restore']['calls'], 1)
                    self.assertNotIn('final_storage', run)
                self.assertEqual(len(run['failures']), 2 if prior_timeout else 1)
                self.assertEqual(run['failures'][-1], {
                    'error_type': 'OSError', 'category': 'os-error', 'errno': 5,
                    'operation': 'workspace', 'check': 'workspace-cleanup'})


if __name__ == '__main__':
    unittest.main()

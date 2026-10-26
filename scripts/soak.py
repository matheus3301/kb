#!/usr/bin/env python3
"""Exercise real CLI processes on disposable synthetic knowledge, never an existing root."""
import argparse
from concurrent.futures import ThreadPoolExecutor
from collections import Counter
import json
import math
from pathlib import Path
import platform
import random
import shutil
import subprocess
import tempfile
import threading
import time

from acceptance import revision


class CheckFailure(RuntimeError):
    """An invariant label composed only from harness-controlled synthetic values."""

    def __init__(self, check):
        self.check = check
        super().__init__(check)


class CommandFailure(RuntimeError):
    def __init__(self, details):
        self.details = details
        super().__init__(details['check'])


def public_failure(error):
    """Never expose exception commands, environment, response bytes or paths."""
    if isinstance(error, CommandFailure):
        return dict(error.details)
    result = {'error_type': type(error).__name__}
    if isinstance(error, CheckFailure):
        return dict(result, category='invariant', check=error.check)
    if isinstance(error, subprocess.TimeoutExpired):
        return dict(result, category='timeout', check='cli-command-timeout')
    if isinstance(error, OSError):
        return dict(result, category='os-error', check='filesystem-or-process-operation', errno=error.errno)
    if isinstance(error, json.JSONDecodeError):
        return dict(result, category='protocol', check='invalid-cli-json')
    return dict(result, category='unexpected-error', check='workload-execution')


def require(condition, message):
    if not condition:
        raise CheckFailure(message)


def inventory(root):
    """Authoritative bytes for this harness's deliberately small, known layout."""
    return {p.relative_to(root).as_posix(): revision(p.read_bytes())
            for p in root.rglob('*') if p.is_file()
            and not any(part in {'.kb', '__pycache__'} for part in p.relative_to(root).parts)} | {
                p.relative_to(root).as_posix(): revision(p.read_bytes())
                for pattern in ('.kb/config.toml', '.kb/state/receipts/*.json')
                for p in root.glob(pattern)}


def check_replay(original, replay):
    require(original.get('replayed') is False, 'initial mutation unexpectedly replayed')
    require(replay == dict(original, replayed=True), 'replay receipt changed or missing replay marker')


def check_round(writers, results):
    winners = [row for row in results if row['won']]
    require(len(results) == writers and len(winners) == 1,
            f'CAS requires one winner among {writers} writers, observed {len(winners)}')
    return winners[0]


def check_counts(writers, rounds, counts):
    expected = {'mutations': rounds * (writers + 1), 'replays': rounds * (writers + 1),
                'conflicts': rounds * (writers - 1), 'reads': rounds * 2 * (writers + 1),
                'searches': rounds}
    for key, value in expected.items():
        require(counts[key] == value, f'{key}: observed {counts[key]}, expected {value}')


class CLI:
    def __init__(self, binary, root, environment, timeout):
        self.binary, self.root, self.environment, self.timeout = binary, root, environment, timeout
        self.lock = threading.Lock()
        self.counts = Counter()
        self.acknowledgements = Counter()
        self.timings = {}

    def call(self, *args, payload=None, conflict=False, root=None):
        start = time.monotonic()
        outcome = 'error'
        try:
            result = subprocess.run([self.binary, '--root', str(root or self.root), '--json', *args],
                                    input=None if payload is None else json.dumps(payload), text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                    env=self.environment, timeout=self.timeout)
            value = json.loads(result.stdout)
            require(value.get('schema_version') == 'kb.v1', 'invalid CLI schema')
            require(value.get('ok') is (result.returncode == 0), 'exit/envelope mismatch')
            if not value['ok']:
                code = value.get('error', {}).get('code')
                require(conflict and result.returncode == 4 and code == 'REVISION_CONFLICT',
                        'unexpected exit or error code')
                outcome = 'conflict'
                return None
            if args[0] in {'create', 'patch'}:
                receipt = value['data']
                require(type(receipt.get('replayed')) is bool, 'mutation missing replay classification')
                with self.lock:
                    self.acknowledgements['replays' if receipt['replayed'] else 'mutations'] += 1
            outcome = 'success'
            return value['data']
        except Exception as error:
            details = public_failure(error)
            details['operation'] = args[0] if args[0] in {
                'init', 'create', 'patch', 'read', 'search', 'backup', 'restore', 'index'} else 'cli'
            raise CommandFailure(details) from None
        finally:
            elapsed = time.monotonic() - start
            with self.lock:
                self.counts[outcome] += 1
                timing = self.timings.setdefault(args[0], {'calls': 0, 'total_seconds': 0, 'max_seconds': 0})
                timing['calls'] += 1
                timing['total_seconds'] += elapsed
                timing['max_seconds'] = max(timing['max_seconds'], elapsed)


def request(key, body, **fields):
    return dict(fields, body=body, idempotency_key=key, actor='process:reliability-soak',
                summary='Record synthetic soak effect ' + key)

def check_log(root, keys, maintain_log, initial):
    raw = (root / 'log.md').read_bytes()
    if not maintain_log:
        require(raw == initial, 'immutable event log changed in no-log profile')
        return
    events = [line for line in raw.decode().splitlines() if line.startswith('- Record synthetic soak effect ')]
    require(len(events) == len(keys), 'missing, duplicate or unexpected logical log events')
    for key in keys:
        prefix = f'- Record synthetic soak effect {key} (process:reliability-soak): '
        require(sum(line.startswith(prefix) for line in events) == 1,
                'incorrect log effect or actor provenance')


def storage(root):
    receipts = list((root / '.kb/state/receipts').glob('*.json'))
    return {'receipt_files': len(receipts), 'receipt_bytes': sum(p.stat().st_size for p in receipts),
            'log_bytes': (root / 'log.md').stat().st_size}


def check_provenance(data):
    require(data['note']['metadata'].get('sources') == [{'resource': '/raw/source.bin'}],
            'confirmed note lost immutable-source provenance')



class Workload:
    def __init__(self, cli, seed, writers, maintain_log=False):
        self.cli, self.seed, self.writers = cli, seed, writers
        self.maintain_log = maintain_log
        self.rng = random.Random(seed)
        self.counts = Counter()
        self.receipts = {}
        self.latest = None
        self.source = bytes(range(256)) * 8

    def remember(self, receipt):
        key = receipt['idempotency_key']
        require(receipt.get('replayed') is False and not receipt.get('dry_run'), 'unconfirmed effect')
        require(receipt.get('changes'), 'mutation acknowledged without changes')
        self.receipts[key] = receipt

    def setup(self):
        self.cli.call('init')
        # Retained receipts embed full-file diffs; the sustained no-log profile
        # bounds each receipt. The optional short profile also exercises logs.
        config = self.cli.root / '.kb/config.toml'
        config.write_text('version = 1\n[write]\nenabled = true\nmaintain_indexes = true\n'
                          f'maintain_log = {str(self.maintain_log).lower()}\n')
        self.initial_log = (self.cli.root / 'log.md').read_bytes()
        raw = self.cli.root / 'raw'
        raw.mkdir()
        (raw / 'source.bin').write_bytes(self.source)
        for path in ['shared.md'] + [f'writer-{i}.md' for i in range(self.writers)]:
            receipt = self.cli.call('create', '--input', '-', payload=request(
                f'{self.seed}/setup/{path}', '# Synthetic ledger\n\nInitial state.\n',
                path=path, type='Reference', metadata={'sources': [{'resource': '/raw/source.bin'}]}))
            self.remember(receipt)
            self.counts['setup_mutations'] += 1
        self.verify_receipts()

    def verify_receipts(self):
        require((self.cli.root / 'raw/source.bin').read_bytes() == self.source, 'immutable source changed')
        require(not (self.cli.root / '.kb/state/pending.json').exists(), 'unresolved pending journal')
        stored = {}
        for path in (self.cli.root / '.kb/state/receipts').glob('*.json'):
            receipt = json.loads(path.read_bytes())['receipt']
            key = receipt['idempotency_key']
            require(key not in stored, 'duplicate durable receipt')
            stored[key] = receipt
        require(stored == self.receipts, 'lost, changed or unexpected durable receipt')
        check_log(self.cli.root, self.receipts, self.maintain_log, self.initial_log)

    def round(self, number, pool):
        shared = self.cli.call('read', 'shared.md')['note']['revision']
        tokens = [f'soakledger seed {self.seed} round {number} writer {i} token {self.rng.getrandbits(64):016x}'
                  for i in range(self.writers)]
        barrier = threading.Barrier(self.writers)

        def writer(i):
            path = f'writer-{i}.md'
            old = self.cli.call('read', path)['note']['revision']
            body = '# Synthetic ledger\n\n' + tokens[i] + '\n'
            patch = request(f'{self.seed}/round/{number}/writer/{i}', body, expected_revision=old)
            receipt = self.cli.call('patch', path, '--input', '-', payload=patch)
            replay = self.cli.call('patch', path, '--input', '-', payload=patch)
            check_replay(receipt, replay)
            data = self.cli.call('read', path)
            require(data['markdown'].endswith(body), 'confirmed private effect missing')
            check_provenance(data)
            change = next(c for c in receipt['changes'] if c['path'] == path)
            require(data['note']['revision'] == change['after_revision'], 'confirmed revision differs')
            attempt = request(f'{self.seed}/round/{number}/cas/{i}', body, expected_revision=shared)
            barrier.wait(timeout=self.cli.timeout)
            won = self.cli.call('patch', 'shared.md', '--input', '-', payload=attempt, conflict=True)
            if won is not None:
                check_replay(won, self.cli.call('patch', 'shared.md', '--input', '-', payload=attempt))
            return {'won': won is not None, 'receipt': receipt, 'cas': won, 'body': body, 'request': patch, 'path': path}

        results = list(pool.map(writer, range(self.writers)))
        winner = check_round(self.writers, results)
        shared_data = self.cli.call('read', 'shared.md')
        require(shared_data['markdown'].endswith(winner['body']), 'CAS loser changed confirmed winner')
        check_provenance(shared_data)
        change = next(c for c in winner['cas']['changes'] if c['path'] == 'shared.md')
        require(shared_data['note']['revision'] == change['after_revision'], 'CAS winner revision missing')
        for row in results:
            self.remember(row['receipt'])
        self.remember(winner['cas'])
        self.latest = (winner['path'], winner['request'], winner['receipt'])
        search = self.cli.call('search', 'soakledger', '--limit', str(self.writers + 1))
        require({h['path'] for h in search['hits']} == {'shared.md'} | {f'writer-{i}.md' for i in range(self.writers)},
                'search omitted confirmed effects')
        self.counts.update(mutations=self.writers + 1, replays=self.writers + 1,
                           conflicts=self.writers - 1, reads=2 * (self.writers + 1), searches=1)

    def backup(self, number):
        start = time.monotonic()
        self.verify_receipts()
        before = inventory(self.cli.root)
        archive = f'.kb/backups/soak-{number}.tar'
        destination = f'.kb/restores/soak-{number}'
        backup = self.cli.call('backup', archive)
        restored = self.cli.call('restore', archive, destination)
        require(backup['manifest_revision'] == restored['manifest_revision'], 'restore manifest mismatch')
        root = self.cli.root / destination
        require(before == inventory(root), 'restored authoritative bytes differ')
        path, patch, original = self.latest
        check_replay(original, self.cli.call('patch', path, '--input', '-', payload=patch, root=root))
        self.cli.call('index', 'rebuild', root=root)
        require(self.cli.call('read', path, root=root)['markdown'].endswith(patch['body']), 'restored read lost effect')
        old = self.cli.call('read', path, root=root)['note']['revision']
        continued = request(f'{self.seed}/restored/{number}', '# Restored continuation\n', expected_revision=old)
        receipt = self.cli.call('patch', path, '--input', '-', payload=continued, root=root)
        check_replay(receipt, self.cli.call('patch', path, '--input', '-', payload=continued, root=root))
        require(self.cli.call('read', path, root=root)['markdown'].endswith(continued['body']), 'restored write lost')
        require(before == inventory(self.cli.root), 'backup/restore modified original authority')
        require((root / 'raw/source.bin').read_bytes() == self.source, 'restored continuation changed source')
        check_log(root, [*self.receipts, continued['idempotency_key']], self.maintain_log, self.initial_log)
        shutil.rmtree(root)
        (self.cli.root / archive).unlink()
        self.counts['backup_checks'] += 1
        self.counts['restored_mutations'] += 1
        self.counts['restored_replays'] += 2
        return {'round': number, 'seconds': time.monotonic() - start, 'files': backup['files'],
                'bytes': backup['bytes'], 'manifest_revision': backup['manifest_revision'],
                'storage': storage(self.cli.root)}


def run_seed(binary, seed, writers, duration, backup_interval, timeout, maintain_log=False):
    report = {'seed': seed, 'ok': False, 'rounds': 0, 'counts': {}, 'backup_checks': [], 'failures': []}
    start = time.monotonic()
    with tempfile.TemporaryDirectory(prefix='kb-soak-') as directory:
        workspace = Path(directory).resolve()
        home = workspace / 'home'
        home.mkdir()
        environment = {'PATH': '', 'HOME': str(home), 'TMPDIR': str(workspace), 'LANG': 'C.UTF-8', 'GOMAXPROCS': '2'}
        cli = CLI(binary, workspace / 'knowledge', environment, timeout)
        workload = Workload(cli, seed, writers, maintain_log)
        try:
            workload.setup()
            report['initial_storage'] = storage(cli.root)
            active_start = time.monotonic()
            next_backup = active_start + backup_interval
            with ThreadPoolExecutor(max_workers=writers) as pool:
                while report['rounds'] == 0 or time.monotonic() - active_start < duration:
                    workload.round(report['rounds'], pool)
                    report['rounds'] += 1
                    if time.monotonic() >= next_backup:
                        report['backup_checks'].append(workload.backup(report['rounds']))
                        next_backup = time.monotonic() + backup_interval
            report['work_seconds'] = time.monotonic() - active_start
            report['backup_checks'].append(workload.backup(report['rounds']))
            report['final_storage'] = storage(cli.root)
            report['acknowledged_per_work_second'] = (workload.counts['mutations'] + workload.counts['replays']) / report['work_seconds']
            check_counts(writers, report['rounds'], workload.counts)
            require(cli.counts['conflict'] == workload.counts['conflicts'], 'unaccounted conflicts')
            require(cli.counts['error'] == 0, 'unaccounted command errors')
            require(cli.acknowledgements['mutations'] == workload.counts['mutations']
                    + workload.counts['setup_mutations'] + workload.counts['restored_mutations'],
                    'unaccounted acknowledged mutations')
            require(cli.acknowledgements['replays'] == workload.counts['replays']
                    + workload.counts['restored_replays'], 'unaccounted acknowledged replays')
            report['ok'] = True
        except Exception as error:
            report['failures'].append(public_failure(error))
        report['counts'] = dict(workload.counts)
        report['commands'] = {key: cli.counts[key] for key in ('success', 'conflict', 'error')}
        report['all_root_acknowledgements'] = {key: cli.acknowledgements[key] for key in ('mutations', 'replays')}
        report['timings'] = cli.timings
        report['confirmed_receipts'] = len(workload.receipts)
    report['elapsed_seconds'] = time.monotonic() - start
    return report


def qualification(runs, seeds, minimum):
    require(len(runs) == len(seeds) and [r['seed'] for r in runs] == seeds, 'missing or reordered seed run')
    require(all(r['ok'] and not r['failures'] for r in runs), 'one or more seed runs failed')
    acknowledged = sum(r['counts'].get('mutations', 0) + r['counts'].get('replays', 0) for r in runs)
    require(acknowledged >= minimum, f'acknowledged mutations/replays {acknowledged} below minimum {minimum}')
    return acknowledged


def publish_report(report, destination):
    try:
        output = Path(destination)
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps(report, indent=2) + '\n')
    except Exception as error:
        failure = public_failure(error)
        failure.update(operation='report-output', check='report-output-write')
        report['failures'].append(failure)
        report['ok'] = False
    print(json.dumps({'ok': report['ok'], 'rounds': [r['rounds'] for r in report['runs']],
                      'acknowledged_work': report.get('acknowledged_work'), 'failures': report['failures']}))
    return 0 if report['ok'] else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', required=True)
    parser.add_argument('--output', required=True)
    parser.add_argument('--duration', type=float, default=1800, help='Total active workload seconds, divided evenly across seeds')
    parser.add_argument('--seeds', type=int, nargs='+', default=[1729, 2718, 31415])
    parser.add_argument('--writers', type=int, default=4)
    parser.add_argument('--minimum-work', type=int, default=10000, help='Aggregate acknowledged main-root mutations plus replays; excludes setup/restore work')
    parser.add_argument('--backup-interval', type=float, default=60)
    parser.add_argument('--command-timeout', type=float, default=30)
    parser.add_argument('--maintain-log', action='store_true', help='Enable event-log maintenance; use a bounded short profile because retained full-log diffs grow quadratically')
    args = parser.parse_args()
    if (args.writers < 4 or args.minimum_work < 1 or len(set(args.seeds)) != len(args.seeds)
            or any(not math.isfinite(n) or n <= 0 for n in (args.duration, args.backup_interval, args.command_timeout))):
        parser.error('require >=4 writers, unique seeds, and positive finite durations/minimum-work')
    report = {'schema_version': 'kb.soak.v1', 'ok': False, 'platform': {'system': platform.system(),
              'release': platform.release(), 'machine': platform.machine(), 'python': platform.python_version()},
              'binary_revision': None, 'writers': args.writers, 'seeds': args.seeds,
              'requested_seconds': args.duration, 'minimum_work': args.minimum_work,
              'backup_interval_seconds': args.backup_interval, 'command_timeout_seconds': args.command_timeout,
              'profile': {'maintain_indexes': True, 'maintain_log': args.maintain_log, 'notes': args.writers + 1},
              'runs': [], 'failures': [], 'scope': 'finite synthetic local-process correctness; not power-loss proof'}
    preflight_started = time.monotonic()
    try:
        binary = str(Path(args.binary).resolve())
        report['binary_revision'] = revision(Path(binary).read_bytes())
    except Exception as error:
        failure = public_failure(error)
        failure.update(operation='binary-preflight', check='binary-identity-read')
        report['failures'].append(failure)
        report.update(acknowledged_work=0, totals={}, elapsed_seconds=time.monotonic() - preflight_started)
        return publish_report(report, args.output)
    started = time.monotonic()
    for seed in args.seeds:
        run = run_seed(binary, seed, args.writers, args.duration / len(args.seeds), args.backup_interval, args.command_timeout, args.maintain_log)
        report['runs'].append(run)
        if not run['ok']:
            break
    report['acknowledged_work'] = sum(r['counts'].get('mutations', 0) + r['counts'].get('replays', 0)
                                      for r in report['runs'])
    report['totals'] = dict(sum((Counter(r['counts']) for r in report['runs']), Counter()))
    try:
        report['acknowledged_work'] = qualification(report['runs'], args.seeds, args.minimum_work)
        report['ok'] = True
    except RuntimeError as error:
        report['failures'].append(public_failure(error))
    report['elapsed_seconds'] = time.monotonic() - started
    return publish_report(report, args.output)


if __name__ == '__main__':
    raise SystemExit(main())

#!/usr/bin/env python3
"""Reproducible synthetic CLI/MCP latency and process-RSS measurements."""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import statistics
import threading
import selectors
import subprocess
import tempfile
import time


def percentile(values, fraction):
    return sorted(values)[max(0, math.ceil(len(values) * fraction) - 1)]


def measured_cli(binary, root, args, request=None):
    argv = [binary, '--root', str(root), *args]
    if request is not None:
        argv += ['--input', '-']
    start = time.perf_counter_ns()
    with tempfile.TemporaryFile() as output, tempfile.TemporaryFile() as errors:
        process = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=output, stderr=errors)
        if request is not None:
            process.stdin.write(json.dumps(request).encode())
        process.stdin.close()
        watchdog = threading.Timer(120, process.kill)
        watchdog.start()
        try:
            _, status, usage = os.wait4(process.pid, 0)
        finally:
            watchdog.cancel()
        process.returncode = os.waitstatus_to_exitcode(status)
        output.seek(0)
        errors.seek(0)
        stdout = output.read()
        stderr = errors.read()
    elapsed = (time.perf_counter_ns() - start) / 1e6
    result = json.loads(stdout)
    if process.returncode or not result['ok']:
        raise RuntimeError((args, process.returncode, stderr.decode(), result.get('error')))
    return result, elapsed, usage.ru_maxrss / ((1 << 20) if platform.system() == 'Darwin' else 1024)


class MCP:
    def __init__(self, binary, root):
        self.process = subprocess.Popen([binary, '--root', str(root), 'mcp'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.sequence = 0
        self.rpc('initialize', {'protocolVersion': '2025-11-25', 'capabilities': {}, 'clientInfo': {'name': 'kb-benchmark', 'version': '1'}})
        self.process.stdin.write(json.dumps({'jsonrpc': '2.0', 'method': 'notifications/initialized'}).encode() + b'\n')
        self.process.stdin.flush()

    def rpc(self, method, params):
        self.sequence += 1
        self.process.stdin.write(json.dumps({'jsonrpc': '2.0', 'id': self.sequence, 'method': method, 'params': params}).encode() + b'\n')
        self.process.stdin.flush()
        selector = selectors.DefaultSelector()
        selector.register(self.process.stdout, selectors.EVENT_READ)
        try:
            if not selector.select(120):
                raise TimeoutError('MCP benchmark response timed out')
            response = json.loads(self.process.stdout.readline())
        finally:
            selector.close()
        if response.get('id') != self.sequence or 'error' in response:
            raise RuntimeError(response)
        return response['result']

    def call(self, action, request):
        start = time.perf_counter_ns()
        result = self.rpc('tools/call', {'name': 'kb_' + action, 'arguments': request})
        elapsed = (time.perf_counter_ns() - start) / 1e6
        envelope = result['structuredContent']
        if not envelope['ok']:
            raise RuntimeError(envelope['error'])
        return envelope, elapsed

    def close(self):
        self.process.stdin.close()
        _, status, usage = os.wait4(self.process.pid, 0)
        self.process.returncode = os.waitstatus_to_exitcode(status)
        stderr = self.process.stderr.read()
        self.process.stdout.close()
        self.process.stderr.close()
        if self.process.returncode:
            raise RuntimeError(stderr.decode())
        return usage.ru_maxrss / ((1 << 20) if platform.system() == 'Darwin' else 1024)


def summarize(latencies, rss=None):
    value = {'samples': len(latencies), 'p50_ms': round(statistics.median(latencies), 3), 'p95_ms': round(percentile(latencies, .95), 3)}
    if rss:
        value['peak_rss_mib'] = round(max(rss), 3)
    return value


def fixture(binary, root, count):
    measured_cli(binary, root, ['init'])
    for i in range(count):
        folder = root / f'group-{i // 500:03d}'
        folder.mkdir(exist_ok=True)
        header = f'---\ntype: Reference\ntitle: Synthetic note {i:06d}\ntags: [synthetic, benchmark]\n---\n# Evidence\nSynthetic quartzbeam recovery ação {i:06d}.\n'
        text = header + ('Bounded local evidence for reliable retrieval. ' * 60)
        (folder / f'note-{i:06d}.md').write_bytes(text.encode()[:2048])


def run(binary, root, count, samples, repair_samples):
    target = 'group-000/note-000000.md'
    requests = {
        'search': (['search', 'quartzbeam recovery'], {'query': 'quartzbeam recovery'}),
        'read': (['read', target], {'path': target}),
        'validate': (['validate'], {'profile': 'okf'}),
    }
    row = {'notes': count, 'note_bytes': 2048, 'cli': {}, 'mcp': {}}
    for action, (args, _) in requests.items():
        timings, rss = [], []
        for _ in range(samples):
            result, elapsed, peak = measured_cli(binary, root, args)
            if action == 'search':
                assert result['data']['total'] == count
            timings.append(elapsed)
            rss.append(peak)
        row['cli'][action] = summarize(timings, rss)
    connection = MCP(binary, root)
    try:
        for action, (_, request) in requests.items():
            _, first = connection.call(action, request)
            timings = [connection.call(action, request)[1] for _ in range(samples)]
            row['mcp'][action] = {**summarize(timings), 'first_ms': round(first, 3)}
    finally:
        row['mcp']['process_peak_rss_mib'] = round(connection.close(), 3)
    # One invalid source exercises the bounded repair preview without fsync cost.
    raw = b'# Legacy evidence\nPreserve the exact body.\n'
    (root / 'legacy.md').write_bytes(raw)
    plan = {'actor': 'process:benchmark', 'summary': 'Preview one explicit synthetic format repair', 'idempotency_key': 'benchmark-repair', 'repair': True, 'incremental': True, 'dry_run': True, 'operations': [{'kind': 'patch', 'path': 'legacy.md', 'expected_revision': 'sha256:' + hashlib.sha256(raw).hexdigest(), 'document': '---\ntype: Reference\n---\n' + raw.decode()}]}
    timings, rss = [], []
    for _ in range(repair_samples):
        result, elapsed, peak = measured_cli(binary, root, ['apply'], plan)
        assert result['data']['repair_progress']['conformant']
        timings.append(elapsed)
        rss.append(peak)
    row['cli']['repair_preview'] = summarize(timings, rss)
    timings, rss = [], []
    for sample in range(repair_samples):
        (root / 'legacy.md').write_bytes(raw)
        plan['dry_run'] = False
        plan['idempotency_key'] = 'benchmark-committed-repair-' + str(sample)
        result, elapsed, peak = measured_cli(binary, root, ['apply'], plan)
        assert result['data']['repair_progress']['conformant']
        timings.append(elapsed)
        rss.append(peak)
    row['cli']['repair_apply'] = {**summarize(timings, rss), 'first_ms': round(timings[0], 3)}
    return row



def rebuild(binary, root, count, samples):
    timings, rss = [], []
    for _ in range(samples):
        result, elapsed, peak = measured_cli(binary, root, ['index', 'rebuild'])
        assert result['data']['index_stale'] is False
        timings.append(elapsed)
        rss.append(peak)
    return {'notes': count, 'note_bytes': 2048, 'cli': {'index_rebuild': summarize(timings, rss)}, 'mcp': {}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', required=True)
    parser.add_argument('--output', required=True)
    parser.add_argument('--sizes', nargs='+', type=int, default=[1000, 10000, 50000])
    parser.add_argument('--samples', type=int, default=20)
    parser.add_argument('--repair-samples', type=int, default=3)
    parser.add_argument('--rebuild-only', action='store_true', help='Measure explicit derived-index rebuilding separately')
    args = parser.parse_args()
    if args.samples < 1 or args.repair_samples < 1 or any(n < 1 for n in args.sizes):
        parser.error('sizes and samples must be positive')
    binary = str(Path(args.binary).resolve())
    report = {'environment': {'system': platform.system(), 'machine': platform.machine(), 'cpus': os.cpu_count(), 'python': platform.python_version()}, 'binary_version': json.loads(subprocess.check_output([binary, 'version']))['data'], 'method': 'Fresh CLI process per sample; one reused stdio MCP process per corpus; OS page cache not evicted; RSS excludes fixture creation; repair preview includes validation/navigation planning; committed repair includes fsync, journal and receipt publication, with first navigation creation reported separately.', 'rebuild_only': args.rebuild_only, 'corpora': []}
    for count in args.sizes:
        with tempfile.TemporaryDirectory(prefix='kb-benchmark-') as tmp:
            root = Path(tmp).resolve() / 'knowledge'
            fixture(binary, root, count)
            row = rebuild(binary, root, count, args.samples) if args.rebuild_only else run(binary, root, count, args.samples, args.repair_samples)
            report['corpora'].append(row)
            Path(args.output).write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(row), flush=True)


if __name__ == '__main__':
    main()

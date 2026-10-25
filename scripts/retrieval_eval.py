#!/usr/bin/env python3
"""Bounded lexical/context quality evaluation on disposable synthetic roots."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import queue
import subprocess
import tempfile
import threading


AS_OF = '2026-01-01T00:00:00Z'
CORPUS = Path(__file__).resolve().parents[1] / 'testdata/retrieval/corpus.json'


class EvaluationFailure(Exception):
    """A consumer-visible quality invariant failed."""


def require(condition, message):
    if not condition:
        raise EvaluationFailure(message)


def revision(raw):
    return 'sha256:' + hashlib.sha256(raw).hexdigest()


def metrics(ranked, relevant):
    """Recall uses all relevant documents; RR is censored at five."""
    if len(set(ranked)) != len(ranked) or len(set(relevant)) != len(relevant):
        raise ValueError('unique document judgments/results required')
    if not relevant:
        return {'recall_at5': None, 'reciprocal_rank_at5': None}
    selected = ranked[:5]
    relevant = set(relevant)
    return {'recall_at5': len(relevant.intersection(selected)) / len(relevant),
            'reciprocal_rank_at5': next((1 / rank for rank, path in enumerate(selected, 1)
                                         if path in relevant), 0.0)}


def aggregate(rows):
    positive = [row for row in rows if row['recall_at5'] is not None]
    return {'positive_cases': len(positive), 'negative_cases': len(rows) - len(positive),
            'macro_recall_at5': sum(row['recall_at5'] for row in positive) / len(positive) if positive else None,
            'mrr_at5': sum(row['reciprocal_rank_at5'] for row in positive) / len(positive) if positive else None}


def document(note):
    # JSON scalars/arrays/maps are valid YAML values; no YAML dependency needed.
    if not note['metadata']:
        return note['body'].encode('utf-8')
    header = '\n'.join(key + ': ' + json.dumps(value, ensure_ascii=False)
                       for key, value in note['metadata'].items())
    return ('---\n' + header + '\n---\n' + note['body']).encode('utf-8')


def load_corpus(path):
    corpus = json.loads(path.read_text(encoding='utf-8'))
    paths = [note['path'] for note in corpus['notes']]
    require(len(paths) == len(set(paths)), 'duplicate corpus paths')
    for path in paths:
        require(not Path(path).is_absolute() and '..' not in Path(path).parts,
                f'unsafe fixture path: {path}')
    for group in ('search_cases', 'context_cases'):
        ids = [case['id'] for case in corpus[group]]
        require(len(ids) == len(set(ids)), f'duplicate case IDs in {group}')
        for case in corpus[group]:
            for field in ('relevant', 'matches', 'paths'):
                values = case.get(field, [])
                require(len(values) == len(set(values)) and set(values) <= set(paths),
                        f'{case["id"]}: invalid {field} judgments')
            if group == 'search_cases':
                require(set(case['relevant']) <= set(case['matches']),
                        f'{case["id"]}: relevance must be eligible')
    return corpus


def cli(binary, root, environment, action, request=None):
    command = [binary, '--root', str(root), '--as-of', AS_OF, action]
    if request is not None:
        command += ['--input', '-']
    result = subprocess.run(command, input=None if request is None else json.dumps(request),
                            text=True, encoding='utf-8', capture_output=True,
                            env=environment, timeout=20)
    require(result.returncode == 0, f'CLI {action}: exit {result.returncode}: {result.stderr} {result.stdout}')
    envelope = json.loads(result.stdout)
    require(envelope['ok'], f'CLI {action}: {envelope}')
    return envelope['data']


class MCP:
    """Independent newline-framed stdio consumer; never loads the CLI skill."""
    def __init__(self, binary, root, environment):
        self.errors = tempfile.TemporaryFile()
        self.process = subprocess.Popen([binary, '--root', str(root), '--as-of', AS_OF, 'mcp'],
                                        stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        stderr=self.errors, env=environment)
        self.sequence = 0
        self.responses = queue.Queue()
        self.reader = threading.Thread(target=self._read, daemon=True)
        self.reader.start()

    def _read(self):
        for line in self.process.stdout:
            self.responses.put(line)
        self.responses.put(b'')

    def send(self, message):
        self.process.stdin.write(json.dumps(message).encode() + b'\n')
        self.process.stdin.flush()

    def rpc(self, method, params):
        self.sequence += 1
        self.send({'jsonrpc': '2.0', 'id': self.sequence, 'method': method, 'params': params})
        try:
            raw = self.responses.get(timeout=20)
        except queue.Empty as error:
            raise EvaluationFailure(f'MCP {method}: timed out') from error
        require(bool(raw), f'MCP {method}: unexpected EOF')
        value = json.loads(raw)
        require(value.get('id') == self.sequence and 'error' not in value,
                f'MCP {method}: invalid response: {value}')
        return value['result']

    def initialize(self):
        initialized = self.rpc('initialize', {'protocolVersion': '2025-11-25', 'capabilities': {},
                                             'clientInfo': {'name': 'kb-retrieval-eval', 'version': '1'}})
        require(bool(initialized.get('instructions')), 'MCP missing independent initialization guidance')
        self.send({'jsonrpc': '2.0', 'method': 'notifications/initialized'})
        tools = self.rpc('tools/list', {})['tools']
        for name in ('kb_search', 'kb_context', 'kb_read'):
            require(any(tool['name'] == name and tool.get('description') and tool.get('inputSchema')
                        for tool in tools), f'MCP missing descriptive schema: {name}')

    def call(self, action, request):
        response = self.rpc('tools/call', {'name': 'kb_' + action, 'arguments': request})
        envelope = response['structuredContent']
        require(envelope['ok'] and not response.get('isError'), f'MCP {action}: {envelope}')
        return envelope['data']

    def close(self):
        self.process.stdin.close()
        try:
            self.process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()
            raise EvaluationFailure('MCP did not exit after stdin EOF')
        finally:
            self.reader.join(timeout=2)
            self.process.stdout.close()
            self.errors.close()
        require(self.process.returncode == 0, f'MCP exited {self.process.returncode}')


def authoritative_files(root):
    return {path.relative_to(root).as_posix(): revision(path.read_bytes())
            for path in root.rglob('*') if path.is_file()
            and not path.relative_to(root).as_posix().startswith('.kb/cache/')
            and path.relative_to(root).as_posix() != '.kb/state/lock'}


def check_purity(before, after):
    changed = sorted(path for path in before.keys() | after.keys() if before.get(path) != after.get(path))
    require(not changed, f'queries changed authoritative files: {changed}')


def check_parity(cli_data, mcp_data):
    # Floating-point term summation order is not a transport/ranking contract.
    left, right = json.loads(json.dumps(cli_data)), json.loads(json.dumps(mcp_data))
    for a, b in zip(left.get('hits', []), right.get('hits', [])):
        require(math.isclose(a.pop('score'), b.pop('score'), rel_tol=1e-12, abs_tol=1e-12),
                'CLI/MCP score mismatch')
    require(left == right, f'CLI/MCP mismatch: CLI={left} MCP={right}')


def check_search(case, data, notes):
    paths = [hit['path'] for hit in data['hits']]
    require(len(paths) == len(set(paths)), f'duplicate results: {paths}')
    require(data['total'] == len(case['matches']),
            f'expected total {len(case["matches"])}, got {data["total"]}; hits={paths}')
    require(len(paths) == min(5, len(case['matches'])) and set(paths) <= set(case['matches']),
            f'unexpected top-5: {paths}; eligible={case["matches"]}')
    result = metrics(paths, case['relevant'])
    if case['relevant']:
        floor = min(5, len(case['relevant'])) / len(case['relevant'])
        require(result['recall_at5'] >= floor and result['reciprocal_rank_at5'] == 1,
                f'relevance regression: {result}; relevant={case["relevant"]}; hits={paths}')
    for higher, lower in case.get('before', []):
        require(higher in paths and lower in paths and paths.index(higher) < paths.index(lower),
                f'expected {higher} before {lower}; got {paths}')
    for hit in data['hits']:
        require(hit['revision'] == revision(notes[hit['path']]), f'wrong revision for {hit["path"]}')
        require(hit['parse_status'] == 'valid' and hit['okf'], f'invalid hit {hit["path"]}')
        if 'stale' in case:
            require(hit['stale'] == case['stale'], f'wrong stale flag for {hit["path"]}')
        lines = notes[hit['path']].decode('utf-8').splitlines()
        for snippet in hit.get('snippets') or []:
            start, end = snippet['line_start'], snippet['line_end']
            require(1 <= start <= end <= len(lines), f'invalid snippet lines: {hit["path"]}')
            require('\n'.join(lines[start - 1:end]).startswith(snippet['text']),
                    f'non-original snippet: {hit["path"]}')
    return {'id': case['id'], 'ranked': paths, 'total': data['total'], **result}


def check_context(case, data, notes):
    text = data['text']
    encoded = text.encode('utf-8')
    require(len(encoded) <= min(case['max_bytes'], case['max_tokens'] * 4), 'context exceeded budget')
    require(data['estimated_tokens'] == (len(encoded) + 3) // 4
            and data['estimator'] == 'utf8-bytes/4-ceiling', 'incorrect token estimate')
    require(data['truncated'] == case['truncated'] and data['omitted'] == case['omitted'],
            f'wrong truncation/omission: {data["truncated"]}/{data["omitted"]}')
    paths = [source['path'] for source in data['sources']]
    require(len(paths) == len(set(paths)) and set(paths) == set(case['paths']),
            f'wrong context sources: {paths}; expected {case["paths"]}')
    remaining = text
    for index, source in enumerate(data['sources']):
        path, start, end = source['path'], source['line_start'], source['line_end']
        require(path not in ('index.md', 'log.md'), f'reserved navigation in context: {path}')
        require(source['revision'] == revision(notes[path]), f'wrong citation revision: {path}')
        lines = notes[path].decode('utf-8').splitlines(keepends=True)
        # A final LF has a zero-byte final line in the runtime's physical-line model.
        if notes[path].endswith(b'\n'):
            lines.append('')
        require(1 <= start <= end <= len(lines), f'invalid citation range {path}:{start}-{end}')
        heading = next((line.strip().lstrip('#').strip() for line in reversed(lines[:start])
                        if line.strip().startswith('#')), '')
        require(source['heading'] == heading, f'wrong citation heading: {path}')
        prefix = f'[{path}:{start}-{end} {source["revision"]}]\n'
        require(remaining.startswith(prefix), f'missing exact citation prefix: {prefix!r}')
        remaining = remaining[len(prefix):]
        original = ''.join(lines[start - 1:end])
        if index < len(data['sources']) - 1 or not data['truncated'] or remaining == original:
            require(remaining.startswith(original), f'non-original excerpt: {path}')
            remaining = remaining[len(original):]
        else:
            require(bool(remaining) and original.startswith(remaining), f'non-original partial excerpt: {path}')
            require(start + remaining.count('\n') - int(remaining.endswith('\n')) == end,
                    f'partial citation range does not describe returned bytes: {path}')
            remaining = ''
    require(not remaining, 'uncited/generated text in context')
    return {'id': case['id'], 'paths': paths, 'bytes': len(encoded),
            'estimated_tokens': data['estimated_tokens'], 'truncated': data['truncated'],
            'omitted': data['omitted']}


def exercise(binary, corpus_path):
    corpus = load_corpus(corpus_path)
    concept_count = sum(bool(note['metadata']) for note in corpus['notes'])
    report = {'corpus_sha256': revision(corpus_path.read_bytes()), 'as_of': AS_OF,
              'notes': concept_count, 'fixture_files': len(corpus['notes']),
              'search_cases': [], 'context_cases': [], 'failures': []}
    with tempfile.TemporaryDirectory(prefix='kb-retrieval-eval-') as directory:
        workspace = Path(directory).resolve()
        root = workspace / 'knowledge'
        home = workspace / 'home'
        home.mkdir()
        environment = {'PATH': '', 'HOME': str(home), 'TMPDIR': str(workspace), 'LANG': 'C.UTF-8'}
        cli(binary, root, environment, 'init')
        notes = {note['path']: document(note) for note in corpus['notes']}
        for path, raw in notes.items():
            target = root / path
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(raw)
        status = cli(binary, root, environment, 'status')
        require(status['notes'] == concept_count and status['invalid'] == 0 and not status['degraded'],
                f'invalid synthetic fixture: {status}')
        before = authoritative_files(root)
        connection = MCP(binary, root, environment)
        try:
            connection.initialize()
            for action in ('search', 'context'):
                for case in corpus[action + '_cases']:
                    request = {key: case[key] for key in ('query', 'filters', 'literal', 'max_tokens', 'max_bytes') if key in case}
                    if action == 'search':
                        request['limit'] = 5
                    try:
                        first = cli(binary, root, environment, action, request)
                        second = connection.call(action, request)
                        check_parity(first, second)
                        check = check_search if action == 'search' else check_context
                        row = check(case, first, notes)
                        check(case, second, notes)
                        report[action + '_cases'].append({**row, 'cli_mcp_parity': True})
                    except (EvaluationFailure, ValueError, KeyError) as error:
                        report['failures'].append(f'{action}/{case["id"]}: {error}')
            # Confirm source reads through both transports, independent of context metadata.
            for path in sorted({path for case in corpus['context_cases'] for path in case['paths']}):
                first = cli(binary, root, environment, 'read', {'path': path})
                second = connection.call('read', {'path': path})
                check_parity(first, second)
                require(first['markdown'].encode() == notes[path]
                        and first['note']['revision'] == revision(notes[path]), f'source read mismatch: {path}')
            report['source_reads'] = len({path for case in corpus['context_cases'] for path in case['paths']})
        finally:
            connection.close()
        check_purity(before, authoritative_files(root))
        report['query_purity'] = True
        report['isolated_home_unchanged'] = not list(home.iterdir())
        require(report['isolated_home_unchanged'], 'query changed isolated home')
    report['metrics'] = aggregate(report['search_cases'])
    report['passed'] = not report['failures']
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', required=True)
    parser.add_argument('--output', required=True)
    parser.add_argument('--corpus', type=Path, default=CORPUS)
    args = parser.parse_args()
    report = exercise(str(Path(args.binary).resolve()), args.corpus)
    Path(args.output).write_text(json.dumps(report, indent=2, ensure_ascii=False) + '\n', encoding='utf-8')
    print(json.dumps({key: report[key] for key in ('notes', 'metrics', 'query_purity', 'passed')}))
    for failure in report['failures']:
        print(failure)
    return 0 if report['passed'] else 1


if __name__ == '__main__':
    raise SystemExit(main())

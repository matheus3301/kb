"""Consumer-visible regressions for judgments, metrics and context validation."""
import copy
from pathlib import Path
import tempfile
import unittest

import retrieval_eval as evaluation


class MetricTests(unittest.TestCase):
    def test_recall_denominator_is_all_relevant_not_k(self):
        ranked = ['a', 'b', 'c', 'd', 'e', 'f']
        self.assertAlmostEqual(evaluation.metrics(ranked, ranked)['recall_at5'], 5 / 6)
        self.assertEqual(evaluation.metrics(['x', 'a', 'y', 'b', 'z'], ['a', 'b', 'c'])['recall_at5'], 2 / 3)

    def test_reciprocal_rank_uses_first_relevant_and_censors_at_five(self):
        result = evaluation.metrics(['x', 'y', 'a', 'b'], ['a', 'b'])
        self.assertEqual(result['reciprocal_rank_at5'], 1 / 3)
        self.assertEqual(evaluation.metrics(['x', 'y', 'z', 'w', 'v', 'a'], ['a'])['reciprocal_rank_at5'], 0)
        self.assertEqual(evaluation.metrics([], ['a'])['recall_at5'], 0)

    def test_no_relevant_is_undefined_not_perfect_recall(self):
        empty = evaluation.metrics([], [])
        self.assertEqual(empty, {'recall_at5': None, 'reciprocal_rank_at5': None})
        rows = [evaluation.metrics(['a'], ['a']), evaluation.metrics(['x', 'b'], ['b', 'c']), empty]
        summary = evaluation.aggregate(rows)
        self.assertEqual(summary, {'positive_cases': 2, 'negative_cases': 1,
                                  'macro_recall_at5': .75, 'mrr_at5': .75})

    def test_duplicate_results_or_judgments_cannot_inflate_metrics(self):
        for ranked, relevant in [(['a', 'a'], ['a', 'b']), (['a'], ['a', 'a'])]:
            with self.subTest(ranked=ranked, relevant=relevant), self.assertRaises(ValueError):
                evaluation.metrics(ranked, relevant)


class QualityFailureTests(unittest.TestCase):
    def setUp(self):
        self.notes = {'a.md': b'---\ntype: Reference\n---\n# Evidence\nOriginal memory.\nSecond line.\n'}
        self.case = {'id': 'context', 'max_bytes': 1000, 'max_tokens': 250,
                     'paths': ['a.md'], 'truncated': False, 'omitted': 0}
        source = {'path': 'a.md', 'revision': evaluation.revision(self.notes['a.md']),
                  'heading': 'Evidence', 'line_start': 5, 'line_end': 6}
        text = f'[a.md:5-6 {source["revision"]}]\nOriginal memory.\nSecond line.\n'
        self.context = {'sources': [source], 'text': text, 'estimated_tokens': (len(text.encode()) + 3) // 4,
                        'estimator': 'utf8-bytes/4-ceiling', 'truncated': False, 'omitted': 0}

    def test_citation_checker_accepts_original_full_and_partial_bytes(self):
        evaluation.check_context(self.case, self.context, self.notes)
        partial = copy.deepcopy(self.context)
        partial['truncated'] = True
        partial['text'] = partial['text'][:-4]
        partial['estimated_tokens'] = (len(partial['text'].encode()) + 3) // 4
        evaluation.check_context(dict(self.case, truncated=True), partial, self.notes)

    def test_complete_last_excerpt_can_precede_omitted_notes(self):
        data = dict(self.context, truncated=True, omitted=2)
        evaluation.check_context(dict(self.case, truncated=True, omitted=2), data, self.notes)

    def test_rejects_fabricated_content_revision_lines_and_budget(self):
        mutations = [('generated answer', lambda data: data.update(text=data['text'] + 'Invented.')),
                     ('revision', lambda data: data['sources'][0].update(revision='sha256:wrong')),
                     ('line', lambda data: data['sources'][0].update(line_start=0)),
                     ('heading', lambda data: data['sources'][0].update(heading='Invented')),
                     ('truncation', lambda data: data.update(truncated=True)),
                     ('omission', lambda data: data.update(omitted=1)),
                     ('tokens', lambda data: data.update(estimated_tokens=0))]
        for name, mutate in mutations:
            with self.subTest(name=name), self.assertRaises(evaluation.EvaluationFailure):
                data = copy.deepcopy(self.context)
                mutate(data)
                evaluation.check_context(self.case, data, self.notes)
        with self.assertRaisesRegex(evaluation.EvaluationFailure, 'budget'):
            evaluation.check_context(dict(self.case, max_bytes=10), self.context, self.notes)

    def test_budget_counts_utf8_bytes_not_characters(self):
        notes = {'a.md': 'mémoire 🧪'.encode()}
        source = {'path': 'a.md', 'revision': evaluation.revision(notes['a.md']), 'heading': '',
                  'line_start': 1, 'line_end': 1}
        text = f'[a.md:1-1 {source["revision"]}]\nmémoire 🧪'
        data = dict(self.context, sources=[source], text=text,
                    estimated_tokens=(len(text.encode()) + 3) // 4)
        evaluation.check_context(self.case, data, notes)
        with self.assertRaisesRegex(evaluation.EvaluationFailure, 'budget'):
            evaluation.check_context(dict(self.case, max_bytes=len(text)), data, notes)

    def test_search_rejects_correct_set_with_bad_field_ranking(self):
        notes = {'title.md': b'title', 'body.md': b'body'}
        case = {'matches': list(notes), 'relevant': ['title.md'], 'before': [['title.md', 'body.md']]}
        hits = [{'path': path, 'revision': evaluation.revision(raw), 'parse_status': 'valid', 'okf': True}
                for path, raw in notes.items()]
        data = {'total': 2, 'hits': list(reversed(hits))}
        with self.assertRaisesRegex(evaluation.EvaluationFailure, 'relevance regression'):
            evaluation.check_search(case, data, notes)

    def test_no_match_cannot_pass_by_omitting_relevance_metrics(self):
        with self.assertRaisesRegex(evaluation.EvaluationFailure, 'expected total 0'):
            evaluation.check_search({'matches': [], 'relevant': []},
                                    {'total': 1, 'hits': [{'path': 'a.md'}]}, self.notes)

    def test_parity_checks_order_revision_and_context_not_just_ids(self):
        for key, value in [('text', 'wrong'), ('omitted', 3), ('sources', [])]:
            with self.subTest(key=key), self.assertRaises(evaluation.EvaluationFailure):
                evaluation.check_parity(self.context, dict(self.context, **{key: value}))
        one = {'hits': [{'path': 'a', 'score': 1.0000000000000002}]}
        evaluation.check_parity(one, {'hits': [{'path': 'a', 'score': 1.0}]})
        with self.assertRaises(evaluation.EvaluationFailure):
            evaluation.check_parity(one, {'hits': [{'path': 'b', 'score': 1.0}]})

    def test_purity_detects_modified_added_and_deleted_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'a.md').write_bytes(b'original')
            before = evaluation.authoritative_files(root)
            (root / '.kb/cache').mkdir(parents=True)
            (root / '.kb/cache/generation').write_bytes(b'derived')
            evaluation.check_purity(before, evaluation.authoritative_files(root))
            for name, content in [('a.md', b'changed'), ('log.md', b'unexpected')]:
                with self.subTest(name=name):
                    (root / name).write_bytes(content)
                    with self.assertRaisesRegex(evaluation.EvaluationFailure, name):
                        evaluation.check_purity(before, evaluation.authoritative_files(root))
                    if name == 'a.md':
                        (root / name).write_bytes(b'original')
                    else:
                        (root / name).unlink()
            (root / 'a.md').unlink()
            with self.assertRaises(evaluation.EvaluationFailure):
                evaluation.check_purity(before, evaluation.authoritative_files(root))


if __name__ == '__main__':
    unittest.main()

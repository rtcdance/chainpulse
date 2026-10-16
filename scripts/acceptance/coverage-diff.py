#!/usr/bin/env python3
"""Validate raw profiles and compare authoritative `go tool cover` summaries.

Each summary must be produced in the matching source checkout. Raw profile
block totals can include initializers that Go's function report omits, so they
are diagnostic only. The existing MIN_COVERAGE gate remains authoritative.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import sys
import tempfile


def read_profile(path):
    text = path.read_text()
    lines = text.splitlines()
    if not lines or lines[0] not in ('mode: set', 'mode: count', 'mode: atomic'):
        raise ValueError(f'{path}: invalid or missing coverage mode')
    blocks = {}
    for index, line in enumerate(lines[1:], 2):
        match = re.fullmatch(r'(\S+:\d+\.\d+,\d+\.\d+)\s+(\d+)\s+(\d+)', line)
        if not match:
            raise ValueError(f'{path}:{index}: malformed coverage block')
        key, statements, count = match.groups()
        statements, count = int(statements), int(count)
        previous = blocks.get(key)
        if previous is not None and previous[0] != statements:
            raise ValueError(f'{path}:{index}: inconsistent duplicate statement count')
        blocks[key] = (statements, count > 0 or (previous is not None and previous[1]))
    total = sum(statements for statements, _ in blocks.values())
    covered = sum(statements for statements, hit in blocks.values() if hit)
    if total == 0:
        raise ValueError(f'{path}: no statements; comparison cannot execute')
    return {'mode': lines[0].split()[1], 'raw_block_statement_count': total,
            'raw_block_covered_statements': covered, 'raw_block_percent': 100 * covered / total,
            'profile_sha256': hashlib.sha256(path.read_bytes()).hexdigest()}



def read_summary(path):
    totals = [line for line in path.read_text().splitlines() if line.startswith('total:')]
    if len(totals) != 1:
        raise ValueError(f'{path}: expected exactly one Go coverage total')
    match = re.fullmatch(r'total:\s+\(statements\)\s+([0-9]+(?:\.[0-9]+)?)%', totals[0])
    if not match or not 0 <= float(match[1]) <= 100:
        raise ValueError(f'{path}: invalid Go coverage total')
    return float(match[1])


def self_test():
    with tempfile.TemporaryDirectory(prefix='chainpulse-coverage-control-') as directory:
        path = Path(directory) / 'coverage.out'
        path.write_text('mode: atomic\nexample/a.go:1.1,2.1 2 1\nexample/a.go:3.1,4.1 2 0\n')
        assert read_profile(path)['raw_block_percent'] == 50
        path.write_text(path.read_text() + 'example/a.go:3.1,4.1 2 1\n')
        assert read_profile(path)['raw_block_statement_count'] == 4
        assert read_profile(path)['raw_block_percent'] == 100
        for invalid in ('', 'mode: atomic\n', 'mode: unknown\n',
                        'mode: atomic\nexample/a.go:1.1,2.1 1 NaN\n',
                        'mode: atomic\nexample/a.go:1.1,2.1 1 -1\n',
                        'mode: atomic\nexample/a.go:1.1,2.1 0 1\n',
                        'mode: atomic\nexample/a.go:1.1,2.1 2 0\nexample/a.go:1.1,2.1 3 1\n'):
            path.write_text(invalid)
            try:
                read_profile(path)
            except ValueError:
                continue
            raise AssertionError('invalid profile accepted: ' + repr(invalid))
        summary = Path(directory) / 'functions.txt'
        summary.write_text('example/a.go:1: Function 100.0%\ntotal: (statements) 69.6%\n')
        assert read_summary(summary) == 69.6
        for invalid in ('', 'total: (statements) NaN%', 'total: (statements) 101%',
                        'total: (statements) 1%\ntotal: (statements) 2%'):
            summary.write_text(invalid)
            try:
                read_summary(summary)
            except ValueError:
                continue
            raise AssertionError('invalid native summary accepted')
        path.unlink()
        try:
            read_profile(path)
        except FileNotFoundError:
            pass
        else:
            raise AssertionError('missing profile accepted')
    print('coverage-diff self-test passed: native Go summaries, raw validation and invalid/missing inputs')
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', type=Path)
    parser.add_argument('--candidate', type=Path)
    parser.add_argument('--base-summary', type=Path)
    parser.add_argument('--base-status', type=Path)
    parser.add_argument('--candidate-summary', type=Path)
    parser.add_argument('--base-sha')
    parser.add_argument('--candidate-sha')
    parser.add_argument('--out', type=Path)
    parser.add_argument('--self-test', action='store_true')
    args = parser.parse_args()
    if args.self_test:
        return self_test()
    if not all((args.base, args.candidate, args.base_summary, args.candidate_summary, args.out)):
        parser.error('--base, --candidate, --base-summary, --candidate-summary and --out are required')
    for sha in (args.base_sha, args.candidate_sha):
        if sha is not None and not re.fullmatch(r'[0-9a-f]{40}', sha):
            parser.error('source SHAs must be full commit identifiers')
    try:
        candidate = read_profile(args.candidate)
        candidate['percent'] = read_summary(args.candidate_summary)
        if args.base_status:
            availability = json.loads(args.base_status.read_text())
            if availability.get('comparison_status') == 'unavailable_base_tests_failed':
                doc = {'schema':'chainpulse.coverage-comparison.v1',
                       'comparison_status':'unavailable_base_tests_failed',
                       'base_sha':args.base_sha,'candidate_sha':args.candidate_sha,
                       'candidate':candidate,'base_execution':availability}
                args.out.parent.mkdir(parents=True, exist_ok=True)
                args.out.with_suffix('.json').write_text(json.dumps(doc,indent=2)+'\n')
                args.out.write_text('# Coverage comparison unavailable\n\n'
                                    'The base commit tests failed. Its partial profile is not a valid baseline; no coverage difference was calculated.\n\n'
                                    f'Candidate native Go coverage: {candidate["percent"]:.1f}%. '
                                    'The required candidate coverage job retains its existing threshold gate.\n')
                print(args.out.read_text())
                return 0  # Informational report generated; comparison is explicitly unavailable.
            if availability.get('comparison_status') != 'available':
                raise ValueError('unrecognized base comparison status')
        base = read_profile(args.base)
        base['percent'] = read_summary(args.base_summary)
        candidate['percent'] = read_summary(args.candidate_summary)
        difference = candidate['percent'] - base['percent']
        doc = {'schema': 'chainpulse.coverage-comparison.v1', 'comparison_status':'available', 'base_sha': args.base_sha,
               'candidate_sha': args.candidate_sha, 'base': base, 'candidate': candidate,
               'difference_percentage_points': difference,
               'note': 'Percentages come from go tool cover in the matching checkout. Raw block totals are diagnostic and can include initializers omitted from function coverage. Existing CI MIN_COVERAGE remains authoritative.'}
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.with_suffix('.json').write_text(json.dumps(doc, indent=2) + '\n')
        args.out.write_text(f'# Coverage comparison (pkg, short tests)\n\n'
                            f'Base: {base["percent"]:.1f}%\n\n'
                            f'Candidate: {candidate["percent"]:.1f}%\n\n'
                            f'Difference: {difference:+.2f} percentage points\n\n{doc["note"]}\n')
        print(args.out.read_text())
        return 0
    except (OSError, ValueError) as error:
        print(f'coverage comparison failed: {error}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())

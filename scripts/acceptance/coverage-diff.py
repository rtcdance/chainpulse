#!/usr/bin/env python3
"""Compare real Go statement coverage profiles without depending on source layout.

The existing MIN_COVERAGE gate remains authoritative. This tool reports the
comparison; missing, malformed or statement-free profiles fail instead of
turning into a fabricated zero percentage.
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
    return {'mode': lines[0].split()[1], 'statement_count': total,
            'covered_statements': covered, 'percent': 100 * covered / total,
            'profile_sha256': hashlib.sha256(path.read_bytes()).hexdigest()}


def self_test():
    with tempfile.TemporaryDirectory(prefix='chainpulse-coverage-control-') as directory:
        path = Path(directory) / 'coverage.out'
        path.write_text('mode: atomic\nexample/a.go:1.1,2.1 2 1\nexample/a.go:3.1,4.1 2 0\n')
        assert read_profile(path)['percent'] == 50
        path.write_text(path.read_text() + 'example/a.go:3.1,4.1 2 1\n')
        assert read_profile(path)['statement_count'] == 4
        assert read_profile(path)['percent'] == 100
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
        path.unlink()
        try:
            read_profile(path)
        except FileNotFoundError:
            pass
        else:
            raise AssertionError('missing profile accepted')
    print('coverage-diff self-test passed: weighted statements, duplicate merge and invalid/missing inputs')
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', type=Path)
    parser.add_argument('--candidate', type=Path)
    parser.add_argument('--base-sha')
    parser.add_argument('--candidate-sha')
    parser.add_argument('--out', type=Path)
    parser.add_argument('--self-test', action='store_true')
    args = parser.parse_args()
    if args.self_test:
        return self_test()
    if not all((args.base, args.candidate, args.out)):
        parser.error('--base, --candidate and --out are required')
    for sha in (args.base_sha, args.candidate_sha):
        if sha is not None and not re.fullmatch(r'[0-9a-f]{40}', sha):
            parser.error('source SHAs must be full commit identifiers')
    try:
        base, candidate = read_profile(args.base), read_profile(args.candidate)
        difference = candidate['percent'] - base['percent']
        doc = {'schema': 'chainpulse.coverage-comparison.v1', 'base_sha': args.base_sha,
               'candidate_sha': args.candidate_sha, 'base': base, 'candidate': candidate,
               'difference_percentage_points': difference,
               'note': 'Statement-weighted coverage; code changed between versions. Existing CI MIN_COVERAGE remains authoritative.'}
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.with_suffix('.json').write_text(json.dumps(doc, indent=2) + '\n')
        args.out.write_text(f'# Coverage comparison (pkg, short tests)\n\n'
                            f'Base: {base["percent"]:.2f}% ({base["covered_statements"]}/{base["statement_count"]} statements)\n\n'
                            f'Candidate: {candidate["percent"]:.2f}% ({candidate["covered_statements"]}/{candidate["statement_count"]} statements)\n\n'
                            f'Difference: {difference:+.2f} percentage points\n\n{doc["note"]}\n')
        print(args.out.read_text())
        return 0
    except (OSError, ValueError) as error:
        print(f'coverage comparison failed: {error}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())

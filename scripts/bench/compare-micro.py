#!/usr/bin/env python3
"""Compare frozen sources and the candidate on the SAME host/toolchain/corpus.

Both versions always use five samples. At most one paired remeasurement is
allowed; in that case ALL ten samples feed the existing fixed 15% gate.
This is a decoder microbenchmark, not the full ingestion/soak acceptance.
"""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import sys
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-sha', required=True)
    parser.add_argument('--out', type=Path, default=Path('metrics'))
    parser.add_argument('--benchstat-bin', default=os.environ.get('BENCHSTAT_BIN', 'benchstat'))
    args = parser.parse_args()
    if not re.fullmatch(r'[0-9a-f]{40}', args.base_sha):
        parser.error('--base-sha must be a frozen full commit SHA')
    benchstat = shutil.which(args.benchstat_bin)
    if not benchstat:
        parser.error('pinned benchstat binary is required')
    root = Path(__file__).resolve().parents[2]
    out = args.out.resolve()
    out.mkdir(parents=True, exist_ok=True)
    baseline_raw, candidate_raw = out / 'baseline-results.txt', out / 'benchmark-results.txt'
    baseline, gate_raw = out / 'paired-baseline.json', out / 'bench-gate.json'
    corpus = Path('pkg/evm/event_decoder_bench_test.go')
    command = ['go', 'test', '-run', '^$', '-bench', 'BenchmarkDecodeEventDataTransfer', '-benchmem', '-count=5', './pkg/evm/']
    command_text = "go test -run '^$' -bench BenchmarkDecodeEventDataTransfer -benchmem -count=5 ./pkg/evm/"
    gate = root / 'scripts/acceptance/bench-gate.py'
    temporary = Path(tempfile.mkdtemp(prefix='chainpulse-bench-pair-'))
    base = temporary / 'baseline'
    provenance = {'schema': 'chainpulse.bench-pair.v1', 'base_sha': args.base_sha,
                  'candidate_sha': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
                  'go_version': subprocess.check_output(['go', 'version'], cwd=root, text=True).strip(),
                  'platform': platform.platform(), 'machine': platform.machine(),
                  'cpu_count': os.cpu_count(), 'command': command_text, 'threshold': 0.15,
                  'candidate_tree_dirty': bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=root, text=True).strip()),
                  'generated_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  'note': 'Same-runner paired decoder measurements; not the 8-vCPU absolute-load reference or full E14.'}
    added = False
    result_code = 2
    try:
        subprocess.run(['git', 'worktree', 'add', '--detach', str(base), args.base_sha], cwd=root, check=True)
        added = True
        digest = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
        if digest(root / corpus) != digest(base / corpus):
            raise RuntimeError('benchmark corpus differs from the frozen source; review before comparing')
        provenance['corpus_sha256'] = digest(root / corpus)
        for attempt in range(1, 3):
            mode = 'w' if attempt == 1 else 'a'
            for cwd, output in [(base, baseline_raw), (root, candidate_raw)]:
                with output.open(mode) as stream:
                    subprocess.run(command, cwd=cwd, stdout=stream, stderr=subprocess.STDOUT, check=True, timeout=300)
            subprocess.run([sys.executable, str(gate), '--bench-output', str(baseline_raw), '--write-baseline', str(baseline)], cwd=root, check=True)
            document = json.loads(baseline.read_text())
            document['provenance'] = dict(provenance, measured_sha=args.base_sha, attempts=attempt, samples_per_benchmark=attempt * 5)
            baseline.write_text(json.dumps(document, indent=2) + '\n')
            with (out / 'gate-result.txt').open('w') as stream:
                result = subprocess.run([sys.executable, str(gate), '--bench-output', str(candidate_raw), '--baseline', str(baseline), '--command', command_text, '--emit-json', str(gate_raw)], cwd=root, stdout=stream, stderr=subprocess.STDOUT)
            with (out / 'benchstat.txt').open('w') as stream:
                subprocess.run([benchstat, str(baseline_raw), str(candidate_raw)], stdout=stream, stderr=subprocess.STDOUT, check=True)
            result_code = result.returncode
            provenance.update(attempts=attempt, samples_per_benchmark=attempt * 5, exit_code=result_code,
                              baseline_raw_sha256=digest(baseline_raw), candidate_raw_sha256=digest(candidate_raw))
            (out / 'paired-provenance.json').write_text(json.dumps(provenance, indent=2) + '\n')
            print((out / 'gate-result.txt').read_text(), flush=True)
            if result_code != 1:
                break
        return result_code
    finally:
        if added:
            subprocess.run(['git', 'worktree', 'remove', str(base)], cwd=root, check=True)
        temporary.rmdir()


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        print(f'paired benchmark failed: {error}', file=sys.stderr)
        raise SystemExit(2)

#!/usr/bin/env python3
"""Execute one isolated, real CI gate fault; exit 1 means the gate rejected it.

A passing Go test is run first so missing tools/network cannot masquerade as
successful fault injection. No repository source or dependency is modified.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

CONTROLS = ('compile', 'assertion', 'timeout', 'empty', 'required-skip', 'coverage')


def execute(control, output):
    root = Path(__file__).resolve().parents[2]
    output.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, GOWORK='off')
    report = {'schema': 'chainpulse.ci-negative.v1', 'control': control,
              'candidate_sha': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
              'go_version': subprocess.check_output(['go', 'version'], text=True).strip(),
              'positive_control_passed': False, 'injection_rejected': False}
    with tempfile.TemporaryDirectory(prefix='chainpulse-ci-negative-') as directory:
        fixture = Path(directory)
        (fixture / 'go.mod').write_text('module chainpulse.invalid/gatefixture\n\ngo 1.26.0\n')
        source = fixture / 'gate_test.go'
        source.write_text('package gatefixture\nimport "testing"\nfunc TestGate(t *testing.T) {}\n')

        def run_go(label, extra=()):
            command = ['go', 'test', '-json', '-count=1', *extra, './...']
            result = subprocess.run(command, cwd=fixture, env=env, capture_output=True, text=True, timeout=60)
            path = output / (label + '.jsonl')
            path.write_text(result.stdout)
            (output / (label + '.stderr')).write_text(result.stderr)
            verified = subprocess.run([sys.executable, str(root / 'scripts/acceptance/verify-tests.py'),
                                       '--jsonl', str(path), '--min-tests', '1', '--fail-on-unknown-skip'],
                                      capture_output=True, text=True, timeout=10)
            (output / (label + '-verifier.log')).write_text(verified.stdout + verified.stderr)
            return result.returncode, verified.returncode

        report['positive_exit'], report['positive_verifier_exit'] = run_go('positive')
        if report['positive_exit'] or report['positive_verifier_exit']:
            raise RuntimeError('positive Go fixture failed; injection cannot be credited')
        report['positive_control_passed'] = True
        extra = ()
        if control == 'compile':
            source.write_text('package gatefixture\nfunc malformed( {\n')
        elif control == 'assertion':
            source.write_text('package gatefixture\nimport "testing"\nfunc TestGate(t *testing.T) { t.Fatal("injected assertion failure") }\n')
        elif control == 'timeout':
            source.write_text('package gatefixture\nimport "testing"\nfunc TestGate(t *testing.T) { select {} }\n')
            extra = ('-timeout=100ms',)
        elif control == 'empty':
            extra = ('-run=^NoMatchingTest$',)
        elif control == 'required-skip':
            source.write_text('package gatefixture\nimport "testing"\nfunc TestGate(t *testing.T) { t.Skip("injected missing required dependency") }\n')
        elif control == 'coverage':
            # Run a real test without -coverprofile, then exercise the exact
            # non-empty profile invariant used in the coverage workflow.
            extra = ()
        report['injected_go_exit'], report['injected_verifier_exit'] = run_go('injected', extra)
        if control == 'coverage':
            profile = fixture / 'coverage.out'
            report['profile_present_nonempty'] = profile.is_file() and profile.stat().st_size > 0
            report['injection_rejected'] = (report['injected_go_exit'] == 0 and
                                             report['injected_verifier_exit'] == 0 and
                                             not report['profile_present_nonempty'])
        elif control in ('compile', 'assertion', 'timeout'):
            report['injection_rejected'] = report['injected_go_exit'] != 0 and report['injected_verifier_exit'] != 0
        else:
            report['injection_rejected'] = report['injected_go_exit'] == 0 and report['injected_verifier_exit'] != 0
    (output / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
    if not report['injection_rejected']:
        raise RuntimeError('fault escaped the required gate')
    print(json.dumps(report), flush=True)
    print('Intentional required-check failure: controlled injection rejected.', file=sys.stderr)
    return 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--control', choices=CONTROLS, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    try:
        return execute(args.control, args.out.resolve())
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        print(f'negative control infrastructure failure: {error}', file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())

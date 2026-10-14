#!/usr/bin/env python3
"""bench-prometheus.py - turn a bench-gate result into Prometheus exposition text.

The workflow this replaces wrote its metrics with a quoted heredoc containing
hard-coded zeros:

    chainpulse_test_coverage_percent 0
    chainpulse_benchmark_ops_per_sec 0

Section 7.4 requires that a missing or unparseable artifact fail rather than
produce a zero-valued fake report, so this reads the real numbers out of the
bench-gate JSON instead of asserting them. Anything the input does not contain is
omitted rather than defaulted to zero.

Usage
-----
    python3 scripts/acceptance/bench-prometheus.py metrics/bench-gate.json
    python3 scripts/acceptance/bench-prometheus.py metrics/bench-gate.json --self-test

Exit codes
----------
  0  exposition text written to stdout
  1  the input is missing, unparseable, or contains no benchmark results
  2  usage error
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

METRICS = (
    ("ns_per_op", "chainpulse_benchmark_ns_per_op", "Benchmark mean nanoseconds per operation"),
    ("bytes_per_op", "chainpulse_benchmark_bytes_per_op", "Benchmark mean bytes per operation"),
)


def render(summary: dict) -> str:
    """Return Prometheus exposition text, or raise ValueError if there is no data."""
    benchmarks = summary.get("benchmarks") or {}
    if not benchmarks:
        raise ValueError("no benchmark results in the input")

    lines: list[str] = []
    for key, metric, help_text in METRICS:
        lines.append(f"# HELP {metric} {help_text}")
        lines.append(f"# TYPE {metric} gauge")
        for name in sorted(benchmarks):
            mean = (benchmarks[name] or {}).get("mean") or {}
            value = mean.get(key)
            if value is None:
                continue
            lines.append(f'{metric}{{benchmark="{name}"}} {value}')
        lines.append("")

    if not any(line.startswith("chainpulse_") for line in lines):
        raise ValueError("benchmark results present but no mean values to report")
    return "\n".join(lines).rstrip("\n") + "\n"


def self_test() -> int:
    ok = True

    def check(label: str, got, want) -> None:
        nonlocal ok
        if got == want:
            print(f"[PASS] self-test: {label}")
        else:
            ok = False
            print(f"[FAIL] self-test: {label} (got {got!r}, want {want!r})")

    good = {
        "benchmarks": {
            "BenchmarkX": {"mean": {"ns_per_op": 1552.4, "bytes_per_op": 904.0}},
        }
    }
    text = render(good)
    check("emits the real ns/op", 'chainpulse_benchmark_ns_per_op{benchmark="BenchmarkX"} 1552.4' in text, True)
    check("emits the real bytes/op", 'chainpulse_benchmark_bytes_per_op{benchmark="BenchmarkX"} 904.0' in text, True)
    check("does not invent a zero metric", " 0\n" in text, False)

    # The defect this replaces: a fabricated zero must never appear when the data
    # is absent - the tool must refuse instead.
    for label, payload in (
        ("no benchmarks key", {}),
        ("empty benchmarks", {"benchmarks": {}}),
        ("means missing", {"benchmarks": {"B": {}}}),
    ):
        try:
            render(payload)
            check(f"refuses {label}", "no error", "ValueError")
        except ValueError:
            check(f"refuses {label}", "ValueError", "ValueError")

    print("self-test:", "OK" if ok else "FAILED")
    return 0 if ok else 1


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path, nargs="?")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()
    if args.input is None:
        parser.error("input is required (or use --self-test)")
    if not args.input.is_file():
        print(f"error: {args.input} not found", file=sys.stderr)
        return 1

    try:
        summary = json.loads(args.input.read_text())
    except (OSError, json.JSONDecodeError) as exc:
        print(f"error: cannot read {args.input}: {exc}", file=sys.stderr)
        return 1

    try:
        sys.stdout.write(render(summary))
    except ValueError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

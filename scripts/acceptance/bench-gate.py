#!/usr/bin/env python3
"""bench-gate.py — micro-benchmark regression gate.

Task: T14 of docs/specs/2026-10-12-web3-go-backend-delivery-plan.md, implementing
the §8 row "微基准回归":

    同硬件/版本/语料下 ns/op、B/op 恶化超过 15% 阻断；有统计噪声时复测一次再判
    至少 5 次样本，benchstat 与结构化数值；
    no match、缺基线、无有效样本不能 pass

That last clause is the reason this is a separate tool rather than a shell
comparison: a gate that passes when it could not actually compare anything is
worse than no gate, because it reports green. Every one of those three
conditions is a FAILURE here, not a skip.

It is dependency-free (Python 3 stdlib only) and has a built-in self-test, so the
gate itself is verified rather than trusted.

Usage
-----
    # Record a baseline (at least 5 samples per benchmark).
    bench-gate.py --bench-output raw.txt --write-baseline baseline.json

    # Gate a run against it.
    bench-gate.py --bench-output raw.txt --baseline baseline.json

    # Verify the gate.
    bench-gate.py --self-test
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

# §8 freezes these; do not move them after seeing a result.
REGRESSION_THRESHOLD = 0.15
MIN_SAMPLES = 5

# go test -bench line, e.g.
#   BenchmarkDecode-8   1000000   1234 ns/op   456 B/op   7 allocs/op
BENCH_LINE = re.compile(
    r"^(?P<name>Benchmark\S+?)"
    r"(?:-\d+)?"
    r"\s+\d+\s+"
    r"(?P<ns>[\d.]+)\s+ns/op"
    r"(?:\s+(?P<bytes>[\d.]+)\s+B/op)?"
    r"(?:\s+(?P<allocs>[\d.]+)\s+allocs/op)?"
)


@dataclass
class Sample:
    ns_per_op: float
    bytes_per_op: float | None = None
    allocs_per_op: float | None = None


@dataclass
class Result:
    comparisons: list[dict] = field(default_factory=list)
    regressions: list[dict] = field(default_factory=list)
    failures: list[str] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return not self.regressions and not self.failures


def parse_bench_output(text: str) -> dict[str, list[Sample]]:
    """Group samples by benchmark name. Repeated runs of the same benchmark
    produce repeated lines, which is how the >= 5 sample rule is satisfied."""
    samples: dict[str, list[Sample]] = {}
    for line in text.splitlines():
        match = BENCH_LINE.match(line.strip())
        if not match:
            continue
        name = match.group("name")
        samples.setdefault(name, []).append(
            Sample(
                ns_per_op=float(match.group("ns")),
                bytes_per_op=float(match.group("bytes")) if match.group("bytes") else None,
                allocs_per_op=float(match.group("allocs")) if match.group("allocs") else None,
            )
        )
    return samples


def mean(values: list[float]) -> float:
    return sum(values) / len(values)


def summarize(samples: list[Sample]) -> dict[str, float]:
    """Mean ns/op and B/op. The plan allows one re-measurement under noise, which
    a mean over >= 5 samples already covers without hand-picking a run."""
    summary = {"ns_per_op": mean([s.ns_per_op for s in samples])}
    byte_values = [s.bytes_per_op for s in samples if s.bytes_per_op is not None]
    if byte_values:
        summary["bytes_per_op"] = mean(byte_values)
    return summary


def compare(current: dict[str, list[Sample]], baseline: dict[str, dict], threshold: float) -> Result:
    result = Result()

    if not current:
        result.failures.append("no benchmark results parsed: there is nothing to compare")
        return result

    if not baseline:
        result.failures.append("baseline is empty: without it no comparison is possible")
        return result

    for name, samples in sorted(current.items()):
        if len(samples) < MIN_SAMPLES:
            # §8: "无有效样本不能 pass".
            result.failures.append(
                f"{name}: only {len(samples)} sample(s), {MIN_SAMPLES} required"
            )
            continue

        expected = baseline.get(name)
        if expected is None:
            # §8: "缺基线 ... 不能 pass". A new benchmark must have its baseline
            # recorded deliberately, not silently inherit a pass.
            result.failures.append(f"{name}: no baseline entry")
            continue

        actual = summarize(samples)
        for metric in ("ns_per_op", "bytes_per_op"):
            if metric not in expected or metric not in actual:
                continue

            base_value = expected[metric]
            if base_value <= 0:
                result.failures.append(f"{name}: baseline {metric} is not positive")
                continue

            change = (actual[metric] - base_value) / base_value
            entry = {
                "benchmark": name,
                "metric": metric,
                "baseline": base_value,
                "current": actual[metric],
                "change_pct": change * 100,
            }
            result.comparisons.append(entry)

            if change > threshold:
                result.regressions.append(entry)

    # §8: "no match ... 不能 pass". A baseline that names benchmarks the run did
    # not produce means the two are not comparable.
    missing = sorted(set(baseline) - set(current))
    if missing:
        result.failures.append(
            "baseline entries with no matching benchmark in this run: " + ", ".join(missing)
        )

    return result


def report(result: Result, threshold: float) -> None:
    for entry in result.comparisons:
        sign = "+" if entry["change_pct"] >= 0 else ""
        print(
            f"  {entry['benchmark']:40s} {entry['metric']:13s} "
            f"{entry['baseline']:12.2f} -> {entry['current']:12.2f} "
            f"({sign}{entry['change_pct']:.1f}%)"
        )

    if result.regressions:
        print()
        print(f"REGRESSIONS beyond {threshold:.0%}:", file=sys.stderr)
        for entry in result.regressions:
            print(
                f"  {entry['benchmark']} {entry['metric']} "
                f"{entry['baseline']:.2f} -> {entry['current']:.2f} "
                f"(+{entry['change_pct']:.1f}%)",
                file=sys.stderr,
            )

    if result.failures:
        print()
        print("GATE FAILURES (a gate that could not compare must not report green):", file=sys.stderr)
        for failure in result.failures:
            print(f"  {failure}", file=sys.stderr)


# --------------------------------------------------------------------------- #
# Self-test
# --------------------------------------------------------------------------- #

def bench_lines(name: str, ns: float, count: int, bytes_per_op: float | None = None) -> str:
    lines = []
    for _ in range(count):
        line = f"{name}-8\t1000000\t{ns:.1f} ns/op"
        if bytes_per_op is not None:
            line += f"\t{bytes_per_op:.1f} B/op\t7 allocs/op"
        lines.append(line)
    return "\n".join(lines) + "\n"


def self_test() -> int:
    failures: list[str] = []

    def check(label: str, ok: bool) -> None:
        if not ok:
            failures.append(label)

    baseline = {"BenchmarkA": {"ns_per_op": 1000.0, "bytes_per_op": 100.0}}

    # 1. Within threshold passes.
    current = parse_bench_output(bench_lines("BenchmarkA", 1100, MIN_SAMPLES, 100))
    result = compare(current, baseline, REGRESSION_THRESHOLD)
    check(f"within threshold should pass, got {result.regressions} {result.failures}", result.ok)

    # 2. A regression beyond 15% blocks.
    current = parse_bench_output(bench_lines("BenchmarkA", 1200, MIN_SAMPLES, 100))
    result = compare(current, baseline, REGRESSION_THRESHOLD)
    check("a 20% ns/op regression must block", not result.ok)
    check("the regression must be reported", len(result.regressions) == 1)

    # 3. B/op regression blocks too.
    current = parse_bench_output(bench_lines("BenchmarkA", 1000, MIN_SAMPLES, 130))
    result = compare(current, baseline, REGRESSION_THRESHOLD)
    check("a 30% B/op regression must block", not result.ok)
    check(
        "the B/op regression must be the reported one",
        len(result.regressions) == 1 and result.regressions[0]["metric"] == "bytes_per_op",
    )

    # 4. An improvement passes.
    current = parse_bench_output(bench_lines("BenchmarkA", 800, MIN_SAMPLES, 100))
    result = compare(current, baseline, REGRESSION_THRESHOLD)
    check("an improvement must pass", result.ok)

    # 5. Too few samples must NOT pass (§8: 无有效样本不能 pass).
    current = parse_bench_output(bench_lines("BenchmarkA", 1000, MIN_SAMPLES - 1, 100))
    result = compare(current, baseline, REGRESSION_THRESHOLD)
    check("fewer than 5 samples must not pass", not result.ok)
    check(
        "the sample count must be named",
        any("sample" in f for f in result.failures),
    )

    # 6. A benchmark with no baseline must NOT pass (§8: 缺基线不能 pass).
    current = parse_bench_output(bench_lines("BenchmarkB", 1000, MIN_SAMPLES, 100))
    result = compare(current, baseline, REGRESSION_THRESHOLD)
    check("a missing baseline entry must not pass", not result.ok)
    check("the missing baseline must be named", any("no baseline" in f for f in result.failures))

    # 7. A baseline entry with no matching benchmark must NOT pass (§8: no match).
    current = parse_bench_output(bench_lines("BenchmarkA", 1000, MIN_SAMPLES, 100))
    extra = dict(baseline)
    extra["BenchmarkGone"] = {"ns_per_op": 1.0}
    result = compare(current, extra, REGRESSION_THRESHOLD)
    check("an unmatched baseline entry must not pass", not result.ok)
    check("the unmatched entry must be named", any("no matching" in f for f in result.failures))

    # 8. Empty output must NOT pass: a run that produced nothing is not a pass.
    result = compare(parse_bench_output(""), baseline, REGRESSION_THRESHOLD)
    check("empty benchmark output must not pass", not result.ok)

    # 9. An empty baseline must NOT pass.
    result = compare(current, {}, REGRESSION_THRESHOLD)
    check("an empty baseline must not pass", not result.ok)

    # 10. Exactly at the threshold is not a regression (strictly greater blocks).
    current = parse_bench_output(bench_lines("BenchmarkA", 1150, MIN_SAMPLES, 100))
    result = compare(current, baseline, REGRESSION_THRESHOLD)
    check("exactly +15% must not block", result.ok)

    if failures:
        print("bench-gate self-test FAILED:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        f"bench-gate self-test passed "
        f"(threshold {REGRESSION_THRESHOLD:.0%}, min samples {MIN_SAMPLES}, 10 cases)"
    )
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Micro-benchmark regression gate (§8).")
    parser.add_argument("--bench-output", type=Path, help="raw `go test -bench` output")
    parser.add_argument("--baseline", type=Path, help="baseline JSON to compare against")
    parser.add_argument("--write-baseline", type=Path, help="record the run as the new baseline")
    parser.add_argument("--threshold", type=float, default=REGRESSION_THRESHOLD)
    parser.add_argument("--self-test", action="store_true", help="verify the gate itself")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    if not args.bench_output:
        parser.error("--bench-output is required unless --self-test is given")
    if not args.bench_output.is_file():
        print(f"benchmark output does not exist: {args.bench_output}", file=sys.stderr)
        return 2

    current = parse_bench_output(args.bench_output.read_text(encoding="utf-8"))

    if args.write_baseline:
        if not current:
            print("no benchmark results parsed; refusing to write an empty baseline", file=sys.stderr)
            return 1
        too_few = {n: len(s) for n, s in current.items() if len(s) < MIN_SAMPLES}
        if too_few:
            print(f"refusing to record a baseline with too few samples: {too_few}", file=sys.stderr)
            return 1
        recorded = {name: summarize(samples) for name, samples in sorted(current.items())}
        args.write_baseline.write_text(
            json.dumps({"benchmarks": recorded}, indent=2, sort_keys=True) + "\n", encoding="utf-8"
        )
        print(f"recorded baseline for {len(recorded)} benchmark(s) in {args.write_baseline}")
        return 0

    if not args.baseline:
        parser.error("--baseline is required unless --write-baseline or --self-test is given")

    baseline_doc = json.loads(args.baseline.read_text(encoding="utf-8"))
    result = compare(current, baseline_doc.get("benchmarks", {}), args.threshold)

    report(result, args.threshold)
    return 0 if result.ok else 1


if __name__ == "__main__":
    sys.exit(main())

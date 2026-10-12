#!/usr/bin/env python3
"""verify-tests.py — Go `go test -json` (JSONL) verifier and empty-run detector.

Task: T01 of docs/specs/2026-10-12-web3-go-backend-delivery-plan.md (finding F21).

Why this exists
---------------
`go test` exiting 0 is NOT proof that tests ran. A suite can "pass" because:
  * it matched zero tests (`-run` typo, build tags not set),
  * the package has no test files at all (`ok ... [no test files]`),
  * every test was skipped,
  * a fixture silently skipped because a dependency was not ready.

This tool parses the JSONL that `go test -json` emits and makes those cases
fail explicitly. It is intentionally dependency-free (Python 3 stdlib only) so
CI can run it before any toolchain is installed.

Counting rules (per the plan's §7.2)
------------------------------------
  * Test-level actions   = JSONL events that carry a `Test` field.
                           -> pass / fail / skip counts.
  * Top-level tests      = distinct `Test` values WITHOUT a "/" (parent/child
                           subtest pass events are NOT counted as tests).
  * Package results      = events WITHOUT a `Test` field, Action in
                           {pass, fail, skip}.
  * No-test packages     = packages that emitted no test-level event at all.

Usage
-----
  verify-tests.py --jsonl unit.jsonl --jsonl integration.jsonl \
      [--required-cases manifest.json] [--min-tests N] \
      [--fail-on-unknown-skip] [--report report.json]

  verify-tests.py --self-test

Required-cases manifest format
------------------------------
  {
    "scenarios": [
      {"id": "E02", "lane": "contracts",
       "tests": ["TestCheckpointAdvances", "TestDuplicateBatch"]}
    ]
  }
Each listed test must exist and must have PASSED; a listed test that was
SKIPPED is a failure (required skips must be 0).

Exit codes
----------
  0  verification passed
  1  verification failed (details on stdout, JSON in --report)
  2  usage / IO error
"""

from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Iterable


# --------------------------------------------------------------------------- #
# Data model
# --------------------------------------------------------------------------- #
@dataclass
class TestStat:
    """Aggregated result for a single test name (across all packages)."""

    name: str
    package: str
    passed: bool = False
    failed: bool = False
    skipped: bool = False

    @property
    def final(self) -> str:
        if self.failed:
            return "fail"
        if self.passed:
            return "pass"
        if self.skipped:
            return "skip"
        return "unknown"


@dataclass
class PackageStat:
    name: str
    result: str = "unknown"  # pass | fail | skip | unknown
    has_tests: bool = False


@dataclass
class Report:
    test_actions: dict[str, int] = field(
        default_factory=lambda: {"pass": 0, "fail": 0, "skip": 0}
    )
    tests: dict[tuple[str, str], TestStat] = field(default_factory=dict)
    packages: dict[str, PackageStat] = field(default_factory=dict)
    parse_errors: list[str] = field(default_factory=list)

    @property
    def top_level_tests(self) -> dict[str, TestStat]:
        return {n: t for (p, n), t in self.tests.items() if "/" not in n}

    @property
    def no_test_packages(self) -> list[str]:
        return sorted(p for p, s in self.packages.items() if not s.has_tests)


# --------------------------------------------------------------------------- #
# Parsing
# --------------------------------------------------------------------------- #
def parse_jsonl(paths: Iterable[Path], report: Report) -> None:
    for path in paths:
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError as exc:  # pragma: no cover - defensive
            report.parse_errors.append(f"cannot read {path}: {exc}")
            continue
        for lineno, line in enumerate(text.splitlines(), start=1):
            line = line.strip()
            if not line or not line.startswith("{"):
                continue
            try:
                event: dict[str, Any] = json.loads(line)
            except json.JSONDecodeError:
                report.parse_errors.append(f"{path}:{lineno}: invalid JSON")
                continue
            _ingest(event, report)


def _ingest(event: dict[str, Any], report: Report) -> None:
    action = event.get("Action")
    package = event.get("Package", "")
    test = event.get("Test")

    if test:
        # Test-level event. Only terminal actions count.
        if action in ("pass", "fail", "skip"):
            key = (package, test)
            stat = report.tests.get(key)
            if stat is None:
                stat = TestStat(name=test, package=package)
                report.tests[key] = stat
            if action == "pass":
                stat.passed = True
                report.test_actions["pass"] += 1
            elif action == "fail":
                stat.failed = True
                report.test_actions["fail"] += 1
            elif action == "skip":
                stat.skipped = True
                report.test_actions["skip"] += 1
            pkg = report.packages.setdefault(package, PackageStat(name=package))
            pkg.has_tests = True
        return

    # Package-level event.
    if action in ("pass", "fail", "skip") and package:
        pkg = report.packages.setdefault(package, PackageStat(name=package))
        pkg.result = action


# --------------------------------------------------------------------------- #
# Verification
# --------------------------------------------------------------------------- #
def load_required_cases(path: Path) -> dict[str, list[str]]:
    data = json.loads(path.read_text(encoding="utf-8"))
    out: dict[str, list[str]] = {}
    for scenario in data.get("scenarios", []):
        out[scenario["id"]] = list(scenario.get("tests", []))
    return out


def verify(
    report: Report,
    required_cases: dict[str, list[str]],
    min_tests: int | None,
    fail_on_unknown_skip: bool,
    require_packages: list[str],
) -> list[str]:
    """Return a list of human-readable failures (empty == passed)."""
    failures: list[str] = []

    if report.parse_errors:
        failures.extend(f"parse error: {e}" for e in report.parse_errors)

    # 1. No test executed at all -> empty run.
    if not report.tests:
        failures.append(
            "empty-run: no test-level events found in any JSONL input "
            "(suite matched zero tests, or build tags were not set)"
        )
        return failures

    # 2. Real failures.
    for (pkg, name), stat in sorted(report.tests.items()):
        if stat.failed:
            failures.append(f"test failed: {pkg}::{name}")

    # 3. Minimum test count.
    if min_tests is not None and len(report.top_level_tests) < min_tests:
        failures.append(
            f"empty-run: expected >= {min_tests} top-level tests, "
            f"found {len(report.top_level_tests)}"
        )

    # 4. Packages that were required to run but emitted no tests.
    for pkg in require_packages:
        stat = report.packages.get(pkg)
        if stat is None:
            failures.append(f"required package did not run: {pkg}")
        elif not stat.has_tests:
            failures.append(f"required package ran no tests (empty-run): {pkg}")

    # 5. Required cases.
    #
    # A manifest that declares scenarios but requires ZERO tests is a
    # placeholder, and a gate with nothing to check cannot fail. That is the
    # same class of defect as an empty suite (F13): it reports green while
    # proving nothing. Fail loudly instead of passing vacuously.
    if required_cases and not any(required_cases.values()):
        failures.append(
            "required-cases manifest requires 0 tests across "
            f"{len(required_cases)} scenario(s): it is a placeholder, so this "
            "gate cannot fail (fill in the test names, or drop --required-cases)"
        )
    for scenario_id, names in sorted(required_cases.items()):
        for name in names:
            matches = [
                stat for (_pkg, tname), stat in report.tests.items() if tname == name
            ]
            if not matches:
                failures.append(
                    f"required case {scenario_id}: test '{name}' did not run"
                )
                continue
            final = matches[0].final
            if final == "skip":
                failures.append(
                    f"required case {scenario_id}: test '{name}' was SKIPPED "
                    "(required skips must be 0)"
                )
            elif final != "pass":
                failures.append(
                    f"required case {scenario_id}: test '{name}' result = {final}"
                )

    # 6. Unknown skips (optional, off by default).
    if fail_on_unknown_skip:
        for (pkg, name), stat in sorted(report.tests.items()):
            if stat.final == "skip":
                failures.append(f"unknown skip (not allow-listed): {pkg}::{name}")

    return failures


def to_json(report: Report, failures: list[str]) -> dict[str, Any]:
    return {
        "schema": "chainpulse.tests.v1",
        "ok": not failures,
        "test_actions": report.test_actions,
        "top_level_test_count": len(report.top_level_tests),
        "package_count": len(report.packages),
        "packages": {p: s.result for p, s in sorted(report.packages.items())},
        "no_test_packages": report.no_test_packages,
        "failed_tests": sorted(
            f"{p}::{n}" for (p, n), s in report.tests.items() if s.failed
        ),
        "skipped_tests": sorted(
            f"{p}::{n}" for (p, n), s in report.tests.items() if s.final == "skip"
        ),
        "failures": failures,
    }


# --------------------------------------------------------------------------- #
# Self-test (no Go toolchain required)
# --------------------------------------------------------------------------- #
_SELF_TEST_FIXTURES: list[tuple[str, list[str], bool]] = [
    # (name, jsonl lines, expect_ok)
    (
        "happy path",
        [
            '{"Action":"run","Package":"p/foo","Test":"TestA"}',
            '{"Action":"pass","Package":"p/foo","Test":"TestA"}',
            '{"Action":"pass","Package":"p/foo","Elapsed":0.1}',
        ],
        True,
    ),
    (
        "assertion failure",
        [
            '{"Action":"run","Package":"p/foo","Test":"TestA"}',
            '{"Action":"fail","Package":"p/foo","Test":"TestA"}',
            '{"Action":"fail","Package":"p/foo","Elapsed":0.1}',
        ],
        False,
    ),
    (
        "empty run (no tests at all)",
        [
            '{"Action":"run","Package":"p/foo"}',
            '{"Action":"pass","Package":"p/foo","Elapsed":0.1}',
        ],
        False,
    ),
    (
        "subtest pass does not inflate top-level count",
        [
            '{"Action":"run","Package":"p/foo","Test":"TestA"}',
            '{"Action":"pass","Package":"p/foo","Test":"TestA/sub"}',
            '{"Action":"pass","Package":"p/foo","Test":"TestA"}',
            '{"Action":"pass","Package":"p/foo","Elapsed":0.1}',
        ],
        True,
    ),
]


def self_test() -> int:
    ok = True
    for name, lines, expect_ok in _SELF_TEST_FIXTURES:
        report = Report()
        for line in lines:
            _ingest(json.loads(line), report)
        failures = verify(report, {}, None, False, [])
        got_ok = not failures
        status = "PASS" if got_ok == expect_ok else "FAIL"
        if got_ok != expect_ok:
            ok = False
        print(f"[{status}] self-test: {name} (failures={failures})")

    # Top-level counting rule.
    report = Report()
    for line in _SELF_TEST_FIXTURES[3][1]:
        _ingest(json.loads(line), report)
    if len(report.top_level_tests) != 1:
        ok = False
        print(
            f"[FAIL] self-test: top-level count = {len(report.top_level_tests)}, want 1"
        )
    else:
        print("[PASS] self-test: top-level count excludes subtests")

    # Required-case: a skipped required test must fail.
    report = Report()
    for line in [
        '{"Action":"run","Package":"p/foo","Test":"TestB"}',
        '{"Action":"skip","Package":"p/foo","Test":"TestB"}',
        '{"Action":"pass","Package":"p/foo","Elapsed":0.1}',
    ]:
        _ingest(json.loads(line), report)
    failures = verify(report, {"E01": ["TestB"]}, None, False, [])
    if failures:
        print("[PASS] self-test: required skip is a failure")
    else:
        ok = False
        print("[FAIL] self-test: required skip was not treated as a failure")

    # Required-case: a placeholder manifest (scenarios declared, zero tests)
    # must FAIL rather than pass vacuously.
    report = Report()
    for line in [
        '{"Action":"run","Package":"p/foo","Test":"TestA"}',
        '{"Action":"pass","Package":"p/foo","Test":"TestA"}',
        '{"Action":"pass","Package":"p/foo","Elapsed":0.1}',
    ]:
        _ingest(json.loads(line), report)
    failures = verify(report, {"E01": [], "E02": []}, None, False, [])
    if any("placeholder" in f for f in failures):
        print("[PASS] self-test: placeholder required-cases manifest is a failure")
    else:
        ok = False
        print("[FAIL] self-test: placeholder manifest passed vacuously")

    print("self-test:", "OK" if ok else "FAILED")
    return 0 if ok else 1


# --------------------------------------------------------------------------- #
# CLI
# --------------------------------------------------------------------------- #
def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--jsonl", action="append", default=[], type=Path)
    parser.add_argument("--required-cases", type=Path)
    parser.add_argument("--min-tests", type=int)
    parser.add_argument("--require-package", action="append", default=[])
    parser.add_argument("--fail-on-unknown-skip", action="store_true")
    parser.add_argument("--report", type=Path)
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    if not args.jsonl:
        parser.error("at least one --jsonl is required (or use --self-test)")

    report = Report()
    parse_jsonl(args.jsonl, report)

    required = load_required_cases(args.required_cases) if args.required_cases else {}
    failures = verify(
        report,
        required,
        args.min_tests,
        args.fail_on_unknown_skip,
        args.require_package,
    )

    summary = to_json(report, failures)
    if args.report:
        args.report.write_text(json.dumps(summary, indent=2), encoding="utf-8")

    print(
        "tests: "
        f"pass={report.test_actions['pass']} "
        f"fail={report.test_actions['fail']} "
        f"skip={report.test_actions['skip']} "
        f"top_level={len(report.top_level_tests)} "
        f"packages={len(report.packages)}"
    )
    if report.no_test_packages:
        print(f"no-test packages ({len(report.no_test_packages)}):")
        for pkg in report.no_test_packages:
            print(f"  - {pkg}")

    if failures:
        print(f"\n✗ verification FAILED ({len(failures)} issue(s)):")
        for f in failures:
            print(f"  - {f}")
        return 1

    print("\n✓ verification passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""negative-cases.py - prove the acceptance gates actually fire.

Plan section 7.4 ("CI 本身的反例验收") requires each injected fault to produce a
failure, and each gate to be shown failing rather than assumed to. It lists the
injections in a table:

    | injection                                   | expected |
    | compile error / assertion failure / race     | failure  |
    | timeout / panic / exit 1 with no `--- FAIL`  | failure, not text-grep |
    | `-run` matches zero tests / runner deleted   | manifest/empty-run failure |
    | Foundry missing / DB not ready / required skip | failure |
    | coverage or perf artifact missing, empty, NaN, wrong schema | failure |
    | throughput drop, p99 over budget, checksum   | failure  |
    | prior job skipped / CD credential missing    | release/deploy failure |

The plan wants those injections run on an isolated branch and the failing run
URLs kept. The run URL needs GitHub; the decision logic does not. This script
covers the half that is checkable here: for every row, it feeds the repository's
own acceptance tool an adversarial input and asserts the tool exits non-zero,
then feeds it the corresponding clean input and asserts it exits zero.

Why the second half matters
---------------------------
A tool that always fails would satisfy every "expected: failure" row while being
useless. Each negative case is therefore paired with a positive control, and the
control must pass.

Why not just trust the tools' own --self-test
---------------------------------------------
Because a self-test can be wrong in the same way the tool is. Several of this
session's findings were exactly that: a gate whose own test asserted the
behaviour the gate already had. This script drives each tool from the outside,
through its CLI, and only reads the exit code.

Usage
-----
    python3 scripts/acceptance/negative-cases.py            # run the matrix
    python3 scripts/acceptance/negative-cases.py --json out.json
    python3 scripts/acceptance/negative-cases.py --list

Exit code 0 only when every case produced its expected verdict.

Not covered here (needs GitHub or a deployment target): the release/deploy rows -
prior job skipped, CD credential missing, smoke failure - are only partly
reachable. The `acceptance-gate` skipped/neutral/missing cases below cover the
decision logic those rows depend on.
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass, field
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
TOOLS = REPO_ROOT / "scripts" / "acceptance"
PYTHON = sys.executable


@dataclass
class Case:
    name: str
    row: str  # the section 7.4 row this covers
    tool: str
    args: list[str]
    expect: str  # "fail" | "pass"
    files: dict[str, str] = field(default_factory=dict)
    note: str = ""


# --------------------------------------------------------------------------- #
# Fixtures
# --------------------------------------------------------------------------- #
CLEAN_JSONL = "\n".join(
    [
        '{"Action":"run","Package":"p/foo","Test":"TestA"}',
        '{"Action":"pass","Package":"p/foo","Test":"TestA","Elapsed":0.01}',
        '{"Action":"pass","Package":"p/foo","Elapsed":0.02}',
    ]
)

ASSERTION_FAIL_JSONL = "\n".join(
    [
        '{"Action":"run","Package":"p/foo","Test":"TestA"}',
        '{"Action":"output","Package":"p/foo","Test":"TestA","Output":"    x_test.go:9: got 1, want 2\\n"}',
        '{"Action":"output","Package":"p/foo","Test":"TestA","Output":"--- FAIL: TestA (0.00s)\\n"}',
        '{"Action":"fail","Package":"p/foo","Test":"TestA","Elapsed":0.01}',
        '{"Action":"fail","Package":"p/foo","Elapsed":0.02}',
    ]
)

# The shape of a panic outside any test, a TestMain that exits non-zero, or a
# timeout kill: `go test` exits 1 and the package reports `fail`, but no Test
# ever carries a fail action, so there is no `--- FAIL` to grep for.
NO_FAIL_LINE_JSONL = "\n".join(
    [
        '{"Action":"run","Package":"p/good","Test":"TestGood"}',
        '{"Action":"pass","Package":"p/good","Test":"TestGood","Elapsed":0.01}',
        '{"Action":"pass","Package":"p/good","Elapsed":0.02}',
        '{"Action":"output","Package":"p/broken","Output":"panic: boom\\n"}',
        '{"Action":"fail","Package":"p/broken","Elapsed":9.0}',
    ]
)

RACE_JSONL = "\n".join(
    [
        '{"Action":"run","Package":"p/foo","Test":"TestRace"}',
        '{"Action":"output","Package":"p/foo","Test":"TestRace",'
        '"Output":"==================\\nWARNING: DATA RACE\\n"}',
        '{"Action":"output","Package":"p/foo","Test":"TestRace",'
        '"Output":"--- FAIL: TestRace (0.03s)\\n"}',
        '{"Action":"fail","Package":"p/foo","Test":"TestRace","Elapsed":0.03}',
        '{"Action":"fail","Package":"p/foo","Elapsed":0.04}',
    ]
)

EMPTY_JSONL = '{"Action":"output","Package":"p/foo","Output":"?   p/foo\\t[no test files]\\n"}\n{"Action":"skip","Package":"p/foo","Elapsed":0}'

REQUIRED_SKIP_JSONL = "\n".join(
    [
        '{"Action":"run","Package":"p/foo","Test":"TestNeedsDB"}',
        '{"Action":"skip","Package":"p/foo","Test":"TestNeedsDB"}',
        '{"Action":"pass","Package":"p/foo","Elapsed":0.1}',
    ]
)

NO_TEST_PACKAGE_JSONL = "\n".join(
    [
        '{"Action":"run","Package":"p/foo","Test":"TestA"}',
        '{"Action":"pass","Package":"p/foo","Test":"TestA","Elapsed":0.01}',
        '{"Action":"pass","Package":"p/foo","Elapsed":0.02}',
        '{"Action":"skip","Package":"p/empty","Elapsed":0}',
    ]
)

REQUIRED_CASES_OK = json.dumps({"scenarios": [{"id": "E01", "tests": ["TestA"]}]})
REQUIRED_CASES_MISSING = json.dumps(
    {"scenarios": [{"id": "E01", "tests": ["TestNeverRuns"]}]}
)

BENCH_CLEAN = """goos: linux
goarch: amd64
BenchmarkDecodeEventDataTransfer-8   	  100000	      1550 ns/op	     904 B/op	       5 allocs/op
BenchmarkDecodeEventDataTransfer-8   	  100000	      1560 ns/op	     904 B/op	       5 allocs/op
BenchmarkDecodeEventDataTransfer-8   	  100000	      1545 ns/op	     904 B/op	       5 allocs/op
BenchmarkDecodeEventDataTransfer-8   	  100000	      1555 ns/op	     904 B/op	       5 allocs/op
BenchmarkDecodeEventDataTransfer-8   	  100000	      1552 ns/op	     904 B/op	       5 allocs/op
PASS
ok  	x/evm	1.2s
"""

BENCH_REGRESSION = BENCH_CLEAN.replace("1550", "4000").replace(
    "1560", "4100"
).replace("1545", "4050").replace("1555", "4090").replace("1552", "4020")

BENCH_NAN = (
    "BenchmarkDecodeEventDataTransfer-8   	  100000	       NaN ns/op	     904 B/op	       5 allocs/op\n"
)


def build_cases(work: Path) -> list[Case]:
    """Write fixtures under `work` and return the case list."""
    f = work

    def w(name: str, text: str) -> str:
        (f / name).write_text(text if text.endswith("\n") else text + "\n")
        return str(f / name)

    clean = w("clean.jsonl", CLEAN_JSONL)
    assertion = w("assertion.jsonl", ASSERTION_FAIL_JSONL)
    nofail = w("nofail.jsonl", NO_FAIL_LINE_JSONL)
    race = w("race.jsonl", RACE_JSONL)
    empty = w("empty.jsonl", EMPTY_JSONL)
    req_skip = w("reqskip.jsonl", REQUIRED_SKIP_JSONL)
    no_test_pkg = w("notestpkg.jsonl", NO_TEST_PACKAGE_JSONL)
    rc_ok = w("required_ok.json", REQUIRED_CASES_OK)
    rc_missing = w("required_missing.json", REQUIRED_CASES_MISSING)

    bench_clean = w("bench_clean.txt", BENCH_CLEAN)
    bench_reg = w("bench_regression.txt", BENCH_REGRESSION)
    bench_nan = w("bench_nan.txt", BENCH_NAN)
    bench_empty = w("bench_empty.txt", "")

    checks_ok = w(
        "checks_ok.json",
        json.dumps(
            {"checks": [{"name": n, "conclusion": "success"} for n in ("test", "lint")]}
        ),
    )
    checks_skipped = w(
        "checks_skipped.json",
        json.dumps(
            {
                "checks": [
                    {"name": "test", "conclusion": "success"},
                    {"name": "lint", "conclusion": "skipped"},
                ]
            }
        ),
    )
    checks_neutral = w(
        "checks_neutral.json",
        json.dumps(
            {
                "checks": [
                    {"name": "test", "conclusion": "success"},
                    {"name": "lint", "conclusion": "neutral"},
                ]
            }
        ),
    )
    checks_missing = w(
        "checks_missing.json",
        json.dumps({"checks": [{"name": "test", "conclusion": "success"}]}),
    )

    dep_changed = w("changed_dep.txt", "go.mod\ngo.sum")
    dep_approved = w(
        "changed_dep_ok.txt", "go.mod\ndocs/project/DEPENDENCY_APPROVAL.md"
    )

    baseline = str(REPO_ROOT / "test" / "benchmarks" / "micro_baseline.json")
    clean_scan = w("scan_empty.json", "")

    V = "verify-tests.py"
    A = "acceptance-gate.py"
    B = "bench-gate.py"
    D = "dep-approval-gate.py"
    G = "govulncheck-summary.py"

    return [
        # ---- verify-tests: the test-execution gate -------------------------
        Case("assertion-failure", "compile error / assertion failure / race",
             V, ["--jsonl", assertion, "--min-tests", "1"], "fail"),
        Case("race-detector-failure", "compile error / assertion failure / race",
             V, ["--jsonl", race, "--min-tests", "1"], "fail"),
        Case("package-fail-without-FAIL-line",
             "timeout / panic / exit 1 with no --- FAIL",
             V, ["--jsonl", nofail, "--min-tests", "1"], "fail",
             note="the gap this session found and fixed"),
        Case("run-matches-zero-tests", "-run matches zero tests / runner deleted",
             V, ["--jsonl", empty, "--min-tests", "1"], "fail"),
        Case("required-case-never-ran",
             "-run matches zero tests / runner deleted",
             V, ["--jsonl", clean, "--required-cases", rc_missing], "fail"),
        Case("required-skip", "Foundry missing / DB not ready / required skip",
             V, ["--jsonl", req_skip, "--fail-on-unknown-skip"], "fail"),
        # controls
        Case("control-happy-path", "(control)", V,
             ["--jsonl", clean, "--min-tests", "1"], "pass"),
        Case("control-required-case-present", "(control)", V,
             ["--jsonl", clean, "--required-cases", rc_ok], "pass"),
        Case("control-no-test-package", "(control)", V,
             ["--jsonl", no_test_pkg, "--min-tests", "1"], "pass",
             note="a package with no test files must not be a failure"),
        # ---- acceptance-gate: the aggregation gate -------------------------
        Case("prior-check-skipped", "prior job skipped / CD credential missing",
             A, ["--checks", checks_skipped, "--required", "test", "lint"], "fail"),
        Case("prior-check-neutral", "prior job skipped / CD credential missing",
             A, ["--checks", checks_neutral, "--required", "test", "lint"], "fail"),
        Case("prior-check-missing", "prior job skipped / CD credential missing",
             A, ["--checks", checks_missing, "--required", "test", "lint"], "fail"),
        Case("control-all-checks-success", "(control)", A,
             ["--checks", checks_ok, "--required", "test", "lint"], "pass"),
        # ---- bench-gate: the performance artifact gate ---------------------
        Case("perf-artifact-empty", "coverage/perf artifact missing, empty",
             B, ["--bench-output", bench_empty, "--baseline", baseline], "fail"),
        Case("perf-artifact-nan", "coverage/perf artifact NaN / wrong schema",
             B, ["--bench-output", bench_nan, "--baseline", baseline], "fail"),
        Case("perf-regression-over-budget", "throughput drop / p99 over budget",
             B, ["--bench-output", bench_reg, "--baseline", baseline], "fail"),
        Case("control-perf-within-budget", "(control)", B,
             ["--bench-output", bench_clean, "--baseline", baseline], "pass"),
        # ---- dep-approval-gate --------------------------------------------
        Case("dependency-changed-without-approval",
             "dependency manifest changed without approval record",
             D, ["--changed-files", dep_changed], "fail"),
        Case("control-dependency-approved", "(control)", D,
             ["--changed-files", dep_approved], "pass"),
        # ---- govulncheck-summary: the security artifact --------------------
        Case("security-artifact-undecodable",
             "security artifact wrong schema / decode broken",
             G, ["--input", clean_scan], "fail"),
    ]


def run_case(case: Case) -> tuple[bool, int, str]:
    """Run one case. Returns (as_expected, exit_code, first line of output)."""
    tool = TOOLS / case.tool
    cmd = [PYTHON, str(tool), *case.args]
    proc = subprocess.run(
        cmd, cwd=REPO_ROOT, capture_output=True, text=True, timeout=600
    )
    rc = proc.returncode
    got = "fail" if rc != 0 else "pass"
    first = ""
    for line in (proc.stdout + proc.stderr).splitlines():
        line = line.strip()
        if line:
            first = line
            break
    return got == case.expect, rc, first


def real_build_failure_check(work: Path) -> tuple[bool, str]:
    """Reproduce the no-`--- FAIL` case with a real `go test`, not a fixture.

    Two packages: one that compiles and passes, one that does not compile. That
    is the case a line-oriented reader misses - `go test` exits 1, the JSONL has
    a package-level fail, and no Test carries a fail action.
    """
    go = shutil.which("go")
    if go is None:
        return True, "skipped (no go toolchain)"

    root = work / "real"
    (root / "good").mkdir(parents=True, exist_ok=True)
    (root / "broken").mkdir(parents=True, exist_ok=True)
    (root / "go.mod").write_text("module negcase\n\ngo 1.26.9\n")
    (root / "good" / "good_test.go").write_text(
        "package good\n\nimport \"testing\"\n\nfunc TestGood(t *testing.T) {}\n"
    )
    (root / "broken" / "broken_test.go").write_text(
        "package broken\n\nimport \"testing\"\n\n"
        "func TestBroken(t *testing.T) { undefinedSymbolHere() }\n"
    )

    env = dict(os.environ)
    env.setdefault("GOPROXY", "https://goproxy.cn,direct")
    env.setdefault("GOCACHE", "/tmp/chainpulse-go-build-cache")

    proc = subprocess.run(
        [go, "test", "-json", "-count=1", "./..."],
        cwd=root, capture_output=True, text=True, env=env, timeout=600,
    )
    if proc.returncode == 0:
        return False, "go test unexpectedly exited 0"

    jsonl = work / "real_build_failure.jsonl"
    jsonl.write_text(proc.stdout)
    verifier = subprocess.run(
        [PYTHON, str(TOOLS / "verify-tests.py"), "--jsonl", str(jsonl),
         "--min-tests", "1"],
        cwd=REPO_ROOT, capture_output=True, text=True, timeout=120,
    )
    if verifier.returncode == 0:
        return False, f"verifier passed a run where go test exited {proc.returncode}"
    return True, f"go test exit={proc.returncode}, verifier exit={verifier.returncode}"


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--json", type=Path, default=None)
    parser.add_argument("--list", action="store_true")
    args = parser.parse_args(argv)

    with tempfile.TemporaryDirectory(prefix="negcases-") as td:
        work = Path(td)
        cases = build_cases(work)

        if args.list:
            for c in cases:
                print(f"{c.expect:4}  {c.name:38} {c.row}")
            return 0

        results = []
        ok = True
        print(f"{'result':7} {'expect':7} {'case':38} tool")
        print("-" * 78)
        for c in cases:
            as_expected, rc, first = run_case(c)
            ok = ok and as_expected
            results.append(
                {
                    "case": c.name,
                    "section_7_4_row": c.row,
                    "tool": c.tool,
                    "expected": c.expect,
                    "exit_code": rc,
                    "as_expected": as_expected,
                    "note": c.note,
                }
            )
            mark = "ok" if as_expected else "MISMATCH"
            print(f"{mark:7} {c.expect:7} {c.name:38} {c.tool}")

        real_ok, real_detail = real_build_failure_check(work)
        ok = ok and real_ok
        results.append(
            {
                "case": "real-build-failure-no-FAIL-line",
                "section_7_4_row": "timeout / panic / exit 1 with no --- FAIL",
                "tool": "verify-tests.py",
                "expected": "fail",
                "detail": real_detail,
                "as_expected": real_ok,
            }
        )
        print(f"{'ok' if real_ok else 'MISMATCH':7} {'fail':7} "
              f"{'real-build-failure-no-FAIL-line':38} verify-tests.py")
        print(f"        detail: {real_detail}")

    passed = sum(1 for r in results if r["as_expected"])
    print(f"\nnegative-case matrix: {passed}/{len(results)} as expected")

    if args.json:
        args.json.write_text(
            json.dumps(
                {"cases": results, "passed": passed, "total": len(results)},
                indent=2,
            )
            + "\n"
        )
        print(f"wrote {args.json}")

    if not ok:
        print("\n✗ the acceptance gates did not all fire as required")
        return 1
    print("✓ every gate fired on its negative input and passed its positive control")
    return 0


if __name__ == "__main__":
    sys.exit(main())

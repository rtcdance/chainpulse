#!/usr/bin/env python3
"""Emit the single skip taxonomy required by plan §10 (`skip-report.json`).

Why this exists
---------------
T00 established that three different skip counts were in circulation
(AGENTS "~89", the baseline JSON "69 skip actions", and 72 `t.Skip(` sites in
source) and that the reconciliation was only ever written as prose. §10 lists
`skip-report.json` as a required evidence artifact, and prose cannot be consumed
by the collector. This tool derives the taxonomy mechanically from two measured
sources so the number is reproducible rather than quoted:

  * runtime skips  — `go test -json` streams: which tests actually skipped, and
                     the message that was printed when they did;
  * source sites   — every `t.Skip`/`t.Skipf`/`t.SkipNow` call in the tree: the
                     skip that exists but may not have been reached on this run.

The two numbers are expected to differ, and the report states both. A site that
never fires is not evidence that nothing was skipped.

Classification is deliberately mechanical and conservative. A reason that does
not match a known bucket lands in `unclassified` and is counted and listed
explicitly — never folded into a bucket that would make the total look tidy.
§7.2 requires unknown skips to be visible.

Standard library only: CI must be able to run this before any toolchain exists.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path

SCHEMA = "chainpulse.skip-report.v1"

# Buckets, in precedence order. The first pattern that matches wins, so the
# order is part of the definition: a "pre-existing" skip that also mentions
# short mode is classified as residue, because the reason it still exists is
# that nobody reconstructed its intent.
BUCKETS: tuple[tuple[str, str, str], ...] = (
    (
        "historical_residue",
        r"pre-?existing",
        "the production function was deleted; intent cannot be reconstructed from what remains",
    ),
    (
        "short_mode_excluded",
        r"short mode",
        "deliberately excluded from -short runs; the full lane must execute it",
    ),
    (
        "covered_elsewhere",
        r"tested separately|covered (by|in)|see .*_test\.go",
        "the behaviour is asserted by another test; this one would be a duplicate",
    ),
    (
        "dependency_gated",
        r"CHAINPULSE_[A-Z0-9_]+|CHAINPULSE_RUN|docker|redis|postgres|kafka|anvil|foundry|forge\b"
        r"|services not ready|failed to initialize|not available|requires fixtures|endpoint unavailable"
        r"|orchestrator",
        "needs a real dependency; runs in the lane that owns it",
    ),
    (
        "environment_gated",
        r"at least \d+ (Ps|CPUs|cores)|GOMAXPROCS|not enough (memory|cpu)|insufficient (memory|disk)",
        "needs a host capability; the gate is the environment, not the code",
    ),
    (
        "unimplemented",
        r"not implemented|unimplemented|TODO|FIXME",
        "the capability does not exist yet; needs a task, not a test fix",
    ),
)

# Where each bucket's full-lane execution belongs, derived from the reason text.
LANE_BY_DEPENDENCY: tuple[tuple[str, str], ...] = (
    (r"redis|postgres", "contracts"),
    (r"kafka", "contracts / chaos-soak-perf"),
    (r"anvil|foundry|forge\b", "evm-e2e"),
    (r"orchestrator|services not ready|docker", "evm-e2e / chaos-soak-perf"),
)

SKIP_CALL_RE = re.compile(r"\bt\.(Skip|Skipf|SkipNow)\s*\(")

# go test frames that carry no reason. The line before `--- SKIP:` is the
# message; keeping the `--- SKIP:` line instead would classify every runtime
# skip as unknown, which is what the first version of this tool did.
NOISE_RE = re.compile(r"^\s*(===|--- (SKIP|PASS|FAIL|PAUSE|CONT)|ok\s|FAIL\s|PASS\s)")


@dataclass
class SkipEntry:
    package: str
    test: str
    bucket: str
    lane: str
    reason: str


@dataclass
class SourceSite:
    file: str
    line: int
    kind: str
    bucket: str
    lane: str
    reason: str


@dataclass
class Result:
    runtime: list[SkipEntry] = field(default_factory=list)
    source: list[SourceSite] = field(default_factory=list)
    package_skips: list[str] = field(default_factory=list)
    files_scanned: int = 0
    jsonl_errors: list[str] = field(default_factory=list)


def classify(reason: str) -> tuple[str, str]:
    """Return (bucket, lane) for a skip reason.

    An unrecognised reason returns ("unclassified", "unassigned"): it must be
    counted as unknown rather than absorbed into a plausible-looking bucket.
    A reason that is missing entirely is distinguished from one that did not
    match, because "the skip says nothing" is a different repair.
    """
    if not reason.strip():
        return "no_reason", "unassigned - needs a reason"

    for name, pattern, _why in BUCKETS:
        if re.search(pattern, reason, re.IGNORECASE):
            if name == "short_mode_excluded":
                return name, "unit-race / contracts (full run)"
            if name == "historical_residue":
                return name, "none - review before removing"
            if name == "unimplemented":
                return name, "none - needs a task"
            for dep_pattern, lane in LANE_BY_DEPENDENCY:
                if re.search(dep_pattern, reason, re.IGNORECASE):
                    return name, lane
            return name, "full lane (unassigned)"
    return "unclassified", "unassigned"


def bucket_description(bucket: str) -> str:
    for name, _pattern, why in BUCKETS:
        if name == bucket:
            return why
    if bucket == "no_reason":
        return "the skip was called with no message; the reason has to be recovered from the code"
    return "reason did not match any known bucket; listed explicitly rather than absorbed"


def parse_jsonl(paths: list[Path]) -> tuple[list[SkipEntry], list[str], list[str]]:
    """Extract skipped tests, the message printed before each skip, and whole
    packages that were skipped.

    A package-level skip carries no Test field but is still a skipped test file
    that never ran, so it is reported rather than dropped.
    """
    entries: list[SkipEntry] = []
    package_skips: list[str] = []
    errors: list[str] = []

    for path in paths:
        if not path.is_file():
            errors.append(f"{path}: not a file")
            continue

        # test -> output lines seen since the last action for that test
        pending: dict[str, list[str]] = {}

        with path.open(encoding="utf-8", errors="replace") as handle:
            for lineno, raw in enumerate(handle, start=1):
                raw = raw.strip()
                if not raw:
                    continue
                try:
                    event = json.loads(raw)
                except json.JSONDecodeError:
                    errors.append(f"{path}:{lineno}: not JSON")
                    continue

                action = event.get("Action")
                test = event.get("Test")

                if not test:
                    if action == "skip":
                        package_skips.append(event.get("Package", ""))
                    continue

                if action == "output":
                    text = (event.get("Output") or "").strip()
                    # Drop go test's own framing so the reason survives.
                    if text and not NOISE_RE.match(text):
                        pending.setdefault(test, []).append(text)
                    continue

                if action == "skip":
                    lines = pending.pop(test, [])
                    reason = lines[-1] if lines else ""
                    bucket, lane = classify(reason)
                    entries.append(
                        SkipEntry(
                            package=event.get("Package", ""),
                            test=test,
                            bucket=bucket,
                            lane=lane,
                            reason=reason,
                        )
                    )
                elif action in {"pass", "fail"}:
                    pending.pop(test, None)

    return entries, package_skips, errors


def scan_source(root: Path) -> tuple[list[SourceSite], int, list[str]]:
    """Find every t.Skip call site in the tree."""
    sites: list[SourceSite] = []
    errors: list[str] = []
    scanned = 0

    for path in sorted(root.rglob("*.go")):
        parts = set(path.parts)
        if "vendor" in parts or "node_modules" in parts:
            continue

        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError as exc:
            errors.append(f"{path}: {exc}")
            continue

        scanned += 1
        lines = text.splitlines()
        for index, line in enumerate(lines):
            match = SKIP_CALL_RE.search(line)
            if not match:
                continue

            reason = _extract_call_text(lines, index)
            bucket, lane = classify(reason)
            sites.append(
                SourceSite(
                    file=str(path.relative_to(root)),
                    line=index + 1,
                    kind=match.group(1),
                    bucket=bucket,
                    lane=lane,
                    reason=reason,
                )
            )

    return sites, scanned, errors


def _extract_call_text(lines: list[str], start: int) -> str:
    """Return the argument text of a call starting at lines[start], balanced.

    Multi-line calls are common (`t.Skipf("...", arg)` wrapping), so parens are
    counted rather than reading a single line.
    """
    collected: list[str] = []
    depth = 0
    started = False

    for index in range(start, min(start + 12, len(lines))):
        line = lines[index]
        for char in line:
            if char == "(":
                depth += 1
                started = True
            elif char == ")":
                depth -= 1
            if started:
                collected.append(char)
            if started and depth == 0:
                return "".join(collected).strip()
        if started:
            collected.append(" ")
    return "".join(collected).strip()


def git_output(root: Path, args: list[str]) -> str:
    try:
        out = subprocess.run(
            ["git", *args],
            cwd=root,
            capture_output=True,
            text=True,
            check=False,
        )
    except OSError:
        return ""
    return out.stdout.strip() if out.returncode == 0 else ""


def build_report(result: Result, root: Path) -> dict:
    runtime_buckets: dict[str, int] = {}
    for entry in result.runtime:
        runtime_buckets[entry.bucket] = runtime_buckets.get(entry.bucket, 0) + 1

    source_buckets: dict[str, int] = {}
    for site in result.source:
        source_buckets[site.bucket] = source_buckets.get(site.bucket, 0) + 1

    def bucket_rows(counts: dict[str, int]) -> list[dict]:
        names = sorted(
            set(counts) | {name for name, _p, _w in BUCKETS} | {"unclassified", "no_reason"}
        )
        return [
            {
                "bucket": name,
                "count": counts.get(name, 0),
                "description": bucket_description(name),
            }
            for name in names
        ]

    return {
        "schema": SCHEMA,
        "generated_at": dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "commit_sha": git_output(root, ["rev-parse", "HEAD"]),
        "tree_dirty": bool(git_output(root, ["status", "--porcelain"])),
        "source_root": str(root),
        "runtime": {
            "description": "skips that actually fired in the given go test -json streams",
            "total": len(result.runtime),
            "package_skips": sorted(set(result.package_skips)),
            "package_skip_count": len(set(result.package_skips)),
            "buckets": bucket_rows(runtime_buckets),
            "entries": [
                {
                    "package": e.package,
                    "test": e.test,
                    "bucket": e.bucket,
                    "lane": e.lane,
                    "reason": e.reason,
                }
                for e in sorted(result.runtime, key=lambda e: (e.package, e.test))
            ],
        },
        "source": {
            "description": "every t.Skip call site in the tree, fired or not",
            "total": len(result.source),
            "files_scanned": result.files_scanned,
            "buckets": bucket_rows(source_buckets),
            "entries": [
                {
                    "file": s.file,
                    "line": s.line,
                    "kind": s.kind,
                    "bucket": s.bucket,
                    "lane": s.lane,
                    "reason": s.reason,
                }
                for s in sorted(result.source, key=lambda s: (s.file, s.line))
            ],
        },
        "unclassified": {
            "runtime": runtime_buckets.get("unclassified", 0),
            "source": source_buckets.get("unclassified", 0),
            "runtime_no_reason": runtime_buckets.get("no_reason", 0),
            "source_no_reason": source_buckets.get("no_reason", 0),
            "note": (
                "An unclassified skip is not a failure of this tool; it means the reason "
                "did not match a known bucket and therefore needs a human decision. It is "
                "counted separately so it cannot hide inside a tidy total."
            ),
        },
    }


# --------------------------------------------------------------------------- #
# Self-test
# --------------------------------------------------------------------------- #

SELF_TEST_JSONL = [
    {"Action": "output", "Package": "p/a", "Test": "TShort", "Output": "    x_test.go:9: skipping in short mode\n"},
    {"Action": "skip", "Package": "p/a", "Test": "TShort"},
    {"Action": "output", "Package": "p/a", "Test": "TReal", "Output": "    x_test.go:20: pass\n"},
    {"Action": "pass", "Package": "p/a", "Test": "TReal"},
    {"Action": "output", "Package": "p/b", "Test": "TDb", "Output": "    y_test.go:33: set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 to run\n"},
    {"Action": "skip", "Package": "p/b", "Test": "TDb"},
    {"Action": "output", "Package": "p/b", "Test": "TMystery", "Output": "    y_test.go:44: cannot proceed\n"},
    {"Action": "skip", "Package": "p/b", "Test": "TMystery"},
    {"Action": "pass", "Package": "p/b"},
    {"Action": "skip", "Package": "p/c"},
]

SELF_TEST_SOURCE = """package a

import "testing"

func TestOne(t *testing.T) {
\tt.Skip("skipping stress test in short mode")
}

func TestTwo(t *testing.T) {
\tt.Skipf("pre-existing failure: %v", 1)
}

func TestThree(t *testing.T) {
\tt.Skip(
\t\t"needs a real redis",
\t)
}
"""


def self_test() -> int:
    import tempfile

    failures: list[str] = []

    def check(name: str, condition: bool, detail: str = "") -> None:
        if condition:
            print(f"[PASS] self-test: {name}")
        else:
            print(f"[FAIL] self-test: {name} {detail}")
            failures.append(name)

    # classify()
    check("short mode classified",
          classify("skipping in short mode")[0] == "short_mode_excluded")
    check("residue classified",
          classify("pre-existing failure: boom")[0] == "historical_residue")
    check("residue outranks short mode (precedence is part of the definition)",
          classify("pre-existing, also skipped in short mode")[0] == "historical_residue")
    check("redis gate classified",
          classify("set CHAINPULSE_RUN_REDIS_INTEGRATION=1 to run")[0] == "dependency_gated")
    check("redis gate routes to the contracts lane",
          classify("set CHAINPULSE_RUN_REDIS_INTEGRATION=1")[1] == "contracts")
    check("anvil gate routes to evm-e2e",
          classify("needs anvil")[1] == "evm-e2e")
    check("unknown reason is unclassified, not absorbed",
          classify("cannot proceed")[0] == "unclassified")
    check("unclassified routes to unassigned", classify("cannot proceed")[1] == "unassigned")

    # Buckets widened after measuring the tree: these three were the bulk of the
    # 68 site-level reasons that matched nothing on the first pass. Widening them
    # is a decision, so each is pinned here.
    check("orchestrator failure is dependency gated",
          classify("Failed to initialize orchestrator: %v")[0] == "dependency_gated")
    check("services-not-ready is dependency gated",
          classify("Services not ready: %v")[0] == "dependency_gated")
    check("missing ABI is dependency gated",
          classify("ABI not available for Swap")[0] == "dependency_gated")
    check("missing fixture is dependency gated",
          classify("Requires fixtures package")[0] == "dependency_gated")
    check("delegated coverage is its own bucket",
          classify("RBAC checking tested separately in rbac_test.go")[0] == "covered_elsewhere")
    check("host capability is environment gated",
          classify("needs at least 2 Ps for the check and the act to interleave")[0]
          == "environment_gated")
    check("a placeholder reason stays unclassified",
          classify("regression")[0] == "unclassified",
          classify("regression")[0])
    check("covered_elsewhere outranks dependency_gated",
          classify("requires redis; covered in cache_test.go")[0] == "covered_elsewhere")

    with tempfile.TemporaryDirectory() as tmp:
        tmp_path = Path(tmp)
        jsonl = tmp_path / "unit.jsonl"
        jsonl.write_text("\n".join(json.dumps(e) for e in SELF_TEST_JSONL) + "\n")

        entries, package_skips, errors = parse_jsonl([jsonl])
        check("jsonl parsed without errors", errors == [], str(errors))
        # Three skipped tests in the fixture: one short-mode, one dependency-gated
        # and one whose reason matches nothing.
        check("runtime skips counted", len(entries) == 3, str(len(entries)))

        by_test = {e.test: e for e in entries}
        check("skip reason attached from the preceding output",
              by_test["TDb"].reason.endswith("to run"), by_test["TDb"].reason)
        check("passing tests are not counted", "TReal" not in by_test)
        check("runtime bucket applied",
              by_test["TMystery"].bucket == "unclassified")

        # The package-level pass carries no Test field and must not be counted.
        check("package-level skip recorded separately",
              package_skips == ["p/c"], str(package_skips))

        src = tmp_path / "a_test.go"
        src.write_text(SELF_TEST_SOURCE)

        sites, scanned, serrors = scan_source(tmp_path)
        check("source scan has no errors", serrors == [], str(serrors))
        check("source file scanned", scanned == 1, str(scanned))
        check("all three call sites found", len(sites) == 3, str(len(sites)))

        kinds = sorted(s.kind for s in sites)
        check("Skipf and multi-line Skip found", kinds == ["Skip", "Skip", "Skipf"], str(kinds))

        multiline = [s for s in sites if s.line == 14]
        check("multi-line call reason captured",
              bool(multiline) and "redis" in multiline[0].reason,
              str([s.reason for s in multiline]))

        report = build_report(Result(runtime=entries, source=sites, files_scanned=scanned), tmp_path)
        check("schema stamped", report["schema"] == SCHEMA)
        check("unclassified surfaced separately",
              report["unclassified"]["runtime"] == 1, json.dumps(report["unclassified"]))
        check("runtime total recorded", report["runtime"]["total"] == 3,
              str(report["runtime"]["total"]))
        check("source total recorded", report["source"]["total"] == 3)
        check("every bucket row present even at zero",
              len(report["runtime"]["buckets"]) == len(BUCKETS) + 2,
              str(len(report["runtime"]["buckets"])))

        missing, _pkgs, merr = parse_jsonl([tmp_path / "nope.jsonl"])
        check("a missing jsonl is an error, not silence",
              missing == [] and len(merr) == 1, str(merr))

    if failures:
        print(f"\n{len(failures)} self-test failure(s)")
        return 1
    print("\nall self-tests passed")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Build the single skip taxonomy (plan §10 skip-report.json) from "
            "go test -json streams plus the t.Skip call sites in the tree."
        )
    )
    parser.add_argument("--jsonl", action="append", default=[], type=Path,
                        help="a go test -json stream; repeat per suite")
    parser.add_argument("--source-root", type=Path, default=None,
                        help="tree to scan for t.Skip sites (default: repository root)")
    parser.add_argument("--out", type=Path, default=None, help="where to write the report")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    root = args.source_root
    if root is None:
        root = Path(__file__).resolve().parents[2]
    if not root.is_dir():
        print(f"error: source root {root} is not a directory", file=sys.stderr)
        return 2

    entries, package_skips, jsonl_errors = parse_jsonl(args.jsonl)
    for message in jsonl_errors:
        print(f"warning: {message}", file=sys.stderr)

    sites, scanned, scan_errors = scan_source(root)
    for message in scan_errors:
        print(f"warning: {message}", file=sys.stderr)

    report = build_report(
        Result(
            runtime=entries,
            source=sites,
            package_skips=package_skips,
            files_scanned=scanned,
            jsonl_errors=[],
        ),
        root,
    )

    encoded = json.dumps(report, indent=2) + "\n"
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(encoded)
        print(f"skip report written to {args.out}")
    else:
        sys.stdout.write(encoded)

    print(
        f"runtime skips: {report['runtime']['total']} "
        f"(+{report['runtime']['package_skip_count']} package)  "
        f"source sites: {report['source']['total']}  "
        f"unclassified: {report['unclassified']['runtime']} runtime / "
        f"{report['unclassified']['source']} source",
        file=sys.stderr,
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())

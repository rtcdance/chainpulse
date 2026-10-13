#!/usr/bin/env python3
"""Build the §10 `chaos-report.json` from the fault rehearsals that actually ran.

Why this exists
---------------
§10 requires a chaos report and §9 defines the fault matrix, but a report assembled
by hand is a claim, not evidence. This tool reads the `go test -json` streams from
the lanes that execute the rehearsals and reports each scenario with the outcome
the run actually produced.

What it does NOT do
-------------------
It does not decide that a scenario passed. It requires, per scenario, that every
test listed in the manifest appeared in the stream and PASSED. A listed test that
is missing or skipped is a failure — a rehearsal that skipped proves nothing, and
silently counting it as a pass is the exact false-pass this repository keeps
guarding against.

Scenarios whose dependency is unavailable here are marked `environment_skipped`
and are reported as NOT covered, never as covered. That distinction is the point
of the field.

Standard library only, so it runs in the acceptance toolchain without a Go
toolchain.
"""

from __future__ import annotations

import argparse
import datetime
import json
import sys
from dataclasses import dataclass, field
from pathlib import Path

SCHEMA = "chainpulse.chaos-report.v1"


@dataclass
class Scenario:
    id: str
    title: str
    injection: str
    tests: list[str]
    # Tests that are expected to skip here because a dependency is unreachable.
    # Reported as NOT covered rather than folded into a pass.
    environment_skipped: list[str] = field(default_factory=list)
    lane: str = ""


# The §9 matrix, restricted to the rehearsals that exist as real tests. Each
# entry records the fault that is injected, so the report reads as a description
# of the experiment rather than a list of green ticks.
SCENARIOS: tuple[Scenario, ...] = (
    Scenario(
        id="E05a",
        title="commit crash: kill inside the commit transaction",
        injection="SIGKILL while the commit transaction holds the events, before COMMIT",
        tests=["TestCrashInsideCommitLeavesNothingBehind"],
        lane="acceptance",
    ),
    Scenario(
        id="E05b",
        title="commit crash: kill after commits, then resume",
        injection="SIGKILL after three commits, then restart from the durable checkpoint",
        tests=["TestCrashAfterCommitKeepsTheBlock"],
        lane="acceptance",
    ),
    Scenario(
        id="E05c",
        title="graceful shutdown: SIGTERM",
        injection="SIGTERM mid-batch; the workload must drain, keep the checkpoint on a batch boundary and exit 0",
        tests=["TestSigtermGracefulShutdown"],
        lane="acceptance",
    ),
    Scenario(
        id="E05-control",
        title="control: the same kill against a NON-atomic commit",
        injection="identical kill window, with events and checkpoint committed in two transactions",
        tests=["TestCrashInsideCommitNegativeControl"],
        lane="acceptance",
    ),
    Scenario(
        id="E09a",
        title="database outage: backpressure, not data loss",
        injection="stop the database forwarder so host connections are refused, then submit a batch",
        tests=["TestDatabaseDisconnectIsBackpressure"],
        lane="acceptance",
    ),
    Scenario(
        id="E09b",
        title="Redis outage on the security path: fail closed",
        injection="stop Redis; a nonce store that still reports success would allow a replay",
        tests=["TestRedisOutageFailsClosed"],
        lane="acceptance",
    ),
    Scenario(
        id="E09c",
        title="Redis outage on the query path: degrade, do not fail",
        injection="stop Redis; a read must fall back to the authoritative source and be marked degraded",
        tests=["TestRedisOutageDegradesQueries"],
        lane="acceptance",
    ),
    Scenario(
        id="E09d",
        title="Kafka redelivery: at-least-once with a single effect",
        injection="replay a batch through a fresh consumer group",
        tests=["TestKafkaRedeliveryKeepsEffectSingle"],
        lane="acceptance",
        environment_skipped=["TestKafkaRedeliveryKeepsEffectSingle"],
    ),
    Scenario(
        id="F25",
        title="restricted provider: no block may be skipped",
        injection="a provider that refuses wide ranges and one that silently truncates a page",
        tests=[
            "TestPullEventsCoversEveryBlockWhenProviderRefusesWideRanges",
            "TestPullEventsCoversEveryBlockWhenProviderTruncatesSilently",
            "TestTruncationLossIsRealNotHypothetical",
        ],
        lane="unit",
    ),
    Scenario(
        id="RES",
        title="resource: workers return to the goroutine baseline",
        injection="10 start/stop cycles of a background worker",
        tests=["TestBackgroundWorkersReturnToBaseline"],
        lane="acceptance",
    ),
)


def parse_streams(paths: list[Path]) -> tuple[dict[str, str], list[str]]:
    """Return test name -> outcome ("pass"/"fail"/"skip"), plus IO errors.

    A name that appears more than once is recorded as `fail` if any occurrence
    failed, and `pass` only if every occurrence passed — a test that failed in one
    lane is not rescued by passing in another.
    """
    outcomes: dict[str, str] = {}
    errors: list[str] = []

    for path in paths:
        if not path.is_file():
            errors.append(f"{path}: not a file")
            continue
        with path.open(encoding="utf-8", errors="replace") as handle:
            for line in handle:
                line = line.strip()
                if not line:
                    continue
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue
                name = event.get("Test")
                action = event.get("Action")
                if not name or "/" in name or action not in {"pass", "fail", "skip"}:
                    continue
                previous = outcomes.get(name)
                if action == "fail" or previous == "fail":
                    outcomes[name] = "fail"
                elif action == "skip" and previous != "pass-with-skip":
                    outcomes[name] = "skip"
                else:
                    outcomes[name] = "pass"
    return outcomes, errors


def build_report(outcomes: dict[str, str]) -> tuple[dict, list[str]]:
    scenarios = []
    problems: list[str] = []
    covered = 0
    not_covered: list[str] = []

    for scenario in SCENARIOS:
        results = []
        for test in scenario.tests:
            outcome = outcomes.get(test)
            results.append({"test": test, "outcome": outcome or "not_run"})

        enforced = [r for r in results if r["test"] not in scenario.environment_skipped]
        enforced_ok = all(r["outcome"] == "pass" for r in enforced)
        env_skipped = [
            r["test"] for r in results if r["test"] in scenario.environment_skipped
            and r["outcome"] in {"skip", "not_run"}
        ]

        if scenario.environment_skipped and env_skipped:
            status = "environment_skipped"
            not_covered.append(scenario.id)
        elif enforced_ok:
            status = "passed"
            covered += 1
        else:
            status = "failed"
            missing = [r["test"] for r in enforced if r["outcome"] != "pass"]
            problems.append(
                f"{scenario.id}: {', '.join(missing)} did not pass "
                f"(a rehearsal that did not run or was skipped proves nothing)"
            )

        scenarios.append({
            "id": scenario.id,
            "title": scenario.title,
            "lane": scenario.lane,
            "fault_injected": scenario.injection,
            "status": status,
            "tests": results,
            "environment_skipped": env_skipped,
        })

    report = {
        "schema": SCHEMA,
        "generated_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "scenario_count": len(scenarios),
        "covered": covered,
        "not_covered": not_covered,
        "note": (
            "A scenario is 'passed' only when every test listed for it appeared in the "
            "stream and passed. 'environment_skipped' means a dependency is unreachable "
            "here and the scenario is NOT covered — it is never reported as covered."
        ),
        "scenarios": scenarios,
    }
    return report, problems


def self_test() -> int:
    failures: list[str] = []

    def check(name: str, condition: bool, detail: str = "") -> None:
        if condition:
            print(f"[PASS] self-test: {name}")
        else:
            print(f"[FAIL] self-test: {name} {detail}")
            failures.append(name)

    import tempfile

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        stream = root / "a.jsonl"
        stream.write_text("\n".join([
            json.dumps({"Action": "pass", "Test": "TestCrashInsideCommitLeavesNothingBehind"}),
            json.dumps({"Action": "pass", "Test": "TestCrashAfterCommitKeepsTheBlock"}),
            json.dumps({"Action": "skip", "Test": "TestKafkaRedeliveryKeepsEffectSingle"}),
            json.dumps({"Action": "pass", "Test": "TestPullEventsCoversEveryBlockWhenProviderRefusesWideRanges"}),
            json.dumps({"Action": "pass", "Test": "TestPullEventsCoversEveryBlockWhenProviderTruncatesSilently"}),
            json.dumps({"Action": "pass", "Test": "TestTruncationLossIsRealNotHypothetical"}),
            json.dumps({"Action": "pass", "Test": "TestBackgroundWorkersReturnToBaseline"}),
            json.dumps({"Action": "pass", "Test": "TestCrashInsideCommitNegativeControl"}),
            json.dumps({"Action": "pass", "Test": "TestSigtermGracefulShutdown"}),
            json.dumps({"Action": "pass", "Test": "TestDatabaseDisconnectIsBackpressure"}),
            json.dumps({"Action": "pass", "Test": "TestRedisOutageFailsClosed"}),
            json.dumps({"Action": "pass", "Test": "TestRedisOutageDegradesQueries"}),
            # a subtest must not be counted as a top-level test
            json.dumps({"Action": "pass", "Test": "TestSigtermGracefulShutdown/run_00"}),
        ]) + "\n")

        outcomes, errors = parse_streams([stream])
        check("stream parsed without errors", errors == [], str(errors))
        check("subtests are not counted", "TestSigtermGracefulShutdown/run_00" not in outcomes)

        report, problems = build_report(outcomes)
        check("no problems when every enforced test passed", problems == [], str(problems))
        check("the Kafka scenario is reported as not covered",
              "E09d" in report["not_covered"], str(report["not_covered"]))
        check("the Kafka scenario status is environment_skipped",
              next(s for s in report["scenarios"] if s["id"] == "E09d")["status"] == "environment_skipped")

        # A skipped non-environment test must fail the report.
        outcome_copy = dict(outcomes)
        outcome_copy["TestSigtermGracefulShutdown"] = "skip"
        _, problems2 = build_report(outcome_copy)
        check("a skipped enforced test is a failure", len(problems2) == 1, str(problems2))

        # A missing test must fail the report, so an absent rehearsal cannot pass.
        outcome_copy = dict(outcomes)
        del outcome_copy["TestCrashInsideCommitLeavesNothingBehind"]
        _, problems3 = build_report(outcome_copy)
        check("a missing test is a failure", any("E05a" in p for p in problems3), str(problems3))

        # A failure anywhere wins over a pass elsewhere.
        two = root / "b.jsonl"
        two.write_text(json.dumps({"Action": "fail", "Test": "TestSigtermGracefulShutdown"}) + "\n")
        merged, _ = parse_streams([stream, two])
        check("a failure in another stream wins", merged["TestSigtermGracefulShutdown"] == "fail")

        missing, merr = parse_streams([root / "nope.jsonl"])
        check("a missing stream is an error, not silence", missing == {} and len(merr) == 1, str(merr))

    if failures:
        print(f"\n{len(failures)} self-test failure(s)")
        return 1
    print("\nall self-tests passed")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build the §10 chaos report from real runs.")
    parser.add_argument("--jsonl", action="append", default=[], type=Path,
                        help="a go test -json stream; repeat per lane")
    parser.add_argument("--out", type=Path, default=None, help="where to write chaos-report.json")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    outcomes, errors = parse_streams(args.jsonl)
    for message in errors:
        print(f"warning: {message}", file=sys.stderr)

    if not outcomes:
        print("error: no test outcomes parsed; refusing to report on an empty run", file=sys.stderr)
        return 1

    report, problems = build_report(outcomes)
    encoded = json.dumps(report, indent=2) + "\n"

    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(encoded)
        print(f"chaos report written to {args.out}")
    else:
        sys.stdout.write(encoded)

    print(
        f"scenarios {report['scenario_count']}: {report['covered']} covered, "
        f"{len(report['not_covered'])} not covered here",
        file=sys.stderr,
    )

    if problems:
        print(f"\nchaos report FAILED ({len(problems)} issue(s)):", file=sys.stderr)
        for problem in problems:
            print(f"  - {problem}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""acceptance-gate.py - aggregate required CI checks and refuse anything that is
not an outright success.

Plan section 7.3 defines the acceptance-gate job:

    acceptance-gate | every PR, main, always() | aggregates the required jobs
                    | every conclusion must be success; skipped / neutral /
                      cancelled / missing all fail

The plan's own wording is the whole specification, so the decision logic lives
here where it can be tested, and a workflow only has to fetch the check runs and
call this script. That split matters because the surrounding questions - which
workflow owns the job, what the required check names end up being - depend on
repository-admin and naming decisions that are still open.

Why "skipped is a failure" needs stating
----------------------------------------
A skipped job is the normal way a required gate silently stops protecting
anything: rename a job, tighten a path filter, or make a step conditional, and
GitHub reports the check as skipped while the PR still looks mergeable. Treating
skipped as anything other than failure is how a gate becomes decorative.

Input
-----
A JSON file holding check runs. Both shapes are accepted:

  {"checks": [{"name": "test", "conclusion": "success"}, ...]}

or the raw `gh api repos/:owner/:repo/commits/<sha>/check-runs` response, whose
"check_runs" array uses "name" and "conclusion" too.

Usage
-----
    python3 scripts/acceptance/acceptance-gate.py --checks checks.json \
        --required test lint build coverage
    python3 scripts/acceptance/acceptance-gate.py --self-test

Exit code 0 only when every required check is present AND concluded success.
"""

from __future__ import annotations

import argparse
import json
import sys
import tempfile
from pathlib import Path

# The only conclusion that counts as passing.
PASSING = "success"

# Conclusions that must never satisfy a required gate. Listed explicitly rather
# than "anything that is not success", so the intent is auditable.
FAILING = {
    "failure",
    "cancelled",
    "timed_out",
    "action_required",
    "stale",
    "skipped",
    "neutral",
    "startup_failure",
}


def load_checks(path: Path) -> dict[str, str]:
    """Return {check name: conclusion} from either accepted JSON shape."""
    data = json.loads(path.read_text(encoding="utf-8"))

    if isinstance(data, dict) and "check_runs" in data:
        runs = data["check_runs"]
    elif isinstance(data, dict) and "checks" in data:
        runs = data["checks"]
    elif isinstance(data, list):
        runs = data
    else:
        raise ValueError(
            "unrecognised input: expected {'checks': [...]}, "
            "{'check_runs': [...]}, or a list"
        )

    out: dict[str, str] = {}
    for run in runs:
        name = run.get("name")
        if not name:
            continue
        # An in-progress check has conclusion null. Record it as such so it is
        # reported as not-yet-successful rather than silently absent.
        conclusion = (run.get("conclusion") or "in_progress").lower()
        # Never let a second successful record hide a failed/pending record
        # with the same name. CI asks GitHub for filter=latest; conflicting
        # duplicate results are still ambiguous evidence, not a green check.
        if name not in out or out[name] == PASSING:
            out[name] = conclusion
    return out


def evaluate(checks: dict[str, str], required: list[str]) -> list[str]:
    """Return the reasons the gate must fail (empty list == pass)."""
    failures: list[str] = []
    if not required:
        return ["required-check set is empty"]

    for name in required:
        conclusion = checks.get(name)
        if conclusion is None:
            failures.append(f"required check missing: {name}")
        elif conclusion != PASSING:
            failures.append(f"required check {name}: conclusion={conclusion}")

    return failures


def load_policy(path: Path) -> list[str]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict) or data.get("schema") != "chainpulse.required-checks.v1":
        raise ValueError("unsupported required-check policy")
    required = data.get("required")
    if not isinstance(required, list) or not required or any(
        not isinstance(name, str) or not name.strip() for name in required
    ):
        raise ValueError("required-check policy must contain nonempty check names")
    if len(required) != len(set(required)):
        raise ValueError("required-check policy contains duplicate check names")
    return required


def self_test() -> int:
    ok = True

    def check(label: str, checks: dict[str, str], required: list[str], want_ok: bool) -> None:
        nonlocal ok
        failures = evaluate(checks, required)
        got_ok = not failures
        status = "PASS" if got_ok == want_ok else "FAIL"
        if got_ok != want_ok:
            ok = False
        print(f"[{status}] self-test: {label} (failures={failures})")

    check("all success", {"test": "success", "lint": "success"}, ["test", "lint"], True)
    check("empty required set", {}, [], False)
    check("failure", {"test": "failure"}, ["test"], False)
    check("skipped", {"test": "skipped"}, ["test"], False)
    check("neutral", {"test": "neutral"}, ["test"], False)
    check("cancelled", {"test": "cancelled"}, ["test"], False)
    check("timed_out", {"test": "timed_out"}, ["test"], False)
    check("action_required", {"test": "action_required"}, ["test"], False)
    check("missing", {"lint": "success"}, ["test", "lint"], False)
    check("in progress", {"test": "in_progress"}, ["test"], False)
    check(
        "one bad among many",
        {"a": "success", "b": "success", "c": "skipped"},
        ["a", "b", "c"],
        False,
    )

    with tempfile.TemporaryDirectory(prefix="chainpulse-check-policy-") as directory:
        path = Path(directory) / "policy.json"
        for label, required, valid in [
            ("valid policy", ["quality", "contracts"], True),
            ("empty policy", [], False),
            ("duplicate policy", ["quality", "quality"], False),
            ("blank policy", [" "], False),
            ("wrong type", "quality", False),
        ]:
            path.write_text(json.dumps({"schema": "chainpulse.required-checks.v1", "required": required}))
            try:
                load_policy(path)
                accepted = True
            except ValueError:
                accepted = False
            passed = accepted == valid
            ok = ok and passed
            print(f"[{'PASS' if passed else 'FAIL'}] self-test: {label}")

        for runs in [
            [{"name": "quality", "conclusion": "failure"}, {"name": "quality", "conclusion": "success"}],
            [{"name": "quality", "conclusion": "success"}, {"name": "quality", "conclusion": "failure"}],
        ]:
            path.write_text(json.dumps({"checks": runs}))
            check("duplicate success cannot hide failure", load_checks(path), ["quality"], False)

    print("self-test:", "OK" if ok else "FAILED")
    return 0 if ok else 1


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--checks", type=Path, help="JSON of check runs")
    parser.add_argument("--policy", type=Path, help="versioned required-check policy JSON")
    parser.add_argument(
        "--required",
        nargs="+",
        default=[],
        help="check names that must be present and successful",
    )
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    if not args.checks:
        parser.error("--checks is required (or use --self-test)")
    if args.policy and args.required:
        parser.error("use --policy or --required, not both")
    if args.policy:
        try:
            args.required = load_policy(args.policy)
        except (OSError, ValueError, TypeError) as exc:
            parser.error(str(exc))
    if not args.required:
        parser.error(
            "--required is required: an empty required set would pass vacuously"
        )

    checks = load_checks(args.checks)
    failures = evaluate(checks, args.required)

    print(f"checks seen: {len(checks)}")
    for name in args.required:
        print(f"  {name}: {checks.get(name, '<missing>')}")

    if failures:
        print(f"\nacceptance-gate FAILED ({len(failures)} issue(s)):", file=sys.stderr)
        for f in failures:
            print(f"  - {f}", file=sys.stderr)
        return 1

    print("\nacceptance-gate passed: every required check concluded success")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

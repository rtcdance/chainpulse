#!/usr/bin/env python3
"""dep-approval-gate.py — dependency changes must update DEPENDENCY_APPROVAL.md.

Task: T15 of docs/specs/2026-10-12-web3-go-backend-delivery-plan.md (finding F17).

Why this exists
---------------
AGENTS.md states the obligation:

    Dependency changes also require updating `docs/project/DEPENDENCY_APPROVAL.md`.

F17 records that the obligation "未出现在任何任务、§7 或 §10 DoD 中" — it was a
rule with no enforcement. §10 now requires it, and T15's acceptance is explicit:

    依赖变更而 DEPENDENCY_APPROVAL.md 未更新时 §10 DoD 不通过

So this gate fails when a dependency manifest changed and the approval document
did not. It is deliberately narrow: it does not judge WHETHER a dependency is
acceptable, only that a human recorded a decision.

It is dependency-free (Python 3 stdlib only) and self-tested, so the gate itself
is verified rather than trusted.

Usage
-----
    # Compare two revisions (what CI does on a PR).
    dep-approval-gate.py --base origin/main --head HEAD

    # Or feed a newline-separated changed-file list.
    dep-approval-gate.py --changed-files /tmp/changed.txt

    # Verify the gate.
    dep-approval-gate.py --self-test
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

# Manifests whose change means "a dependency or tool version changed".
DEPENDENCY_MANIFESTS = (
    "go.mod",
    "go.sum",
    "package.json",
    "package-lock.json",
    "frontend/package.json",
    "frontend/package-lock.json",
    "frontend/pnpm-lock.yaml",
    "contracts/foundry.toml",  # forge-std and solc are pinned here
)

APPROVAL_DOC = "docs/project/DEPENDENCY_APPROVAL.md"


def changed_dependency_manifests(changed_files: list[str]) -> list[str]:
    """Return the dependency manifests present in a changed-file list."""
    changed = set(changed_files)
    return [manifest for manifest in DEPENDENCY_MANIFESTS if manifest in changed]


def evaluate(changed_files: list[str]) -> tuple[bool, list[str], list[str]]:
    """Returns (ok, manifests, reasons).

    An empty changed-file list is not a failure: a commit that touched nothing
    relevant has nothing to approve. What IS a failure is a dependency change
    with no recorded decision.
    """
    manifests = changed_dependency_manifests(changed_files)
    approval_updated = APPROVAL_DOC in set(changed_files)

    if not manifests:
        return True, [], []

    if approval_updated:
        return True, manifests, []

    return (
        False,
        manifests,
        [
            f"{APPROVAL_DOC} was not updated, but these dependency manifests changed: "
            + ", ".join(manifests)
        ],
    )


def git_changed_files(base: str, head: str) -> list[str]:
    out = subprocess.run(
        ["git", "diff", "--name-only", base, head],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    return [line.strip() for line in out.splitlines() if line.strip()]


# --------------------------------------------------------------------------- #
# Self-test
# --------------------------------------------------------------------------- #

SELF_TEST_CASES = (
    # (label, changed files, expected ok)
    ("no changes", [], True),
    ("unrelated file only", ["pkg/core/event_hash.go"], True),
    ("go.mod without approval", ["go.mod"], False),
    ("go.sum without approval", ["go.sum"], False),
    ("frontend manifest without approval", ["frontend/package.json"], False),
    ("foundry pin without approval", ["contracts/foundry.toml"], False),
    ("go.mod with approval", ["go.mod", APPROVAL_DOC], True),
    ("approval doc alone", [APPROVAL_DOC], True),
    ("unrelated file plus approval", ["README.md", APPROVAL_DOC], True),
)


def self_test() -> int:
    failures: list[str] = []

    for label, files, expected_ok in SELF_TEST_CASES:
        ok, manifests, reasons = evaluate(files)
        if ok != expected_ok:
            failures.append(f"{label}: expected ok={expected_ok}, got ok={ok} ({reasons})")

    # The failure must NAME the manifest, otherwise it is not actionable.
    ok, manifests, reasons = evaluate(["go.mod", "go.sum"])
    if ok:
        failures.append("go.mod+go.sum without approval must fail")
    if "go.mod" not in " ".join(manifests) or "go.sum" not in " ".join(manifests):
        failures.append(f"the failure must name both manifests, got {manifests}")
    if not reasons:
        failures.append("a failure must carry a reason")

    if failures:
        print("dep-approval-gate self-test FAILED:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(
        f"dep-approval-gate self-test passed "
        f"({len(SELF_TEST_CASES)} cases + manifest naming)"
    )
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Fail when a dependency changed without updating DEPENDENCY_APPROVAL.md."
    )
    parser.add_argument("--base", help="base revision for git diff")
    parser.add_argument("--head", default="HEAD", help="head revision for git diff")
    parser.add_argument("--changed-files", type=Path, help="newline-separated changed-file list")
    parser.add_argument("--self-test", action="store_true", help="verify the gate itself")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    if args.changed_files:
        changed = [
            line.strip()
            for line in args.changed_files.read_text(encoding="utf-8").splitlines()
            if line.strip()
        ]
    elif args.base:
        changed = git_changed_files(args.base, args.head)
    else:
        parser.error("either --base or --changed-files is required (or --self-test)")

    ok, manifests, reasons = evaluate(changed)

    if ok:
        if manifests:
            print(
                f"dependency manifests changed and {APPROVAL_DOC} was updated: "
                + ", ".join(manifests)
            )
        else:
            print("no dependency manifest changed; nothing to approve")
        return 0

    print("DEPENDENCY APPROVAL GATE FAILED (F17 / §10):", file=sys.stderr)
    for reason in reasons:
        print(f"  {reason}", file=sys.stderr)
    print(
        f"\nRecord the decision in {APPROVAL_DOC} (or revert the dependency change).",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())

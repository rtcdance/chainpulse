#!/usr/bin/env python3
"""reverted-fixes.py - find fix commits whose changes are no longer in HEAD.

Why this exists
---------------
A defect found in this repository was not "never implemented" but "implemented,
then silently undone": cf345a2 added Kafka at-least-once delivery, 8f3c475 put
the auto-commit timer back the next day, and nothing went red for 146 days
(see docs/project/2026-10-14-silent-revert-kafka.md).

Finding that by hand took a lucky grep of the CHANGELOG. This script looks for
the whole class: for each commit that claims to fix something, measure how much
of what it added still exists in HEAD.

Method
------
For every commit matching the message filter (default: fix / security /
concurrency / resilience / correctness):

  1. collect the lines it ADDED (unified=0 diff, additions only)
  2. drop lines too trivial to be evidence (blank, brace-only, import lines,
     very short lines)
  3. build the set of lines present in HEAD for the files that commit touched
  4. report the fraction of added lines that are still present

A low survival ratio means the change is largely gone. That is a CANDIDATE, not
a verdict: legitimate refactors, renames and file moves also remove lines. The
point is to turn an unbounded search into a short list worth reading.

KNOWN LIMITATION - read this before trusting the output
-------------------------------------------------------
This measures survival at the WHOLE-COMMIT level, so it cannot see a surgical
revert inside a large commit. The case that motivated this script is exactly
that: 8f3c475 reverted Kafka at-least-once with TWO lines, inside a 292-line
commit about CI and ZeroMQ. Its survival ratio is about 99% and it never
appears here.

What this tool is good for: commits whose bulk disappeared (a wholesale revert
or a reverted branch). What it is not good for: a two-line correctness fix
removed by an unrelated large commit.

For that class the effective method is different and was the one that actually
found the Kafka revert: start from a CLAIM (CHANGELOG, README, a comment, a
commit message), then verify the claim against the code, then trace with
git log -S when they disagree. Claims are few and bounded; lines are not.

Usage
-----
    python3 scripts/audit/reverted-fixes.py                 # last 400 commits
    python3 scripts/audit/reverted-fixes.py --limit 800
    python3 scripts/audit/reverted-fixes.py --threshold 0.5 # only report <=50% survival
    python3 scripts/audit/reverted-fixes.py --json out.json
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

# Commit messages that claim a fix. Broad on purpose; the survival ratio does
# the filtering.
FIX_FILTER = r"fix|security|concurren|resilien|correctness|P0|P1|race|leak"

# Lines that are never evidence of a fix surviving.
TRIVIAL = re.compile(
    r"^\s*$"  # blank
    r"|^\s*[{}()\[\]]+,?\s*$"  # brace only
    r"|^\s*//"  # comment
    r"|^\s*$"  # blank again
)


def git(*args: str) -> str:
    return subprocess.run(
        ["git", *args], capture_output=True, text=True, check=False
    ).stdout


def head_lines(repo: Path, path: str) -> set[str]:
    """Set of stripped lines in `path` at HEAD. Empty if the file is gone."""
    full = repo / path
    if not full.is_file():
        return set()
    try:
        return {ln.strip() for ln in full.read_text(encoding="utf-8", errors="replace").splitlines()}
    except OSError:
        return set()


def added_lines(sha: str) -> tuple[list[str], list[str]]:
    """Return (added lines, files touched) for a commit."""
    diff = git("show", "--format=", "--unified=0", "--no-color", sha)
    files: list[str] = []
    added: list[str] = []
    for line in diff.splitlines():
        if line.startswith("+++ b/"):
            files.append(line[6:])
        elif line.startswith("+") and not line.startswith("+++"):
            added.append(line[1:])
    return added, files


def is_evidence(line: str) -> bool:
    stripped = line.strip()
    if TRIVIAL.match(line):
        return False
    if len(stripped) < 12:  # too short to be distinctive
        return False
    if stripped.startswith("import") or stripped.startswith('"'):
        return False
    return True


def touches_production_code(files: list[str]) -> bool:
    """True if the commit changed non-test Go under pkg/ or cmd/.

    Without this the report is dominated by documentation, prompt and CI churn:
    those files get rewritten constantly, so their survival ratio says nothing
    about whether a fix was reverted. The question here is about code.
    """
    for f in files:
        if not f.endswith(".go") or f.endswith("_test.go"):
            continue
        if f.startswith("pkg/") or f.startswith("cmd/"):
            return True
    return False


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--limit", type=int, default=400, help="how many commits to scan")
    parser.add_argument("--threshold", type=float, default=0.5,
                        help="report commits at or below this survival ratio")
    parser.add_argument("--json", type=Path)
    args = parser.parse_args(argv)

    repo = Path(git("rev-parse", "--show-toplevel").strip() or ".")
    head = git("rev-parse", "HEAD").strip()

    log = git(
        "log",
        f"-{args.limit}",
        "--format=%H%x09%ad%x09%s",
        "--date=short",
        "-i",
        f"--grep={FIX_FILTER}",
        "-E",
    )
    commits = [ln.split("\t", 2) for ln in log.splitlines() if ln.count("\t") >= 2]

    print(f"HEAD: {head[:12]}")
    print(f"fix-like commits scanned: {len(commits)} (of the last {args.limit})")
    print()

    cache: dict[str, set[str]] = {}
    rows = []

    for sha, date, subject in commits:
        added, files = added_lines(sha)
        if not touches_production_code(files):
            continue
        evidence = [ln for ln in added if is_evidence(ln)]
        if not evidence:
            continue

        present = set()
        for f in files:
            if f not in cache:
                cache[f] = head_lines(repo, f)
            present |= cache[f]

        survived = sum(1 for ln in evidence if ln.strip() in present)
        ratio = survived / len(evidence)
        if ratio <= args.threshold:
            rows.append(
                {
                    "sha": sha[:12],
                    "date": date,
                    "subject": subject,
                    "added_lines": len(evidence),
                    "survived": survived,
                    "survival": round(ratio, 3),
                }
            )

    rows.sort(key=lambda r: (r["survival"], r["date"]))
    print(f"commits at or below {args.threshold:.0%} survival: {len(rows)}")
    print()
    for r in rows:
        print(
            f"  {r['survival']:>5.0%}  {r['sha']}  {r['date']}  "
            f"{r['survived']}/{r['added_lines']:<4} {r['subject'][:70]}"
        )

    if args.json:
        args.json.write_text(json.dumps(rows, indent=2) + "\n", encoding="utf-8")
        print(f"\nwrote {args.json}")

    print()
    print("These are candidates, not verdicts: refactors and file moves also")
    print("remove lines. Read the survivors before concluding anything.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

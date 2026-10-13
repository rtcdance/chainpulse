#!/usr/bin/env python3
"""unwired-components.py - find components that are built and tested but never
constructed by production code.

Why this exists
---------------
Several findings in this repository have the same shape: a component is
implemented, unit-tested, and then never constructed outside its tests, so the
tests pass while the feature does nothing in the deployed binary. Examples found
by hand: pkg/observability's MetricLabelGuard and LagRecorder, the IndexerMetrics
/ IndexerHealth pair, pkg/plugins/cache's VersionedCache, and reorg's
OutboxDrainerLoop.

Hand-auditing that class one component at a time is slow and easy to give up on.
This script enumerates it.

What it measures
----------------
For every exported constructor-shaped function (name starts with New or Build)
defined in a non-test file under the given roots, count references in:

  prod  - non-test .go files, excluding the definition itself
  test  - _test.go files

A constructor with prod == 0 and test > 0 is "test-only": something constructs it
in a test, nothing constructs it in the binary.

Caveats (do not treat the output as a defect list)
--------------------------------------------------
- prod == 0 does NOT always mean dead code. A constructor may be a public API
  entry point for callers outside this repository, or reachable only through
  reflection/DI codegen. Read each hit before acting.
- A component can be wired without its constructor being called directly (e.g.
  built by a generic factory). This finds candidates, not verdicts.
- References are textual, so a name that appears in a comment counts.

Usage
-----
    python3 scripts/audit/unwired-components.py [--roots pkg cmd] [--json out.json]
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

DEF_RE = re.compile(r"^func\s+(?:\([^)]*\)\s*)?(New[A-Z]\w*|Build[A-Z]\w*)\s*\(")
BLOCK_COMMENT_RE = re.compile(r"/\*.*?\*/", re.S)

# Names that mark test doubles. These are supposed to have no production caller,
# so listing them buries the real hits.
TEST_DOUBLE_RE = re.compile(r"Mock|Fake|Stub|Dummy|Test")

# Directories that exist to serve tests.
TEST_SUPPORT_DIRS = ("pkg/testhelpers",)


def is_test_double(symbol: str) -> bool:
    return bool(TEST_DOUBLE_RE.search(symbol))


def is_test_support(path: str) -> bool:
    return any(f"/{d}/" in path or path.startswith(f"{d}/") for d in TEST_SUPPORT_DIRS)


def code_lines(text: str) -> list[tuple[int, str]]:
    """Yield (lineno, line) for lines that are not comments.

    A doc comment that merely NAMES a constructor ("// NewFoo creates ...") is
    not a call site. Counting it as one produced false negatives: the first
    version of this script missed NewLagRecorder and NewOutboxDrainerLoop
    precisely because their own doc comments referenced them.

    Full-line comments and block comments are dropped. A trailing comment on a
    code line is left in place, which can only ever over-count, never hide a
    test-only component.
    """
    text = BLOCK_COMMENT_RE.sub("", text)
    out: list[tuple[int, str]] = []
    for lineno, line in enumerate(text.splitlines(), start=1):
        if line.lstrip().startswith("//"):
            continue
        out.append((lineno, line))
    return out


def go_files(root: Path) -> list[Path]:
    return sorted(p for p in root.rglob("*.go") if p.is_file())


def collect(files: list[Path]) -> tuple[dict[str, str], dict[str, int], dict[str, int]]:
    """Return (definitions, prod_refs, test_refs).

    definitions maps symbol -> "path:line". prod_refs/test_refs count textual
    occurrences of the symbol in non-test and test files respectively.
    """
    definitions: dict[str, str] = {}
    prod_refs: dict[str, int] = {}
    test_refs: dict[str, int] = {}

    for path in files:
        text = path.read_text(encoding="utf-8", errors="replace")
        is_test = path.name.endswith("_test.go")

        for lineno, line in code_lines(text):
            if not is_test:
                m = DEF_RE.match(line)
                if m:
                    definitions.setdefault(m.group(1), f"{path}:{lineno}")

            for sym in re.findall(r"\b(?:New[A-Z]\w*|Build[A-Z]\w*)\b", line):
                if is_test:
                    test_refs[sym] = test_refs.get(sym, 0) + 1
                else:
                    prod_refs[sym] = prod_refs.get(sym, 0) + 1

    return definitions, prod_refs, test_refs


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--roots", nargs="+", default=["pkg", "cmd"])
    parser.add_argument("--json", type=Path, help="also write the full result as JSON")
    parser.add_argument(
        "--include-test-doubles",
        action="store_true",
        help="also list constructors named Mock/Fake/Stub/Dummy/Test",
    )
    parser.add_argument(
        "--include-test-helpers",
        action="store_true",
        help="also list constructors defined under pkg/testhelpers",
    )
    args = parser.parse_args(argv)

    repo = Path(__file__).resolve().parents[2]
    files: list[Path] = []
    for r in args.roots:
        root = repo / r
        if not root.is_dir():
            print(f"warning: {root} is not a directory", file=sys.stderr)
            continue
        files.extend(go_files(root))

    definitions, prod_refs, test_refs = collect(files)

    test_only: list[dict[str, object]] = []
    filtered_doubles = 0
    filtered_support = 0

    for sym, where in sorted(definitions.items()):
        # The definition line itself is one production occurrence; anything more
        # means production code references it.
        prod = prod_refs.get(sym, 0) - 1
        test = test_refs.get(sym, 0)
        if prod > 0 or test <= 0:
            continue

        if not args.include_test_doubles and is_test_double(sym):
            filtered_doubles += 1
            continue
        if not args.include_test_helpers and is_test_support(where):
            filtered_support += 1
            continue

        test_only.append(
            {"symbol": sym, "defined_at": where, "prod_refs": prod, "test_refs": test}
        )

    # Rank by how well tested the component is: a component with many tests and
    # no production caller is the most suspicious shape.
    test_only.sort(key=lambda r: (-int(r["test_refs"]), str(r["symbol"])))

    print(f"scanned {len(files)} Go files under {' '.join(args.roots)}")
    print(f"exported New*/Build* constructors: {len(definitions)}")
    print(
        f"test-only, after filtering {filtered_doubles} test double(s) and "
        f"{filtered_support} test-support file(s): {len(test_only)}"
    )
    print()
    for row in test_only:
        print(f"  {row['symbol']:<46} test_refs={row['test_refs']:<4} {row['defined_at']}")

    if args.json:
        args.json.write_text(
            json.dumps(
                {
                    "scanned_files": len(files),
                    "constructors": len(definitions),
                    "test_only": test_only,
                },
                indent=2,
            )
            + "\n",
            encoding="utf-8",
        )
        print(f"\nwrote {args.json}")

    # Exit 0: this is an inventory, not a gate. A non-zero exit would tempt
    # callers to wire things up just to make CI green.
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

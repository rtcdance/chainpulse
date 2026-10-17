#!/usr/bin/env python3
"""Check that docs/api/openapi.yaml still describes the API the code serves.

Why this exists
---------------
Plan §4.4 requires an OpenAPI/contract diff test: a documented contract that is
not checked against the implementation is worse than no contract, because it is
believed. This repository's spec had already drifted — it documents
`EventResponse` without the `network` property that the handler returns — and it
cannot express the plan's §4.1 rule that amounts and large integers travel as
JSON strings, because its `eventData` is an untyped `additionalProperties: true`.

Scope is deliberately narrow: it compares the property NAMES of specific schemas
against the JSON tags of their Go structs, and checks the few documented value
conventions that carry a hard rule. It is not an OpenAPI validator.

Two failure modes are treated as failures rather than passes:

* a schema where the parser found no properties at all, because an extraction bug
  would otherwise compare two empty sets and "pass";
* a struct the parser could not locate, for the same reason.

Standard library only, and no YAML dependency, so this can run in the acceptance
toolchain alongside the other gates on a machine with no Go toolchain.
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

# schema name -> (spec file, Go source file, Go struct name)
SCHEMAS: tuple[tuple[str, str, str], ...] = (
    (
        "EventResponse",
        "pkg/plugins/api/event_query_handler.go",
        "EventResponse",
    ),
)


@dataclass
class Drift:
    schema: str
    missing_in_spec: list[str] = field(default_factory=list)
    missing_in_code: list[str] = field(default_factory=list)
    problems: list[str] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return not self.missing_in_spec and not self.missing_in_code and not self.problems


def _indent(line: str) -> int:
    return len(line) - len(line.lstrip(" "))


def parse_schema_properties(spec_text: str, schema: str) -> tuple[list[str], list[str]]:
    """Return (property names, problems) for `components.schemas.<schema>`.

    The parse is indentation-based and bounded to the schema's own block, so a
    neighbouring schema cannot leak into the result.
    """
    problems: list[str] = []
    lines = spec_text.splitlines()

    schema_re = re.compile(r"^(\s+)" + re.escape(schema) + r":\s*$")
    start = None
    schema_indent = 0
    for index, line in enumerate(lines):
        match = schema_re.match(line)
        if match:
            start = index
            schema_indent = len(match.group(1))
            break

    if start is None:
        return [], [f"schema {schema!r} not found in the spec"]

    properties_indent = None
    names: list[str] = []

    for line in lines[start + 1:]:
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        indent = _indent(line)

        # A line at or below the schema's own indentation ends the block.
        if indent <= schema_indent:
            break

        if properties_indent is None:
            if line.strip() == "properties:":
                properties_indent = indent
            continue

        # Inside `properties:`, direct children are the property names.
        if indent == properties_indent + 2 and line.rstrip().endswith(":"):
            name = line.strip()[:-1].strip()
            if name and not name.startswith("-"):
                names.append(name)
        elif indent <= properties_indent:
            break

    if properties_indent is None:
        problems.append(f"schema {schema!r} has no properties block")
    elif not names:
        problems.append(
            f"schema {schema!r} parsed to zero properties — refusing to pass on an empty comparison"
        )

    return names, problems


def parse_struct_json_tags(go_text: str, struct_name: str) -> tuple[list[str], list[str]]:
    """Return (json tag names, problems) for a Go struct, skipping `-` and empty."""
    problems: list[str] = []

    marker = re.compile(r"^type\s+" + re.escape(struct_name) + r"\s+struct\s*\{")
    start = None
    for index, line in enumerate(go_text.splitlines()):
        if marker.match(line.strip()):
            start = index
            break

    if start is None:
        return [], [f"Go struct {struct_name!r} not found"]

    names: list[str] = []
    for line in go_text.splitlines()[start + 1:]:
        if line.startswith("}"):
            break
        tag = re.search(r'json:"([^"]*)"', line)
        if not tag:
            continue
        name = tag.group(1).split(",")[0]
        if name and name != "-":
            names.append(name)

    if not names:
        problems.append(
            f"Go struct {struct_name!r} parsed to zero json tags — refusing to pass on an empty comparison"
        )

    return names, problems


def check(repo_root: Path) -> tuple[list[Drift], list[str]]:
    spec_path = repo_root / "docs/api/openapi.yaml"
    if not spec_path.is_file():
        return [], [f"spec not found at {spec_path}"]

    spec_text = spec_path.read_text(encoding="utf-8")

    results: list[Drift] = []
    for schema, go_rel, struct_name in SCHEMAS:
        go_path = repo_root / go_rel
        if not go_path.is_file():
            results.append(Drift(schema=schema, problems=[f"Go source not found at {go_rel}"]))
            continue

        spec_names, spec_problems = parse_schema_properties(spec_text, schema)
        go_names, go_problems = parse_struct_json_tags(go_path.read_text(encoding="utf-8"), struct_name)

        drift = Drift(schema=schema, problems=spec_problems + go_problems)
        if not drift.problems:
            drift.missing_in_spec = sorted(set(go_names) - set(spec_names))
            drift.missing_in_code = sorted(set(spec_names) - set(go_names))
        results.append(drift)

    return results, []


def check_money_encoding(spec_text: str) -> list[str]:
    """The spec must state that decoded amounts travel as strings.

    §4.1 forbids a float64 asset value and requires large integers as JSON
    strings. A spec whose `eventData` is `additionalProperties: true` cannot say
    so, and a client that reads the spec has no way to know a uint256 arrives as
    a string rather than a number.
    """
    problems: list[str] = []

    if "decimal string" not in spec_text:
        problems.append(
            "the spec never states that decoded amounts / large integers are decimal "
            "strings; §4.1 requires that rule to be documented"
        )

    # The example must demonstrate the rule, not contradict it.
    value_example = re.search(r"^\s+value:\s*(\S+)\s*$", spec_text, re.MULTILINE)
    if value_example and not value_example.group(1).startswith('"'):
        problems.append(
            f"the eventData example shows value as {value_example.group(1)!r}; "
            "a bare number in the example teaches clients the wrong encoding"
        )

    return problems


# --------------------------------------------------------------------------- #
# Self-test
# --------------------------------------------------------------------------- #

FIXTURE_SPEC_OK = """components:
  schemas:
    EventResponse:
      type: object
      properties:
        eventId:
          type: string
        network:
          type: string
    Other:
      type: object
      properties:
        unrelated:
          type: string
"""

FIXTURE_GO_OK = """package api

type EventResponse struct {
\tEventID string `json:"eventId"`
\tNetwork string `json:"network"`
}
"""

FIXTURE_GO_DRIFT = """package api

type EventResponse struct {
\tEventID string `json:"eventId"`
\tNetwork string `json:"network"`
\tExtra   string `json:"extra"`
}
"""


def self_test() -> int:
    import tempfile

    failures: list[str] = []

    def check_case(name: str, condition: bool, detail: str = "") -> None:
        if condition:
            print(f"[PASS] self-test: {name}")
        else:
            print(f"[FAIL] self-test: {name} {detail}")
            failures.append(name)

    names, problems = parse_schema_properties(FIXTURE_SPEC_OK, "EventResponse")
    check_case("properties parsed", names == ["eventId", "network"], str(names))
    check_case("no problems for a well-formed schema", problems == [], str(problems))
    check_case(
        "a neighbouring schema does not leak in",
        "unrelated" not in names,
        str(names),
    )

    tags, tag_problems = parse_struct_json_tags(FIXTURE_GO_OK, "EventResponse")
    check_case("json tags parsed", tags == ["eventId", "network"], str(tags))
    check_case("no problems for a well-formed struct", tag_problems == [], str(tag_problems))

    drift_tags, _ = parse_struct_json_tags(FIXTURE_GO_DRIFT, "EventResponse")
    check_case(
        "drift is detectable in the code direction",
        sorted(set(drift_tags) - set(names)) == ["extra"],
    )

    _, missing_problems = parse_schema_properties(FIXTURE_SPEC_OK, "NoSuchSchema")
    check_case("a missing schema is reported", len(missing_problems) == 1, str(missing_problems))

    empty_spec = "components:\n  schemas:\n    EventResponse:\n      type: object\n"
    empty_names, empty_problems = parse_schema_properties(empty_spec, "EventResponse")
    check_case(
        "a schema with no properties block is a failure, not an empty pass",
        empty_names == [] and len(empty_problems) == 1,
        str(empty_problems),
    )

    empty_go = "package api\n\ntype EventResponse struct {\n}\n"
    _, empty_go_problems = parse_struct_json_tags(empty_go, "EventResponse")
    check_case(
        "a struct with no json tags is a failure, not an empty pass",
        len(empty_go_problems) == 1,
        str(empty_go_problems),
    )

    check_case(
        "a spec without the money rule is flagged",
        check_money_encoding("eventData: {}") != [],
    )
    check_case(
        "a bare-number example is flagged",
        check_money_encoding('decoded as decimal string\n            value: 500\n') != [],
    )
    check_case(
        "a quoted example with the rule passes",
        check_money_encoding('decoded as decimal string\n            value: "500"\n') == [],
    )

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        (root / "docs/api").mkdir(parents=True)
        (root / "pkg/plugins/api").mkdir(parents=True)
        (root / "docs/api/openapi.yaml").write_text(FIXTURE_SPEC_OK)
        (root / "pkg/plugins/api/event_query_handler.go").write_text(FIXTURE_GO_OK)

        results, errors = check(root)
        check_case("end-to-end check has no IO errors", errors == [], str(errors))
        check_case("end-to-end check passes on matching fixtures",
                   len(results) == 1 and results[0].ok,
                   str([(r.schema, r.missing_in_spec, r.missing_in_code, r.problems) for r in results]))

        (root / "pkg/plugins/api/event_query_handler.go").write_text(FIXTURE_GO_DRIFT)
        results, _ = check(root)
        check_case("end-to-end check reports drift",
                   len(results) == 1 and results[0].missing_in_spec == ["extra"],
                   str(results[0] if results else None))

    if failures:
        print(f"\n{len(failures)} self-test failure(s)")
        return 1
    print("\nall self-tests passed")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Check docs/api/openapi.yaml against the Go response structs."
    )
    parser.add_argument("--repo-root", type=Path, default=None,
                        help="repository root (default: inferred from this script)")
    parser.add_argument("--report", type=Path, default=None, help="write a JSON report here")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    root = args.repo_root
    if root is None:
        root = Path(__file__).resolve().parents[2]
    if not root.is_dir():
        print(f"error: repo root {root} is not a directory", file=sys.stderr)
        return 2

    results, errors = check(root)
    problems = list(errors)
    for drift in results:
        problems.extend(f"{drift.schema}: {p}" for p in drift.problems)
        for name in drift.missing_in_spec:
            problems.append(f"{drift.schema}: {name!r} is returned by the handler but is not in the spec")
        for name in drift.missing_in_code:
            problems.append(f"{drift.schema}: {name!r} is documented but no longer returned")

    spec_path = root / "docs/api/openapi.yaml"
    if spec_path.is_file():
        problems.extend(f"money encoding: {p}" for p in check_money_encoding(spec_path.read_text(encoding="utf-8")))

    label = "ok" if not problems else "DRIFT"
    for drift in results:
        print(f"  {drift.schema}: {'ok' if drift.ok else 'drift'}")

    if args.report:
        import json

        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(
            {
                "result": label,
                "schemas": [
                    {
                        "schema": d.schema,
                        "missing_in_spec": d.missing_in_spec,
                        "missing_in_code": d.missing_in_code,
                        "problems": d.problems,
                    }
                    for d in results
                ],
                "problems": problems,
            },
            indent=2,
        ) + "\n")

    if problems:
        print(f"\nopenapi drift check FAILED ({len(problems)} issue(s)):", file=sys.stderr)
        for problem in problems:
            print(f"  - {problem}", file=sys.stderr)
        return 1

    print("openapi drift check passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())

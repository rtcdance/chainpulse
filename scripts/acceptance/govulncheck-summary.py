#!/usr/bin/env python3
"""Summarize `govulncheck -json` output into the §10 security artifact.

govulncheck emits a STREAM of pretty-printed JSON objects, not JSONL — one object
per record, each spanning many lines. Reading it line by line silently yields
nothing, which is how a broken parser reports "no vulnerabilities" from a scan
that found many. This decodes the stream properly with raw_decode.

A finding is "callable" when its trace reaches a function symbol; a finding
without one is a dependency that is present but not reached from this module.
Only the callable set is reported as actionable, because that is the set
govulncheck itself treats as a vulnerability.

Usage: govulncheck-summary.py --input govulncheck.json --out-dir <sha dir>
"""
from __future__ import annotations

import argparse
import collections
import datetime
import json
import sys
from pathlib import Path


def decode_stream(text: str) -> list[dict]:
    decoder = json.JSONDecoder()
    objects: list[dict] = []
    index = 0
    while index < len(text):
        while index < len(text) and text[index] in " \n\r\t":
            index += 1
        if index >= len(text):
            break
        obj, index = decoder.raw_decode(text, index)
        objects.append(obj)
    return objects


def build_summary(objects: list[dict]) -> dict:
    config = next((o["config"] for o in objects if "config" in o), {})
    advisories = {o["osv"]["id"]: o["osv"] for o in objects if "osv" in o}
    findings = [o["finding"] for o in objects if "finding" in o]

    callable_ids = sorted(
        {f["osv"] for f in findings if any(fr.get("function") for fr in (f.get("trace") or []))}
    )

    by_module: dict[str, dict[str, set]] = collections.defaultdict(lambda: {"ids": set(), "fixes": set()})
    for osv_id in callable_ids:
        for affected in advisories[osv_id].get("affected") or []:
            package = (affected.get("package") or {}).get("name", "?")
            by_module[package]["ids"].add(osv_id)
            for rng in affected.get("ranges") or []:
                for event in rng.get("events") or []:
                    if event.get("fixed"):
                        by_module[package]["fixes"].add(event["fixed"])

    distinct = set()
    for value in by_module.values():
        distinct |= value["ids"]

    return {
        "schema": "chainpulse.security-scan.v1",
        "generated_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "scanner": {
            "name": config.get("scanner_name"),
            "version": config.get("scanner_version"),
            "db": config.get("db"),
            "db_last_modified": config.get("db_last_modified"),
            "scan_level": config.get("scan_level"),
            "scan_mode": config.get("scan_mode"),
        },
        "toolchain_go_version": config.get("go_version"),
        "totals": {
            "advisories_considered": len(advisories),
            "finding_records": len(findings),
            "callable_findings": sum(
                1 for f in findings if any(fr.get("function") for fr in (f.get("trace") or []))
            ),
            "callable_advisories": len(distinct),
            "module_advisory_memberships": sum(len(v["ids"]) for v in by_module.values()),
        },
        "by_module": {
            package: {"advisories": sorted(v["ids"]), "fixes": sorted(v["fixes"])}
            for package, v in sorted(by_module.items(), key=lambda kv: (-len(kv[1]["ids"]), kv[0]))
        },
        "callable_advisory_ids": callable_ids,
        "raw_artifact": "security/govulncheck.json",
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--out-dir", type=Path, default=None)
    parser.add_argument("--sbom-dir", type=Path, default=None, help="also write the SBOM record here")
    args = parser.parse_args(argv)

    if not args.input.is_file():
        print(f"error: {args.input} not found", file=sys.stderr)
        return 2

    objects = decode_stream(args.input.read_text())
    summary = build_summary(objects)
    totals = summary["totals"]

    if totals["advisories_considered"] == 0 or totals["finding_records"] == 0:
        print("error: parsed zero records — the stream decode is wrong, refusing to report a clean scan",
              file=sys.stderr)
        return 1

    encoded = json.dumps(summary, indent=2) + "\n"
    if args.out_dir:
        target = args.out_dir / "security" / "summary.json"
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(encoded)
        print(f"wrote {target}")

    if args.sbom_dir:
        sbom = next((o["SBOM"] for o in objects if "SBOM" in o), None)
        if sbom is None:
            print("error: no SBOM record in the scan output", file=sys.stderr)
            return 1
        target = args.sbom_dir / "sbom" / "govulncheck-sbom.json"
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(json.dumps(
            {
                "schema": "chainpulse.sbom.v1",
                "produced_by": f"{summary['scanner']['name']} {summary['scanner']['version']}",
                "note": "software bill of materials emitted by govulncheck for this module graph",
                **sbom,
            },
            indent=2,
        ) + "\n")
        print(f"wrote {target}")

    print(f"advisories considered : {totals['advisories_considered']}")
    print(f"callable advisories   : {totals['callable_advisories']}")
    print(f"callable findings     : {totals['callable_findings']}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

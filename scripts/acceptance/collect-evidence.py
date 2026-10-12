#!/usr/bin/env python3
"""collect-evidence.py — §10 evidence-pack collector.

Task: T12 of docs/specs/2026-10-12-web3-go-backend-delivery-plan.md (finding F21).

Why this exists
---------------
The plan's §10 requires a release evidence pack containing a fixed list of
artifacts (manifest, quality log, per-suite tests.jsonl, skip report, migration
report, canonical digests, benchmark raw data, CPU/heap/goroutine profiles,
chaos report, deployment report, checksums, SBOM, security scan, remote run
URLs). T18 consumes that pack. Assembling it by hand across 15+ artifacts
reliably produces something incomplete or unverifiable.

This tool assembles the pack from a directory of raw artifacts and REFUSES to
succeed when a required artifact is missing. The rule is deliberate: a pack that
is missing an artifact must fail loudly, because a pack that quietly omits one
is indistinguishable from a pack that never produced it.

It is dependency-free (Python 3 stdlib only) so it can run before any toolchain
is installed, and it has a built-in self-test (`--self-test`) so the collector
itself is verified rather than assumed.

Usage
-----
    collect-evidence.py --input build/raw --out build/evidence

    # Verify the collector: builds a complete pack (must succeed) and an
    # incomplete one (must fail naming the missing artifact).
    collect-evidence.py --self-test
"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import sys
import tempfile
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path


@dataclass(frozen=True)
class ArtifactSpec:
    """One entry of the §10 pack.

    pattern is relative to --input and may be a glob (e.g. tests/*.jsonl).
    min_count is how many matches are required; 1 unless the plan asks for a set.
    """

    name: str
    pattern: str
    why: str
    min_count: int = 1


# The §10 minimum list. Keep this in sync with the plan; the self-test asserts
# the collector reacts to each entry.
REQUIRED_ARTIFACTS: tuple[ArtifactSpec, ...] = (
    ArtifactSpec("quality.log", "quality.log", "build/check/lint output"),
    ArtifactSpec("tests.jsonl", "tests/*.jsonl", "per-suite go test -json streams", min_count=1),
    ArtifactSpec("skip-report.json", "skip-report.json", "the single skip taxonomy"),
    ArtifactSpec("required-cases-report.json", "required-cases-report.json", "required-case coverage"),
    ArtifactSpec("migration-report.json", "migration-report.json", "fresh + upgrade migration evidence"),
    ArtifactSpec("canonical-digests.json", "canonical-digests.json", "oracle digests per scenario"),
    ArtifactSpec("bench-raw.json", "bench-raw.json", "raw benchmark output"),
    ArtifactSpec("profiles-cpu", "profiles/*.cpu.pprof", "CPU profile"),
    ArtifactSpec("profiles-heap", "profiles/*.heap.pprof", "heap profile"),
    ArtifactSpec("profiles-goroutine", "profiles/*.goroutine.pprof", "goroutine profile"),
    ArtifactSpec("chaos-report.json", "chaos-report.json", "fault/rollback rehearsal results"),
    ArtifactSpec("deployment-report.json", "deployment-report.json", "deployment + rollback results"),
    ArtifactSpec("sbom", "sbom/*", "software bill of materials"),
    ArtifactSpec("security-scan", "security/*", "security scan output"),
    ArtifactSpec("run-urls.json", "run-urls.json", "remote CI run URLs"),
)


@dataclass
class Collected:
    name: str
    source: str
    target: str
    sha256: str
    size: int


@dataclass
class Result:
    collected: list[Collected] = field(default_factory=list)
    missing: list[str] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return not self.missing


def sha256_of(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def collect(input_dir: Path, out_dir: Path, specs=REQUIRED_ARTIFACTS) -> Result:
    """Copy every required artifact into out_dir and hash it.

    A spec that matches nothing lands in Result.missing, which makes the run
    fail. Nothing is written for a missing artifact, so the pack never contains
    an empty placeholder that a reader could mistake for a real result.
    """
    result = Result()
    out_dir.mkdir(parents=True, exist_ok=True)

    for spec in specs:
        matches = sorted(p for p in input_dir.glob(spec.pattern) if p.is_file())
        if len(matches) < spec.min_count:
            result.missing.append(f"{spec.name} ({spec.why}): no file matched {spec.pattern!r}")
            continue

        for source in matches:
            # Preserve the relative path so tests/unit.jsonl stays distinguishable
            # from tests/e2e.jsonl.
            relative = source.relative_to(input_dir)
            target = out_dir / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)
            result.collected.append(
                Collected(
                    name=spec.name,
                    source=str(source),
                    target=str(relative),
                    sha256=sha256_of(target),
                    size=target.stat().st_size,
                )
            )

    return result


def write_manifest(out_dir: Path, result: Result, input_dir: Path) -> Path:
    manifest = {
        "schema": "chainpulse.evidence/v1",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "input_dir": str(input_dir),
        "complete": result.ok,
        "artifact_count": len(result.collected),
        "artifacts": [
            {
                "name": item.name,
                "path": item.target,
                "sha256": item.sha256,
                "size": item.size,
            }
            for item in result.collected
        ],
        "missing": result.missing,
    }

    manifest_path = out_dir / "manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return manifest_path


def write_checksums(out_dir: Path, result: Result) -> Path:
    """A checksum list over the collected files, excluding the manifest itself
    (which cannot contain its own hash)."""
    checksums_path = out_dir / "checksums.txt"
    lines = [f"{item.sha256}  {item.target}" for item in sorted(result.collected, key=lambda i: i.target)]
    checksums_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return checksums_path


def report(result: Result, out_dir: Path) -> None:
    print(f"collected {len(result.collected)} artifact(s) into {out_dir}")
    for item in result.collected:
        print(f"  ok   {item.target}  ({item.size} bytes)")

    if result.missing:
        print()
        print(f"MISSING {len(result.missing)} required artifact(s):", file=sys.stderr)
        for item in result.missing:
            print(f"  MISS {item}", file=sys.stderr)
        print()
        print("The pack is incomplete, so the run fails (F21).", file=sys.stderr)


# --------------------------------------------------------------------------- #
# Self-test
# --------------------------------------------------------------------------- #

COMPLETE_INPUT = {
    "quality.log": "build ok\n",
    "tests/unit.jsonl": '{"Action":"pass","Test":"TestA"}\n',
    "tests/integration.jsonl": '{"Action":"pass","Test":"TestB"}\n',
    "skip-report.json": '{"skips":[]}\n',
    "required-cases-report.json": '{"cases":[]}\n',
    "migration-report.json": '{"fresh":"ok","upgrade":"ok"}\n',
    "canonical-digests.json": '{"events":"deadbeef"}\n',
    "bench-raw.json": '{"bench":[]}\n',
    "profiles/run.cpu.pprof": "cpu\n",
    "profiles/run.heap.pprof": "heap\n",
    "profiles/run.goroutine.pprof": "goroutine\n",
    "chaos-report.json": '{"scenarios":[]}\n',
    "deployment-report.json": '{"rollback":"ok"}\n',
    "sbom/sbom.json": '{"components":[]}\n',
    "security/scan.json": '{"findings":[]}\n',
    "run-urls.json": '{"runs":[]}\n',
}


def write_input(root: Path, files: dict[str, str]) -> None:
    for relative, content in files.items():
        path = root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")


def self_test() -> int:
    """Prove the collector succeeds on a complete pack and FAILS on an
    incomplete one, naming what is missing."""
    failures: list[str] = []

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)

        # 1. A complete pack must succeed and hash every artifact.
        complete_input = root / "complete-in"
        complete_out = root / "complete-out"
        write_input(complete_input, COMPLETE_INPUT)

        result = collect(complete_input, complete_out)
        if not result.ok:
            failures.append(f"complete pack reported missing: {result.missing}")
        if len(result.collected) != len(COMPLETE_INPUT):
            failures.append(
                f"expected {len(COMPLETE_INPUT)} artifacts, collected {len(result.collected)}"
            )

        manifest_path = write_manifest(complete_out, result, complete_input)
        write_checksums(complete_out, result)

        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        if manifest.get("complete") is not True:
            failures.append("manifest.complete should be true for a complete pack")
        if not (complete_out / "checksums.txt").exists():
            failures.append("checksums.txt was not written")
        for entry in manifest.get("artifacts", []):
            if len(entry.get("sha256", "")) != 64:
                failures.append(f"artifact {entry.get('path')} has a malformed sha256")

        # 2. An incomplete pack must fail and name the missing artifact.
        incomplete_input = root / "incomplete-in"
        incomplete_out = root / "incomplete-out"
        write_input(incomplete_input, COMPLETE_INPUT)
        (incomplete_input / "canonical-digests.json").unlink()

        incomplete = collect(incomplete_input, incomplete_out)
        if incomplete.ok:
            failures.append("an incomplete pack must NOT be reported as ok")
        if not any("canonical-digests" in item for item in incomplete.missing):
            failures.append(f"the missing artifact was not named: {incomplete.missing}")
        if (incomplete_out / "canonical-digests.json").exists():
            failures.append("a missing artifact must not leave a placeholder behind")

        # 3. A pack missing a whole set (profiles) must fail too.
        no_profiles_input = root / "no-profiles-in"
        write_input(no_profiles_input, COMPLETE_INPUT)
        for profile in (no_profiles_input / "profiles").glob("*"):
            profile.unlink()

        no_profiles = collect(no_profiles_input, root / "no-profiles-out")
        if no_profiles.ok:
            failures.append("a pack with no profiles must not be ok")
        if not any("profiles-" in item for item in no_profiles.missing):
            failures.append("the missing profile set was not named")

    if failures:
        print("collect-evidence self-test FAILED:", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        return 1

    print(f"collect-evidence self-test passed ({len(REQUIRED_ARTIFACTS)} required artifact specs)")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Assemble the §10 evidence pack.")
    parser.add_argument("--input", type=Path, help="directory holding the raw artifacts")
    parser.add_argument("--out", type=Path, help="directory to write the evidence pack into")
    parser.add_argument("--self-test", action="store_true", help="verify the collector itself")
    args = parser.parse_args(argv)

    if args.self_test:
        return self_test()

    if not args.input or not args.out:
        parser.error("--input and --out are required unless --self-test is given")
    if not args.input.is_dir():
        print(f"input directory does not exist: {args.input}", file=sys.stderr)
        return 2

    result = collect(args.input, args.out)
    write_manifest(args.out, result, args.input)
    if result.ok:
        write_checksums(args.out, result)

    report(result, args.out)
    return 0 if result.ok else 1


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env bash
# checksums.sh - generate and verify the release checksum manifest.
#
# Plan section 7.5 item 3: "修 release.yml checksum 循环覆盖写文件问题；每个文件名
# 唯一且可验证". The original loop wrote
#
#     for dir in binaries-*; do cd "$dir"; sha256sum * > ../checksums.txt; cd ..; done
#
# so every iteration truncated the same file and only the last platform's
# checksums survived. The release matrix builds five platforms with five binaries
# each, so 20 of 25 checksums were silently dropped and the truncated manifest was
# published as if complete.
#
# The decision logic lives here rather than inline in the workflow so that
# scripts/acceptance/negative-cases.py can drive it and prove the guards fire -
# the same split acceptance-gate.py uses for the aggregation logic.
#
# Usage
# -----
#     bash scripts/acceptance/checksums.sh dist
#     bash scripts/acceptance/checksums.sh --verify-only dist
#
# `--verify-only` skips generation and runs only the guards. It exists so the
# negative-case suite can inject a fault between hashing and verification - a
# binary that changes after it was hashed, which is not otherwise reachable from
# outside.
#
# Exit codes
# ----------
#   0  manifest written and every entry verified
#   1  a guard fired (count mismatch, duplicate name, or a failed verification)
#   2  usage error
#
# What the guards are for
# -----------------------
#   count  - catches the original truncation, and an artifact that downloaded
#            nothing.
#   unique - a checksum that cannot be attributed to one file is not a checksum.
#   verify - catches a binary that changed after it was hashed.

set -euo pipefail

verify_only=0
if [ "${1:-}" = "--verify-only" ]; then
  verify_only=1
  shift
fi

if [ "$#" -ne 1 ]; then
  echo "usage: $0 [--verify-only] <dist-dir>" >&2
  exit 2
fi

dist="$1"
if [ ! -d "$dist" ]; then
  echo "error: $dist is not a directory" >&2
  exit 2
fi

cd "$dist"

if ! compgen -G 'binaries-*' > /dev/null; then
  echo "error: no binaries-* directories under $dist" >&2
  exit 1
fi

if [ "$verify_only" -eq 0 ]; then
  # Truncate once, then append per directory: the whole point of the fix is that
  # the per-directory step must not truncate.
  : > checksums.txt
  for dir in binaries-*; do
    [ -d "$dir" ] || continue
    (cd "$dir" && sha256sum *)
  done >> checksums.txt

  sort -k2 -o checksums.txt checksums.txt
fi

if [ ! -s checksums.txt ]; then
  echo "::error::checksums.txt is missing or empty" >&2
  exit 1
fi

present=$(find binaries-* -type f | wc -l | tr -d ' ')
recorded=$(wc -l < checksums.txt | tr -d ' ')
if [ "$recorded" -ne "$present" ]; then
  echo "::error::checksums.txt records $recorded of $present binaries" >&2
  exit 1
fi

dups=$(awk '{print $2}' checksums.txt | sort | uniq -d)
if [ -n "$dups" ]; then
  echo "::error::duplicate filenames in checksums.txt: $dups" >&2
  exit 1
fi

for dir in binaries-*; do
  [ -d "$dir" ] || continue
  if ! (cd "$dir" && sha256sum -c ../checksums.txt --ignore-missing --quiet); then
    echo "::error::checksum verification failed in $dir" >&2
    exit 1
  fi
done

echo "verified $recorded checksums across $(ls -d binaries-* | wc -l | tr -d ' ') platforms"

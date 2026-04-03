# Phase 304 - gRPC Streaming Data Plane Assessment

## Status
Status: Approved

## Summary

Assess the current gRPC streaming data-plane work after the metrics surface
reached compact source posture, delivery posture, and reliability hint.

## Problem

After phase 303, the gRPC streaming metrics surface now exposes:

- source posture
- delivery posture
- reliability hint

This gives the gRPC streaming path the same broad facts-to-posture-to-hint
shape used elsewhere in the architecture work, but the repository still lacks a
clear statement of whether that is enough to treat the current gRPC slice as a
stable baseline with a stop-line.

## Decision

Classify the current gRPC streaming work as:

- `stage-complete for the gRPC streaming data-plane baseline`

This means:

- the current gRPC streaming metrics surface is strong enough to pause by
  default
- the baseline already exposes compact operational semantics for stream source,
  delivery, and reliability

It does **not** mean:

- protobuf-level stream metadata parity
- a broader gRPC query/control plane
- fully expanded gRPC runtime surfaces

## Scope

In scope:

- gRPC streaming data-plane assessment
- explicit stop-line for the current gRPC streaming baseline
- architecture/index documentation updates

Out of scope:

- protobuf changes
- new gRPC API families
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase304-grpc-streaming-data-plane-assessment.md`

## Exit Criteria

- The docs explicitly describe the current gRPC streaming work as a stable
  baseline with a stop-line.
- Future gRPC streaming expansion is treated as an explicit reopen rather than
  default continuation.

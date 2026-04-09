Title: H5 Dashboard Probe Summary Overflow Fix
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: frontend/src/components/Dashboard.tsx, frontend/src/components/Runtime.tsx

## Status

Approved for implementation.

## Problem Statement

The H5 dashboard and runtime service cards render probe response previews
without wrapping or containment.

Observed fact:

1. long JSON snippets and long error strings can overflow the card body and
   damage readability

## Scope

This bugfix will:

1. keep probe summaries inside their cards
2. preserve readable wrapped text for long response bodies
3. apply the same containment treatment to dashboard and runtime cards

## Non-Goals

This bugfix will not:

1. redesign the dashboard layout
2. truncate probe summaries to a single line
3. alter probe fetch behavior

## Selected Approach

1. render probe summaries in bounded monospace preview blocks
2. enable line wrapping and overflow scrolling inside the preview block
3. preserve the existing response preview text content

## Test Strategy

1. `bash scripts/spec-approval-check.sh docs/specs/2026-04-09-h5-dashboard-probe-summary-overflow-fix.md`
2. `cd frontend && npm run build`


Title: H5 Service Card Overflow Containment
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: frontend/src/components/Dashboard.tsx, frontend/src/components/Runtime.tsx

## Status

Approved for implementation.

## Problem Statement

The H5 dashboard service cards can still overflow horizontally even after probe
summary wrapping was improved.

Observed fact:

1. long service base URLs and probe paths can prevent flex children from
   shrinking, causing the whole card to exceed its grid cell

## Scope

This bugfix will:

1. allow service cards to shrink inside the grid
2. wrap long base URLs and probe paths inside the card boundary
3. keep status badges visible without forcing horizontal overflow

## Selected Approach

1. add `min-w-0` to the card and flex children that hold long text
2. allow probe header rows to wrap instead of forcing a single line
3. mark status badges as non-shrinking while the text column absorbs wrapping

## Test Strategy

1. `bash scripts/spec-approval-check.sh docs/specs/2026-04-09-h5-service-card-overflow-containment.md`
2. `cd frontend && npm run build`


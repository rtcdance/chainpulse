---
name: "event-ordering-finality"
description: "Make ordering/finality semantics explicit per chain. Test out-of-order, duplicate, and reorg scenarios. Invoke when touching puller/indexer/query paths that depend on block/tx/log ordering or chain finality."
---

# Skill: event-ordering-finality

## Trigger

Use this skill when touching puller/indexer/query paths that depend on block/tx/log ordering or chain finality.

## Must Do

1. Define ordering model explicitly:
   - block number
   - tx index
   - log index
2. Define chain finality policy per chain/network where relevant.
3. Separate "observed" vs "finalized" data handling when needed.
4. Ensure reorg-aware replay preserves ordering guarantees.
5. Add tests for out-of-order, late, duplicate, and reorg scenarios.

## Must Not

- No implicit ordering assumptions.
- No mixing finalized and non-finalized semantics without labels/contracts.
- No chain-agnostic finality assumption for all networks.

## Exit Criteria

- Ordering and finality rules are explicit, tested, and documented.
- Reorg handling remains consistent with declared policy.

# Skill: schema-migration-safety

## Trigger

Use this skill when changing DB schema, event model fields, storage indices, or query contracts.

## Must Do

1. Define backward-compatibility strategy:
   - additive-first changes
   - dual-read/dual-write window if needed
2. Provide migration and rollback plan.
3. Add data integrity verification steps.
4. Add tests for old/new schema compatibility on critical paths.
5. Document migration order for monolith and microservice deployments.

## Must Not

- No destructive schema change without rollback path.
- No query contract break without compatibility handling.

## Exit Criteria

- Migration plan is reversible.
- Integrity checks and compatibility tests pass.

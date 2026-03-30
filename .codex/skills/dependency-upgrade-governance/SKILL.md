# Skill: dependency-upgrade-governance

## Trigger

Use this skill when adding/updating/removing dependencies in `go.mod` or upgrading toolchain/library versions.

## Must Do

1. Classify dependency change:
   - security patch
   - bugfix
   - feature
   - breaking major upgrade
2. Record impact analysis:
   - touched modules
   - runtime risk
   - compatibility concerns
3. Verify with focused tests on impacted areas.
4. Provide rollback strategy for risky upgrades.
5. Keep dependency changes minimal and intentional.

## Must Not

- No broad "upgrade all" without explicit reason and verification.
- No hidden transitive-risk upgrade without review note.
- No production dependency upgrade without regression checks.

## Exit Criteria

- Dependency change is justified, verified, and reversible.
- Upgrade scope and risk are documented.

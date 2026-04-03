---
name: "web3-go-architecture-guardrails"
description: "Keep domain/application/adapters/platform boundaries clean. Preserve monolith debug and microservice deploy consistency. Invoke when changing architecture, module boundaries, service interfaces, or startup wiring."
---

# Skill: web3-go-architecture-guardrails

## Trigger

Use this skill when changing architecture, module boundaries, service interfaces, or startup wiring.

## Must Do

1. Keep business logic in `domain` and `application` layers only.
2. Keep transport and infrastructure concerns in adapters.
3. Keep bootstrap code (`cmd/*`) for wiring and lifecycle only.
4. Preserve monolith debug mode and microservice mode behavior parity.
5. Document boundary changes in architecture docs when behavior changes.

## ChainPulse Pointers

- Monolith entry: `cmd/monolithic/chainpulse/main.go`
- Microservice entries: `cmd/microservices/*`
- Architecture prompt: `docs/ARCHITECTURE_PROMPT.md`
- Constraint framework: `docs/guides/ENGINEERING_CONSTRAINT_FRAMEWORK.md`

## Must Not

- No business rule logic in adapters/platform/bootstrap.
- No mode-specific business branches (monolith vs microservice).
- No hidden coupling across module boundaries.

## Exit Criteria

- Boundaries respected in changed files.
- Startup paths still support both modes.
- Docs updated if architecture contracts changed.

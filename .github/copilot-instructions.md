# ChainPulse — Copilot Instructions

The project's agent rules live in a single place: **[`AGENTS.md`](../AGENTS.md)** at the repository root. Read that file before generating code.

It covers: dev environment setup (`GOPROXY`, Go 1.26, linter build), build/test commands, the Definition of Done gate, code style (DDD layering — note there is **no `pkg/adapters`**), testing instructions, security considerations, and commit/PR guidelines.

GitHub Copilot's coding agent reads `AGENTS.md` natively; this file exists only so the instruction stays discoverable. **Do not add rules here** — add them to `AGENTS.md`.

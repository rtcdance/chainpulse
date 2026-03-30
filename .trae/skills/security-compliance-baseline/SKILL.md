# Skill: security-compliance-baseline

## Trigger

Use this skill when touching auth, secrets, network endpoints, config, data access, or API surfaces.

## Must Do

1. Classify data sensitivity and access path.
2. Ensure secrets are environment-based and never hardcoded.
3. Validate least-privilege access for DB/MQ/RPC credentials.
4. Add security checks to verification:
   - static lint/security scan where available
   - input validation and error sanitization
5. Document security impact in spec and PR notes.

## Must Not

- No plaintext secrets in code/tests/docs.
- No broad wildcard permissions without explicit justification.
- No external error leakage through API responses.

## Exit Criteria

- Security impact reviewed and documented.
- No secret leakage and no privilege broadening without approval.

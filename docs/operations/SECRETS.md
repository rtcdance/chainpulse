# ChainPulse Secrets Management

## Overview

ChainPulse K8s manifests use Kubernetes Secrets for sensitive configuration. **All secrets in the repository are placeholders** (`REPLACE_ME`) and must be replaced before any production deployment.

## Required Secrets

The `chainpulse-secrets` Secret (`k8s/base/secret.yaml`) contains:

| Key | Description | Example Generation |
|-----|-------------|-------------------|
| `DATABASE_URL` | PostgreSQL connection string | `echo -n 'postgres://chainpulse:YOUR_PASSWORD@postgres:5432/chainpulse' \| base64` |
| `POSTGRES_PASSWORD` | PostgreSQL user password | `openssl rand -base64 32` then base64-encode |
| `JWT_SECRET` | HMAC-SHA256 signing key (min 32 chars) | `openssl rand -base64 48` then base64-encode |
| `KAFKA_USERNAME` | Kafka SASL username | `echo -n 'your_kafka_user' \| base64` |
| `KAFKA_PASSWORD` | Kafka SASL password | `openssl rand -base64 32` then base64-encode |

## Production Deployment Process

### Option 1: Manual Secret Replacement

```bash
# Generate and apply secrets directly (never commit to git)
kubectl create secret generic chainpulse-secrets \
  --namespace chainpulse \
  --from-literal=DATABASE_URL='postgres://chainpulse:STRONG_PASSWORD@postgres:5432/chainpulse' \
  --from-literal=POSTGRES_PASSWORD='STRONG_PASSWORD' \
  --from-literal=JWT_SECRET='MINIMUM_32_CHARACTER_SECRET_HERE' \
  --from-literal=KAFKA_USERNAME='kafka_user' \
  --from-literal=KAFKA_PASSWORD='KAFKA_PASSWORD' \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Option 2: External Secret Management (Recommended)

For production, use one of:

- **SealedSecrets** (`bitnami-labs/sealed-secrets`) — encrypt secrets in-git, decrypt in-cluster
- **External Secrets Operator** — sync from AWS Secrets Manager / GCP Secret Manager / Vault
- **Vault** — inject secrets via CSI driver

Example with SealedSecrets:
```bash
# Install SealedSecrets controller first
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
# Commit sealed-secret.yaml (safe) — controller decrypts at runtime
```

## Pre-Deployment Checklist

- [ ] All `REPLACE_ME` placeholders replaced with real credentials
- [ ] `JWT_SECRET` is at least 32 characters
- [ ] `POSTGRES_PASSWORD` is not a dictionary word
- [ ] Kafka SASL credentials differ from development defaults (`admin`/`admin`)
- [ ] Secrets are NOT committed to version control (verify with `git diff`)
- [ ] External secret management is configured for production clusters

## Development Credentials

Development credentials are in `docker/.env` (gitignored). Defaults:
- PostgreSQL: `chainpulse_dev` / `chainpulse_dev`
- Kafka: `admin` / `admin`

These are acceptable for local development only.

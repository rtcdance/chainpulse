# Docker Runtime Recovery

This document records the current compose-readiness blocker discovered while
running the docker-compose microservice profile from this repository.

## Current Failure Boundary

When running:

```bash
bash scripts/verify-docker-compose-microservices-readiness.sh
```

the current machine failed before compose startup with:

- Docker CLI available
- Docker context: `desktop-linux`
- Docker daemon unreachable
- socket path expected by the CLI:
  - `unix:///Users/<user>/.docker/run/docker.sock`
- attempting to open Docker Desktop reported:
  - `kLSNoExecutableErr: The executable is missing`

This means the current blocker is **Docker runtime availability**, not compose
syntax or service wiring.

Another compose-start blocker to check is Docker credential helper wiring. If
`~/.docker/config.json` contains:

```json
"credsStore": "desktop"
```

then the machine must also have:

```bash
docker-credential-desktop
```

available on `PATH`. If that helper is missing, `docker compose up` can fail
while pulling images with:

- `error getting credentials`
- `docker-credential-desktop: executable file not found in $PATH`

## Fast Checks

Run these first:

```bash
docker context show
docker info
docker ps
cat ~/.docker/config.json
which docker-credential-desktop
```

Expected healthy state:

- `docker context show` returns the intended context
- `docker info` succeeds
- `docker ps` succeeds

If `docker info` fails with daemon/socket errors, the compose readiness smoke
cannot proceed.

If `docker info` succeeds but image pulls fail with credential-helper errors,
fix the Docker credential helper configuration before retrying compose startup.

## Docker Desktop Checks (macOS)

Check whether Docker Desktop exists:

```bash
ls /Applications | rg '^Docker\\.app$'
```

If the app exists but cannot be opened, try:

```bash
open /Applications/Docker.app
```

If Launch Services returns `kLSNoExecutableErr`, treat the Docker Desktop
installation as damaged or incomplete.

## Recovery Steps

1. Start or reinstall Docker Desktop.
2. Confirm the Docker daemon is reachable:
   ```bash
   docker info
   docker ps
   ```
3. If `~/.docker/config.json` uses `"credsStore": "desktop"`, either:
   - install/restore `docker-credential-desktop`, or
   - remove/replace that `credsStore` setting
4. Re-run the compose service-set check:
   ```bash
   COMPOSE_FILE=docker/docker-compose.microservices.yml \
     bash scripts/verify-docker-compose-stack.sh
   ```
5. Re-run the full compose readiness smoke:
   ```bash
   bash scripts/verify-docker-compose-microservices-readiness.sh
   ```

## Related Entry Points

- [`RUNNABLE_APP.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/RUNNABLE_APP.md)
- [`docker/README.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docker/README.md)
- [`verify-docker-compose-stack.sh`](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/verify-docker-compose-stack.sh)
- [`verify-docker-compose-microservices-readiness.sh`](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/verify-docker-compose-microservices-readiness.sh)

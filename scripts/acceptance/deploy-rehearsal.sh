#!/usr/bin/env bash
# Local deployment rehearsal for the monolithic image (§7.5, T00's frozen target).
#
# What it actually does, in order:
#   1. build the previous release image from CHAINPULSE_DEPLOY_REHEARSAL_PREV_REF
#      (its own source tree, cross-compiled) and confirm it serves /health, so the
#      rollback target is a verified artefact rather than a hopeful digest;
#   2. migrate the deployed database to HEAD (a deploy is migrate-then-start; the
#      current release correctly refuses to start below schema 20 - that is the
#      fail-closed behaviour E15 pins, not something to work around here);
#   3. cross-compile HEAD, build the current image and record its immutable digest;
#   4. start it against the already-running PostgreSQL/MongoDB/Redis, wait for the
#      container healthcheck and smoke three surfaces (health, a route, metrics);
#   5. roll back to the PREVIOUS image digest, smoke the same surfaces again and
#      compare the indexed row count before and after, which is what "the rollback
#      restored the old release without changing the data" means at this level.
#
# What it does NOT do, and says so in the report rather than implying otherwise:
# no registry push, no multi-platform build, no real chain traffic, and no
# observation window. The row-count comparison is an equality check, not a load
# test: this rehearsal generates no new events.
#
# Exits non-zero when any container never becomes healthy or a smoke probe fails,
# because a deployment rehearsal that cannot fail is not a rehearsal.
set -euo pipefail

SHA="$(git rev-parse HEAD)"
TAG="chainpulse:${SHA:0:12}"
PREV_REF="${CHAINPULSE_DEPLOY_REHEARSAL_PREV_REF:-7636755}"
PREV_TAG="chainpulse:prev-${PREV_REF:0:12}"
OUT_DIR="${1:-build/acceptance/${SHA:0:12}}"
DEP_NETWORK="$(docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' chainpulse-postgres)"
CONTAINER=cp-deploy-rehearsal
REPORT="${OUT_DIR}/deployment-report.json"
PREV_SRC=""

cleanup() {
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  rm -f ./chainpulse-linux ./chainpulse-migrate
  if [ -n "$PREV_SRC" ]; then rm -rf "$PREV_SRC"; fi
}
trap cleanup EXIT

# start_target <image> -> runs the container with the rehearsal's environment.
start_target() {
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  docker run -d --name "$CONTAINER" --network "$DEP_NETWORK" \
    -e MONGODB_URI='mongodb://chainpulse-mongodb:27017' \
    -e DATABASE_URL='postgres://chainpulse:chainpulse_dev@chainpulse-postgres:5432/chainpulse_dev?sslmode=disable' \
    -e DATABASE_SSLMODE=disable \
    -e DATABASE_TYPE=postgres \
    -e REDIS_URL='redis://chainpulse-redis:6379' \
    -e CACHE_CONNECTION_URL='redis://chainpulse-redis:6379' \
    "$1" >/dev/null
}

# wait_healthy -> echoes yes/no; never fails the script on its own.
wait_healthy() {
  local state=missing
  for _ in $(seq 1 30); do
    state="$(docker inspect --format='{{.State.Health.Status}}' "$CONTAINER" 2>/dev/null || echo missing)"
    if [ "$state" = "healthy" ]; then echo yes; return 0; fi
    if ! docker ps --filter "name=${CONTAINER}" --format '{{.Names}}' | grep -q "$CONTAINER"; then break; fi
    sleep 1
  done
  echo no
}

# event_rows -> indexed rows in the rehearsal database, or "unknown" if the
# container/psql pairing is unavailable (recorded as null, never as zero).
event_rows() {
  docker exec chainpulse-postgres psql -U chainpulse -d chainpulse_dev -tAc \
    "select count(*) from blockchain_events" 2>/dev/null | tr -d '[:space:]' || echo unknown
}

echo "== previous release image (rollback target) =="
PREV_SRC="$(mktemp -d)"
git archive --format=tar -o "$PREV_SRC/revision.tar" "$PREV_REF"
tar -x -f "$PREV_SRC/revision.tar" -C "$PREV_SRC"
prev_ups="$(ls "$PREV_SRC"/migrations/*.up.sql | wc -l | tr -d ' ')"
head_ups="$(ls migrations/*.up.sql | wc -l | tr -d ' ')"
echo "previous release migrations: ${prev_ups}; HEAD migrations: ${head_ups}"
(
  cd "$PREV_SRC"
  CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -o "$PREV_SRC/chainpulse-linux" ./cmd/monolithic/chainpulse
)
cp "$PREV_SRC/chainpulse-linux" ./chainpulse-linux
docker build -f docker/Dockerfile.prebuilt -t "$PREV_TAG" . >/dev/null
prev_digest="$(docker inspect --format='{{.Id}}' "$PREV_TAG")"
prev_image_bytes="$(docker inspect --format='{{.Size}}' "$PREV_TAG")"
rm -f ./chainpulse-linux

echo "== verify the rollback target serves =="
start_target "$PREV_TAG"
prev_healthy="$(wait_healthy)"
if [ "$prev_healthy" != "yes" ]; then
  echo "FAILED: the previous release never became healthy, so there is no verified rollback target" >&2
  docker logs "$CONTAINER" 2>&1 | tail -15 >&2
  exit 1
fi
prev_digest_health="$(docker exec "$CONTAINER" curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/health || echo 000)"

echo "== migrate the deployed database to HEAD =="
# This is the deploy's first real action. Skipping it is what made the current
# release refuse to start on a database the previous release accepted: the
# canonical aggregate cache requires schema >= 20 with a clean ledger, which is
# E15's fail-closed contract, so the rehearsal has to perform the upgrade rather
# than pretend the environment was already current.
#
# The migration runs INSIDE the deployment network (its own cross-compiled
# binary in an Alpine container) instead of from the host, because the container's
# published host port is not a reliable route: a developer-local PostgreSQL bound
# to 127.0.0.1:5432 shadows it (observed: `role "chainpulse" does not exist`).
# Resolving `chainpulse-postgres` on the network has no such ambiguity.
REHEARSAL_DATABASE_URL="${CHAINPULSE_REHEARSAL_DATABASE_URL:-postgres://chainpulse:chainpulse_dev@chainpulse-postgres:5432/chainpulse_dev?sslmode=disable}"
MIGRATE_BINARY=./chainpulse-migrate
schema_query='select version from schema_migrations limit 1'
schema_before="$(docker exec chainpulse-postgres psql -U chainpulse -d chainpulse_dev -tAc "$schema_query" | tr -d '[:space:]')"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$MIGRATE_BINARY" ./cmd/migrate
docker run --rm --network "$DEP_NETWORK" \
  -v "$(pwd)/migrations:/migrations:ro" \
  -v "$(pwd)/${MIGRATE_BINARY#./}:/chainpulse-migrate:ro" \
  -e DATABASE_URL="$REHEARSAL_DATABASE_URL" \
  alpine:3.19 /chainpulse-migrate -path /migrations up
schema_after="$(docker exec chainpulse-postgres psql -U chainpulse -d chainpulse_dev -tAc "$schema_query" | tr -d '[:space:]')"
echo "schema ${schema_before} -> ${schema_after}"
rm -f "$MIGRATE_BINARY"

echo "== cross-compile linux/arm64 =="
start_build=$(python3 -c 'import time;print(time.time())')
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -ldflags "-X main.Version=deploy-rehearsal -X main.Commit=${SHA}" \
  -o ./chainpulse-linux ./cmd/monolithic/chainpulse
binary_bytes=$(wc -c < ./chainpulse-linux | tr -d ' ')

echo "== docker build =="
docker build -f docker/Dockerfile.prebuilt -t "$TAG" . >/dev/null
digest="$(docker inspect --format='{{.Id}}' "$TAG")"
image_bytes="$(docker inspect --format='{{.Size}}' "$TAG")"
end_build=$(python3 -c 'import time;print(time.time())')

echo "== start =="
start_run=$(python3 -c 'import time;print(time.time())')
start_target "$TAG"

healthy="$(wait_healthy)"
end_run=$(python3 -c 'import time;print(time.time())')
rows_before_rollback="$(event_rows)"

echo "== smoke =="
# The events route enforces the explicit chain filter (I01): an unfiltered query
# is refused with 503 query_unavailable ("an explicit chain filter is required"),
# so the smoke has to name the chain this deployment serves. Deriving it from the
# database keeps the probe valid whatever the rehearsal database holds, instead of
# asserting a status the route is designed to reject.
smoke_chain="$(docker exec chainpulse-postgres psql -U chainpulse -d chainpulse_dev -tAc \
  'select chain_id from blockchain_events group by chain_id order by count(*) desc limit 1' 2>/dev/null | tr -d '[:space:]')"
smoke_chain="${smoke_chain:-ethereum}"
events_path="/api/v1/events?chain_id=${smoke_chain}&limit=5"
smoke() {  # path -> "http_code time_total"
  docker exec "$CONTAINER" curl -s -o /dev/null -w '%{http_code} %{time_total}' "http://localhost:8080$1" 2>/dev/null || echo "000 0"
}
read -r health_code health_time <<<"$(smoke /health)"
read -r events_code events_time <<<"$(smoke "$events_path")"
read -r metrics_code metrics_time <<<"$(smoke /metrics)"
# Written to a FILE rather than interpolated into the Python below: the body is
# JSON and can end on a quote at the truncation boundary, which would terminate
# the Python string literal early. Embedding data into generated source is how
# that bug happens; passing a path removes the hazard whatever the content is.
mkdir -p "${OUT_DIR}/deploy"
health_body_file="${OUT_DIR}/deploy/health-body.json"
docker exec "$CONTAINER" curl -s "http://localhost:8080/health" 2>/dev/null | head -c 600 > "$health_body_file" || true

echo "== rollback to the previous release digest =="
# The rollback target is the PREVIOUS image built and health-verified above, not
# a recreated copy of the current one: a rehearsal that rolls back to its own
# digest exercises nothing.
start_target "$PREV_TAG"
rollback_healthy="$(wait_healthy)"
read -r rollback_health_code rollback_health_time <<<"$(smoke /health)"
read -r rollback_events_code rollback_events_time <<<"$(smoke "$events_path")"
read -r rollback_metrics_code rollback_metrics_time <<<"$(smoke /metrics)"
rows_after_rollback="$(event_rows)"

python3 - "$REPORT" <<PYEOF
import json, sys, datetime, subprocess, os
report_path = sys.argv[1]
os.makedirs(os.path.dirname(report_path), exist_ok=True)
report = {
  "schema": "chainpulse.deployment-report.v1",
  "generated_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
  "commit_sha": "${SHA}",
  "tree_dirty": bool(subprocess.run(["git","status","--porcelain"],capture_output=True,text=True).stdout.strip()),
  "target": {
    "kind": "local-docker",
    "note": "T00 froze the CD target as a local Docker demonstration; §7.5 allows a local isolated environment.",
    "network": "${DEP_NETWORK}",
  },
  "artifact": {
    "tag": "${TAG}",
    "digest": "${digest}",
    "image_bytes": int("${image_bytes}"),
    "binary_bytes": int("${binary_bytes}"),
    "build_seconds": round(${end_build} - ${start_build}, 2),
    "start_seconds": round(${end_run} - ${start_run}, 2),
    "platform": "linux/arm64",
  },
  "previous_artifact": {
    "ref": "${PREV_REF}",
    "tag": "${PREV_TAG}",
    "digest": "${prev_digest}",
    "image_bytes": int("${prev_image_bytes}"),
    "health_status": "${prev_digest_health}",
    "migrations": int("${prev_ups}"),
    "note": "the rollback target: built from its own revision, and verified to serve /health before it is used",
  },
  "head_migrations": int("${head_ups}"),
  "smoke_chain": "${smoke_chain}",
  "migration": {
    "from": "${schema_before}",
    "to": "${schema_after}",
    "note": "the expand migration the deploy applies before the new release starts",
  },
  "health": {"status": "${healthy}", "after_rollback_health": "${rollback_healthy}"},
  "smoke": [
    {"probe": "/health", "http_status": int("${health_code}"), "seconds": float("${health_time}")},
    {"probe": "${events_path}", "http_status": int("${events_code}"), "seconds": float("${events_time}")},
    {"probe": "/metrics", "http_status": int("${metrics_code}"), "seconds": float("${metrics_time}")},
  ],
  "health_body_excerpt": (open("${health_body_file}").read() if os.path.exists("${health_body_file}") else ""),
  "rollback": {
    "from_digest": "${digest}",
    "to_digest": "${prev_digest}",
    "distinct_digests": "${digest}" != "${prev_digest}",
    "health_after_rollback": "${rollback_healthy}",
    "rows_before_rollback": (int("${rows_before_rollback}") if "${rows_before_rollback}".isdigit() else None),
    "rows_after_rollback": (int("${rows_after_rollback}") if "${rows_after_rollback}".isdigit() else None),
    "smoke": [
      {"probe": "/health", "http_status": int("${rollback_health_code}"), "seconds": float("${rollback_health_time}")},
      {"probe": "${events_path}", "http_status": int("${rollback_events_code}"), "seconds": float("${rollback_events_time}")},
      {"probe": "/metrics", "http_status": int("${rollback_metrics_code}"), "seconds": float("${rollback_metrics_time}")},
    ],
    "note": "rolled back to the previous release's image digest on the same database; the row counts are an equality check across the swap, not a load result",
  },
  "not_covered": [
    "registry push and immutable pull by digest",
    "multi-arch build (only the deployed platform was built)",
    "real chain traffic through the deployed service",
    "the 15-minute observation window of §7.5 step 6",
    "a data digest comparison under concurrent writes (E15's acceptance test owns that)",
  ],
}
json.dump(report, open(report_path, "w"), indent=2)
print("deployment report written to", report_path)
PYEOF

echo "== verdict =="
if [ "$healthy" != "yes" ] || [ "$rollback_healthy" != "yes" ]; then
  echo "FAILED: container did not become healthy (current=${healthy} after_rollback=${rollback_healthy})" >&2
  exit 1
fi
if [ "$digest" = "$prev_digest" ]; then
  echo "FAILED: the rollback target digest equals the deployed digest, so nothing was rolled back" >&2
  exit 1
fi
for pair in \
  "${health_code}:health" \
  "${events_code}:/api/v1/events" \
  "${metrics_code}:/metrics" \
  "${rollback_health_code}:health after rollback" \
  "${rollback_events_code}:/api/v1/events after rollback" \
  "${rollback_metrics_code}:/metrics after rollback"; do
  code="${pair%%:*}"; name="${pair#*:}"
  if [ "$code" != "200" ]; then echo "FAILED: ${name} returned ${code}" >&2; exit 1; fi
done
# Only enforced when both sides were measurable; an unreadable count is recorded
# as null in the report rather than turned into a pass or a flake.
if [ "$rows_before_rollback" != "unknown" ] && [ "$rows_after_rollback" != "unknown" ] \
  && [ "$rows_before_rollback" != "$rows_after_rollback" ]; then
  echo "FAILED: the indexed row count changed across the rollback (${rows_before_rollback} -> ${rows_after_rollback})" >&2
  exit 1
fi
echo "deployment rehearsal PASSED (current=${digest:0:19}..., rolled back to=${prev_digest:0:19}..., rows=${rows_after_rollback})"

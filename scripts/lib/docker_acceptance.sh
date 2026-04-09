#!/usr/bin/env bash

docker_acceptance_require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker is not installed or not in PATH" >&2
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    local context
    context="$(docker context show 2>/dev/null || echo unknown)"
    echo "docker daemon is not reachable for context '${context}'; start Docker Desktop or the Docker daemon first" >&2
    echo "expected Docker socket: ${DOCKER_HOST:-unix:///Users/${USER}/.docker/run/docker.sock}" >&2
    exit 1
  fi
}

docker_acceptance_prepare_temp_docker_config() {
  local source_config current_context
  if [[ -n "${TEMP_DOCKER_CONFIG:-}" ]]; then
    return 0
  fi

  source_config="${HOME}/.docker"
  TEMP_DOCKER_CONFIG="$(mktemp -d "${TMPDIR:-/tmp}/chainpulse-docker-config.XXXXXX")"

  if [[ -d "${source_config}/contexts" ]]; then
    cp -R "${source_config}/contexts" "${TEMP_DOCKER_CONFIG}/contexts"
  fi
  if [[ -d "${source_config}/cli-plugins" ]]; then
    cp -R "${source_config}/cli-plugins" "${TEMP_DOCKER_CONFIG}/cli-plugins"
  fi

  current_context="$(sed -n 's/.*"currentContext"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${source_config}/config.json" 2>/dev/null | head -n 1)"
  if [[ -n "${current_context}" ]]; then
    cat >"${TEMP_DOCKER_CONFIG}/config.json" <<EOF
{
  "auths": {},
  "currentContext": "${current_context}"
}
EOF
  else
    cat >"${TEMP_DOCKER_CONFIG}/config.json" <<'EOF'
{
  "auths": {}
}
EOF
  fi
}

docker_acceptance_check_docker_credential_helper() {
  local docker_config helper
  docker_config="${HOME}/.docker/config.json"
  if [[ ! -f "${docker_config}" ]]; then
    return 0
  fi

  helper="$(sed -n 's/.*"credsStore"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${docker_config}" | head -n 1)"
  if [[ -z "${helper}" ]]; then
    return 0
  fi

  if ! command -v "docker-credential-${helper}" >/dev/null 2>&1; then
    "${DOCKER_ACCEPTANCE_LOG_FN}" "docker credential helper missing: docker-credential-${helper}; using isolated docker config"
    docker_acceptance_prepare_temp_docker_config
    return 0
  fi

  if ! "docker-credential-${helper}" list >/dev/null 2>&1; then
    "${DOCKER_ACCEPTANCE_LOG_FN}" "docker credential helper unhealthy: docker-credential-${helper}; using isolated docker config"
    docker_acceptance_prepare_temp_docker_config
  fi
}

docker_acceptance_compose() {
  if [[ -n "${TEMP_DOCKER_CONFIG:-}" ]]; then
    env DOCKER_CONFIG="${TEMP_DOCKER_CONFIG}" docker compose "$@"
    return
  fi
  docker compose "$@"
}

docker_acceptance_pull_anvil_image_with_retry() {
  local attempt=1
  local delay=2

  if docker image inspect "${ANVIL_IMAGE}" >/dev/null 2>&1; then
    "${DOCKER_ACCEPTANCE_LOG_FN}" "Found cached anvil image: ${ANVIL_IMAGE}"
    return 0
  fi

  while (( attempt <= ANVIL_IMAGE_PULL_RETRIES )); do
    "${DOCKER_ACCEPTANCE_LOG_FN}" "Pre-pulling anvil image (${attempt}/${ANVIL_IMAGE_PULL_RETRIES}): ${ANVIL_IMAGE}"
    if docker pull "${ANVIL_IMAGE}"; then
      "${DOCKER_ACCEPTANCE_LOG_FN}" "Pulled anvil image successfully: ${ANVIL_IMAGE}"
      return 0
    fi

    if (( attempt == ANVIL_IMAGE_PULL_RETRIES )); then
      break
    fi

    "${DOCKER_ACCEPTANCE_LOG_FN}" "Anvil image pull failed; retrying in ${delay}s"
    sleep "${delay}"
    delay=$((delay * 2))
    attempt=$((attempt + 1))
  done

  echo "failed to pre-pull anvil image after ${ANVIL_IMAGE_PULL_RETRIES} attempts: ${ANVIL_IMAGE}" >&2
  exit 1
}

docker_acceptance_wait_for_http() {
  local label="$1"
  local url="$2"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      "${DOCKER_ACCEPTANCE_LOG_FN}" "Ready: ${label}"
      return 0
    fi
    sleep 2
  done

  echo "timed out waiting for ${label}: ${url}" >&2
  exit 1
}

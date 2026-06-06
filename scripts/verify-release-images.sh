#!/usr/bin/env bash
# 上线前验证：用与生产相同的 Dockerfile 本地构建候选镜像，
# 按用户部署路径（compose 仅 image、nginx 反代 /api）做健康检查与 smoke。
#
# 通过后再执行: git tag v0.x.x && git push --tags
#
# 环境变量:
#   RELEASE_VERIFY_TAG          镜像 tag（默认 release-candidate）
#   RELEASE_VERIFY_SERVER_PORT  宿主机 server 端口（默认 17422）
#   RELEASE_VERIFY_FRONTEND_PORT 宿主机 frontend 端口（默认 18080）
#   ANCHOR_API_TOKEN            验证用 token（默认 release-verify-token）
#   SKIP_BUILD=1                跳过镜像构建（复用已构建候选镜像）
#   SKIP_UNIT=1                 跳过 go vet / go test
#   SKIP_SMOKE=1                跳过 Playwright smoke（仅健康检查）
#   KEEP_RUNNING=1              验证结束后不 down compose
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RELEASE_VERIFY_TAG="${RELEASE_VERIFY_TAG:-release-candidate}"
RELEASE_VERIFY_SERVER_PORT="${RELEASE_VERIFY_SERVER_PORT:-17422}"
RELEASE_VERIFY_FRONTEND_PORT="${RELEASE_VERIFY_FRONTEND_PORT:-18080}"
ANCHOR_API_TOKEN="${ANCHOR_API_TOKEN:-release-verify-token}"
COMPOSE_FILE="docker-compose.release-verify.yml"
SERVER_HEALTH_URL="http://localhost:${RELEASE_VERIFY_SERVER_PORT}/health"
FRONTEND_URL="http://localhost:${RELEASE_VERIFY_FRONTEND_PORT}/"
API_WORKERS_URL="http://localhost:${RELEASE_VERIFY_SERVER_PORT}/workers"

log() { echo "[release-verify] $*"; }
fail() { echo "[release-verify] ERROR: $*" >&2; exit 1; }

detect_targetarch() {
  local arch
  arch="$(uname -m)"
  case "${arch}" in
    x86_64)  echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) fail "unsupported host architecture: ${arch}" ;;
  esac
}

wait_http_ok() {
  local url="$1"
  local label="$2"
  local max_ms="${3:-120000}"
  local start
  start="$(date +%s)"
  while true; do
    if curl -sf --noproxy '*' "${url}" >/dev/null 2>&1; then
      log "${label} ready (${url})"
      return 0
    fi
    if (( ($(date +%s) - start) * 1000 >= max_ms )); then
      fail "${label} not ready within ${max_ms}ms (${url})"
    fi
    sleep 2
  done
}

wait_worker_online() {
  local max_ms="${1:-180000}"
  local start
  start="$(date +%s)"
  while true; do
    local body
    body="$(curl -sf --noproxy '*' \
      -H "Authorization: Bearer ${ANCHOR_API_TOKEN}" \
      "${API_WORKERS_URL}" 2>/dev/null || true)"
    if echo "${body}" | grep -qE '"status"\s*:\s*"(online|busy)"'; then
      log "worker registered and online"
      return 0
    fi
    if (( ($(date +%s) - start) * 1000 >= max_ms )); then
      docker compose -f "${COMPOSE_FILE}" ps >&2 || true
      docker compose -f "${COMPOSE_FILE}" logs --tail=80 worker >&2 || true
      fail "worker did not register within ${max_ms}ms"
    fi
    sleep 3
  done
}

run_unit_gate() {
  if [[ "${SKIP_UNIT:-0}" == "1" ]]; then
    log "SKIP_UNIT=1 — skipping go vet / go test"
    return 0
  fi
  log "unit gate: go vet ./..."
  go vet ./...
  log "unit gate: go test ./..."
  go test ./...
}

build_candidate_images() {
  if [[ "${SKIP_BUILD:-0}" == "1" ]]; then
    log "SKIP_BUILD=1 — reusing existing images anchor-*:${RELEASE_VERIFY_TAG}"
    return 0
  fi

  local targetarch
  targetarch="$(detect_targetarch)"
  log "building linux/${targetarch} binary..."
  make build-linux TARGETARCH="${targetarch}"

  log "building anchor-server:${RELEASE_VERIFY_TAG} (Dockerfile.server, RELEASE_VERSION=local)..."
  docker build -f Dockerfile.server \
    --build-arg RELEASE_VERSION=local \
    -t "anchor-server:${RELEASE_VERIFY_TAG}" .

  log "building anchor-worker:${RELEASE_VERIFY_TAG} (Dockerfile.worker — may take several minutes)..."
  docker build -f Dockerfile.worker \
    --build-arg RELEASE_VERSION=local \
    -t "anchor-worker:${RELEASE_VERIFY_TAG}" .

  log "building anchor-frontend:${RELEASE_VERIFY_TAG} (Dockerfile.frontend)..."
  docker build -f Dockerfile.frontend \
    -t "anchor-frontend:${RELEASE_VERIFY_TAG}" .
}

start_stack() {
  log "starting release-verify stack (ports ${RELEASE_VERIFY_FRONTEND_PORT}/17421 → nginx, ${RELEASE_VERIFY_SERVER_PORT}/17421 → API)..."
  docker compose -f "${COMPOSE_FILE}" down --remove-orphans 2>/dev/null || true
  RELEASE_VERIFY_TAG="${RELEASE_VERIFY_TAG}" \
  RELEASE_VERIFY_SERVER_PORT="${RELEASE_VERIFY_SERVER_PORT}" \
  RELEASE_VERIFY_FRONTEND_PORT="${RELEASE_VERIFY_FRONTEND_PORT}" \
  ANCHOR_API_TOKEN="${ANCHOR_API_TOKEN}" \
    docker compose -f "${COMPOSE_FILE}" up -d
}

run_health_checks() {
  wait_http_ok "${SERVER_HEALTH_URL}" "server health"
  wait_http_ok "${FRONTEND_URL}" "frontend (nginx)"
  wait_worker_online
}

run_api_smoke() {
  log "API smoke: create project via authenticated API..."
  local resp
  resp="$(curl -sf --noproxy '*' \
    -X POST "${SERVER_HEALTH_URL%/health}/projects" \
    -H "Authorization: Bearer ${ANCHOR_API_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"name":"Release Verify Project","org":"CI","description":"pre-tag gate"}')"
  echo "${resp}" | grep -q '"id"' || fail "project create response missing id: ${resp}"
  log "API smoke: project created"
}

run_playwright_smoke() {
  if [[ "${SKIP_SMOKE:-0}" == "1" ]]; then
    log "SKIP_SMOKE=1 — skipping Playwright smoke"
    return 0
  fi
  if [[ ! -d frontend/node_modules ]]; then
    log "installing frontend deps for Playwright..."
    (cd frontend && npm ci)
  fi
  log "ensuring Playwright chromium is installed..."
  (cd frontend && npx playwright install chromium)
  log "Playwright smoke (user path: nginx :${RELEASE_VERIFY_FRONTEND_PORT}, /api proxy)..."
  (
    cd frontend
    RELEASE_VERIFY_FRONTEND_PORT="${RELEASE_VERIFY_FRONTEND_PORT}" \
    RELEASE_VERIFY_SERVER_PORT="${RELEASE_VERIFY_SERVER_PORT}" \
    ANCHOR_API_BASE="http://localhost:${RELEASE_VERIFY_SERVER_PORT}" \
    ANCHOR_API_TOKEN="${ANCHOR_API_TOKEN}" \
      npx playwright test e2e/tests/release-verify-smoke.spec.ts \
        --config=playwright.release-verify.config.ts \
        --project=chromium
  )
}

cleanup() {
  if [[ "${KEEP_RUNNING:-0}" == "1" ]]; then
    log "KEEP_RUNNING=1 — stack left up for manual inspection"
    log "  frontend: ${FRONTEND_URL}"
    log "  API:      http://localhost:${RELEASE_VERIFY_SERVER_PORT}/"
    log "  token:    ${ANCHOR_API_TOKEN}"
    return 0
  fi
  docker compose -f "${COMPOSE_FILE}" down --remove-orphans
  log "stack torn down"
}

main() {
  log "=== Anchor release image verification ==="
  log "tag=${RELEASE_VERIFY_TAG} frontend_port=${RELEASE_VERIFY_FRONTEND_PORT} server_port=${RELEASE_VERIFY_SERVER_PORT}"

  run_unit_gate
  build_candidate_images
  start_stack
  trap cleanup EXIT
  run_health_checks
  run_api_smoke
  run_playwright_smoke

  log "=== PASSED — safe to tag and push ==="
  log "Next: git tag v0.x.x && git push --tags"
}

main "$@"

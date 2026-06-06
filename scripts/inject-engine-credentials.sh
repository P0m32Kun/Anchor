#!/usr/bin/env bash
# 从 functional-test.env 读取 FOFA / Hunter / Quake 密钥并写入 Anchor API
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/functional-test-env.sh
source "${SCRIPT_DIR}/lib/functional-test-env.sh"
load_functional_test_env "${SCRIPT_DIR}/.." || true

API="${ANCHOR_API_BASE:-http://localhost:17421}"
TOKEN="${ANCHOR_API_TOKEN:-test-e2e-token}"
AUTH="Authorization: Bearer ${TOKEN}"

inject_cred() {
  local engine="$1" api_key="$2" extra="${3:-}"
  local body
  if [[ -n "$extra" ]]; then
    body=$(python3 -c "import json,sys; print(json.dumps({'engine':sys.argv[1],'api_key':sys.argv[2],'extra':sys.argv[3]}))" "$engine" "$api_key" "$extra")
  else
    body=$(python3 -c "import json,sys; print(json.dumps({'engine':sys.argv[1],'api_key':sys.argv[2]}))" "$engine" "$api_key")
  fi
  curl -sf --max-time 15 -X POST \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "$body" "${API}/engines/credentials" >/dev/null
  echo "  ${engine}: ok (${api_key:0:4}****)"
}

echo "[inject] Anchor ${API}"
INJECTED=0

if [[ -n "${FOFA_API_KEY:-}" ]]; then
  inject_cred "fofa" "$FOFA_API_KEY" "${FOFA_EMAIL:-${FOFA_EXTRA:-}}"
  INJECTED=$((INJECTED + 1))
fi
if [[ -n "${HUNTER_API_KEY:-}" ]]; then
  inject_cred "hunter" "$HUNTER_API_KEY"
  INJECTED=$((INJECTED + 1))
fi
if [[ -n "${QUAKE_API_KEY:-}" ]]; then
  inject_cred "quake" "$QUAKE_API_KEY"
  INJECTED=$((INJECTED + 1))
fi

if [[ "$INJECTED" -eq 0 ]]; then
  echo "FAIL: 未配置任何引擎密钥。请复制 functional-test.env.example → functional-test.env 并填写 FOFA/Hunter/Quake API key"
  exit 2
fi

echo "[inject] 已注入 ${INJECTED} 个引擎凭证"

#!/usr/bin/env bash
# 内网靶场资产驱动扫描验收（rangefield Redis）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/functional-test-env.sh
source "${SCRIPT_DIR}/lib/functional-test-env.sh"
load_functional_test_env "${SCRIPT_DIR}/.." || true

API="${ANCHOR_API_BASE:-http://localhost:17421}"
TOKEN="${ANCHOR_API_TOKEN:-test-e2e-token}"
AUTH="Authorization: Bearer ${TOKEN}"
REDIS_IP="172.31.0.13"

api() {
  local method="$1" path="$2"
  shift 2
  curl -sf -X "$method" -H "$AUTH" -H "Content-Type: application/json" "$@" "${API}${path}"
}

echo "[internal] 创建项目"
PROJECT=$(api POST /projects -d "{\"name\":\"内网资产驱动-$(date +%s)\",\"organization\":\"FT\",\"purpose\":\"rangefield redis\"}")
PROJECT_ID=$(echo "$PROJECT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "  project_id=$PROJECT_ID"

echo "[internal] scope + 目标 ${REDIS_IP}"
api POST /scope-rules -d "{\"project_id\":\"${PROJECT_ID}\",\"action\":\"exclude\",\"type\":\"ip\",\"value\":\"10.255.255.255\",\"reason\":\"FT placeholder\"}" >/dev/null
api POST "/projects/${PROJECT_ID}/targets" -d "{\"type\":\"ip\",\"value\":\"${REDIS_IP}\"}" >/dev/null

echo "[internal] 启动内网资产驱动扫描（6379）"
SCAN=$(api POST "/projects/${PROJECT_ID}/scan" -d '{
  "mode": "internal",
  "config": {
    "port_range": "6379",
    "enable_subfinder": false,
    "enable_dnsx": false,
    "enable_cdn_filter": false,
    "enable_ffuf": false,
    "enable_katana": false,
    "enable_nuclei": true,
    "enable_httpx": true,
    "enable_nmap_service": true,
    "nuclei_scan_depth": "tags"
  }
}')
RUN_ID=$(echo "$SCAN" | python3 -c "import sys,json; print(json.load(sys.stdin)['run_id'])")
echo "  run_id=$RUN_ID"

STATUS="running"
for i in $(seq 1 60); do
  sleep 10
  BODY=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}")
  STATUS=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))")
  METRICS=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}/metrics" 2>/dev/null || echo '{}')
  WORKS_DONE=$(echo "$METRICS" | python3 -c "import sys,json; m=json.load(sys.stdin); print(m.get('works_done',0))" 2>/dev/null || echo "0")
  echo "  [$i] status=$STATUS works_done=$WORKS_DONE"
  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" || "$STATUS" == "cancelled" ]]; then
    break
  fi
done

WORKS=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}/works")
WORK_TOTAL=$(echo "$WORKS" | python3 -c "import sys,json; w=json.load(sys.stdin); print(w.get('total', len(w.get('items',[]))))")

ASSETS=$(api GET "/projects/${PROJECT_ID}/assets?page_size=50")
ASSET_HIT=$(echo "$ASSETS" | python3 -c "import sys,json; b=json.load(sys.stdin); d=b.get('data',[]); print(any('${REDIS_IP}' in str(a) for a in d))")

FINDINGS=$(api GET "/projects/${PROJECT_ID}/findings?page_size=50")
FINDING_COUNT=$(echo "$FINDINGS" | python3 -c "import sys,json; b=json.load(sys.stdin); d=b.get('data',[]); print(len(d))")

echo "  work_items_total=$WORK_TOTAL asset_has_redis=$ASSET_HIT findings=$FINDING_COUNT status=$STATUS"

FAIL=0
[[ "$WORK_TOTAL" -gt 0 ]] || { echo "FAIL: 无 work items"; FAIL=1; }
[[ "$ASSET_HIT" == "True" ]] || { echo "FAIL: 资产页未见 ${REDIS_IP}"; FAIL=1; }
[[ "$FINDING_COUNT" -gt 0 ]] || { echo "FAIL: 无 findings"; FAIL=1; }
[[ "$STATUS" == "completed" ]] || { echo "FAIL: run 未 completed"; FAIL=1; }

echo ""
echo "[internal] 工具调用校验"
bash "${SCRIPT_DIR}/verify-tool-calls.sh" "$PROJECT_ID" "$RUN_ID" --profile internal || FAIL=1

exit $FAIL

#!/usr/bin/env bash
# 外网资产驱动扫描验收（纯 FOFA mock，不访问真实外网）
# 真实外网验收见: scripts/functional-test-external-real.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/functional-test-env.sh
source "${SCRIPT_DIR}/lib/functional-test-env.sh"
load_functional_test_env "${SCRIPT_DIR}/.." || true

API="${ANCHOR_API_BASE:-http://localhost:17421}"
TOKEN="${ANCHOR_API_TOKEN:-test-e2e-token}"
AUTH="Authorization: Bearer ${TOKEN}"
POLL_SEC="${FT_POLL_SEC:-5}"
POLL_MAX="${FT_POLL_MAX:-36}"   # 默认最多 3 分钟

api() {
  local method="$1" path="$2"
  shift 2
  curl -sf --max-time 15 -X "$method" -H "$AUTH" -H "Content-Type: application/json" "$@" "${API}${path}"
}

echo "[external-mock] 注入 FOFA 凭证（指向 fofa-mock）"
api POST /engines/credentials -d '{"engine":"fofa","api_key":"e2e-mock-key","extra":"e2e@test.local"}' >/dev/null || true

echo "[external-mock] 创建项目"
PROJECT=$(api POST /projects -d "{\"name\":\"外网mock-$(date +%s)\",\"organization\":\"FT\",\"purpose\":\"FOFA mock only\"}")
PROJECT_ID=$(echo "$PROJECT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "  project_id=$PROJECT_ID"

echo "[external-mock] scope 占位 + 目标（仅 company，不扫真实 lixiang.com）"
api POST /scope-rules -d "{\"project_id\":\"${PROJECT_ID}\",\"action\":\"exclude\",\"type\":\"ip\",\"value\":\"10.255.255.255\",\"reason\":\"FT placeholder\"}" >/dev/null
api POST "/projects/${PROJECT_ID}/targets" -d '{"type":"company","value":"北京车励行信息技术有限公司"}' >/dev/null

echo "[external-mock] 启动外网资产驱动扫描（轻量配置，避免 top100 拖死）"
SCAN=$(api POST "/projects/${PROJECT_ID}/scan" -d '{
  "mode": "external",
  "config": {
    "port_range": "80",
    "enable_subfinder": false,
    "enable_dnsx": false,
    "enable_cdn_filter": false,
    "enable_ffuf": false,
    "enable_katana": false,
    "enable_nuclei": false,
    "enable_httpx": false,
    "enable_nmap_service": false,
    "nuclei_scan_depth": "tags"
  }
}')
RUN_ID=$(echo "$SCAN" | python3 -c "import sys,json; print(json.load(sys.stdin)['run_id'])")
echo "  run_id=$RUN_ID"

echo "[external-mock] 轮询（每 ${POLL_SEC}s，最多 $((POLL_MAX * POLL_SEC))s）"
STATUS="running"
MOCK_ASSETS=0
WORK_TOTAL=0
for i in $(seq 1 "$POLL_MAX"); do
  sleep "$POLL_SEC"
  BODY=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}")
  STATUS=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))")
  ASSETS=$(api GET "/projects/${PROJECT_ID}/assets?page_size=50" 2>/dev/null || echo '{"data":[]}')
  MOCK_ASSETS=$(echo "$ASSETS" | python3 -c "import sys,json; b=json.load(sys.stdin); d=b.get('data',[]); print(sum(1 for a in d if 'testcorp' in str(a.get('value',''))))")
  WORKS=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}/works" 2>/dev/null || echo '{"items":[],"total":0}')
  WORK_TOTAL=$(echo "$WORKS" | python3 -c "import sys,json; w=json.load(sys.stdin); print(w.get('total', len(w.get('items',[]))))")
  echo "  [$i] status=$STATUS mock_assets=$MOCK_ASSETS works=$WORK_TOTAL"
  if [[ "$MOCK_ASSETS" -ge 1 ]]; then
    echo "  OK: FOFA mock 种子已入库"
    break
  fi
  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" || "$STATUS" == "cancelled" ]]; then
    break
  fi
done

echo "[external-mock] 资产清单（testcorp 子域）"
echo "$ASSETS" | python3 -c "
import sys,json
b=json.load(sys.stdin)
for a in b.get('data',[]):
    if 'testcorp' in str(a.get('value','')):
        print(f\"  - {a.get('type')}: {a.get('value')}\")
"

FINDINGS=$(api GET "/projects/${PROJECT_ID}/findings?page_size=20" 2>/dev/null || echo '{"data":[]}')
FINDING_COUNT=$(echo "$FINDINGS" | python3 -c "import sys,json; b=json.load(sys.stdin); d=b.get('data',[]); print(len(d))")
echo "  findings_count=$FINDING_COUNT status=$STATUS"

FAIL=0
[[ "$MOCK_ASSETS" -ge 1 ]] || { echo "FAIL: assets 中未见 FOFA mock 的 testcorp 子域"; FAIL=1; }
if [[ "$WORK_TOTAL" -eq 0 ]]; then
  echo "  WARN: 轻量配置下可能无 work items（种子入库即 mock 验收通过）"
fi

echo ""
echo "[external-mock] 工具调用校验"
bash "${SCRIPT_DIR}/verify-tool-calls.sh" "$PROJECT_ID" "$RUN_ID" --profile external-mock || FAIL=1

exit $FAIL

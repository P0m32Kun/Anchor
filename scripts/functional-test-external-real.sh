#!/usr/bin/env bash
# 外网资产驱动扫描 — 真实目标（访问公网 + 真实 FOFA API）
#
# 与 functional-test-external-asset-driven.sh（FOFA mock）完全独立，勿混用。
#
# 前置：
#   - anchor-server / worker 已启动，且 server 未将 FOFA_BASE_URL 指向 mock
#   - 本机可访问公网（DNS、fofa.info）
#
# 用法：
#   cp functional-test.env.example functional-test.env   # 填写 FOFA/Hunter/Quake 密钥
#   FT_REAL_EXTERNAL=1 bash scripts/functional-test-external-real.sh
#
# 也可单独注入凭证: bash scripts/inject-engine-credentials.sh
#
# 可选环境变量：
#   ANCHOR_API_BASE / ANCHOR_API_TOKEN
#   FT_DOMAIN          默认 lixiang.com
#   FT_COMPANY         默认 北京车励行信息技术有限公司
#   FT_PORT_RANGE      默认 80,443（比 top100 快；全量可设 top100）
#   FT_POLL_SEC        默认 30
#   FT_POLL_MAX        默认 30（即最多约 15 分钟）
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/functional-test-env.sh
source "${SCRIPT_DIR}/lib/functional-test-env.sh"
load_functional_test_env "${SCRIPT_DIR}/.." || true

API="${ANCHOR_API_BASE:-http://localhost:17421}"
TOKEN="${ANCHOR_API_TOKEN:-test-e2e-token}"
AUTH="Authorization: Bearer ${TOKEN}"
DOMAIN="${FT_DOMAIN:-lixiang.com}"
COMPANY="${FT_COMPANY:-北京车励行信息技术有限公司}"
PORT_RANGE="${FT_PORT_RANGE:-80,443}"
POLL_SEC="${FT_POLL_SEC:-30}"
POLL_MAX="${FT_POLL_MAX:-30}"

if [[ "${FT_REAL_EXTERNAL:-}" != "1" ]]; then
  echo "拒绝执行：此脚本会访问真实外网与 FOFA API。"
  echo "确认后请设置: FT_REAL_EXTERNAL=1"
  exit 2
fi

if [[ -z "${FOFA_API_KEY:-}" ]]; then
  echo "FAIL: 需要 FOFA_API_KEY。请填写 functional-test.env 或 export FOFA_API_KEY"
  exit 2
fi

if [[ -n "${FOFA_BASE_URL:-}" && "${FOFA_BASE_URL}" == *mock* ]]; then
  echo "WARN: 本 shell 设置了 FOFA_BASE_URL=${FOFA_BASE_URL}，请确认 server 进程未指向 mock"
fi

api() {
  local method="$1" path="$2"
  shift 2
  curl -sf --max-time 30 -X "$method" -H "$AUTH" -H "Content-Type: application/json" "$@" "${API}${path}"
}

echo "[external-real] 健康检查"
api GET /health >/dev/null
echo "  API OK: ${API}"

echo "[external-real] 注入搜索引擎凭证（functional-test.env）"
bash "${SCRIPT_DIR}/inject-engine-credentials.sh"

echo "[external-real] 创建项目"
PROJECT=$(api POST /projects -d "{\"name\":\"外网真实-$(date +%s)\",\"organization\":\"FT\",\"purpose\":\"${DOMAIN}+${COMPANY}\"}")
PROJECT_ID=$(echo "$PROJECT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "  project_id=$PROJECT_ID"

echo "[external-real] scope 占位 + 目标: ${DOMAIN} + ${COMPANY}"
api POST /scope-rules -d "{\"project_id\":\"${PROJECT_ID}\",\"action\":\"exclude\",\"type\":\"ip\",\"value\":\"10.255.255.255\",\"reason\":\"FT placeholder\"}" >/dev/null
api POST "/projects/${PROJECT_ID}/targets" -d "{\"type\":\"domain\",\"value\":\"${DOMAIN}\"}" >/dev/null
api POST "/projects/${PROJECT_ID}/targets" -d "{\"type\":\"company\",\"value\":\"${COMPANY}\"}" >/dev/null

TARGETS=$(api GET "/projects/${PROJECT_ID}/targets")
TARGET_COUNT=$(echo "$TARGETS" | python3 -c "import sys,json; b=json.load(sys.stdin); d=b.get('data',b) if isinstance(b,dict) else b; print(len(d))")
echo "  targets: ${TARGET_COUNT} 条"

echo "[external-real] 启动外网资产驱动扫描（port_range=${PORT_RANGE}）"
SCAN=$(api POST "/projects/${PROJECT_ID}/scan" -d "{
  \"mode\": \"external\",
  \"config\": {
    \"port_range\": \"${PORT_RANGE}\",
    \"enable_ffuf\": false,
    \"enable_katana\": false,
    \"nuclei_scan_depth\": \"tags\"
  }
}")
RUN_ID=$(echo "$SCAN" | python3 -c "import sys,json; print(json.load(sys.stdin)['run_id'])")
echo "  run_id=$RUN_ID"

echo "[external-real] 轮询（每 ${POLL_SEC}s，最多 $((POLL_MAX * POLL_SEC))s）"
STATUS="running"
WORK_TOTAL=0
ASSET_COUNT=0
for i in $(seq 1 "$POLL_MAX"); do
  sleep "$POLL_SEC"
  BODY=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}")
  STATUS=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))")
  ENGINE_STATE=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('engine_state',''))")
  METRICS=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}/metrics" 2>/dev/null || echo '{}')
  WORKS_DONE=$(echo "$METRICS" | python3 -c "import sys,json; m=json.load(sys.stdin); print(m.get('works_done', m.get('works_completed', '?')))" 2>/dev/null || echo "?")
  ASSETS=$(api GET "/projects/${PROJECT_ID}/assets?page_size=100" 2>/dev/null || echo '{"data":[]}')
  ASSET_COUNT=$(echo "$ASSETS" | python3 -c "import sys,json; b=json.load(sys.stdin); d=b.get('data',[]); print(len(d))")
  WORKS=$(api GET "/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}/works" 2>/dev/null || echo '{"items":[],"total":0}')
  WORK_TOTAL=$(echo "$WORKS" | python3 -c "import sys,json; w=json.load(sys.stdin); print(w.get('total', len(w.get('items',[]))))")
  echo "  [$i] status=$STATUS engine=$ENGINE_STATE assets=$ASSET_COUNT works=$WORK_TOTAL works_done=$WORKS_DONE"
  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" || "$STATUS" == "cancelled" ]]; then
    break
  fi
done

echo "[external-real] 资产抽样（含 ${DOMAIN} 或公司展开）"
echo "$ASSETS" | python3 -c "
import sys, json, os
domain = os.environ.get('FT_DOMAIN', 'lixiang.com')
b = json.load(sys.stdin)
shown = 0
for a in b.get('data', []):
    v = str(a.get('value', ''))
    if domain in v or shown < 8:
        print(f\"  - {a.get('type')}: {v}\")
        shown += 1
    if shown >= 12:
        break
" FT_DOMAIN="$DOMAIN"

FINDINGS=$(api GET "/projects/${PROJECT_ID}/findings?page_size=20" 2>/dev/null || echo '{"data":[]}')
FINDING_COUNT=$(echo "$FINDINGS" | python3 -c "import sys,json; b=json.load(sys.stdin); d=b.get('data',[]); print(len(d))")
echo "  findings_count=$FINDING_COUNT"

DOMAIN_ASSETS=$(echo "$ASSETS" | python3 -c "
import sys,json,os
domain=os.environ['FT_DOMAIN']
b=json.load(sys.stdin)
print(sum(1 for a in b.get('data',[]) if domain in str(a.get('value',''))))
" FT_DOMAIN="$DOMAIN")

FAIL=0
[[ "$WORK_TOTAL" -gt 0 ]] || { echo "FAIL: 无 work items"; FAIL=1; }
[[ "$ASSET_COUNT" -gt 0 ]] || { echo "FAIL: 无资产入库"; FAIL=1; }
[[ "$DOMAIN_ASSETS" -ge 1 ]] || echo "  WARN: assets 中未见 ${DOMAIN} 相关条目（公司 FOFA 展开可能为主）"
[[ "$STATUS" == "completed" ]] || { echo "WARN: run 未 completed (status=$STATUS)，可增大 FT_POLL_MAX 或缩小 FT_PORT_RANGE"; }

echo ""
echo "[external-real] 工具调用校验"
VERIFY_FAIL=0
bash "${SCRIPT_DIR}/verify-tool-calls.sh" "$PROJECT_ID" "$RUN_ID" --profile external-real || VERIFY_FAIL=1

FAIL=$((FAIL | VERIFY_FAIL))
exit $FAIL

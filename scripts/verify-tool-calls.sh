#!/usr/bin/env bash
# 校验 pipeline run 的工具调用日志：是否调用、成败、是否有产出摘要
#
# 用法:
#   bash scripts/verify-tool-calls.sh <project_id> <run_id> [--profile external-real|external-mock|internal]
#
# 环境变量（可覆盖 profile 默认）:
#   FT_EXPECT_TOOLS=naabu,httpx,dnsx   必须至少 1 次 completed
#   FT_OPTIONAL_TOOLS=nuclei,subfinder   缺失仅 WARN
#   ANCHOR_API_BASE / ANCHOR_API_TOKEN
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/functional-test-env.sh
source "${SCRIPT_DIR}/lib/functional-test-env.sh"
load_functional_test_env "${SCRIPT_DIR}/.." || true

API="${ANCHOR_API_BASE:-http://localhost:17421}"
TOKEN="${ANCHOR_API_TOKEN:-test-e2e-token}"
AUTH="Authorization: Bearer ${TOKEN}"

PROJECT_ID="${1:-}"
RUN_ID="${2:-}"
PROFILE="${FT_TOOL_PROFILE:-}"

shift 2 2>/dev/null || true
while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="$2"
      shift 2
      ;;
    *)
      echo "未知参数: $1" >&2
      exit 2
      ;;
  esac
done

if [[ -z "$PROJECT_ID" || -z "$RUN_ID" ]]; then
  echo "用法: $0 <project_id> <run_id> [--profile external-real|external-mock|internal]" >&2
  exit 2
fi

case "$PROFILE" in
  external-real)
    FT_EXPECT_TOOLS="${FT_EXPECT_TOOLS:-naabu,dnsx,httpx,subfinder}"
    FT_OPTIONAL_TOOLS="${FT_OPTIONAL_TOOLS:-nuclei,nmap_service,cdncheck}"
    ;;
  external-mock)
    FT_EXPECT_TOOLS="${FT_EXPECT_TOOLS:-naabu}"
    FT_OPTIONAL_TOOLS="${FT_OPTIONAL_TOOLS:-httpx,dnsx}"
    ;;
  internal)
    FT_EXPECT_TOOLS="${FT_EXPECT_TOOLS:-naabu,httpx,nmap_service}"
    FT_OPTIONAL_TOOLS="${FT_OPTIONAL_TOOLS:-nuclei,dnsx}"
    ;;
  "")
    FT_EXPECT_TOOLS="${FT_EXPECT_TOOLS:-}"
    FT_OPTIONAL_TOOLS="${FT_OPTIONAL_TOOLS:-}"
    ;;
  *)
    echo "未知 profile: $PROFILE" >&2
    exit 2
    ;;
esac

BODY=$(curl -sf --max-time 30 -H "$AUTH" "${API}/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}/tool-calls")
RUN_STATUS=$(curl -sf --max-time 15 -H "$AUTH" "${API}/projects/${PROJECT_ID}/pipeline/runs/${RUN_ID}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))")

export BODY RUN_STATUS FT_EXPECT_TOOLS FT_OPTIONAL_TOOLS RUN_ID="$RUN_ID"
python3 <<'PY'
import json, os, sys

body = json.loads(os.environ["BODY"])
items = body.get("items") or []
run_status = os.environ.get("RUN_STATUS", "")
expect = [t.strip() for t in os.environ.get("FT_EXPECT_TOOLS", "").split(",") if t.strip()]
optional = [t.strip() for t in os.environ.get("FT_OPTIONAL_TOOLS", "").split(",") if t.strip()]

by_tool: dict[str, dict] = {}
for it in items:
    tool = it.get("tool") or "?"
    st = it.get("status") or "?"
    bucket = by_tool.setdefault(tool, {"completed": 0, "failed": 0, "running": 0, "samples": []})
    if st in bucket:
        bucket[st] += 1
    else:
        bucket[st] = bucket.get(st, 0) + 1
    if len(bucket["samples"]) < 2:
        bucket["samples"].append(it)

def ok_output(it) -> bool:
    if it.get("status") != "completed":
        return False
    tool = it.get("tool") or ""
    ec = it.get("exit_code")
    summary = (it.get("output_summary") or "").strip()
    err = (it.get("error_message") or "").strip()
    if err:
        return False
    if summary:
        return True
    # cdncheck 仅对命中 CDN/cloud/WAF 的 IP 输出 JSONL；空 stdout = 非 CDN，见 internal/cdn/parse.go
    if tool == "cdncheck":
        return ec is None or ec == 0
    return ec is not None and ec == 0

def productive_completed(tool: str) -> int:
    return sum(1 for it in items if it.get("tool") == tool and ok_output(it))

print(f"[tool-calls] run={os.environ.get('RUN_ID','')} status={run_status} total={len(items)}")
print(f"{'tool':<16} {'done':>6} {'fail':>6} {'run':>6} {'产出ok':>8}")
print("-" * 48)
for tool in sorted(by_tool):
    b = by_tool[tool]
    prod = productive_completed(tool)
    print(f"{tool:<16} {b.get('completed',0):6} {b.get('failed',0):6} {b.get('running',0):6} {prod:8}")

running_total = sum(b.get("running", 0) for b in by_tool.values())
if running_total and run_status == "completed":
    print(f"  WARN: run 已 completed 但仍有 {running_total} 条 running 工具日志")

print("\n[tool-calls] 抽样（completed + 有 output_summary）")
shown = 0
for it in items:
    if it.get("status") != "completed":
        continue
    summary = (it.get("output_summary") or "").strip()
    if not summary:
        continue
    tool = it.get("tool", "?")
    action = it.get("action", "?")
    preview = summary.replace("\n", " ")[:120]
    print(f"  - {tool}/{action}: {preview}")
    shown += 1
    if shown >= 5:
        break
if shown == 0:
    print("  (无带 output_summary 的 completed 记录；检查 exit_code=0 的 completed)")

failures = [it for it in items if it.get("status") == "failed"][:5]
if failures:
    print("\n[tool-calls] 失败抽样（最多 5 条）")
    for it in failures:
        tool = it.get("tool", "?")
        action = it.get("action", "?")
        err = (it.get("error_message") or "no message")[:160]
        print(f"  - {tool}/{action}: {err}")

exit_code = 0
if len(items) == 0:
    print("\nFAIL: 无任何工具调用日志（tool_call_logs 为空）")
    exit_code = 1

for tool in expect:
    b = by_tool.get(tool, {})
    done = b.get("completed", 0)
    prod = productive_completed(tool)
    if done == 0:
        print(f"\nFAIL: 期望工具 {tool} 无 completed 调用")
        exit_code = 1
    elif prod == 0:
        print(f"\nFAIL: 期望工具 {tool} 有 completed 但无正常产出（无 output 且 exit_code!=0）")
        exit_code = 1
    else:
        print(f"\nOK: 期望工具 {tool} completed={done} 产出正常={prod}")

for tool in optional:
    if tool not in by_tool:
        print(f"\nWARN: 可选工具 {tool} 未调用")
    elif by_tool[tool].get("completed", 0) == 0:
        print(f"\nWARN: 可选工具 {tool} 无 completed")

if not expect and len(items) > 0:
    print("\nOK: 未配置 FT_EXPECT_TOOLS，仅输出汇总")

sys.exit(exit_code)
PY

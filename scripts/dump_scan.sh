#!/bin/bash
# dump_scan.sh {runId} — 导出扫描的完整工具调用记录
# Usage: bash scripts/dump_scan.sh id-1778703063522477009-9

set -euo pipefail
RUN_ID="${1:?Usage: $0 <runId>}"
API="${ANCHOR_API_BASE:-http://localhost:17421}"
TOKEN="${ANCHOR_API_TOKEN:?请设置 ANCHOR_API_TOKEN}"
AUTH="Authorization: Bearer $TOKEN"
OUTDIR="scan-dump-${RUN_ID: -8}"
mkdir -p "$OUTDIR"

echo "=== 1. 扫描任务与命令 ==="
curl -s -H "$AUTH" "$API/runs/$RUN_ID/tasks" | python3 -c "
import json, sys
for i, t in enumerate(json.load(sys.stdin), 1):
    s = t.get('started_at','')[:19]; f = t.get('finished_at','')[:19] or '...'
    print(f'{i:2d}. [{t[\"tool\"]:10s}] exit={t.get(\"exit_code\",\"?\")}  {s} -> {f}')
    print(f'    {t[\"command_template\"]}'); print()
" | tee "$OUTDIR/01_tasks.txt"

echo ""
echo "=== 2. 核心工具 stdout (nmap/naabu/httpx/nuclei) ==="
curl -s -H "$AUTH" "$API/runs/$RUN_ID/tasks" | python3 -c "
import json, sys, urllib.request, os
tasks = json.load(sys.stdin)
for t in tasks:
    if t['tool'] not in ('nmap','naabu','httpx','nuclei','ffuf'):
        continue
    url = '$API/tasks/' + t['id'] + '/artifacts'
    req = urllib.request.Request(url, headers={'Authorization': 'Bearer $TOKEN'})
    try:
        with urllib.request.urlopen(req, timeout=5) as r:
            arts = json.load(r)
    except Exception as e:
        print(f'--- {t[\"tool\"]} ({t[\"id\"]}) ERROR: {e}'); continue
    for a in arts:
        if a['type'] != 'stdout': continue
        path = a['path']
        print(f'--- {t[\"tool\"]} ({t[\"id\"]}) ---')
        print(f'    file: {path}')
        # Try reading via docker exec
        out = os.popen(f'docker exec anchor-worker cat {path} 2>/dev/null').read()
        if not out:
            out = os.popen(f'docker exec anchor-server cat {path} 2>/dev/null').read()
        if out:
            lines = out.strip().split('\n')
            print(f'    lines: {len(lines)}')
            for l in lines[:15]: print(f'    | {l[:150]}')
            if len(lines) > 15: print(f'    ... truncated ({len(lines)} total)')
        else:
            print('    (binary or unreadable)')
        print()
" 2>&1 | tee "$OUTDIR/02_stdout.txt"

echo ""
echo "=== 3. nuclei evidence (JSON output) ==="
curl -s -H "$AUTH" "$API/runs/$RUN_ID/tasks" | python3 -c "
import json, sys, urllib.request
tasks = json.load(sys.stdin)
for t in tasks:
    if t['tool'] != 'nuclei': continue
    url = '$API/tasks/' + t['id'] + '/artifacts'
    req = urllib.request.Request(url, headers={'Authorization': 'Bearer $TOKEN'})
    try:
        with urllib.request.urlopen(req, timeout=5) as r:
            arts = json.load(r)
    except: continue
    for a in arts:
        if 'evidence' in a.get('path','') and 'nuclei' in a.get('path',''):
            path = a['path']
            out = __import__('os').popen(f'docker exec anchor-server cat {path} 2>/dev/null').read()
            if out:
                try:
                    d = json.loads(out)
                    print(f'  [{d.get(\"template-id\",\"?\")}] {d.get(\"host\",\"?\")}:{d.get(\"port\",\"?\")}')
                except:
                    print(f'  {out[:120]}')
" 2>&1 | tee "$OUTDIR/03_nuclei_evidence.txt"

echo ""
echo "已保存到: $OUTDIR/"
echo "  ls $OUTDIR/"
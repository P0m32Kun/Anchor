#!/bin/bash
# seed-anchor.sh — 向 Anchor SQLite 注入靶场 WebEndpoint，用于直接测试 Web 初筛工作流
# 用法: PROJECT_ID=<id> [ANCHOR_DB=~/.anchor/anchor.db] ./seed-anchor.sh

set -euo pipefail

DB_PATH="${ANCHOR_DB:-$HOME/.anchor/anchor.db}"
PROJECT_ID="${PROJECT_ID:-}"
ANCHOR_API="${ANCHOR_API:-http://localhost:8080}"

if [ -z "$PROJECT_ID" ]; then
    echo "错误: 请设置 PROJECT_ID 环境变量"
    echo "  示例: PROJECT_ID=proj_xxx ./scripts/seed-anchor.sh"
    exit 1
fi

if [ ! -f "$DB_PATH" ]; then
    echo "错误: 找不到 Anchor 数据库: $DB_PATH"
    echo "  请确认 Anchor 已运行过至少一次，或设置 ANCHOR_DB 环境变量"
    exit 1
fi

# --- 生成 UUID ---
gen_uuid() {
    if command -v uuidgen &>/dev/null; then
        uuidgen
    elif command -v python3 &>/dev/null; then
        python3 -c "import uuid; print(uuid.uuid4())"
    else
        # fallback: 使用 /dev/urandom
        xxd -l 16 -p /dev/urandom | sed 's/\(........\)\(....\)\(....\)\(....\)\(............\)/\1-\2-\3-\4-\5/'
    fi
}

# --- 靶场配置 ---
declare -a TARGETS=(
    "nginx|http://127.0.0.1:18080|nginx|nginx|Welcome to nginx"
    "tomcat|http://127.0.0.1:18081|tomcat|tomcat|Apache Tomcat"
    "grafana|http://127.0.0.1:18082|grafana|grafana|Grafana"
)

echo "[+] 数据库: $DB_PATH"
echo "[+] 项目ID: $PROJECT_ID"
echo "[+] 注入 ${#TARGETS[@]} 个 WebEndpoint..."

for entry in "${TARGETS[@]}"; do
    IFS='|' read -r name url asset_type tech webserver title <<< "$entry"

    ASSET_ID=$(gen_uuid)
    EP_ID=$(gen_uuid)

    # 提取 scheme, host, port, path
    scheme=$(echo "$url" | sed -E 's|^(https?)://.*|\1|')
    hostport=$(echo "$url" | sed -E 's|^https?://([^/]+).*|\1|')
    path=$(echo "$url" | sed -E 's|^https?://[^/]+(/.*)?$|\1|')
    path="${path:-/}"

    host=$(echo "$hostport" | cut -d: -f1)
    port=$(echo "$hostport" | cut -d: -f2)
    port="${port:-80}"

    # normalized_value 对 URL 类型就是 URL 本身
    normalized="$url"

    echo "  → $name: $url (asset=$ASSET_ID, ep=$EP_ID)"

    # 插入 asset
    sqlite3 "$DB_PATH" <<EOF
INSERT OR IGNORE INTO assets (
    id, project_id, type, value, normalized_value,
    source_tools, first_seen, last_seen
) VALUES (
    '$ASSET_ID', '$PROJECT_ID', 'url', '$url', '$normalized',
    'rangefield-seed', datetime('now'), datetime('now')
);
EOF

    # 插入 web_endpoint
    sqlite3 "$DB_PATH" <<EOF
INSERT OR IGNORE INTO web_endpoints (
    id, project_id, asset_id, url, scheme, host, port, path,
    status_code, title, technologies, webserver, source_tool, created_at
) VALUES (
    '$EP_ID', '$PROJECT_ID', '$ASSET_ID', '$url', '$scheme', '$host', $port, '$path',
    200, '$title', '$tech', '$webserver', 'rangefield-seed', datetime('now')
);
EOF

done

# --- 同时插入非 Web 资产（redis, mysql）用于端口扫描测试 ---
declare -a NET_TARGETS=(
    "redis|127.0.0.1|ip|redis|16379"
    "mysql|127.0.0.1|ip|mysql|13306"
)

for entry in "${NET_TARGETS[@]}"; do
    IFS='|' read -r name ip asset_type svc port <<< "$entry"

    ASSET_ID=$(gen_uuid)
    PORT_ID=$(gen_uuid)

    echo "  → $name: $ip:$port (asset=$ASSET_ID)"

    sqlite3 "$DB_PATH" <<EOF
INSERT OR IGNORE INTO assets (
    id, project_id, type, value, normalized_value,
    source_tools, first_seen, last_seen
) VALUES (
    '$ASSET_ID', '$PROJECT_ID', 'ip', '$ip', '$ip',
    'rangefield-seed', datetime('now'), datetime('now')
);
EOF

    sqlite3 "$DB_PATH" <<EOF
INSERT OR IGNORE INTO ports (
    id, asset_id, port, protocol, state, source_tool, created_at
) VALUES (
    '$PORT_ID', '$ASSET_ID', $port, 'tcp', 'open',
    'rangefield-seed', datetime('now')
);
EOF

done

echo ""
echo "[✓] 注入完成"
echo ""
echo "现在可以触发 Web 初筛工作流:"
echo "  curl -X POST $ANCHOR_API/projects/$PROJECT_ID/workflows/web-screening"
echo ""
echo "查看 Finding:"
echo "  curl $ANCHOR_API/projects/$PROJECT_ID/findings"

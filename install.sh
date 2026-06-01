#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()  { echo -e "${CYAN}$*${NC}"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}⚠${NC} $*"; }
err()   { echo -e "${RED}✗${NC} $*"; }

# 全局变量
MODE=""
PORT="17421"
TOKEN=""
CORE_URL=""
COMPOSE_CMD=""

check_docker() {
  info "⚓ Anchor 安装向导"
  info "───────────────────────"

  if ! command -v docker &>/dev/null; then
    err "未检测到 Docker"
    echo "请安装 Docker Desktop: https://docs.docker.com/get-docker/"
    exit 1
  fi

  local docker_version
  docker_version=$(docker --version 2>/dev/null | head -1)
  ok "Docker 已安装 ($docker_version)"

  if docker compose version &>/dev/null; then
    COMPOSE_CMD="docker compose"
  elif command -v docker-compose &>/dev/null; then
    COMPOSE_CMD="docker-compose"
  else
    err "未检测到 docker compose"
    echo "请升级 Docker 或安装 docker-compose-plugin"
    exit 1
  fi

  # 检测 Docker daemon 是否运行
  if ! docker info &>/dev/null; then
    err "Docker daemon 未运行"
    echo "请启动 Docker Desktop，或运行: sudo systemctl start docker"
    exit 1
  fi
}

select_mode() {
  echo ""
  echo "请选择部署模式:"
  echo "  1) Server Only    — 仅 API 服务（适合 VPS 部署）"
  echo "  2) Worker Only    — 连接远程 Server 的扫描节点"
  echo "  3) Server+Worker  — 完整功能（本地开发/测试）"
  echo ""

  while true; do
    read -rp "请输入选项 [1-3]: " choice
    case $choice in
      1) MODE="server"; break ;;
      2) MODE="worker"; break ;;
      3) MODE="server_worker"; break ;;
      *) warn "无效选项，请输入 1、2 或 3" ;;
    esac
  done
}

collect_config() {
  echo ""
  case $MODE in
    server)
      info "── Server 配置 ──"
      read -rp "Server 端口 [17421]: " input_port
      PORT="${input_port:-17421}"
      read -rp "API Token（留空自动生成）: " input_token
      if [ -z "$input_token" ]; then
        TOKEN=$(openssl rand -hex 16 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 32)
        ok "自动生成 Token: $TOKEN"
      else
        TOKEN="$input_token"
      fi
      ;;
    worker)
      info "── Worker 配置 ──"
      while true; do
        read -rp "Core Server URL（必填，如 http://192.168.1.100:17421）: " input_url
        if [ -n "$input_url" ]; then
          CORE_URL="$input_url"
          break
        fi
        warn "Core Server URL 不能为空"
      done
      read -rp "API Token（必填）: " input_token
      while [ -z "$input_token" ]; do
        warn "API Token 不能为空"
        read -rp "API Token: " input_token
      done
      TOKEN="$input_token"
      ;;
    server_worker)
      info "── Server+Worker 配置 ──"
      read -rp "Server 端口 [17421]: " input_port
      PORT="${input_port:-17421}"
      read -rp "API Token（留空自动生成）: " input_token
      if [ -z "$input_token" ]; then
        TOKEN=$(openssl rand -hex 16 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 32)
        ok "自动生成 Token: $TOKEN"
      else
        TOKEN="$input_token"
      fi
      ;;
  esac
}

build_images() {
  echo ""
  info "正在构建镜像（首次约需 5-10 分钟）..."

  case $MODE in
    server)
      if docker image inspect anchor-server-base:latest &>/dev/null; then
        ok "Server base 镜像已存在，跳过构建"
      else
        info "构建 Server base 镜像..."
        make setup-server-base
      fi
      info "构建 Server 运行镜像..."
      make build-server
      ;;
    worker)
      if docker image inspect anchor-worker-base:latest &>/dev/null; then
        ok "Worker base 镜像已存在，跳过构建"
      else
        info "构建 Worker base 镜像..."
        make setup-worker-base
      fi
      info "构建 Worker 运行镜像..."
      make build-worker
      ;;
    server_worker)
      if docker image inspect anchor-server-base:latest &>/dev/null; then
        ok "Server base 镜像已存在，跳过构建"
      else
        info "构建 Server base 镜像..."
        make setup-server-base
      fi
      if docker image inspect anchor-worker-base:latest &>/dev/null; then
        ok "Worker base 镜像已存在，跳过构建"
      else
        info "构建 Worker base 镜像..."
        make setup-worker-base
      fi
      info "构建运行镜像..."
      make build-server
      make build-worker
      ;;
  esac

  ok "镜像构建完成"
}

save_env() {
  cat > "$SCRIPT_DIR/.env" <<EOF
ANCHOR_API_TOKEN=$TOKEN
ANCHOR_PORT=$PORT
ANCHOR_MODE=$MODE
EOF
}

start_containers() {
  echo ""
  info "正在启动容器..."
  save_env

  export ANCHOR_API_TOKEN="$TOKEN"
  export ANCHOR_PORT="$PORT"

  case $MODE in
    server)
      $COMPOSE_CMD -f docker-compose.server.yml up -d
      ;;
    worker)
      export ANCHOR_CORE_URL="$CORE_URL"
      export ANCHOR_WORKER_HOST="${ANCHOR_WORKER_HOST:-host.docker.internal}"
      $COMPOSE_CMD -f docker-compose.worker.yml up -d
      ;;
    server_worker)
      $COMPOSE_CMD -f docker-compose.yml up -d
      ;;
  esac
}

wait_healthy() {
  info "等待服务就绪..."
  local max_wait=60
  local elapsed=0
  local interval=2

  case $MODE in
    server|server_worker)
      while [ $elapsed -lt $max_wait ]; do
        if curl -sf "http://localhost:${PORT}/health" &>/dev/null; then
          ok "Server 已就绪"
          return 0
        fi
        sleep $interval
        elapsed=$((elapsed + interval))
        printf "."
      done
      echo ""
      warn "Server 健康检查超时（${max_wait}s），请手动检查: make logs"
      return 1
      ;;
    worker)
      while [ $elapsed -lt $max_wait ]; do
        if docker ps --filter "name=anchor-worker" --filter "status=running" | grep -q anchor-worker; then
          ok "Worker 容器已运行"
          return 0
        fi
        sleep $interval
        elapsed=$((elapsed + interval))
        printf "."
      done
      echo ""
      warn "Worker 启动超时（${max_wait}s），请手动检查: make logs-worker"
      return 1
      ;;
  esac
}

print_result() {
  echo ""
  ok "Anchor 已启动"
  echo ""

  case $MODE in
    server)
      echo "  地址: http://localhost:${PORT}"
      echo "  Token: $TOKEN"
      ;;
    worker)
      echo "  Worker 已连接到: $CORE_URL"
      ;;
    server_worker)
      echo "  地址: http://localhost:${PORT}"
      echo "  Token: $TOKEN"
      ;;
  esac

  echo ""
  echo "  管理命令:"
  echo "    make status    — 查看状态"
  echo "    make logs      — 查看日志"
  echo "    make down      — 停止服务"
  echo ""
}

detect_desktop() {
  if [ "$(uname)" = "Darwin" ]; then
    return 0
  fi
  if [ -n "${DISPLAY:-}" ] || [ -n "${WAYLAND_DISPLAY:-}" ]; then
    return 0
  fi
  return 1
}

maybe_open_app() {
  if detect_desktop; then
    echo ""
    read -rp "检测到桌面环境，是否打开 Anchor 桌面应用？ [Y/n]: " open_choice
    if [[ ! "$open_choice" =~ ^[Nn]$ ]]; then
      local auto_connect_file="/tmp/anchor-auto-connect.json"
      cat > "$auto_connect_file" <<EOF
{"api_base":"http://localhost:${PORT}","api_token":"${TOKEN}"}
EOF

      if [ "$(uname)" = "Darwin" ]; then
        if [ -d "$SCRIPT_DIR/src-tauri/target/release/bundle/macos/Anchor.app" ]; then
          open "$SCRIPT_DIR/src-tauri/target/release/bundle/macos/Anchor.app"
        elif [ -d "/Applications/Anchor.app" ]; then
          open /Applications/Anchor.app
        else
          warn "未找到桌面应用，请先运行 make tauri-build"
          echo "  或手动访问: http://localhost:${PORT}"
        fi
      else
        if [ -x "$SCRIPT_DIR/src-tauri/target/release/anchor" ]; then
          "$SCRIPT_DIR/src-tauri/target/release/anchor" &
        else
          warn "未找到桌面应用，请先运行 make tauri-build"
          echo "  或手动访问: http://localhost:${PORT}"
        fi
      fi
    fi
  else
    echo ""
    info "未检测到桌面环境。"
    echo "  请在本地桌面电脑打开 Anchor App，连接到此 Server:"
    echo "  地址: http://<this-server-ip>:${PORT}"
    echo "  Token: $TOKEN"
  fi
}

check_existing() {
  if [ -f "$SCRIPT_DIR/.env" ]; then
    local existing_mode
    existing_mode=$(grep "^ANCHOR_MODE=" "$SCRIPT_DIR/.env" 2>/dev/null | cut -d= -f2 || true)
    if [ -n "$existing_mode" ]; then
      echo ""
      warn "检测到已有部署配置（$existing_mode）"
      read -rp "是否重新配置？ [y/N]: " reconfig
      if [[ ! "$reconfig" =~ ^[Yy]$ ]]; then
        info "使用现有配置"
        source "$SCRIPT_DIR/.env"
        TOKEN="${ANCHOR_API_TOKEN:-}"
        PORT="${ANCHOR_PORT:-17421}"
        MODE="${ANCHOR_MODE:-server_worker}"
        if docker ps --filter "name=anchor-server" --filter "status=running" | grep -q anchor-server 2>/dev/null || \
           docker ps --filter "name=anchor-worker" --filter "status=running" | grep -q anchor-worker 2>/dev/null; then
          ok "容器已在运行"
          print_result
          maybe_open_app
          exit 0
        else
          info "容器未运行，正在启动..."
          start_containers
          wait_healthy
          print_result
          exit 0
        fi
      fi
    fi
  fi
}

main() {
  check_docker
  check_existing
  select_mode
  collect_config
  build_images
  start_containers
  wait_healthy
  print_result
  maybe_open_app
}

main "$@"

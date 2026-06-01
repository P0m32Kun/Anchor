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
ACTION=""  # install | restart
ACR_REGISTRY="crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun"

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

get_compose_file() {
  source "$SCRIPT_DIR/.env" 2>/dev/null || true
  case "${ANCHOR_MODE:-server_worker}" in
    server) echo "docker-compose.server.yml" ;;
    worker) echo "docker-compose.worker.yml" ;;
    *)      echo "docker-compose.yml" ;;
  esac
}

restart_services() {
  local compose_file
  compose_file=$(get_compose_file)

  if [ ! -f "$SCRIPT_DIR/$compose_file" ]; then
    err "未找到 ${compose_file}，请先运行安装"
    exit 1
  fi

  # 检查容器是否在运行
  if ! docker ps --filter "name=anchor" --filter "status=running" | grep -q anchor 2>/dev/null; then
    warn "没有运行中的 Anchor 容器"
    read -rp "是否启动服务？ [Y/n]: " start_choice
    if [[ ! "$start_choice" =~ ^[Nn]$ ]]; then
      source "$SCRIPT_DIR/.env" 2>/dev/null || true
      export ANCHOR_API_TOKEN="${ANCHOR_API_TOKEN:-}"
      export ANCHOR_PORT="${ANCHOR_PORT:-17421}"
      $COMPOSE_CMD -f "$compose_file" up -d
    else
      return 0
    fi
  else
    info "正在重启服务..."
    $COMPOSE_CMD -f "$compose_file" restart
  fi

  # 等待就绪
  local port="${ANCHOR_PORT:-17421}"
  info "等待服务就绪..."
  local max_wait=60
  local elapsed=0
  while [ $elapsed -lt $max_wait ]; do
    if curl -sf "http://localhost:${port}/health" &>/dev/null; then
      ok "服务已就绪"
      print_result
      return 0
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    printf "."
  done
  echo ""
  warn "健康检查超时，请手动检查: $0 logs"
}

select_mode() {
  echo ""
  echo "请选择操作:"
  echo "  1) Server Only    — 仅 API 服务（适合 VPS 部署）"
  echo "  2) Worker Only    — 连接远程 Server 的扫描节点"
  echo "  3) Server+Worker  — 完整功能（本地开发/测试）"
  echo "  ─────────────────"
  echo "  4) 重启服务       — 重启已运行的容器"
  echo "  5) 查看状态       — 容器运行状态"
  echo "  6) 查看日志       — 实时日志"
  echo "  7) 停止服务       — 停止所有容器"
  echo ""

  while true; do
    read -rp "请输入选项 [1-7]: " choice
    case $choice in
      1) ACTION="install"; MODE="server"; break ;;
      2) ACTION="install"; MODE="worker"; break ;;
      3) ACTION="install"; MODE="server_worker"; break ;;
      4) ACTION="restart"; break ;;
      5) ACTION="status"; break ;;
      6) ACTION="logs"; break ;;
      7) ACTION="down"; break ;;
      *) warn "无效选项，请输入 1-7" ;;
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
  info "拉取镜像..."

  # 从阿里云 ACR 拉取镜像（如果本地不存在）
  pull_if_missing() {
    local image="$1"
    if docker image inspect "${image}:latest" &>/dev/null; then
      ok "${image} 已存在"
    else
      info "拉取 ${image}..."
      docker pull "${ACR_REGISTRY}/${image}:latest"
      docker tag "${ACR_REGISTRY}/${image}:latest" "${image}:latest"
    fi
  }

  case $MODE in
    server)
      pull_if_missing anchor-server
      pull_if_missing anchor-frontend
      ;;
    worker)
      pull_if_missing anchor-worker
      ;;
    server_worker)
      pull_if_missing anchor-server
      pull_if_missing anchor-worker
      pull_if_missing anchor-frontend
      ;;
  esac

  ok "镜像准备完成"
}

save_env() {
  cat > "$SCRIPT_DIR/.env" <<EOF
ANCHOR_API_TOKEN=$TOKEN
ANCHOR_PORT=$PORT
ANCHOR_MODE=$MODE
ANCHOR_REGISTRY=$ACR_REGISTRY
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
    server|server_worker)
      echo "  浏览器访问: http://localhost"
      echo "  API 地址:   http://localhost:${PORT}"
      echo "  Token:      $TOKEN"
      ;;
    worker)
      echo "  Worker 已连接到: $CORE_URL"
      ;;
  esac

  echo ""
  echo "  管理命令:"
  echo "    $0 status    — 查看状态"
  echo "    $0 logs      — 查看日志"
  echo "    $0 down      — 停止服务"
  echo ""
}

check_existing() {
  # .env 已由 load_env 加载，这里只做询问
  local existing_mode
  existing_mode=$(grep "^ANCHOR_MODE=" "$SCRIPT_DIR/.env" 2>/dev/null | cut -d= -f2 || true)
  if [ -n "$existing_mode" ]; then
    echo ""
    warn "检测到已有部署配置（${existing_mode}）"
    read -rp "是否重新配置？ [y/N]: " reconfig
    if [[ ! "$reconfig" =~ ^[Yy]$ ]]; then
      info "使用现有配置"
      if docker ps --filter "name=anchor-server" --filter "status=running" | grep -q anchor-server 2>/dev/null || \
         docker ps --filter "name=anchor-worker" --filter "status=running" | grep -q anchor-worker 2>/dev/null; then
        ok "容器已在运行"
        print_result
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
}

load_env() {
  if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env" 2>/dev/null || true
    TOKEN="${ANCHOR_API_TOKEN:-}"
    PORT="${ANCHOR_PORT:-17421}"
    MODE="${ANCHOR_MODE:-server_worker}"
  fi
}

main() {
  check_docker
  select_mode
  load_env

  local compose_file
  compose_file=$(get_compose_file)

  case "$ACTION" in
    restart)
      restart_services
      ;;
    install)
      check_existing
      collect_config
      build_images
      start_containers
      wait_healthy
      print_result
      ;;
    status)
      $COMPOSE_CMD -f "$compose_file" ps
      ;;
    logs)
      $COMPOSE_CMD -f "$compose_file" logs -f
      ;;
    down)
      info "正在停止服务..."
      $COMPOSE_CMD -f "$compose_file" down
      ok "服务已停止"
      ;;
  esac
}

main "$@"

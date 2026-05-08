.PHONY: build clean
.PHONY: up down up-server down-server up-worker down-worker restart-worker
.PHONY: logs logs-server logs-worker status shell-server shell-worker
.PHONY: build-worker-base build-worker-builder-base push-worker-base pull-worker-base setup-worker-base
.PHONY: build-server-base push-server-base pull-server-base setup-server-base
.PHONY: test test-unit test-e2e test-e2e-smoke test-e2e-full
.PHONY: range-up range-down range-status range-logs
.PHONY: dev-web tauri-dev tauri-build

GO_FILES := $(shell find . -name '*.go' -not -path './frontend/*')

# ============================================================
#  Base Image (预装安全工具，极少更新)
# ============================================================

build-worker-base:
	# 强制 linux/amd64 — 部署目标为 x64 服务器，本地 ARM Mac 通过 QEMU 自动模拟
	docker build --platform linux/amd64 -f Dockerfile.worker-base -t anchor-worker-base:latest .

push-worker-base:
	docker tag anchor-worker-base:latest p0m32kun/anchor-worker-base:latest
	docker push p0m32kun/anchor-worker-base:latest

pull-worker-base:
	docker pull p0m32kun/anchor-worker-base:latest
	docker tag p0m32kun/anchor-worker-base:latest anchor-worker-base:latest

# --- Worker Builder Base Image ---
build-worker-builder-base:
	docker build --platform linux/amd64 -f Dockerfile.worker-builder-base -t anchor-worker-builder-base:latest .

push-worker-builder-base:
	docker tag anchor-worker-builder-base:latest p0m32kun/anchor-worker-builder-base:latest
	docker push p0m32kun/anchor-worker-builder-base:latest

pull-worker-builder-base:
	docker pull p0m32kun/anchor-worker-builder-base:latest
	docker tag p0m32kun/anchor-worker-builder-base:latest anchor-worker-builder-base:latest

# --- Server Base Image ---
build-server-base:
	# 强制 linux/amd64 — 部署目标为 x64 服务器，本地 ARM Mac 通过 QEMU 自动模拟
	docker build --platform linux/amd64 -f Dockerfile.server-base -t anchor-server-base:latest .

push-server-base:
	docker tag anchor-server-base:latest p0m32kun/anchor-server-base:latest
	docker push p0m32kun/anchor-server-base:latest

pull-server-base:
	docker pull p0m32kun/anchor-server-base:latest
	docker tag p0m32kun/anchor-server-base:latest anchor-server-base:latest

setup-server-base: build-server-base

# 首次设置：构建 worker 的两个基础镜像
setup-worker-base: build-worker-base build-worker-builder-base

# ============================================================
#  Development Environment (Docker only)
# ============================================================

# 启动完整环境（server + worker）
up:
	docker compose -f docker-compose.yml up -d --build

# 快速启动（不重建镜像，使用缓存）
up-fast:
	docker compose -f docker-compose.yml up -d

# 强制重建并启动
up-force:
	docker compose -f docker-compose.yml down --remove-orphans
	docker compose -f docker-compose.yml build --no-cache
	docker compose -f docker-compose.yml up -d

# 停止完整环境
down:
	docker compose -f docker-compose.yml down --remove-orphans

# 仅启动 server
up-server:
	docker compose -f docker-compose.server.yml up -d --build

down-server:
	docker compose -f docker-compose.server.yml down --remove-orphans

# 仅启动 worker（连接已有 server）
up-worker:
	docker compose -f docker-compose.worker.yml up -d --build

down-worker:
	docker compose -f docker-compose.worker.yml down --remove-orphans

# 重启 worker
restart-worker: down-worker up-worker
	@echo "Worker restarted"

# ============================================================
#  Logs & Debug
# ============================================================

status:
	docker compose ps

logs:
	docker compose logs -f

logs-server:
	docker compose logs -f server

logs-worker:
	docker compose logs -f worker

logs-server-solo:
	docker compose -f docker-compose.server.yml logs -f server

logs-worker-solo:
	docker compose -f docker-compose.worker.yml logs -f worker

shell-server:
	docker exec -it anchor-server /bin/sh

shell-worker:
	docker exec -it anchor-worker /bin/sh

# ============================================================
#  Testing
# ============================================================

# Go 单元测试（本地运行，不需要 Docker）
test:
	go test ./...

test-unit: test

# E2E 测试：启动完整 Docker 环境后运行 Playwright
test-e2e:
	@echo "[test-e2e] Starting E2E Docker environment..."
	@docker compose -f docker-compose.e2e.yml up -d --build
	@echo "[test-e2e] Waiting for services..."
	@sleep 5
	@cd frontend && npx playwright test --project=chromium

# E2E smoke 测试
test-e2e-smoke:
	@docker compose -f docker-compose.e2e.yml up -d --build
	@sleep 5
	@cd frontend && npx playwright test e2e/tests/smoke.spec.ts --project=chromium

# E2E 完整流程测试（无预置 auth）
test-e2e-full:
	@docker compose -f docker-compose.e2e.yml up -d --build
	@sleep 5
	@cd frontend && npx playwright test e2e/tests/full-flow.spec.ts --project=chromium-auth

# E2E 环境启动（不运行测试，手动调试用）
test-e2e-up:
	docker compose -f docker-compose.e2e.yml up -d --build

# E2E 环境停止
test-e2e-down:
	docker compose -f docker-compose.e2e.yml down --remove-orphans

# ============================================================
#  Rangefield (靶场，独立管理)
# ============================================================

range-up:
	docker compose -f docker-rangefield/docker-compose.yml up -d

range-down:
	docker compose -f docker-rangefield/docker-compose.yml down --remove-orphans

range-status:
	docker compose -f docker-rangefield/docker-compose.yml ps

range-logs:
	docker compose -f docker-rangefield/docker-compose.yml logs -f

test-naabu:
	docker exec -it anchor-worker naabu -host 172.30.0.10 -p 80

# ============================================================
#  Local Build (仅编译二进制，不启动服务)
# ============================================================

build:
	go build -o bin/anchor .

clean:
	rm -rf bin/
	docker compose -f docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.server.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.worker.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.e2e.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-rangefield/docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
	docker network rm anchor-net 2>/dev/null || true

# ============================================================
#  Frontend Dev (本地开发，不依赖 Docker)
# ============================================================

dev-web:
	@echo "Starting Vite dev server..."
	cd frontend && npm install
	./frontend/node_modules/.bin/vite --host

tauri-dev:
	@echo "Starting Tauri dev mode..."
	cd frontend && npm install
	./frontend/node_modules/.bin/tauri dev

tauri-build:
	cd frontend && npm install
	./frontend/node_modules/.bin/tauri build

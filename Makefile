.PHONY: build build-local clean
.PHONY: build-server build-worker
.PHONY: up down up-server down-server up-worker down-worker restart-worker
.PHONY: logs logs-server logs-worker status shell-server shell-worker
.PHONY: build-worker-base push-worker-base push-worker-base-cn pull-worker-base
.PHONY: build-server-runtime-base push-server-runtime-base pull-server-runtime-base
.PHONY: test test-unit test-e2e test-e2e-smoke test-e2e-full
.PHONY: range-up range-down range-status range-logs
.PHONY: dev-web

GO_FILES := $(shell find . -name '*.go' -not -path './frontend/*')

# ============================================================
#  Base Images (预装运行时依赖，极少更新)
# ============================================================

# --- Worker Base Image (预装安全工具) ---
# 本地开发构建 — 自动匹配 host 架构
build-worker-base:
	docker build -f Dockerfile.worker-base -t anchor-worker-base:latest .

# 多平台推送
push-worker-base:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.worker-base \
		-t p0m32kun/anchor-worker-base:latest \
		--push .

# 推送到阿里云 ACR（国内加速）
ACR_REGISTRY ?= registry.cn-hangzhou.aliyuncs.com
ACR_NAMESPACE ?= p0m32kun
push-worker-base-cn:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.worker-base \
		-t $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-worker-base:latest \
		--push .

pull-worker-base:
	docker pull p0m32kun/anchor-worker-base:latest
	docker tag p0m32kun/anchor-worker-base:latest anchor-worker-base:latest

# --- Server Runtime Base Image ---
build-server-runtime-base:
	docker build -f Dockerfile.server-runtime-base -t anchor-server-runtime-base:latest .

push-server-runtime-base:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.server-runtime-base \
		-t p0m32kun/anchor-server-runtime-base:latest \
		--push .

pull-server-runtime-base:
	docker pull p0m32kun/anchor-server-runtime-base:latest
	docker tag p0m32kun/anchor-server-runtime-base:latest anchor-server-runtime-base:latest

# ============================================================
#  Runtime Image Build (从 GitHub Release 下载预编译二进制)
# ============================================================

build-server:
	docker build -f Dockerfile.server -t anchor-server:latest .

build-worker:
	docker build -f Dockerfile.worker -t anchor-worker:latest .

# ============================================================
#  Local Build (本地编译 Go 二进制，需要 Go + gcc + libsqlite3-dev)
# ============================================================

build:
	CGO_ENABLED=1 go build -ldflags="-w -s" -o bin/anchor .

# 编译并构建 Docker 镜像（本地有 Go 环境时使用，不依赖 GitHub Release）
build-local: build
	docker build -f Dockerfile.server -t anchor-server:latest \
		--build-arg RELEASE_VERSION=local .
	@echo "注意：build-local 需要先将 bin/anchor 上传到 release 或修改 Dockerfile 使用 COPY"

# ============================================================
#  Development Environment
# ============================================================

up:
	docker compose -f docker-compose.yml up -d --build

up-fast:
	docker compose -f docker-compose.yml up -d

up-force:
	docker compose -f docker-compose.yml down --remove-orphans
	docker compose -f docker-compose.yml build --no-cache
	docker compose -f docker-compose.yml up -d

down:
	docker compose -f docker-compose.yml down --remove-orphans

up-server:
	docker compose -f docker-compose.server.yml up -d --build

down-server:
	docker compose -f docker-compose.server.yml down --remove-orphans

up-worker:
	docker compose -f docker-compose.worker.yml up -d --build

down-worker:
	docker compose -f docker-compose.worker.yml down --remove-orphans

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

test:
	go test ./...

test-unit: test

test-e2e:
	@echo "[test-e2e] Starting E2E Docker environment..."
	@docker compose -f docker-compose.e2e.yml up -d --build
	@echo "[test-e2e] Waiting for services..."
	@sleep 5
	@cd frontend && npx playwright test --project=chromium

test-e2e-smoke:
	@docker compose -f docker-compose.e2e.yml up -d --build
	@sleep 5
	@cd frontend && npx playwright test e2e/tests/smoke.spec.ts --project=chromium

test-e2e-full:
	@docker compose -f docker-compose.e2e.yml up -d --build
	@sleep 5
	@cd frontend && npx playwright test e2e/tests/full-flow.spec.ts --project=chromium-auth

test-e2e-up:
	docker compose -f docker-compose.e2e.yml up -d --build

test-unit-frontend:
	@cd frontend && npm run test:unit

test-e2e-down:
	docker compose -f docker-compose.e2e.yml down --remove-orphans

# ============================================================
#  Rangefield (靶场)
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
#  Cleanup
# ============================================================

clean:
	rm -rf bin/
	docker compose -f docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.server.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.worker.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.e2e.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-rangefield/docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
	docker network rm anchor-net 2>/dev/null || true

# ============================================================
#  Frontend Dev
# ============================================================

dev-web:
	@echo "Starting Vite dev server..."
	cd frontend && npm install
	./frontend/node_modules/.bin/vite --host

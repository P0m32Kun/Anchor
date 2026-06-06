.PHONY: build build-local clean
.PHONY: build-server build-worker
.PHONY: push-server-cn push-worker-cn push-frontend-cn push-all-cn
.PHONY: up down pull up-server down-server up-worker down-worker restart-worker up-fast up-force
.PHONY: dev-rebuild e2e-local e2e-local-down e2e-local-logs
.PHONY: logs logs-server logs-worker status shell-server shell-worker
.PHONY: build-server-runtime-base push-server-runtime-base pull-server-runtime-base
.PHONY: test test-unit test-e2e test-e2e-smoke test-e2e-scan test-e2e-full
.PHONY: release-verify release-verify-build release-verify-down
.PHONY: range-up range-down range-status range-logs
.PHONY: dev-web

GO_FILES := $(shell find . -name '*.go' -not -path './frontend/*')

# ============================================================
#  Base Images (预装运行时依赖，极少更新)
# ============================================================

ACR_REGISTRY ?= crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com
ACR_NAMESPACE ?= p0m32kun

# --- Server Runtime Base Image ---
build-server-runtime-base:
	docker build -f Dockerfile.server-runtime-base -t anchor-server-runtime-base:latest .

push-server-runtime-base:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.server-runtime-base \
		-t p0m32kun/anchor-server-runtime-base:latest \
		--push .

push-server-runtime-base-cn:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.server-runtime-base \
		-t $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-server-runtime-base:latest \
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

# --- 阿里云 ACR 推送（国内用户拉取用）---
push-server-cn:
	docker tag anchor-server:latest $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-server:latest
	docker push $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-server:latest

push-worker-cn:
	docker tag anchor-worker:latest $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-worker:latest
	docker push $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-worker:latest

push-frontend-cn:
	docker tag anchor-frontend:latest $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-frontend:latest
	docker push $(ACR_REGISTRY)/$(ACR_NAMESPACE)/anchor-frontend:latest

# 一键推送所有镜像到阿里云
push-all-cn: push-server-cn push-worker-cn push-frontend-cn
	@echo "所有镜像已推送到 $(ACR_REGISTRY)/$(ACR_NAMESPACE)/"

# ============================================================
#  Local Build (本地编译 Go 二进制，需要 Go + gcc + libsqlite3-dev)
# ============================================================

build:
	CGO_ENABLED=1 go build -ldflags="-w -s" -o bin/anchor .

# 交叉编译 Linux 二进制（用于 Docker 快速构建）
TARGETARCH ?= arm64
build-linux:
	@echo "[build-linux] Cross-compiling for linux/$(TARGETARCH)..."
	@mkdir -p bin
	@docker buildx build --platform linux/$(TARGETARCH) \
		-f Dockerfile.compile \
		--build-arg TARGETARCH=$(TARGETARCH) \
		-t anchor-compile:latest \
		--target output \
		--output type=local,dest=./bin/ \
		. 2>&1 | tail -5
	@ls -lh bin/anchor-linux-$(TARGETARCH) 2>/dev/null && echo "[build-linux] Done!" || echo "[build-linux] Failed!"

# 编译并构建 Docker 镜像（本地有 Go 环境时使用，不依赖 GitHub Release）
build-local: build
	docker build -f Dockerfile.server -t anchor-server:latest \
		--build-arg RELEASE_VERSION=local .
	@echo "注意：build-local 需要先将 bin/anchor 上传到 release 或修改 Dockerfile 使用 COPY"

# ============================================================
#  Fast Build (基于 base 镜像的快速构建，适合测试迭代)
# ============================================================

# 首次使用或工具版本更新时，构建 base 镜像（耗时较长，只需执行一次）
build-base:
	@echo "[build-base] Building server runtime base..."
	docker build -f Dockerfile.server-runtime-base -t anchor-server-base:latest .
	@echo "[build-base] Building worker runtime base (this takes a while)..."
	docker build -f Dockerfile.worker-runtime-base -t anchor-worker-base:latest .
	@echo "[build-base] Done! Base images are ready."

# 快速构建应用镜像（基于 base，只需复制二进制，约 10-30 秒）
build-fast: build-linux
	@echo "[build-fast] Building server image (TARGETARCH=$(TARGETARCH))..."
	docker build -f Dockerfile.server-fast --build-arg TARGETARCH=$(TARGETARCH) -t anchor-server:local .
	@echo "[build-fast] Building worker image (TARGETARCH=$(TARGETARCH))..."
	docker build -f Dockerfile.worker-fast --build-arg TARGETARCH=$(TARGETARCH) -t anchor-worker:local .
	@echo "[build-fast] Done! Images: anchor-server:local, anchor-worker:local"

# ============================================================
#  用户部署（仅拉 ACR 三镜像，不 build）
# ============================================================

pull:
	docker compose -f docker-compose.yml pull
	docker compose -f docker-compose.server.yml pull
	docker compose -f docker-compose.worker.yml pull

up: pull
	docker compose -f docker-compose.yml up -d

down:
	docker compose -f docker-compose.yml down --remove-orphans

# 已废弃别名：部署请用 up（pull + up）；本地重建请用 dev-rebuild
up-fast: up
up-force: dev-rebuild

up-server:
	docker compose -f docker-compose.server.yml pull
	docker compose -f docker-compose.server.yml up -d

down-server:
	docker compose -f docker-compose.server.yml down --remove-orphans

up-worker:
	docker compose -f docker-compose.worker.yml pull
	docker compose -f docker-compose.worker.yml up -d

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

E2E_TOKEN ?= test-e2e-token
E2E_PLAYWRIGHT_ENV = ANCHOR_API_TOKEN=$(E2E_TOKEN) ANCHOR_E2E_SKIP_DOCKER=1

test-e2e:
	@echo "[test-e2e] Starting E2E Docker environment (fast suite: chromium project)..."
	@ANCHOR_API_TOKEN=$(E2E_TOKEN) docker compose -f docker-compose.e2e.yml up -d --build
	@echo "[test-e2e] Waiting for services..."
	@sleep 5
	@cd frontend && $(E2E_PLAYWRIGHT_ENV) npx playwright test --project=chromium

test-e2e-smoke:
	@ANCHOR_API_TOKEN=$(E2E_TOKEN) docker compose -f docker-compose.e2e.yml up -d --build
	@sleep 5
	@cd frontend && $(E2E_PLAYWRIGHT_ENV) npx playwright test e2e/tests/smoke.spec.ts --project=chromium

test-e2e-scan:
	@echo "[test-e2e-scan] Long pipeline specs (chromium-scan + chromium-auth)..."
	@ANCHOR_API_TOKEN=$(E2E_TOKEN) docker compose -f docker-compose.e2e.yml up -d --build
	@sleep 5
	@cd frontend && $(E2E_PLAYWRIGHT_ENV) npx playwright test --project=chromium-scan --project=chromium-auth

test-e2e-full:
	@ANCHOR_API_TOKEN=$(E2E_TOKEN) docker compose -f docker-compose.e2e.yml up -d --build
	@sleep 5
	@cd frontend && $(E2E_PLAYWRIGHT_ENV) npx playwright test e2e/tests/full-flow.spec.ts --project=chromium-auth

test-e2e-up:
	ANCHOR_API_TOKEN=$(E2E_TOKEN) docker compose -f docker-compose.e2e.yml up -d --build

test-unit-frontend:
	@cd frontend && npm run test:unit

test-e2e-down:
	docker compose -f docker-compose.e2e.yml down --remove-orphans

# ============================================================
#  上线前验证（tag 推送前必做 — 用户部署镜像路径）
# ============================================================

RELEASE_VERIFY_TAG ?= release-candidate

release-verify:
	@chmod +x scripts/verify-release-images.sh
	@./scripts/verify-release-images.sh

release-verify-build:
	@arch=$$(uname -m); \
	case "$$arch" in x86_64) ta=amd64;; aarch64|arm64) ta=arm64;; *) echo "unsupported arch: $$arch"; exit 1;; esac; \
	$(MAKE) build-linux TARGETARCH=$${TARGETARCH:-$$ta}
	docker build -f Dockerfile.server --build-arg RELEASE_VERSION=local \
		-t anchor-server:$(RELEASE_VERIFY_TAG) .
	docker build -f Dockerfile.worker --build-arg RELEASE_VERSION=local \
		-t anchor-worker:$(RELEASE_VERIFY_TAG) .
	docker build -f Dockerfile.frontend -t anchor-frontend:$(RELEASE_VERIFY_TAG) .
	@echo "Candidate images: anchor-{server,worker,frontend}:$(RELEASE_VERIFY_TAG)"

release-verify-down:
	docker compose -f docker-compose.release-verify.yml down --remove-orphans

# ============================================================
#  本地测试 Docker（build-fast / e2e-local，非用户部署）
# ============================================================

# 强制重建本地 fast 镜像并启动 e2e-local（原 up-force 语义，勿用于生产 compose）
dev-rebuild: build-fast
	docker compose -f docker-compose.e2e-local.yml down --remove-orphans
	docker compose -f docker-compose.e2e-local.yml up -d --build

e2e-local: build-fast
	@echo "[e2e-local] Starting E2E environment with local code..."
	ANCHOR_API_TOKEN=test-e2e-token docker compose -f docker-compose.e2e-local.yml up -d
	@echo "[e2e-local] Waiting for services..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if curl -sf http://localhost:17421/health > /dev/null 2>&1; then \
			echo "Server ready!"; \
			break; \
		fi; \
		sleep 1; \
	done
	@echo "[e2e-local] Services are ready. Run tests manually or use:"
	@echo "  curl -H 'Authorization: Bearer test-e2e-token' http://localhost:17421/health"

# 停止本地 E2E 环境
e2e-local-down:
	docker compose -f docker-compose.e2e-local.yml down --remove-orphans

# 查看本地 E2E 日志
e2e-local-logs:
	docker compose -f docker-compose.e2e-local.yml logs -f

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
	docker exec -it anchor-worker naabu -host 172.31.0.10 -p 80

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
	docker network rm anchor-net anchor-net-e2e 2>/dev/null || true

# ============================================================
#  Frontend Dev
# ============================================================

dev-web:
	@echo "Starting Vite dev server..."
	cd frontend && npm install
	./frontend/node_modules/.bin/vite --host

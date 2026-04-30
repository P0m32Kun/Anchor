.PHONY: build run dev test clean
.PHONY: up up-server up-worker down down-server down-worker restart-worker status logs logs-server logs-worker logs-server-solo logs-worker-solo
.PHONY: dev-desktop dev-web tauri-dev tauri-build
.PHONY: range-up range-down range-status range-logs
.PHONY: up-all down-all
.PHONY: shell-server shell-worker test-naabu

GO_FILES := $(shell find . -name '*.go' -not -path './frontend/*')

# --- Build & Run ---

build:
	go build -o bin/anchor .

run: build
	./bin/anchor

dev:
	@lsof -ti:17421 | xargs kill -9 2>/dev/null || true
	go run .

test:
	go test ./...

clean:
	rm -rf bin/
	docker compose -f docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.server.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-compose.worker.yml down --volumes --remove-orphans 2>/dev/null || true
	docker compose -f docker-rangefield/docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
	docker network rm anchor-net 2>/dev/null || true

# --- Docker Compose Lifecycle ---

# 单机全栈（server + worker）
up:
	docker compose -f docker-compose.yml up -d --build

down:
	docker compose -f docker-compose.yml down --remove-orphans

# 单独 Server
up-server:
	docker compose -f docker-compose.server.yml up -d --build

down-server:
	docker compose -f docker-compose.server.yml down --remove-orphans

# 单独 Worker（需设置 ANCHOR_CORE_URL 指向远端 server）
up-worker:
	docker compose -f docker-compose.worker.yml up -d --build

down-worker:
	docker compose -f docker-compose.worker.yml down --remove-orphans

restart-worker: down-worker up-worker
	@echo "Worker restarted"

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

# --- Development Modes ---

dev-desktop: up
	@echo "Starting Tauri dev..."
	@pkill -f "vite" 2>/dev/null || true
	@pkill -f "target/debug/anchor" 2>/dev/null || true
	cd frontend && npm install
	./frontend/node_modules/.bin/tauri dev

dev-web: up
	@echo "Starting Vite dev server..."
	cd frontend && npm install
	./frontend/node_modules/.bin/vite --host

tauri-dev:
	@pkill -f "vite" 2>/dev/null || true
	@pkill -f "target/debug/anchor" 2>/dev/null || true
	cd frontend && npm install
	./frontend/node_modules/.bin/tauri dev

tauri-build:
	cd frontend && npm install
	./frontend/node_modules/.bin/tauri build

# --- Rangefield ---

range-up:
	docker compose -f docker-rangefield/docker-compose.yml up -d

range-down:
	docker compose -f docker-rangefield/docker-compose.yml down --remove-orphans

range-status:
	docker compose -f docker-rangefield/docker-compose.yml ps

range-logs:
	docker compose -f docker-rangefield/docker-compose.yml logs -f

# --- Combined ---

up-all: up range-up
	@echo "Anchor server, worker, and rangefield services are up"

down-all: down range-down
	@echo "All services stopped"

# --- Utilities ---

shell-server:
	docker exec -it anchor-server /bin/sh

shell-worker:
	docker exec -it anchor-worker /bin/sh

test-naabu:
	docker exec -it anchor-worker naabu -host 172.30.0.10 -p 80

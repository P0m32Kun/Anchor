# 开发者 E2E 测试指南

> 面向**项目开发与 CI 验收**的 Docker + Playwright 测试栈。  
> 客户生产部署请见 [`deployment.md`](deployment.md)，勿混用 compose 文件。

---

## 与部署栈的区别

| | **E2E 测试（本文）** | **客户部署** |
|--|---------------------|-------------|
| 目的 | 验证新代码、跑 Playwright | 交付给客户使用 |
| Compose | `docker-compose.e2e*.yml` | `docker-compose.yml` 等 |
| Server/Worker | 本地 build `anchor-*:local` | ACR 预构建镜像 |
| 前端 | Vite dev `:1420`（Playwright 自动起） | 生产 Nginx `:80` |
| 靶场 / Mock | 内嵌 rangefield、fofa-mock | 无 |
| API Token | `test-e2e-token` | 自设 |

---

## 两种 E2E Compose

### `docker-compose.e2e.yml` — 自动化测试（默认）

Playwright `global-setup` 与 `make test-e2e*` 使用此文件。

- **Build**：`Dockerfile.server-fast` / `Dockerfile.worker-fast` → `anchor-*:local`
- **网络**：独立 `anchor-net-e2e`（`172.31.0.0/24`），避免与生产 compose 冲突
- **内嵌**：rangefield 靶场（nginx / tomcat / grafana / redis / mysql）+ `fofa-mock`
- **无 frontend 容器**：Playwright 通过 `webServer` 启动 Vite dev（`:1420`）

```bash
make test-e2e          # 快速套件（chromium project）
make test-e2e-smoke    # 仅 smoke.spec.ts
make test-e2e-scan     # 长 pipeline（chromium-scan + chromium-auth）
make test-e2e-full     # full-flow.spec.ts
make test-e2e-down     # 停止栈
```

环境变量 `ANCHOR_E2E_SKIP_DOCKER=1` 可跳过 global-setup 的 Docker 启停（Makefile 已设置，配合手动 `test-e2e-up`）。

### `docker-compose.e2e-local.yml` — 手动迭代

适合改完代码后手动 curl / 浏览器验证，不跑 Playwright 全量。

- **Build**：同上 fast Dockerfile
- **Frontend**：仍拉 ACR `anchor-frontend:latest`（`:80`，接近生产 nginx 路径）
- **靶场**：连接外部 `docker-rangefield` 网络（需先 `make range-up`）

```bash
make build-base      # 首次或安全工具版本变更（耗时较长）
make build-linux     # 交叉编译 bin/anchor-linux-*
make build-fast      # 10–30 秒构建 anchor-*:local
make e2e-local       # build-fast + 启动栈
make e2e-local-down
make e2e-local-logs
```

### 可选覆盖：`docker-compose.e2e-real-fofa.override.yml`

去掉 fofa-mock，改用真实 FOFA API（需配置凭证）：

```bash
docker compose -f docker-compose.e2e.yml -f docker-compose.e2e-real-fofa.override.yml up -d --build
```

### 可选覆盖：`docker-compose.e2e-multi-worker.override.yml`

本地验证 **多 Worker 任务分配**（least-loaded + 轮询调度）：

```bash
# 启动 server + 两个 worker + 靶场
ANCHOR_API_TOKEN=test-e2e-token docker compose \
  -f docker-compose.e2e.yml \
  -f docker-compose.e2e-multi-worker.override.yml \
  up -d --build server worker worker-b nginx fofa-mock

# 确认两个节点在线
curl -H "Authorization: Bearer test-e2e-token" http://localhost:17421/workers

# 跑 batch 规模扫描后按 worker_id 统计 task 分布
curl -H "Authorization: Bearer test-e2e-token" \
  http://localhost:17421/runs/{runId}/tasks \
  | jq 'group_by(.worker_id) | map({worker: .[0].worker_id, count: length})'
```

Makefile 快捷目标：`make test-e2e-multi-worker-up` / `make test-e2e-multi-worker-down` / `make test-e2e-multi-worker-scan`。

Worker 本地并发上限：`ANCHOR_WORKER_MAX_CONCURRENCY`（默认 10）；满载时返回 HTTP 503，Server 改派其他 worker。

---

## 快速构建链（fast 镜像）

E2E **不使用**生产 `Dockerfile.server` / `Dockerfile.worker`（那些从 GitHub Release 下载二进制，适合 CI 发布与客户镜像）。

开发迭代使用 **runtime-base + fast** 分层：

```
Dockerfile.server-runtime-base  →  anchor-server-base:latest
Dockerfile.worker-runtime-base  →  anchor-worker-base:latest
         ↓ COPY bin/anchor-linux-*
Dockerfile.server-fast          →  anchor-server:local
Dockerfile.worker-fast          →  anchor-worker:local
```

交叉编译：`Dockerfile.compile` → `make build-linux TARGETARCH=arm64|amd64`

---

## Playwright 运行流程

配置见 `frontend/playwright.config.ts`。

1. `global-setup`：启动 `docker-compose.e2e.yml`，等待 `:17421/health`
2. 生成 `storage-state.json`（API base + test token）
3. `webServer`：启动 Vite dev `:1420`
4. 按 project 跑 spec：
   - `chromium` — 页面 / smoke（默认 2 分钟 timeout）
   - `chromium-scan` — 长 pipeline（单 spec 最长 30 分钟）
   - `chromium-auth` — full-flow（独立 auth 流程）
5. `global-teardown`：停止 Docker 栈，删除 storage-state

**前置条件**：

```bash
docker info                    # Docker 须运行
cd frontend && npm install
npx playwright install chromium
```

测试编写规范见 [`docs/conventions/testing.md`](../conventions/testing.md) 与 [`frontend/e2e/README.md`](../../frontend/e2e/README.md)。

---

## 发布前验证（非 E2E，但相关）

打 `v*` tag 前须走**生产 Dockerfile 路径**，与用户部署一致：

```bash
make release-verify    # Dockerfile.server/worker/frontend + release-verify compose
```

| | `make test-e2e` | `make release-verify` |
|--|-----------------|----------------------|
| Compose | `docker-compose.e2e.yml` | `docker-compose.release-verify.yml` |
| Dockerfile | `*-fast` | `Dockerfile.server` / `.worker` / `.frontend` |
| 前端 | Vite `:1420` | 生产 nginx `:18080` |
| 时机 | 日常开发 | **tag 推送前** |

详见 [`ci-cd-guide.md`](ci-cd-guide.md)。

---

## 靶场（Rangefield）

`docker-rangefield/` 提供独立漏洞靶场，固定 IP `172.31.0.x`：

```bash
make range-up      # 启动靶场
make range-down
make range-status
make test-naabu    # 示例：从 worker 扫靶场
```

`docker-compose.e2e.yml` 已内嵌等价服务；`e2e-local` 模式需单独 `range-up` 并挂载 `rangefield-net`。

---

## Dockerfile 速查（仅开发 / CI）

| 文件 | 用途 | 出现在 |
|------|------|--------|
| `Dockerfile.server` | 生产 Server（Release 二进制） | CI docker-push、release-verify |
| `Dockerfile.worker` | 生产 Worker（含安全工具） | 同上 |
| `Dockerfile.frontend` | 生产 Frontend | 同上 |
| `Dockerfile.*-runtime-base` | 预装运行时依赖 | `make build-base` |
| `Dockerfile.*-fast` | 快速 COPY 本地二进制 | E2E compose |
| `Dockerfile.compile` | 交叉编译 Linux 二进制 | `make build-linux` |

**客户部署 compose 不含上述任何 `build` 段。**

---

## 故障排查

| 问题 | 处理 |
|------|------|
| `Docker daemon is not running` | 启动 Docker Desktop |
| `Backend did not become healthy` | `docker compose -f docker-compose.e2e.yml logs server` |
| `anchor-server-base:latest` not found | 先 `make build-base` |
| 与生产栈端口冲突 | E2E 用独立网络；生产用 `make down` 停掉 |
| Chromium 未安装 | `npx playwright install chromium` |

---

## 相关文档

- 客户部署：[`deployment.md`](deployment.md)
- 测试分层约定：[`conventions/testing.md`](../conventions/testing.md)
- Playwright 用例细节：[`frontend/e2e/README.md`](../../frontend/e2e/README.md)
- CI 流程：[`ci-cd-guide.md`](ci-cd-guide.md)

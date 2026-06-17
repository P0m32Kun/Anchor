# CI/CD 流程指南

## 概述

Anchor 使用 GitHub Actions：**PR/push 跑单元门禁**（`ci.yml`），**tag 触发 Release + Docker 发布**（`release.yml` → `docker-push.yml`）。

## PR 门禁（`ci.yml`）

在 `main` 的 push 与所有 Pull Request 上自动运行：

| Job | 步骤 |
|-----|------|
| `backend` | `go vet ./...`、`go test ./...` |
| `frontend` | `npm ci`、`typecheck`、`test:unit`、`build` |

长耗时 E2E（rangefield Docker 全栈）**不在**默认 PR 门禁内；本地或 nightly 使用 `make test-e2e`（快速套件）/ `make test-e2e-smoke` / `make test-e2e-scan` / `make test-e2e-full`。Playwright 已按 `chromium` / `chromium-scan` 拆分超时，无整次 `globalTimeout`。E2E 使用 `docker-compose.e2e.yml`（`Dockerfile.*-fast` build，网络 `anchor-net-e2e` / `172.31.0.0/24`），与用户部署用的 `docker-compose.yml`（仅 ACR `image`、无 `build`）分离。详见 [`e2e-testing.md`](e2e-testing.md)。

## 发布流程图

```
git tag v0.x.x && git push --tags
         │
         ▼
┌─────────────────┐
│  Release workflow │  构建 Go 二进制
│  (release.yml)   │  创建 GitHub Release
└────────┬────────┘
         │ 完成后自动触发
         ▼
┌─────────────────┐
│ Docker Push      │  构建 Docker 镜像
│ (docker-push.yml)│  推送到阿里云 ACR
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  阿里云 ACR      │  镜像存储
│  (p0m32kun/)     │
└─────────────────┘
```

## 上线前验证（tag 推送前必做）

推送 `v*` tag 会触发 Release + ACR 发布，**不可逆**。在打 tag 之前，必须用与生产相同的 Dockerfile 构建候选镜像，并按**用户部署路径**跑通验证。

### 验证内容

| 步骤 | 说明 |
|------|------|
| 单元门禁 | `go vet ./...`、`go test ./...` |
| 候选镜像构建 | `Dockerfile.server` / `Dockerfile.worker`（`RELEASE_VERSION=local` + 本地 `bin/anchor-linux-*`）+ `Dockerfile.frontend` |
| 用户 compose 路径 | `docker-compose.release-verify.yml` — 仅 `image`、无 `build`，nginx 反代 `/api` |
| 健康检查 | server `/health`、frontend HTTP 200、worker 注册上线 |
| API smoke | 认证后创建项目 |
| UI smoke | Playwright `release-verify-smoke.spec.ts`（`playwright.release-verify.config.ts`，走 `:18080` nginx） |

### 本地执行

```bash
# 完整验证（worker 镜像首次构建可能需数分钟）
make release-verify

# 仅复测已构建候选镜像
SKIP_BUILD=1 make release-verify

# 快速门禁（跳过 Playwright）
SKIP_SMOKE=1 make release-verify

# 仅构建三镜像，不启动栈
make release-verify-build
```

验证通过后：

```bash
git tag v0.x.x
git push --tags
```

### CI 手动触发

GitHub Actions → **Release Verify** → Run workflow（`release-verify.yml`）。与本地 `make release-verify` 等价，适合发布负责人 merge 后、打 tag 前的最后一道门禁。

### 与 E2E 测试栈的区别

| | `make test-e2e` | `make release-verify` |
|--|-----------------|----------------------|
| Compose | `docker-compose.e2e.yml`（含 build / fast 镜像） | `docker-compose.release-verify.yml`（仅 image，同用户 `docker-compose.yml` 结构） |
| 前端 | Vite dev `:1420` | 生产 nginx `:18080`（默认，可 env 覆盖） |
| Dockerfile | `Dockerfile.*-fast` | `Dockerfile.server` / `Dockerfile.worker` / `Dockerfile.frontend` |
| 时机 | 开发 / PR 后 | **tag 推送前** |

## 发布新版本

### 0. 上线前验证

```bash
make release-verify   # 或 Actions: Release Verify
```

### 1. 提交代码

```bash
# 添加修改
git add .

# 提交
git commit -m "feat: your changes"

# 推送到 main 分支
git push origin main
```

### 2. 创建 Tag 并触发 Release

```bash
# 创建 tag（语义化版本号）
git tag v0.5.0

# 推送 tag 触发 CI/CD
git push --tags
```

### 3. 自动化流程

推送 tag 后，GitHub Actions 会自动：

1. **Release workflow** (`release.yml`)
   - 编译 Go 二进制（linux-amd64、linux-arm64）
   - 创建 GitHub Release
   - 附加二进制文件到 Release assets

2. **Docker Push workflow** (`docker-push.yml`)
   - 等待 Release workflow 完成（或 `workflow_dispatch` 指定 tag）
   - Checkout **Release 对应 tag/ref**（非 main HEAD）
   - 构建并推送 `anchor-server`、`anchor-worker`、`anchor-frontend`
   - 镜像同时打 `v0.x.x` 与 `latest`；server/worker 从该 tag 的 GitHub Release 下载二进制

## Docker 镜像

### 镜像仓库

- **Registry**: `crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com`
- **Namespace**: `p0m32kun`

### 镜像列表

| 镜像名 | 说明 | Dockerfile |
|--------|------|------------|
| `anchor-server` | 服务端（API + 任务调度） | `Dockerfile.server` |
| `anchor-worker` | 工作端（执行扫描工具） | `Dockerfile.worker` |
| `anchor-frontend` | 前端（React + Nginx） | `Dockerfile.frontend` |

### 镜像标签

- `v0.x.x`：与 GitHub Release tag 对齐（权威版本）
- `latest`：每次成功 docker-push 同步更新

### 拉取镜像

```bash
# 国内用户（阿里云 ACR，速度快）
docker pull crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/anchor-server:latest
docker pull crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/anchor-worker:latest
docker pull crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/anchor-frontend:latest

# 国际用户（Docker Hub，需要配置）
docker pull p0m32kun/anchor-server:latest
docker pull p0m32kun/anchor-worker:latest
docker pull p0m32kun/anchor-frontend:latest
```

## GitHub Secrets 配置

在 GitHub repo 的 Settings → Secrets and variables → Actions 中配置：

| Secret 名 | 说明 | 示例 |
|-----------|------|------|
| `ACR_REGISTRY` | 阿里云 ACR 地址 | `crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com` |
| `ACR_USERNAME` | ACR 用户名 | `P0m32Kun` |
| `ACR_PASSWORD` | ACR 登录密码 | `***` |

## 镜像构建说明

### 生产 Server / Worker 镜像

`Dockerfile.server` / `Dockerfile.worker` 基于 `debian:bookworm-slim`，通过 `docker/install-anchor-binary.sh` 注入二进制（`RELEASE_VERSION=local` 时用 `bin/anchor-linux-*`，否则从 GitHub Release 下载）。**不**依赖 runtime-base 镜像。

### E2E 快速镜像（仅开发）

`Dockerfile.*-runtime-base` + `Dockerfile.*-fast` 供 `make build-base` / `make build-fast` 与 E2E compose 使用，详见 [`e2e-testing.md`](e2e-testing.md)。

生产 `docker-compose.yml` 中 Server 与 Worker 使用**独立数据卷**（`anchor-server-data` / `anchor-worker-data`），避免 SQLite 共库冲突。

## 常见问题

### Q: 如何更新 Worker runtime-base 镜像？

A: `Dockerfile.worker-runtime-base` 包含系统依赖和安全工具，很少需要更新：

```bash
# 1. 修改 Dockerfile.worker-runtime-base
# 2. 本地重建并验证 E2E 链
make build-base
make build-fast
make test-e2e-smoke
```

### Q: 如何手动触发镜像构建？

A: 可以在 GitHub Actions 页面手动触发 workflow：

1. 进入 Actions 页面
2. 选择 "Docker Push" workflow
3. 点击 "Run workflow"

### Q: 如何回滚到旧版本？

A: 使用特定版本的镜像：

```bash
# 修改 docker-compose.yml
services:
  server:
    image: crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/anchor-server:v0.4.0
```

### Q: 国内用户拉取镜像慢怎么办？

A: 使用阿里云 ACR 镜像：

```bash
# 使用 install.sh 自动配置
curl -fsSL https://raw.githubusercontent.com/P0m32Kun/Anchor/main/install.sh | bash

# 或手动拉取
docker pull crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/anchor-server:latest
```

## 监控和日志

### 查看构建状态

- GitHub Actions: https://github.com/P0m32Kun/Anchor/actions
- Release 页面: https://github.com/P0m32Kun/Anchor/releases

### 查看镜像状态

- 阿里云 ACR 控制台: https://cr.console.aliyun.com/

## 最佳实践

1. **版本号规范**: 使用语义化版本（v0.x.x）
2. **提交信息规范**: 使用 Conventional Commits（feat/fix/docs/chore）
3. **上线前验证**: `make release-verify` 通过后再 `git tag`（见上文「上线前验证」）
4. **测试优先**: PR 门禁 + release-verify 双轨，build/typecheck alone 不算发布就绪
5. **文档同步**: 修改代码时同步更新文档

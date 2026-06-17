# 客户部署指南

> 面向**生产环境 / 客户交付**的 Docker 部署说明。  
> 开发者本地测试与 E2E 请见 [`e2e-testing.md`](e2e-testing.md)，勿混用 compose 文件。

---

## 架构概览

```
[浏览器] --HTTP--> [Nginx :80] --/api/--> [Server :17421] <--HTTP 长轮询-- [Worker]
                           |
                     [React 静态文件]
```

| 组件 | 镜像 | 职责 |
|------|------|------|
| Frontend | `anchor-frontend` | Nginx 静态 serve + `/api/` 反代到 Server |
| Server | `anchor-server` | API、任务调度、SQLite 持久化 |
| Worker | `anchor-worker` | 预装安全工具，长轮询拉取任务 |

---

## 快速开始

```bash
bash install.sh
```

安装向导会：

1. 检测 Docker 环境
2. 选择部署模式（Server Only / Worker Only / Server+Worker）
3. 配置端口与 API Token
4. 从阿里云 ACR **拉取**预构建三镜像（无本地 build）
5. 启动容器并等待健康检查

完成后浏览器访问 `http://localhost`（或配置的端口）。

### 日常管理

```bash
bash install.sh status   # 容器状态
bash install.sh logs     # 实时日志
bash install.sh restart  # 重启（Token 变更后需执行）
bash install.sh down     # 停止服务
```

等价 Makefile 命令（需已配置 `.env`）：

```bash
make up          # pull + docker-compose.yml up
make down
make up-server   # 仅 Server + Frontend
make up-worker   # 仅 Worker
```

---

## 三种部署模式

均使用 **仅 `image`、无 `build`** 的 compose 文件，镜像来自阿里云 ACR。

| 模式 | Compose 文件 | 适用场景 |
|------|-------------|---------|
| Server+Worker | `docker-compose.yml` | 同机完整部署（VPS / 内网单机） |
| Server Only | `docker-compose.server.yml` | VPS 只跑管理面，Worker 远程连接 |
| Worker Only | `docker-compose.worker.yml` | 远程扫描节点，连接已有 Server |

`.env` 中 `ANCHOR_MODE` 决定 `install.sh restart` 使用哪个 compose：

- `server` → `docker-compose.server.yml`
- `worker` → `docker-compose.worker.yml`
- `server_worker`（默认）→ `docker-compose.yml`

---

## 镜像来源与版本

**Registry**（国内默认）：

```
crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/
```

| 镜像 | 说明 |
|------|------|
| `anchor-server` | Go API 服务 |
| `anchor-worker` | 安全工具 + Worker |
| `anchor-frontend` | Nginx + React 构建产物 |

**标签策略**：

- `latest` — `install.sh` 默认拉取
- `v0.x.x` — 与 GitHub Release tag 对齐；锁定版本时改 compose 或 `ANCHOR_REGISTRY` 中的 tag

**客户侧只做 pull，不在 compose 里 build。** 镜像由项目 CI 在 Release 后构建推送（见 [`ci-cd-guide.md`](ci-cd-guide.md)）。

---

## 配置

### API Token

- 安装时写入项目根目录 `.env` 的 `ANCHOR_API_TOKEN`
- 向导**不会**在终端打印完整 Token，请从 `.env` 读取
- 轮换：编辑 `.env` → `bash install.sh restart`

### Worker 网络

- **出站为主**：Worker 通过 `--core-url` 连接 Server（拉任务、心跳、上报），**无需 Worker 公网 IP**
- **Server 入站 Worker（可选）**：UI 实时查看运行中任务 stdout/stderr 时，Server 会请求 Worker 注册的 `endpoint`
  - 不看实时日志：Worker 只需能访问 Server
  - 需要实时日志：`endpoint` 须对 Server 可达（内网 IP、Docker 服务名 `worker`、Tailscale 等）

### 数据卷

同机 `docker-compose.yml` 中 Server 与 Worker 使用**独立** named volume（`anchor-server-data` / `anchor-worker-data`），避免 SQLite 冲突。

### 多平台

镜像支持 `linux/amd64` 与 `linux/arm64`，拉取时自动匹配宿主机架构。

---

## 与 E2E / 开发构建的边界

| | **客户部署（本文）** | **开发者 E2E** |
|--|---------------------|----------------|
| 入口 | `install.sh` / `make up` | `make test-e2e` / `make e2e-local` |
| Compose | `docker-compose.yml` 等 | `docker-compose.e2e*.yml` |
| 镜像 | ACR 预构建 `:latest` | 本地 build `anchor-*:local` |
| 前端 | 生产 Nginx `:80` | Vite dev `:1420` 或 ACR frontend |
| Token | 自设 | 固定 `test-e2e-token` |

**切勿**对客户环境使用 `docker-compose.e2e.yml` 或 `make build-fast`。

---

## 相关文档

- 架构细节：[`architecture.md`](architecture.md)「部署架构」章节
- 发布与 CI：[`ci-cd-guide.md`](ci-cd-guide.md)
- 开发者 E2E：[`e2e-testing.md`](e2e-testing.md)

# ADR-010: Docker 容器化部署

## 状态
已确认 ✅

## 上下文

v0.1 是纯本地二进制运行，部署步骤多（安装 Go、Node、安全工具），环境差异大。需要：
1. 可复现的部署流程
2. Server 和 Worker 可以分离部署
3. 本地开发一键启动

## 决策

使用 **单 Go 二进制 + Docker 镜像分离 + Docker Compose 编排**：

### 单二进制双模式

同一个 `./anchor` 二进制通过命令行标志区分：

```bash
# Server 模式（默认）
./anchor --no-local-worker

# Worker 模式
./anchor --worker --core-url http://server:17421
```

### Docker 镜像分离

- `Dockerfile.server` — 轻量镜像，只包含 anchor 二进制
- `Dockerfile.worker` — 完整镜像，包含 subfinder/naabu/httpx/nuclei + libpcap-dev

### Docker Compose 编排

```yaml
# docker-compose.yml
services:
  server:
    ports: ["17421:17421"]
    command: ["--no-local-worker"]
  worker:
    command: ["--worker", "--core-url", "http://server:17421"]
    profiles: ["worker"]
```

### Makefile 封装

```bash
make up         # 启动 Server
make up-all     # 启动 Server + Worker + Rangefield
make down-all   # 停止全部
```

## 理由

1. **同一二进制减少维护**：Server 和 Worker 共享同一份代码，Bug 修复只需发一个版本
2. **Docker 隔离环境**：安全工具依赖复杂（libpcap、Chromium），Docker 保证一致性
3. **Compose 简化开发**：`make up-all` 一键启动完整环境
4. **网络内置**：同 compose 网络的容器通过 service 名 DNS 互通

## 后果

- **镜像体积**：Worker 镜像较大（含 nuclei-templates）
- **数据持久化**：通过 Docker volume `anchor-data` 共享，跨主机部署需改为 HTTP 传输
- **网络配置**：Worker → Server 使用 service 名（如 `http://server:17421`），非 localhost

## 相关文件

- `Dockerfile.server`
- `Dockerfile.worker`
- `docker-compose.yml`
- `Makefile`

# CI/CD 流程指南

## 概述

Anchor 使用 GitHub Actions 实现自动化构建和部署。代码推送到 GitHub 后，通过 tag 触发 Release，自动构建二进制文件和 Docker 镜像，并推送到阿里云 ACR。

## 流程图

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

## 发布新版本

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
   - 等待 Release workflow 完成
   - 构建 3 个 Docker 镜像：
     - `anchor-server`
     - `anchor-worker`
     - `anchor-frontend`
   - 推送到阿里云 ACR

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

- `latest`: 最新稳定版本
- `v0.x.x`: 特定版本（暂未实现）

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

### Server 镜像

```dockerfile
# 基础镜像
FROM anchor-server-runtime-base:latest

# 从 GitHub Release 下载预编译二进制
RUN curl -fsSL -o /app/anchor "${RELEASE_URL}"

# 内置模板
RUN curl -fsSL -o /tmp/templates.tar.gz "..."
```

### Worker 镜像

```dockerfile
# 基础镜像（预装所有安全工具）
FROM anchor-worker-base:latest

# 从 GitHub Release 下载预编译二进制
RUN curl -fsSL -o /app/anchor "${RELEASE_URL}"
```

### Worker Base 镜像

Worker Base 镜像包含所有依赖，很少更新：

```bash
# 本地构建（需要时手动执行）
make build-worker-base

# 推送到 Docker Hub
make push-worker-base

# 推送到阿里云 ACR
make push-worker-base-cn
```

## 常见问题

### Q: 如何更新 Worker Base 镜像？

A: Worker Base 镜像包含系统依赖和安全工具，很少需要更新：

```bash
# 1. 修改 Dockerfile.worker-base
# 2. 本地构建并测试
make build-worker-base

# 3. 推送到 Docker Hub
make push-worker-base

# 4. 推送到阿里云 ACR
make push-worker-base-cn
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
3. **测试优先**: 推送前确保本地测试通过
4. **文档同步**: 修改代码时同步更新文档

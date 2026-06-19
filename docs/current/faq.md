---
status: active
source_of_truth: true
owner: kun
last_updated: 2026-06-17
---

# Anchor 常见问题（FAQ）

> 收录开发者与运维人员在使用 Anchor 过程中最常遇到的问题与解决方案。

---

## 目录

- [一、扫描配置](#一扫描配置)
- [二、部署与网络](#二部署与网络)
- [三、API 使用](#三api-使用)
- [四、故障排查](#四故障排查)
- [五、架构概念](#五架构概念)

---

## 一、扫描配置

### Q: 扫描完成了但没有发现任何资产/漏洞，为什么？

**常见原因：**

1. **端口范围不对**：默认使用 `top100`，Redis (6379)、MongoDB (27017) 等常见服务端口不在其中。改用 `high-risk` 或手动指定端口。
2. **被动搜索引擎未配置 API Key**：FOFA / Hunter / Quake 需要在「引擎凭证」页面（`/engines/keys`）配置各自的 API Key。未配置时被动搜索静默跳过，不报错。
3. **目标输入方式**：所有目标统一作为 `AssetSubdomain` 种子资产注入，由 `DeriveEligibleWorks()` 自动派生后续动作。确保目标格式正确（域名、IP 或 company 名称）。
4. **扫描引擎未启用**：确认 Server 环境变量 `ANCHOR_SCAN_ENGINE=1` 已设置。
5. **Nuclei 指纹要求**：外网模式下 `nuclei_require_fingerprint` 默认为 `true`，如果 httpx 未获取到指纹，Nuclei 扫描会被跳过。

### Q: 端口范围怎么选？

| 值 | 说明 | 适用场景 |
|---|---|---|
| `top100`（默认） | Top 100 常用端口 | 快速外网扫描 |
| `top1000` | Top 1000 端口 | 更全面的外网扫描 |
| `high-risk` | 高危端口（Redis 6379、MongoDB 27017、ES 9200 等） | 专项安全检查 |
| `full` | 全端口 1-65535 | 内网资产深度清查 |
| `6379` 或 `80,443,8080` | 自定义端口/端口列表 | 指定端口专项扫描 |

> **注意**：端口配置必须放在 `config` 对象内（见 [API 使用](#q-api-请求返回-400-格式错误怎么办) 章节），不能放在请求体顶层。

### Q: Scope（作用域）规则是什么？

Anchor 采用**排除-only** 模型——所有目标默认在范围内，只支持添加 `exclude`（排除）规则。`include` 动作已废弃。

```bash
# 排除特定域名
curl -s -X POST http://localhost:17421/scope-rules \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"<ID>","action":"exclude","type":"domain","value":"admin.example.com"}'
```

这意味着不需要"白名单"——添加目标即为扫描范围，想排除子域名或 IP 用排除规则。

### Q: Nuclei 扫描策略（tags / workflow / both）怎么选？

| 模式 | 说明 | 推荐场景 |
|------|------|---------|
| `tags`（默认） | 按 httpx 指纹标签匹配模板，覆盖广 | 日常扫描 |
| `workflow` | 使用预定义 workflow 串联检测链 | 精确检测特定漏洞 |
| `both` | workflow + tags 双重检测 | 最大覆盖、审计级扫描 |

在前端扫描弹窗 Step 2「Nuclei 扫描策略」面板中切换。模板来源为 RBKD-SEC/RBKD-templates 仓库。

### Q: 扫描太慢或服务器卡死？

可能是资源不足导致 Resource Governor 排队。默认配置：

| 阈值 | 默认值 | 行为 |
|------|--------|------|
| `ANCHOR_GOVERNOR_MEM_PCT` | 85% | 内存超阈值 → 新任务排队等待 |
| `ANCHOR_GOVERNOR_CPU_PCT` | 80% | CPU 超阈值 → 任务延迟 500ms |

可调高阈值或关闭 Governor（`ANCHOR_GOVERNOR_ENABLED=false`），但需自行承担资源风险。也可以降低 Nuclei 并发（`nuclei_concurrency`）和速率（`nuclei_rate_limit`）。

---

## 二、部署与网络

### Q: 怎么安装 Anchor？

```bash
bash install.sh
```

安装向导会：
1. 检测 Docker 环境
2. 选择部署模式（Server Only / Worker Only / Server+Worker）
3. 配置端口与 API Token
4. 从阿里云 ACR 拉取预构建镜像
5. 启动容器并等待健康检查

完成后浏览器访问 `http://localhost`（或配置的端口）。

### Q: 三种部署模式有什么区别？

| 模式 | Compose 文件 | `ANCHOR_MODE` | 适用场景 |
|------|-------------|---------------|---------|
| Server+Worker | `docker-compose.yml` | `server_worker`（默认） | 同机完整部署 |
| Server Only | `docker-compose.server.yml` | `server` | VPS 只跑管理面，Worker 远程连接 |
| Worker Only | `docker-compose.worker.yml` | `worker` | 远程扫描节点，连接已有 Server |

`.env` 中 `ANCHOR_MODE` 决定 `install.sh restart` 使用哪个 compose 文件。

### Q: Worker 怎么连接远程 Server？

Worker 只需**出站**连接 Server，无需公网 IP。在 `.env` 中设置：

```bash
ANCHOR_MODE=worker
ANCHOR_CORE_URL=https://your-server-domain.com
ANCHOR_API_TOKEN=your-shared-token
```

Worker 通过 `--core-url` 长轮询拉取任务、发送心跳、上报结果。

> **UI 实时日志**（可选）：如需在前端实时查看任务 stdout/stderr，Server 需能访问 Worker 的 endpoint（通过内网 IP、Tailscale 等）。不看实时日志则 Worker 只需能访问 Server 即可。

### Q: API Token 怎么管理？

- 安装时写入 `.env` 的 `ANCHOR_API_TOKEN`
- 安装向导**不会**在终端打印完整 Token，请从 `.env` 读取
- 轮换 Token：编辑 `.env` → `bash install.sh restart`（或 `make down && make up`）

### Q: Docker 常用管理命令？

```bash
# install.sh 系列
bash install.sh status    # 容器状态
bash install.sh logs      # 实时日志
bash install.sh restart   # 重启（Token 变更后需执行）
bash install.sh down      # 停止服务

# Makefile 系列（需已配置 .env）
make up          # pull + docker-compose up
make down        # 停止
make up-server   # 仅 Server + Frontend
make up-worker   # 仅 Worker
```

### Q: 镜像拉取失败 / 网络问题？

默认镜像来自阿里云 ACR（国内网络友好）：

```
crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com/p0m32kun/
```

三个镜像：`anchor-server`、`anchor-worker`、`anchor-frontend`。

如果拉取失败，检查：
1. Docker 是否正常运行（`docker info`）
2. 网络是否能访问 `crpi-wthv8jhah5ufmzlr.cn-hangzhou.personal.cr.aliyuncs.com`
3. 是否需要配置 Docker 镜像代理

---

## 三、API 使用

### Q: API 认证方式？

所有需要认证的 API 使用 Bearer Token：

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:17421/...
```

Token 值取自 `.env` 中的 `ANCHOR_API_TOKEN`。

> `/health` 端点不需要认证（用于容器健康检查）。

### Q: 一个完整的 API 调用流程？

```bash
TOKEN="your-anchor-api-token"
BASE="http://localhost:17421"

# 1. 创建项目
PROJECT=$(curl -s -X POST $BASE/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","organization":"Org","purpose":"Test"}')
PROJECT_ID=$(echo $PROJECT | jq -r '.id')

# 2. 添加目标
curl -s -X POST "$BASE/projects/$PROJECT_ID/targets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"domain","value":"example.com"}'

# 3. 启动扫描
SCAN=$(curl -s -X POST "$BASE/projects/$PROJECT_ID/scan" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"mode":"external","config":{"port_range":"top100","enable_nuclei":true}}')
RUN_ID=$(echo $SCAN | jq -r '.run_id')

# 4. 轮询状态
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE/projects/$PROJECT_ID/pipeline/runs/$RUN_ID"

# 5. 获取结果
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE/projects/$PROJECT_ID/assets" | jq .
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE/projects/$PROJECT_ID/findings" | jq .

# 6. 导出报告
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE/projects/$PROJECT_ID/reports/export.md"
```

### Q: API 请求返回 400 / 格式错误怎么办？

确保端口配置在 `config` 对象内：

```json
// ✅ 正确
{
  "mode": "external",
  "config": {
    "port_range": "6379",
    "enable_nuclei": true
  }
}

// ❌ 错误（port_range 放在顶层）
{
  "mode": "external",
  "port_range": "6379"
}
```

### Q: 怎么查看扫描进度和运行指标？

```bash
# 运行状态（running / completed / failed / cancelled）
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE/projects/$PROJECT_ID/pipeline/runs/$RUN_ID"

# 运行指标（引擎状态 + Work 计数）
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE/projects/$PROJECT_ID/pipeline/runs/$RUN_ID/metrics"

# Work 明细（每个 资产×动作 的状态）
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE/projects/$PROJECT_ID/pipeline/runs/$RUN_ID/works"
```

`metrics` 返回示例：

```json
{
  "engine_state": "running",
  "assets_discovered": 12,
  "works_pending": 3,
  "works_running": 2,
  "works_done": 18,
  "works_failed": 0
}
```

---

## 四、故障排查

### Q: 怎么查看日志？

```bash
# install.sh 方式
bash install.sh logs

# Docker 方式
docker logs anchor-server --tail 200 -f   # Server 日志
docker logs anchor-worker --tail 200 -f   # Worker 日志
```

### Q: 怎么做健康检查？

**Server 健康检查**（无需认证）：

```bash
curl http://localhost:17421/health
```

**Worker 健康检查**：

```bash
# Worker 端口默认 17422（同机 compose 中通过 Docker 网络访问）
curl http://worker:17422/health
```

**工具可用性检查**（需认证）：

```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:17421/health/tools
```

### Q: Server 启动后前端白屏？

1. 确认 `anchor-frontend` 容器正在运行：`docker ps | grep frontend`
2. 检查 Nginx 日志：`docker logs anchor-frontend`
3. 前端通过 Nginx 反代 `/api/` 到 `server:17421`，确认 Server 容器健康

### Q: Worker 状态一直显示 offline？

1. 检查 Worker 能否访问 Server：`curl http://server:17421/health`
2. 检查 `ANCHOR_CORE_URL` 是否正确指向 Server
3. 检查 `ANCHOR_API_TOKEN` 是否与 Server 一致
4. 查看 Worker 日志：`docker logs anchor-worker`

### Q: 扫描失败（status=failed）？

1. 查看 Worker 日志定位具体错误
2. 常见原因：
   - 目标不可达（网络限制）
   - 扫描工具未就绪（`/health/tools` 检查）
   - Worker 并发已满（默认 `ANCHOR_WORKER_MAX_CONCURRENCY=10`），返回 503 后 Server 会自动改派到其他 Worker
   - Resource Governor 内存/CPU 超限阻塞

### Q: 内置资源（RBKD 模板/字典/指纹）没有同步？

1. 确认容器能访问 GitHub（`github.com`）
2. 同步失败为 fail-soft，不会阻止启动，但会使用上次落盘的缓存
3. 可通过 `ANCHOR_BUILTIN_SYNC=off` 手动关闭同步
4. 同步产物路径：
   - 字典：`/opt/dict`
   - Nuclei 模板：`/opt/rbkd-templates`
   - httpx 指纹：`/opt/finger/finger.json`

### Q: 数据存在哪里？备份怎么做？

- **Server 数据**：`anchor-server-data` Docker named volume，包含 SQLite 数据库
- **Worker 数据**：`anchor-worker-data` Docker named volume（独立于 Server）
- 备份：`docker run --rm -v anchor-server-data:/data -v $(pwd):/backup alpine tar czf /backup/anchor-data-backup.tar.gz /data`

---

## 五、架构概念

### Q: 资产驱动模型 vs 管线阶段模型？

**Anchor 的扫描执行是资产驱动模型，不是管线阶段模型。**

不存在固定的 P1→P2→P3→P4→P5 执行顺序。核心循环：

```
发现资产 → DeriveEligibleWorks(资产类型) → 派生 Work(资产×动作) → 执行工具 → 输出解析 → 发现新资产 → 循环
```

每个资产类型可派生的动作：

| 资产类型 | 派生的动作 |
|---------|-----------|
| Subdomain | 子域枚举、DNS 解析、CDN 检测 |
| IP | DNS 解析、CDN 检测、端口扫描 |
| IP+Port | 服务指纹 |
| HTTP Service | httpx 指纹、Katana 爬虫、ffuf 目录扫描、Nuclei 漏洞扫描 |
| HTTP Path | Katana 爬虫、Nuclei 扫描、Spoor JS 分析 |

> API 中 `pipeline_runs.stage` 和 `pipeline_run_stages` 只是 UI 投影（进度展示），**不代表**执行顺序。

### Q: 收敛机制（idle_timeout / wind_down）是什么？

扫描不会无限运行。收敛状态机：

```
running → wind_down → stopped
```

- **idle_timeout**：一段时间没有发现新资产后，进入 `wind_down`
- **wind_down**：仅允许 Nuclei/httpx 等低噪声动作执行，不再做主动发现
- **stopped**：所有 Work 完成，扫描结束

### Q: 排除-only Scope 是什么意思？

Anchor 没有白名单机制。添加目标 = 扫描范围内。如果需要排除特定子域名或 IP，使用排除规则（`action: "exclude"`）。

`include` 动作已废弃。这简化了范围管理——你只需要关注「什么不想扫」。

### Q: Worker 调度策略是什么？

Server 侧调度策略：

1. **最少负载优先**：`load = DB running tasks + server in-flight 计数`
2. **同负载轮询**：round-robin
3. **容量上限**：尊重 `worker_nodes.max_concurrency`（默认 10）
4. **故障转移**：不可达时标记 offline 并尝试下一个 Worker
5. **503 改派**：Worker 满载返回 503，Server 自动改派而不标记 offline

### Q: 前端怎么访问 API？

前端通过 Nginx 反向代理自动转发 `/api/` → `server:17421`，无需手动配置 API 地址。前端默认 `apiBase="/api"`。

直接 API 调用使用 Server 端口 `17421`。

### Q: 支持什么平台？

Docker 镜像支持 `linux/amd64`（VPS/PC）和 `linux/arm64`（Mac M1/M2/M3）。拉取时自动匹配宿主机架构。

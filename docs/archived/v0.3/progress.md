---
archived: true
version: "0.3"
status: completed
reason: "v0.3 阶段已结束"
archived_date: "2026-05-01"
---

# Progress

## Status
Completed — v0.3 桌面可用性与可靠性阶段结束（2026-05-01）

## Completed

### Sprint — Token 认证 & 远程部署支持
- **Backend**: API Token 认证中间件 (`TokenAuthMiddleware`)
  - 所有路由（除 `/health`）强制 Bearer Token 验证
  - Token 来源：`ANCHOR_API_TOKEN` 环境变量，或自动生成的随机 32 字节 hex
  - Worker 注册取消独立 token，改用全局 API Token
  - CORS 开放为动态 Origin（支持任意前端地址）
- **Frontend**: API Base + Token 配置流程
  - 首次启动检测 `needsApiBaseConfig()` / `needsApiToken()`，显示配置向导
  - `ApiBaseSetup` 组件：URL 验证 + Token 测试 + 自动保存到 localStorage
  - `AppHealthCheck` 增强：401 诊断 + 网络诊断按钮 + 重置配置入口
  - `SettingsPage` 支持 Token 修改
  - `api.ts` 统一注入 `Authorization: Bearer` Header，导出 `request()` 供页面直接使用
- **Worker**: RemoteClient 适配 Token 认证
  - `NewRemoteClient` 接收 `apiToken` 参数
  - 注册/轮询/上报结果均携带 Bearer Token
  - 远程模式要求 `ANCHOR_API_TOKEN` 环境变量
- **Docker**: 拆分 compose 配置
  - `docker-compose.yml` — 单机全栈（server + worker）
  - `docker-compose.server.yml` — 纯 Server
  - `docker-compose.worker.yml` — 纯 Worker（需指定 `ANCHOR_CORE_URL`）
  - `Makefile` 新增 `up-server` / `up-worker` / `down-server` / `down-worker` / `restart-worker` 等目标
- **Worker 管理**: 离线 Worker 清理
  - 后端 `cleanupStaleWorkers()` 自动删除离线 7 天的 Worker
  - 前端 WorkersPage 支持单个删除 + 一键批量清理
  - 新增 `DELETE /workers/{id}` API
- **Bug 修复**
  - ProjectPage 删除按钮与 '>' 箭头重叠
  - `/projects/:projectId` index 路由重定向到 targets，修复空白页

### Code Simplification
- 提取 `ApiBaseSetup` → `frontend/src/components/ApiBaseSetup.tsx`
- 提取 `validateSetupInput()` 验证函数，分离校验与网络逻辑
- 提取 `WorkersPage` 内联删除逻辑 → `handleDeleteWorker()` / `handleBulkDelete()`
- 移除 `App.tsx` 生产环境调试 `console.log`

## Files Changed
- `internal/api/handlers.go` — Token 认证中间件 + CORS 简化 + 路由保护
- `internal/api/worker_handlers.go` — 移除独立 Worker token；新增 `handleDeleteWorker`
- `internal/db/queries.go` — 新增 `DeleteWorkerNode`
- `internal/worker/remote_client.go` — Bearer Token 注入所有请求
- `internal/scope/import.go` — `detectType` → `DetectType` (export)
- `main.go` — Worker 远程模式强制 `ANCHOR_API_TOKEN`
- `frontend/src/App.tsx` — 健康检查增强 + ApiBaseSetup 集成
- `frontend/src/components/ApiBaseSetup.tsx` — 新增（提取自 App.tsx）
- `frontend/src/lib/api.ts` — Bearer Header + `request` 导出 + JSON fallback
- `frontend/src/lib/config.ts` — Token 读写 + `needsApiBaseConfig` / `needsApiToken`
- `frontend/src/pages/WorkersPage.tsx` — 删除 Worker UI + 一键清理
- `frontend/src/pages/SettingsPage.tsx` — Token 配置 UI
- `frontend/src/pages/ProjectPage.tsx` — 删除按钮布局修复
- `Dockerfile.server` — 移除默认 `--no-local-worker`
- `docker-compose.yml` / `docker-compose.server.yml` / `docker-compose.worker.yml` — 拆分部署
- `Makefile` — 多模式生命周期管理

### v0.3 最终交付 — 网络服务扫描与指纹增强
- **网络服务扫描**：Web 初筛工作流新增非 Web 端口扫描（Redis/MySQL/PostgreSQL/Elasticsearch/MongoDB/Memcached/MSSQL/Oracle）
  - `internal/nuclei/tagmapper.go` — `MapPortToTag` + `GroupPortsByTags` + `PortTarget` 类型
  - `internal/workflow/screenshot.go` — `runNetworkServiceScan()` 集成到 `WebScreeningWorkflow`
  - 按服务标签分组执行 Nuclei，Finding 去重/更新逻辑与 Web 端一致
- **CPE 指纹补充**：httpx 解析器从 CPE 字段提取 product name 作为 tech fallback
  - `internal/parser/httpx.go` — CPE 解析 + 去重合并
  - 解决 404/302 页面 tech 为空导致跳过扫描的问题
- **httpx 增强**：`BuildHttpxCommand` 添加 `-follow-redirects`，提升重定向场景覆盖率
- **靶场修复**：`docker-rangefield/apps/tomcat-vuln/Dockerfile` 适配新版 Tomcat 镜像（webapps.dist 复制、RemoteCIDRValve 放开、移除 LockOutRealm）

## Verification
- `go build ./...` ✅
- `go vet ./...` ✅
- `cd frontend && npx tsc --noEmit` ✅
- `go test ./...` ✅（新增 4 个单元测试全部通过）

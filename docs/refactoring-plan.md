---
status: backlog
source_of_truth: false
owner: kun
last_updated: 2026-05-04
scope: engineering-refactor
---

# Anchor 项目重构计划

> 文档角色：重构候选方案与执行记录，不是当前仓库级计划。

## 背景

当前项目结构存在以下问题，导致迭代困难、bug 定位慢：

1. **handlers.go 1586 行** — 所有 API 逻辑集中在一个文件
2. **parser vs parsers 重复** — 两个包做同样的事，未完成迁移
3. **Server 结构体职责过多** — 7 个不相关的职责耦合在一起
4. **缺少 Service 层** — 业务逻辑和 HTTP 逻辑混在一起
5. **Worker 管理散落各处** — 3 个文件管同一件事

## 依赖关系图（当前）

```
main.go
├── internal/api (2750 行)
│   ├── internal/db
│   ├── internal/errors
│   ├── internal/health
│   ├── internal/models
│   ├── internal/report
│   ├── internal/scope
│   ├── internal/util
│   ├── internal/worker
│   └── internal/workflow
├── internal/db
│   └── internal/models
└── internal/worker
    ├── internal/db
    ├── internal/models
    ├── internal/scope
    └── internal/util

internal/workflow (1970 行)
├── internal/asset
├── internal/cdn
├── internal/db
├── internal/fingerprint
├── internal/models
├── internal/nuclei
├── internal/parser ← 旧版
├── internal/parsers ← 新版
├── internal/resolve
├── internal/scope
├── internal/scoring
├── internal/search
├── internal/util
└── internal/worker
```

## 重构原则

1. **渐进式** — 每个阶段独立可验证，测试必须全部通过
2. **向后兼容** — 不改变外部 API 接口
3. **AI 友好** — 文件小、职责单一、命名清晰
4. **可测试** — 引入接口，便于 mock 和单元测试

---

## 阶段 1：拆分 handlers.go（立即见效） ✅ 已完成

**目标**：将 1586 行的 handlers.go 按功能拆分为独立文件

**结果**：handlers.go 从 1586 行拆分为 12 个文件，最大文件 294 行

### 步骤

1. 创建 `internal/api/middleware.go`
   - `CORSMiddleware`
   - `TokenAuthMiddleware`
   - `writeError`, `writeJSON` helpers

2. 创建 `internal/api/server.go`
   - `Server` 结构体定义
   - `NewServer` 构造函数
   - `Register` 路由注册
   - Worker 清理相关方法

3. 创建 `internal/api/project_handlers.go`
   - `handleCreateProject`
   - `handleListProjects`
   - `handleGetProject`
   - `handleDeleteProject`

4. 创建 `internal/api/target_handlers.go`
   - `handleCreateTarget`
   - `handleListTargets`
   - `handleImportTargets`

5. 创建 `internal/api/scope_handlers.go`
   - `handleCreateScopeRule`
   - `handleListScopeRules`
   - `handleBatchCreateScopeRules`

6. 创建 `internal/api/finding_handlers.go`
   - `handleListFindings`
   - `handleGetFinding`
   - `handlePatchFindingStatus`
   - `handleAddEvidence`
   - `handleBatchUpdateFindingStatus`
   - `handleGetFindingCurl`

7. 创建 `internal/api/report_handlers.go`
   - `handleExportReportMD`
   - `handleExportReportJSON`

8. 创建 `internal/api/pipeline_handlers.go`
   - `handleRunPipeline`
   - `handleListPipelineRuns`
   - `handleGetPipelineRun`
   - `handleGetPipelineConfig`
   - `handleUpdatePipelineConfig`

9. 创建 `internal/api/sse.go`
   - `handleProjectSSE`
   - `broadcastProjectSSE`
   - SSE 客户端管理

10. 保留 `internal/api/handlers.go` 仅包含杂项
    - `handleHealth`
    - `handleToolHealth`
    - `handleListWorkers`
    - `handleListToolTemplates`
    - `handleGetToolTemplate`

**验证**：`go build ./...` + `go test ./...` 全部通过

---

## 阶段 2：统一 parser 包 ✅ 已完成

**目标**：删除重复的 `internal/parsers`，统一使用 `internal/parser`

**结果**：删除了 `internal/parsers` 包，将所有功能合并到 `internal/parser`

### 步骤

1. 将 `parser.ParseNuclei` 迁移到 `parsers` 包
   - 创建 `internal/parsers/nuclei.go`
   - 保持相同的函数签名

2. 更新引用
   - `internal/workflow/pipeline.go` — 改用 `parsers.ParseNuclei`
   - `internal/workflow/discovery.go` — 改用 `parsers.Parse*`
   - `internal/workflow/screenshot.go` — 改用 `parsers.Parse*`
   - `internal/scoring/scoring.go` — 改用 `parsers.Parse*`

3. 删除 `internal/parser/` 目录

4. 重命名 `internal/parsers` → `internal/parser`（可选，更简洁）

**验证**：`go build ./...` + `go test ./...` 全部通过

---

## 阶段 3：引入 Service 层 ✅ 已完成

**目标**：将业务逻辑从 Handler 中解耦，提高可测试性

**结果**：创建了 `internal/service/` 包，实现了 ProjectService、TargetService、FindingService

### 设计

```go
// internal/service/project.go
type ProjectService interface {
    Create(ctx context.Context, req CreateProjectRequest) (*models.Project, error)
    List(ctx context.Context) ([]*models.Project, error)
    Get(ctx context.Context, id string) (*models.Project, error)
    Delete(ctx context.Context, id string) error
}
```

### 步骤

1. 创建 `internal/service/` 目录

2. 实现 `ProjectService`
   - `Create`, `List`, `Get`, `Delete`
   - 从 handlers.go 提取业务逻辑

3. 实现 `TargetService`
   - `Create`, `List`, `Import`
   - 包含 scope 检查逻辑

4. 实现 `FindingService`
   - `List`, `Get`, `UpdateStatus`, `AddEvidence`
   - 包含状态验证逻辑

5. 实现 `PipelineService`
   - `Run`, `ListRuns`, `GetRun`, `UpdateConfig`
   - 包含流水线编排逻辑

6. 重构 Handler 只做 HTTP 解析和响应
   ```go
   func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
       var req service.CreateProjectRequest
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           writeError(w, http.StatusBadRequest, ...)
           return
       }
       project, err := s.projectSvc.Create(r.Context(), req)
       if err != nil {
           // 处理业务错误
           return
       }
       writeJSON(w, http.StatusCreated, project)
   }
   ```

**验证**：为每个 Service 编写单元测试

---

## 阶段 4：整理 Worker 管理 ✅ 已完成

**目标**：将 Worker 生命周期管理统一到一个地方

**结果**：将 Worker 代码拆分为 4 个清晰的文件

### 设计

```
internal/worker/
├── runner.go       # 任务执行
├── commands.go     # 命令构建
├── dispatcher.go   # 远程 Worker 分发
├── server.go       # Worker HTTP 服务
└── remote.go       # 远程 Worker 客户端
```

### 步骤

1. 创建 `internal/worker/commands.go`
   - 将所有 `Build*Command` 函数移到这里
   - 将 `appendRateLimitArgs` 函数移到这里

2. 创建 `internal/worker/dispatcher.go`
   - `Dispatcher` 结构体
   - `PickOnlineWorker`, `DispatchToWorker` 方法
   - 从 Runner 中分离远程分发逻辑

3. 更新 `Runner` 结构体
   - 添加 `dispatcher *Dispatcher` 字段
   - 移除 `httpClient` 字段
   - 使用 `dispatcher` 进行远程分发

**验证**：Worker 注册、心跳、任务分发功能正常

---

## 阶段 5：引入接口提高可测试性 ✅ 已完成

**目标**：定义接口，便于 mock 测试

**结果**：创建了 Repository 接口、Mock 实现和单元测试

### 设计

```go
// internal/service/interfaces.go
type ProjectRepository interface {
    Create(project *models.Project) error
    Get(id string) (*models.Project, error)
    List() ([]*models.Project, error)
    Delete(id string) error
}

type FindingRepository interface {
    Get(id string) (*models.Finding, error)
    ListByProject(projectID string) ([]*models.Finding, error)
    ListByStatus(projectID string, status models.FindingStatus) ([]*models.Finding, error)
    UpdateStatus(id string, status models.FindingStatus) error
    CreateEvidence(evidence *models.Evidence) error
    ListEvidenceByFinding(findingID string) ([]*models.Evidence, error)
}
```

### 步骤

1. 创建 `internal/service/interfaces.go`
   - `ProjectRepository`, `TargetRepository`, `FindingRepository`
   - `ScopeChecker`, `AuditLogger`

2. 创建 `internal/service/mock_test.go`
   - `MockProjectRepository`, `MockTargetRepository`, `MockFindingRepository`
   - `MockScopeChecker`, `MockAuditLogger`

3. 创建 `internal/service/adapter.go`
   - `projectRepoAdapter`, `targetRepoAdapter`, `findingRepoAdapter`
   - `auditLogAdapter`
   - 便捷构造函数 `NewProjectService`, `NewFindingService`

4. 更新 Service 实现
   - `projectService` 使用 `ProjectRepository` 接口
   - `findingService` 使用 `FindingRepository` 接口

5. 创建 `internal/service/project_test.go`
   - 测试 Create, Get, List, Delete
   - 测试错误场景（空名称、负 rate limit、不存在的项目）

**验证**：Service 层单元测试 11 个用例全部通过

---

## 目标结构

```
Anchor/
├── main.go
├── internal/
│   ├── api/                    # HTTP 层（薄层）
│   │   ├── server.go           # Server + 路由注册
│   │   ├── middleware.go       # CORS, Auth
│   │   ├── sse.go             # SSE 推送
│   │   ├── project_handler.go # 项目 API
│   │   ├── target_handler.go  # 目标 API
│   │   ├── finding_handler.go # 漏洞 API
│   │   ├── pipeline_handler.go # 流水线 API
│   │   ├── worker_handler.go  # Worker API
│   │   └── handler_test.go    # Handler 测试
│   ├── service/                # 业务逻辑层
│   │   ├── project.go
│   │   ├── target.go
│   │   ├── finding.go
│   │   ├── pipeline.go
│   │   └── service_test.go
│   ├── worker/                 # Worker 管理
│   │   ├── runner.go           # 任务执行
│   │   ├── manager.go          # 生命周期管理
│   │   ├── server.go           # Worker HTTP 服务
│   │   └── remote.go           # 远程 Worker
│   ├── workflow/               # 工作流编排
│   │   ├── pipeline.go         # 流水线
│   │   ├── discovery.go        # 资产发现
│   │   └── screening.go        # Web 筛查
│   ├── parser/                 # 工具输出解析（统一）
│   │   ├── subfinder.go
│   │   ├── httpx.go
│   │   ├── naabu.go
│   │   └── nuclei.go
│   ├── db/                     # 数据访问
│   ├── models/                 # 数据模型
│   ├── scope/                  # 授权范围
│   ├── asset/                  # 资产管理
│   ├── report/                 # 报告生成
│   ├── health/                 # 健康检查
│   └── util/                   # 工具函数
├── frontend/                   # 前端
└── docs/                       # 文档
```

---

## 执行顺序

| 阶段                | 预计工时 | 风险 | 依赖   |
| ------------------- | -------- | ---- | ------ |
| 1. 拆分 handlers.go | 2-3 小时 | 低   | 无     |
| 2. 统一 parser      | 1 小时   | 低   | 阶段 1 |
| 3. 引入 Service 层  | 4-6 小时 | 中   | 阶段 1 |
| 4. 整理 Worker      | 2-3 小时 | 中   | 阶段 3 |
| 5. 引入接口         | 3-4 小时 | 低   | 阶段 3 |

**总计**：12-17 小时

---

## 验证清单

每个阶段完成后必须验证：

- [x] `go build ./...` 编译通过
- [x] `go test ./...` 全部通过（214 个测试）
- [x] `go vet ./...` 无警告
- [ ] 手动测试核心功能正常
- [x] 更新本文档的进度

---

---

## 阶段 6：上帝文件清理与重复消除 ✅ 已完成（2026-05-08）

**目标**：消除项目中过度复杂的设计、重复逻辑和代码异味，防止项目变成屎山。

**结果**：11 项清理全部完成，编译零错误，测试全绿。

### 后端清理

| 项 | 修改 | 代码变化 |
|---|---|---|
| parser 泛型提取 | `parseJSONLines[T]` + 5 个解析器重构 | -114 行 |
| API 错误处理抽象 | `handleServiceError` helper | -77 行，11 处重复消除 |
| asset handler 性能 | `ListPortsByProject` + `sort.Slice` | N+1 消除，O(n²)→O(n log n) |
| 搜索引擎 HTTP 抽象 | `baseClient.doJSON()` | 3 个引擎共享 |
| models.go 拆分 | 14 个 domain 文件 | 957 行 → 删除 |
| queries.go 拆分 | 10 个 domain 文件 | 2065 行 → 58 行 |
| db.go 拆分 | 15 个 migration 文件（v1~v13） | 1182 行 → db.go 173 行 |
| pipeline.go 拆分 | 5 个职责文件 | 1183 行 → pipeline.go 200 行 |

### 前端清理

| 项 | 修改 |
|---|---|
| useResource hook | 通用数据加载 hook，封装 AbortController + loading + error |
| 4 个页面重构 | TargetPage/AssetPage/RunsPage/FindingsPage 使用 useResource |
| 编译错误修复 | navigate 导入、Users 图标、15 个连带编译问题 |

### 验证

- `go build ./...` ✅ 零错误
- `go test ./internal/...` ✅ 全绿
- `npm run build` ✅ 通过

---

## 风险控制

1. **每个阶段独立提交** — 出问题可回滚
2. **不改变外部 API** — 前端无需改动
3. **保持测试通过** — 每个步骤后运行测试
4. **小步快跑** — 不要一次改太多

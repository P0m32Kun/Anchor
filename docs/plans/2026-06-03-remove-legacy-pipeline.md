# 清除旧流水线扫描模式 实施计划

> 基于调研报告：2026-06-03 清除旧流水线扫描模式

**目标**：完全移除旧的流水线扫描模式（Legacy Pipeline），只保留资产驱动扫描引擎

**架构**：将 `internal/workflow/` 中的独立工作流迁移到新包 `internal/workflows/`，然后删除整个 `workflow` 包和所有 Pipeline 相关代码

**技术栈**：Go 1.26, SQLite, React 18/TypeScript

---

## 文件清单

| 操作 | 文件路径 | 说明 |
|------|---------|------|
| 新建 | `internal/workflows/discovery.go` | 从 workflow/discovery.go 迁移 |
| 新建 | `internal/workflows/screenshot.go` | 从 workflow/screenshot.go 迁移 |
| 修改 | `internal/api/workflow_handlers.go` | 更新 import 路径 |
| 修改 | `internal/api/run_handlers.go` | 更新 import 路径 |
| 修改 | `internal/api/pipeline_handlers.go` | 删除旧分支和 handleRunPipeline |
| 修改 | `internal/api/server.go` | 删除旧路由 |
| 删除 | `internal/workflow/*.go` | 删除整个包（18+ 个文件） |

---

## 任务 1：创建新包 `internal/workflows/` 并迁移文件

**文件**：
- 新建：`internal/workflows/discovery.go`
- 新建：`internal/workflows/screenshot.go`

- [ ] **步骤 1：创建 workflows 目录**

```bash
mkdir -p internal/workflows
```

- [ ] **步骤 2：复制 discovery.go 到新包**

读取 `internal/workflow/discovery.go` 完整内容，修改 package 名为 `workflows`，写入 `internal/workflows/discovery.go`。

关键变更：
```go
package workflows  // 原来是 package workflow
```

其余代码保持不变（AssetDiscoveryWorkflow struct 和所有方法）。

- [ ] **步骤 3：复制 screenshot.go 到新包**

读取 `internal/workflow/screenshot.go` 完整内容，修改 package 名为 `workflows`，写入 `internal/workflows/screenshot.go`。

关键变更：
```go
package workflows  // 原来是 package workflow
```

其余代码保持不变（WebScreeningWorkflow struct 和所有方法）。

- [ ] **步骤 4：验证新包编译通过**

```bash
go build ./internal/workflows/...
```

预期：编译成功

- [ ] **步骤 5：提交**

```bash
git add internal/workflows/
git commit -m "refactor: create workflows package with migrated discovery and screenshot"
```

---

## 任务 2：更新引用新包的 import 路径

**文件**：
- 修改：`internal/api/workflow_handlers.go:1-30`
- 修改：`internal/api/run_handlers.go:1-30`

- [ ] **步骤 1：修改 workflow_handlers.go 的 import**

将：
```go
import (
    // ...
    "github.com/P0m32Kun/Anchor/internal/workflow"
)
```

改为：
```go
import (
    // ...
    "github.com/P0m32Kun/Anchor/internal/workflows"
)
```

同时修改使用处（约第 25 行和第 56 行）：
```go
// 第 25 行
wf := workflows.NewAssetDiscoveryWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir)

// 第 56 行
wf := workflows.NewWebScreeningWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir)
```

- [ ] **步骤 2：修改 run_handlers.go 的 import**

将：
```go
import (
    // ...
    "github.com/P0m32Kun/Anchor/internal/workflow"
)
```

改为：
```go
import (
    // ...
    "github.com/P0m32Kun/Anchor/internal/workflows"
)
```

同时修改使用处（约第 83 行）：
```go
wf := workflows.NewAssetDiscoveryWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir).WithRunID(run.ID)
```

- [ ] **步骤 3：验证编译通过**

```bash
go build ./internal/api/...
```

预期：编译成功

- [ ] **步骤 4：运行测试**

```bash
go test ./internal/api/... -count=1 -timeout 30s
```

预期：测试通过

- [ ] **步骤 5：提交**

```bash
git add internal/api/workflow_handlers.go internal/api/run_handlers.go
git commit -m "refactor: update imports to use new workflows package"
```

---

## 任务 3：删除 pipeline_handlers.go 中的旧代码

**文件**：
- 修改：`internal/api/pipeline_handlers.go`

- [ ] **步骤 1：删除 handleRunPipeline 函数**

删除第 21-96 行的 `handleRunPipeline` 函数（旧的无条件 Pipeline 入口）。

- [ ] **步骤 2：删除 handleCreateScan 中的 legacy 分支**

在 `handleCreateScan` 函数中：

删除第 379 行：
```go
useLegacyPipeline := os.Getenv("ANCHOR_LEGACY_PIPELINE") == "1"
```

将第 400 行：
```go
if !useLegacyPipeline {
```

改为：
```go
// Asset-driven scan engine (the only execution model)
```

删除第 426-444 行的 else 分支：
```go
} else {
    // Legacy pipeline
    pipeline := workflow.NewPipeline(...)
    // ...
}
```

保留新引擎的代码（401-425 行）并调整缩进。

- [ ] **步骤 3：删除未使用的 import**

删除不再需要的 import：
```go
"github.com/P0m32Kun/Anchor/internal/workflow"
```

保留其他 import（`scanengine`, `core`, `evaluator` 等仍被使用）。

- [ ] **步骤 4：验证编译通过**

```bash
go build ./internal/api/...
```

预期：编译成功

- [ ] **步骤 5：运行测试**

```bash
go test ./internal/api/... -count=1 -timeout 30s
```

预期：测试通过

- [ ] **步骤 6：提交**

```bash
git add internal/api/pipeline_handlers.go
git commit -m "refactor: remove legacy pipeline branch from handleCreateScan"
```

---

## 任务 4：删除旧路由

**文件**：
- 修改：`internal/api/server.go:304`

- [ ] **步骤 1：删除旧路由注册**

删除第 304 行：
```go
mux.Handle("POST /projects/{id}/pipeline/run", auth(http.HandlerFunc(s.handleRunPipeline)))
```

- [ ] **步骤 2：验证编译通过**

```bash
go build ./internal/api/...
```

预期：编译成功

- [ ] **步骤 3：提交**

```bash
git add internal/api/server.go
git commit -m "refactor: remove legacy pipeline route POST /projects/{id}/pipeline/run"
```

---

## 任务 5：删除旧 workflow 包

**文件**：
- 删除：`internal/workflow/` 整个目录

- [ ] **步骤 1：确认新包已完全替代**

验证以下命令无输出（表示没有代码再引用旧包）：
```bash
grep -rn "internal/workflow" internal/ --include="*.go" | grep -v "internal/workflows"
```

预期：无输出

- [ ] **步骤 2：删除旧包**

```bash
rm -rf internal/workflow/
```

- [ ] **步骤 3：验证编译通过**

```bash
go build ./...
```

预期：编译成功（忽略 tmp-test 包的已知问题）

- [ ] **步骤 4：运行完整测试**

```bash
go test ./internal/... -count=1 -timeout 60s 2>&1 | grep -E "^(ok|FAIL)"
```

预期：所有包测试通过

- [ ] **步骤 5：提交**

```bash
git add -A internal/workflow/
git commit -m "refactor: remove legacy workflow package (Pipeline, StageEmitter, tool executors)"
```

---

## 任务 6：文档同步

**文件**：
- 修改：`docs/current/architecture.md`
- 修改：`README.md`

- [ ] **步骤 1：更新 architecture.md**

删除或更新以下章节：
- "Baseline Workflow" 中关于 Legacy Pipeline 的描述
- "多目标类型与 Company 目标自动展开" 中关于 `PipelineConfig.runFlow` 的说明
- "What Is Not Baseline Yet" 中如有相关内容

在 "资产驱动扫描引擎" 章节标注为唯一执行模型。

- [ ] **步骤 2：更新 README.md**

删除或更新功能清单中与旧 Pipeline 相关的描述。

- [ ] **步骤 3：提交**

```bash
git add docs/current/architecture.md README.md
git commit -m "docs: remove legacy pipeline references, mark asset-driven as sole execution model"
```

---

## 完成标准

- [ ] `internal/workflow/` 包已完全删除
- [ ] `internal/workflows/` 包包含迁移后的 discovery.go 和 screenshot.go
- [ ] 所有 import 路径已更新
- [ ] `handleRunPipeline` 和旧路由已删除
- [ ] `handleCreateScan` 不再有 legacy 分支
- [ ] `go build ./...` 编译通过
- [ ] `go test ./internal/...` 测试通过
- [ ] 文档已同步更新

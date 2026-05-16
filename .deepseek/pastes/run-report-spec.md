# Run 级别报告 — 实施规格

## 背景

当前 `findings` 表没有 `run_id`，导致 `AggregateByRunWithBatchEvidence` 虽然参数接收 `runID`，但实际聚合的是整个 project 的 findings，与 project 级别报告完全重复。

**目标**：让 run 报告真正按 run 范围过滤，同时保持 project 级别报告不变。

---

## 变更清单

### 1. 新文件: `internal/db/v23.go`

创建 DB migration v23，为 findings 表添加 `run_id` 列。

```go
package db

import (
	"database/sql"
	"fmt"
)

// migrateV23 adds run_id to findings so run-scoped reports can filter
// findings by pipeline run. The column is nullable — findings created
// by project-level workflows (asset discovery, web screening) will
// have NULL run_id and still appear in project-scoped reports.
func migrateV23(db *sql.DB) error {
	var colCount int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('findings') WHERE name = 'run_id'`,
	).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("check run_id column: %w", err)
	}
	if colCount > 0 {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE findings ADD COLUMN run_id TEXT REFERENCES runs(id) ON DELETE SET NULL`); err != nil {
		return fmt.Errorf("add run_id column: %w", err)
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_findings_run ON findings(run_id)`); err != nil {
		return fmt.Errorf("create run_id index: %w", err)
	}

	return nil
}
```

---

### 2. 修改: `internal/db/db.go`

在 migrate 函数末尾（v22 之后）注册 v23。

**当前代码**（约第 249-253 行）：
```go
		if err := migrateV22(db); err != nil {
			return fmt.Errorf("migrate v22 (nuclei_custom_sources.install_path): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 22"); err != nil {
			return fmt.Errorf("set user_version 22: %w", err)
```

**在其后追加**：
```go
		if err := migrateV23(db); err != nil {
			return fmt.Errorf("migrate v23 (findings.run_id): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 23"); err != nil {
			return fmt.Errorf("set user_version 23: %w", err)
```

---

### 3. 修改: `internal/models/finding.go`

在 `Finding` 结构体中，`WebEndpointID` 字段之后添加 `RunID`。

**要添加的字段**：
```go
	RunID           *string         `json:"run_id" db:"run_id"`
```

插入位置：`WebEndpointID   *string` 之后，`SourceTool` 之前。

（Finding struct 的其他字段不变。）

---

### 4. 修改: `internal/db/queries_finding.go`

需要改 7 处：

#### 4a. `CreateFinding`（约第 12-18 行）

**当前**：
```go
func (q *Queries) CreateFinding(f *models.Finding) error {
	_, err := q.db.Exec(`
		INSERT INTO findings (id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.ProjectID, f.AssetID, f.ServiceID, f.WebEndpointID, f.SourceTool, f.SourceRuleID, f.DedupKey, f.Title, f.Severity, f.Confidence, f.Priority, f.Status, f.Summary, f.Remediation, f.CreatedAt, f.UpdatedAt)
	return err
}
```

**改为**（在 `web_endpoint_id` 后加 `run_id`，VALUES 加 `f.RunID`）：
```go
func (q *Queries) CreateFinding(f *models.Finding) error {
	_, err := q.db.Exec(`
		INSERT INTO findings (id, project_id, asset_id, service_id, web_endpoint_id, run_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.ProjectID, f.AssetID, f.ServiceID, f.WebEndpointID, f.RunID, f.SourceTool, f.SourceRuleID, f.DedupKey, f.Title, f.Severity, f.Confidence, f.Priority, f.Status, f.Summary, f.Remediation, f.CreatedAt, f.UpdatedAt)
	return err
}
```

#### 4b. `scanFinding`（约第 133-152 行）

**当前 SELECT 列**：
```sql
SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, ...
```

**改为**（在 `web_endpoint_id` 后加 `run_id`）：
```sql
SELECT id, project_id, asset_id, service_id, web_endpoint_id, run_id, source_tool, ...
```

**Scan 参数**：在 `&webEndpointID` 后加 `&runID`

**Scan 后**：已有
```go
	f.AssetID = nullableString(assetID)
	f.ServiceID = nullableString(serviceID)
	f.WebEndpointID = nullableString(webEndpointID)
	return f, nil
```

**在其前追加**：
```go
	f.RunID = nullableString(runID)
```

（需要先声明 `var runID sql.NullString`）

#### 4c. 所有 SELECT 查询更新

以下每个查询的 SELECT 列都需要在 `web_endpoint_id` 后加 `run_id`：

- `GetFindingByDedupKey`（约第 20 行）
- `GetFinding`（约第 26 行）
- `ListFindingsByProject`（约第 45 行）
- `ListFindingsByStatus`（约第 64 行）
- `ListFindingsByProjectPaginated`（约第 97 行）
- `ListFindingsByStatusPaginated`（约第 116 行）
- `ListFindingsForReport`（约第 156 行）

**模式**：把
```sql
SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
```
全部替换为
```sql
SELECT id, project_id, asset_id, service_id, web_endpoint_id, run_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
```

（所有查询共用同一个 `scanFinding`，所以只需改 SELECT 列，无需改 scan 逻辑。）

#### 4d. 新增: `ListFindingsByRun`

在 `ListFindingsForReport` 之后、`// --- Evidence ---` 注释之前，新增函数：

```go
// ListFindingsByRun returns findings scoped to a specific pipeline run.
func (q *Queries) ListFindingsByRun(projectID, runID string) ([]*models.Finding, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, run_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE project_id = ? AND run_id = ? ORDER BY priority DESC, created_at DESC`, projectID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Finding, 0)
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}
```

---

### 5. 修改: `internal/workflow/pipeline_result.go`

在 `saveNucleiFindings` 函数中，`f := &models.Finding{...}` 结构体字面量末尾，`UpdatedAt` 之后、闭 `}` 之前，添加 `RunID`。

**当前**（约第 115-121 行）：
```go
		f := &models.Finding{
			ID:            util.GenerateID(),
			ProjectID:     p.projectID,
			AssetID:       assetID,
			WebEndpointID: webEndpointID,
			...
			UpdatedAt:     time.Now().UTC(),
		}
```

**在 `UpdatedAt` 后添加**：
```go
			RunID:         &p.runID,
```

但需要注意：当 `p.runID` 为空字符串时（非 pipeline 调用），不应设置 RunID。所以改为条件性地设置：

在结构体字面量末尾添加：
```go
		}
		if p.runID != "" {
			f.RunID = &p.runID
		}
		if err := p.queries.CreateFinding(f); err != nil {
```

即在 `}` 闭合后、`CreateFinding` 调用前插入 3 行。

---

### 6. 修改: `internal/report/aggregate_run.go`

`AggregateByRunWithBatchEvidence` 函数中，将 `ListFindingsForReport(project.ID)` 替换为 `ListFindingsByRun(project.ID, runID)`。

**当前**（约第 120 行）：
```go
	findings, err := q.ListFindingsForReport(project.ID)
```

**改为**：
```go
	findings, err := q.ListFindingsByRun(project.ID, runID)
```

---

### 7. 修改: `internal/report/aggregate_run.go`

`AggregateByRun` 函数同理，约第 50 行：
```
	data, err := Aggregate(ctx, q, project)
```
这里调用了 `Aggregate`，而 `Aggregate` 内部用的是 `ListFindingsForReport`。需要把 `AggregateByRun` 也改为直接聚合（不再复用 `Aggregate`），或者改为调用 `AggregateByRunWithBatchEvidence`。

**最简单方案**：把 `AggregateByRun` 改为直接调用 `AggregateByRunWithBatchEvidence`：

```go
func AggregateByRun(ctx context.Context, q *db.Queries, runID string) (*ReportData, *models.PipelineRun, error) {
	return AggregateByRunWithBatchEvidence(ctx, q, runID)
}
```

---

### 8. 修改: 测试文件 `internal/db/queries_finding_null_test.go`

测试中的 `CreateFinding` 调用不需要修改（`RunID` 是新字段，nullable，旧测试不设置即为 nil）。但需要确认测试文件中的 INSERT 语句已正确更新（如果有直接 SQL 的话）。

---

## 不需要改的文件

- **`internal/workflow/screenshot.go`**: WebScreeningWorkflow 没有 run 概念，findings 的 run_id 为 NULL，正确行为
- **`internal/workflow/discovery.go`**: AssetDiscoveryWorkflow 不创建 findings
- **`internal/workflow/slow_scan.go`**: SlowScanOrchestrator 不直接创建 findings
- **前端**: 暂不需要修改，ReportsPage 用 project 级别报告（不变），RunsPage 用 run 级别报告（后端已修）
- **`internal/service/finding.go`**: 走 `queries.CreateFinding`，自动适配新字段

---

## 实施顺序

1. DB migration (v23.go + db.go)
2. Model (finding.go)
3. Queries (queries_finding.go)
4. Pipeline (pipeline_result.go)
5. Report (aggregate_run.go)
6. 编译验证 (`go build ./...`)
7. 测试 (`go test ./...`)

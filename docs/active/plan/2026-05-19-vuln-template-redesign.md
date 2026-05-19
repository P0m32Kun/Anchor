# 漏洞辞典(Vulnerability Dictionary)实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把漏洞模板模块从「字段覆盖工具」改造成「漏洞辞典」—— 扫描结果自动套用中文明细的词条,长尾漏洞降级为原始数据同位混排;删除所有 JSON/HTML 报告导出与 RunsPage 异步报告流程,统一为 Markdown 同步下载。

**Architecture:**
- **后端:** 模板模型从单 `match_key` 改为 `match_keys []string`,一个词条可挂多个匹配键;匹配逻辑改为「优先 source_rule_id,然后 title」的两级匹配;报告渲染改为按词条聚合,同词条多 finding 合并一节,未命中 finding 单独一节同位混排。
- **前端:** VulnTemplatesPage 改用 chip 输入管理匹配键;ReportsPage 显示「已套/未套」徽章并提供「加入辞典」弹窗;RunsPage 删除异步报告流程,改为「查看报告」按钮跳转。
- **删除:** HTML/JSON 报告导出、RunPage 的整套异步报告持久化流程(handleCreateReport/GenerateHTML/`reports` 表/SSE report_progress)。

**Tech Stack:** Go 1.23+、SQLite、React 18+、TypeScript、Vite

**前置条件:** 使用 `superpowers:using-git-worktrees` 创建隔离 worktree 进行开发(本计划不包含此步骤,由执行前自行完成)。

**依赖文档:** 本计划基于 `docs/design/vuln-template-redesign.md` 编写,执行前请先阅读该 spec。

---

## 文件结构变更

### 新建文件
- `internal/db/v24.go` — 数据库 migration,添加 `match_keys` 列
- `frontend/src/components/TemplateEditor.tsx` — 共享的词条编辑器组件
- `docs/active/plan/2026-05-19-vuln-template-redesign.md` — 本计划文件

### 重大修改文件
- `internal/models/finding_template.go` — `MatchKey` → `MatchKeys []string` + `MatchKeysJSON string`
- `internal/models/scan.go` — 删除 `Report` 相关结构与常量
- `internal/db/queries_finding_template.go` — CRUD 处理 MatchKeys 序列化;`GetFindingTemplateForFinding` 重写
- `internal/db/queries_finding_template_test.go` — 新匹配语义的测试覆盖
- `internal/db/seed_finding_templates.go` — `SeedFindingTemplate.MatchKey` → `MatchKeys`
- `internal/db/seed_finding_templates_test.go` — 同步测试
- `internal/report/report.go` — `ReportData` 加 `Sections`;Aggregate 分桶
- `internal/report/aggregate_run.go` — 同上
- `internal/report/markdown.go` — `GenerateMarkdown` 重写
- `internal/report/report_test.go` — 分桶与渲染测试
- `internal/api/finding_template_handlers.go` — payload `match_keys`;应用层唯一性检查
- `internal/api/report_handlers.go` — 删除所有异步报告 handler,保留 `handleExportReportMD`
- `internal/api/handlers.go` 或 `server.go` — 删除报告异步 endpoint 路由注册
- `internal/api/README.md` — handler 总览与字段索引同步
- `frontend/src/lib/api.ts` — `FindingTemplate.match_keys`;删除 `Report` 与所有异步报告 API 方法
- `frontend/src/pages/VulnTemplatesPage.tsx` — chip 输入、表格 chip 显示、导出文案
- `frontend/src/pages/ReportsPage.tsx` — 词条状态徽章、「加入辞典」弹窗、删 JSON 导出、覆盖率统计
- `frontend/src/pages/RunsPage.tsx` — 删除报告 useState/useEffect/SSE 监听/按钮,改为「查看报告」导航

### 删除文件
- `internal/report/html.go` — 整文件删除
- `internal/report/json.go` — 整文件删除
- `internal/db/queries_report.go` — 整文件删除

### 保留不动
- `internal/db/v14.go` 中的 `CREATE TABLE reports` — 保留 schema,本次不删表(孤儿表),后续单独 PR 清理
- `Finding.MatchedTemplate` 字段 — 保留不删,但新代码不再使用

---

## 任务清单

### Task 1: 修改 FindingTemplate 模型 — 添加 MatchKeys 字段

**目标:** 在模型层增加 `MatchKeys []string` 内存字段和 `MatchKeysJSON string` 存储字段,保留 `MatchKey string` 以兼容老代码。

**文件:**
- Modify: `internal/models/finding_template.go:20-34`

**步骤:**

- [ ] **Step 1: 在 FindingTemplate 结构体中插入新字段**

在 `MatchKey string` 行**之前**插入两行:

```go
// MatchKeys 是内存中的匹配键列表(一个词条可挂多个)。
// MatchKeysJSON 是数据库存储格式(JSON 编码的字符串数组)。
// 老字段 MatchKey 保留一个 release 以兼容回滚;新代码只读写 MatchKeys/MatchKeysJSON。
MatchKeys      []string `json:"match_keys" db:"-"`
MatchKeysJSON  string    `json:"-"            db:"match_keys"`
MatchKey       string    `json:"match_key" db:"match_key"` // 保留兼容,新代码不用
```

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/models/...`
Expected: 编译通过,无错误

- [ ] **Step 3: 提交**

```bash
git add internal/models/finding_template.go
git commit -m "feat(models): add MatchKeys/MatchKeysJSON to FindingTemplate

- MatchKeys []string: 内存中匹配键列表
- MatchKeysJSON string: DB 存储为 JSON 数组
- 保留 MatchKey 字段以兼容回滚

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: 创建 Migration v24 — 添加 match_keys 列并迁移数据

**目标:** 新建 migration 文件,添加 `match_keys TEXT` 列,把现有 `match_key` 的单值迁移成 `["<value>"]` JSON 数组。

**文件:**
- Create: `internal/db/v24.go`

**步骤:**

- [ ] **Step 1: 创建 v24.go 并编写 migration**

创建文件 `internal/db/v24.go`,内容:

```go
package db

import (
	"database/sql"
	"encoding/json"

	_ "modernc.org/sqlite"
)

// migration_24_add_match_keys_column 添加 match_keys 列并迁移现有数据。
func migration_24_add_match_keys_column(db *sql.DB) error {
	// Step 1: 添加新列 match_keys,默认为 JSON 空数组 '[]'
	_, err := db.Exec(`ALTER TABLE finding_templates ADD COLUMN match_keys TEXT NOT NULL DEFAULT '[]'`)
	if err != nil {
		return err
	}

	// Step 2: 把现有 match_key 的单值迁移成 ["<value>"]
	_, err = db.Exec(`
		UPDATE finding_templates 
		SET match_keys = json_array(match_key) 
		WHERE match_key != ''
	`)
	if err != nil {
		return err
	}

	// Step 3: 创建唯一索引(从 (source_tool, match_key) 改为应用层检查,不在此建新索引)
	// 注: 原 UNIQUE INDEX 可能已存在,保留不动。新语义下唯一性由 handler 层保证。
	return nil
}

func init() {
	Migrations = append(Migrations, Migration{
		ID:      24,
		Name:    "add_match_keys_column",
		Forward: migration_24_add_match_keys_column,
	})
}
```

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/db/...`
Expected: 编译通过

- [ ] **Step 3: 运行 migration 验证(可选,本地数据库测试)**

```bash
# 在临时数据库上运行
cd /Users/kun/DEV/p0m32kun
go run main.go --help  # 确认启动逻辑会调用 migration
# 或直接调用 db 测试
go test ./internal/db/... -run TestMigration -v
```

Expected: migration 执行无错误

- [ ] **Step 4: 提交**

```bash
git add internal/db/v24.go
git commit -m "feat(db): add v24 migration for match_keys column

- 添加 match_keys TEXT 列,默认为 '[]'
- 把现有 match_key 单值迁移成 json_array(match_key)
- 保留原 match_key 列作为兼容

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: 修改 queries_finding_template.go — 处理 MatchKeys 序列化

**目标:** 修改 scan/CRUD 函数,让 `MatchKeys` 和 `MatchKeysJSON` 之间正确序列化/反序列化;读取时优先读 `match_keys`,若为空则 fallback 到 `match_key`(兼容老数据)。

**文件:**
- Modify: `internal/db/queries_finding_template.go:124-144` (scanFindingTemplate 函数)
- Modify: `internal/db/queries_finding_template.go:11-17` (CreateFindingTemplate)
- Modify: `internal/db/queries_finding_template.go:78-87` (UpdateFindingTemplate)

**步骤:**

- [ ] **Step 1: 重写 scanFindingTemplate 处理 MatchKeysJSON → MatchKeys**

把 `scanFindingTemplate` 函数(约第 124-144 行)完整替换为:

```go
func scanFindingTemplate(scanner rowScanner) (*models.FindingTemplate, error) {
	t := &models.FindingTemplate{}
	var enabled, isBuiltin, userModified int
	var matchKeysJSON string
	err := scanner.Scan(
		&t.ID, &t.SourceTool, &t.MatchKey, &matchKeysJSON,
		&t.Title, &t.Severity, &t.Summary, &t.Remediation,
		&enabled, &isBuiltin, &userModified, &t.BuiltinPayload,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Enabled = enabled != 0
	t.IsBuiltin = isBuiltin != 0
	t.UserModified = userModified != 0

	// 反序列化 match_keys JSON。如果为空或解析失败,尝试从老字段 match_key 兜底。
	if matchKeysJSON != "" && matchKeysJSON != "[]" {
		if err := json.Unmarshal([]byte(matchKeysJSON), &t.MatchKeys); err == nil {
			return t, nil
		}
	}
	// 兜底:老数据或迁移失败时,用 match_key 构造单元素数组
	if t.MatchKey != "" {
		t.MatchKeys = []string{t.MatchKey}
	} else {
		t.MatchKeys = []string{}
	}
	return t, nil
}
```

- [ ] **Step 2: 修改 CreateFindingTemplate 序列化 MatchKeys → MatchKeysJSON**

在 `CreateFindingTemplate` 函数(约第 11-17 行)中,把 INSERT 语句的 `match_key` 字段改为 `match_keys`,并序列化:

把:
```go
INSERT INTO finding_templates (id, source_tool, match_key, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
```

改为:
```go
matchKeysJSON, _ := json.Marshal(t.MatchKeys) // t.MatchKeys 在 handler 层保证非空
_, err := q.db.Exec(`
	INSERT INTO finding_templates (id, source_tool, match_key, match_keys, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	t.ID, t.SourceTool, t.MatchKey, string(matchKeysJSON), t.Title, t.Severity, t.Summary, t.Remediation,
	boolToInt(t.Enabled), boolToInt(t.IsBuiltin), boolToInt(t.UserModified), t.BuiltinPayload, t.CreatedAt, t.UpdatedAt)
```

- [ ] **Step 3: 修改 UpdateFindingTemplate 同步序列化 MatchKeys**

在 `UpdateFindingTemplate` 函数(约第 78-87 行)中,UPDATE 语句增加 `match_keys = ?`:

把:
```go
UPDATE finding_templates
SET source_tool = ?, match_key = ?, title = ?, severity = ?, summary = ?, remediation = ?, enabled = ?,
    is_builtin = ?, user_modified = ?, builtin_payload = ?, updated_at = ?
WHERE id = ?
```

改为:
```go
matchKeysJSON, _ := json.Marshal(t.MatchKeys)
_, err := q.db.Exec(`
	UPDATE finding_templates
	SET source_tool = ?, match_key = ?, match_keys = ?, title = ?, severity = ?, summary = ?, remediation = ?, enabled = ?,
	    is_builtin = ?, user_modified = ?, builtin_payload = ?, updated_at = ?
	WHERE id = ?`,
	t.SourceTool, t.MatchKey, string(matchKeysJSON), t.Title, t.Severity, t.Summary, t.Remediation, boolToInt(t.Enabled),
	boolToInt(t.IsBuiltin), boolToInt(t.UserModified), t.BuiltinPayload, t.UpdatedAt, t.ID)
```

- [ ] **Step 4: 运行测试验证**

Run: `go test ./internal/db/... -run TestFindingTemplate -v`
Expected: 现有测试通过(因为我们对老数据做了 fallback)

- [ ] **Step 5: 提交**

```bash
git add internal/db/queries_finding_template.go
git commit -m "feat(db): handle MatchKeys serialization in finding_template CRUD

- scanFindingTemplate: 反序列化 match_keys JSON,fallback 到老 match_key
- CreateFindingTemplate/UpdateFindingTemplate: 序列化 MatchKeys → JSON
- 保持对老数据的兼容

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: 重写 GetFindingTemplateForFinding — 新匹配语义

**目标:** 重写匹配函数为 3 参数签名(sourceTool, sourceRuleID, title),按「优先 source_rule_id,然后 title」的两级匹配逻辑,一次拉取 (source_tool, enabled=1) 的全部词条。

**文件:**
- Modify: `internal/db/queries_finding_template.go:95-122` (GetFindingTemplateForFinding 函数)

**步骤:**

- [ ] **Step 1: 添加辅助函数 listEnabledTemplatesByTool**

在 `GetFindingTemplateForFinding` 函数**之前**插入:

```go
// listEnabledTemplatesByTool 读取某工具的所有启用词条。
func (q *Queries) listEnabledTemplatesByTool(sourceTool string) ([]*models.FindingTemplate, error) {
	rows, err := q.db.Query(`
		SELECT id, source_tool, match_key, match_keys, title, severity, summary, remediation,
		       enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at
		FROM finding_templates 
		WHERE source_tool = ? AND enabled = 1`, sourceTool)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.FindingTemplate
	for rows.Next() {
		t, err := scanFindingTemplate(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// containsExact 判断 key 是否在 keys 列表中(精确匹配)。
func containsExact(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 重写 GetFindingTemplateForFinding**

把整个 `GetFindingTemplateForFinding` 函数(约第 95-122 行)替换为:

```go
// GetFindingTemplateForFinding 查找匹配 finding 的启用词条。
// 匹配优先级: source_rule_id → title(都为精确字符串匹配)。
// 返回 (nil, nil) 表示未找到。
func (q *Queries) GetFindingTemplateForFinding(sourceTool, sourceRuleID, title string) (*models.FindingTemplate, error) {
	tool := trim(sourceTool)
	if tool == "" {
		return nil, nil
	}

	templates, err := q.listEnabledTemplatesByTool(tool)
	if err != nil {
		return nil, err
	}

	// Tier 1: 按 source_ruleID 匹配
	if k := trim(sourceRuleID); k != "" {
		for _, t := range templates {
			if containsExact(t.MatchKeys, k) {
				return t, nil
			}
		}
	}

	// Tier 2: 按 title 匹配(兜底)
	if k := trim(title); k != "" {
		for _, t := range templates {
			if containsExact(t.MatchKeys, k) {
				return t, nil
			}
		}
	}

	return nil, nil
}

func trim(s string) string { return strings.TrimSpace(s) }
```

同时在文件顶部确保 `import` 有 `"strings"`。

- [ ] **Step 3: 修改调用点签名**

在 `internal/report/report.go` 第 ~115 行和 `internal/report/aggregate_run.go` 第 ~108 行,把调用从 4 参数改为 3 参数:

原调用:
```go
if tmpl, terr := q.GetFindingTemplateForFinding(f.SourceTool, f.SourceRuleID, f.MatchedTemplate, f.Title); terr == nil {
```

改为:
```go
if tmpl, terr := q.GetFindingTemplateForFinding(f.SourceTool, f.SourceRuleID, f.Title); terr == nil {
```

(两处都改)

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/db/... ./internal/report/... -v`
Expected: 现有测试需要更新(下一步处理)

- [ ] **Step 5: 提交**

```bash
git add internal/db/queries_finding_template.go internal/report/report.go internal/report/aggregate_run.go
git commit -m "feat(db): rewrite GetFindingTemplateForFinding with new match semantics

- 改为 3 参数(sourceTool, sourceRuleID, title)
- 一次拉取 (source_tool, enabled=1) 的全部词条
- 两级匹配: source_ruleID → title
- 添加辅助函数 listEnabledTemplatesByTool / containsExact
- 更新两个调用点签名

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: 更新 queries_finding_template_test.go — 新匹配语义测试

**目标:** 把现有测试改为使用新 3 参数签名,并增加「多匹配键命中任一」的测试用例。

**文件:**
- Modify: `internal/db/queries_finding_template_test.go`

**步骤:**

- [ ] **Step 1: 修改 makeTpl 辅助函数**

把第 10-24 行的 `makeTpl` 函数改为:

```go
func makeTpl(id, tool, key string, enabled bool) *models.FindingTemplate {
	now := time.Now().UTC().Truncate(time.Second)
	return &models.FindingTemplate{
		ID:          id,
		SourceTool:  tool,
		MatchKey:    key,
		MatchKeys:   []string{key}, // 新增:初始化 MatchKeys
		Title:       "T-" + id,
		Severity:    "high",
		Summary:     "summary " + id,
		Remediation: "fix " + id,
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
```

- [ ] **Step 2: 修改 TestGetFindingTemplateForFinding_PriorityFallback**

把整个测试函数(第 78-120 行)替换为:

```go
func TestGetFindingTemplateForFinding_PriorityFallback(t *testing.T) {
	q := New(openTestDB(t))

	// Seed 三个启用模板,不同 match_keys。
	if err := q.CreateFindingTemplate(makeTpl("a", "nuclei", "rule-id-x", true)); err != nil {
		t.Fatal(err)
	}
	// 为 "b" 词条设置多匹配键,验证多键命中任一即可
	b := makeTpl("b", "nuclei", "matched-template-x", true)
	b.MatchKeys = []string{"matched-template-x", "title-key-z"}
	if err := q.CreateFindingTemplate(b); err != nil {
		t.Fatal(err)
	}
	if err := q.CreateFindingTemplate(makeTpl("c", "nuclei", "Title X", true)); err != nil {
		t.Fatal(err)
	}

	// Tier 1: source_ruleID 优先
	tpl, err := q.GetFindingTemplateForFinding("nuclei", "rule-id-x", "Title X")
	if err != nil || tpl == nil || tpl.ID != "a" {
		t.Fatalf("ruleID priority: got %+v err %v", tpl, err)
	}

	// Tier 1: 多匹配键,命中任一即可
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "matched-template-x", "Title X")
	if err != nil || tpl == nil || tpl.ID != "b" {
		t.Fatalf("multi-key match: got %+v err %v", tpl, err)
	}

	// Tier 2: title 兜底
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "", "Title X")
	if err != nil || tpl == nil || tpl.ID != "c" {
		t.Fatalf("title fallback: got %+v err %v", tpl, err)
	}

	// 无匹配返回 (nil, nil)
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "nope", "Nope")
	if err != nil || tpl != nil {
		t.Fatalf("no match: got %+v err %v", tpl, err)
	}

	// 禁用词条不参与
	if err := q.CreateFindingTemplate(makeTpl("d", "hydra", "ssh-weakpass", false)); err != nil {
		t.Fatal(err)
	}
	tpl, err = q.GetFindingTemplateForFinding("hydra", "ssh-weakpass", "", "")
	if err != nil || tpl != nil {
		t.Fatalf("disabled template should not match: got %+v err %v", tpl, err)
	}
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./internal/db/... -run TestGetFindingTemplate -v`
Expected: 所有测试通过

- [ ] **Step 3: 提交**

```bash
git add internal/db/queries_finding_template_test.go
git commit -m "test(db): update finding_template tests for new match semantics

- makeTpl 初始化 MatchKeys
- TestGetFindingTemplateForFinding_PriorityFallback: 
  - 3 参数签名调用
  - 多匹配键命中任一的测试
  - 优先级(规则ID → title)验证

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: 修改 SeedFindingTemplate 和同步逻辑

**目标:** `SeedFindingTemplate` 从 `match_key string` 改为 `match_keys []string`,`SyncFindingTemplatesFromFile` 比对逻辑跟随调整。

**文件:**
- Modify: `internal/db/seed_finding_templates.go:16-27` (SeedFindingTemplate 结构体)
- Modify: `internal/db/seed_finding_templates.go:71-175` (applyFindingTemplateSeeds 函数)

**步骤:**

- [ ] **Step 1: 修改 SeedFindingTemplate 结构体**

把第 16-27 行替换为:

```go
// SeedFindingTemplate 是 docs/templates/vuln-templates.json 中单条记录的形态。
// MatchKey 保留以兼容老文件;MatchKeys 优先。
type SeedFindingTemplate struct {
	SourceTool  string   `json:"source_tool"`
	MatchKey    string   `json:"match_key,omitempty"`    // 兼容老文件
	MatchKeys   []string `json:"match_keys,omitempty"`   // 新形态
	Title       string   `json:"title,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"` // 默认 true
}
```

- [ ] **Step 2: 修改 applyFindingTemplateSeeds 同步逻辑**

在 `applyFindingTemplateSeeds` 函数中找到对 `seed.MatchKey` 的引用(第 95-96 行左右),改为兼容处理:

找到:
```go
match := strings.TrimSpace(s.MatchKey)
```

改为:
```go
// 优先用 MatchKeys,回退到 MatchKey(兼容老文件)
keys := s.MatchKeys
if len(keys) == 0 && s.MatchKey != "" {
	keys = []string{s.MatchKey}
}
```

然后继续往下到 `key{tool, match}` 的构造(第 98 行左右),把:

```go
k := key{tool, match}
```

改为:

```go
// 遍历所有 keys,每个 key 单独处理
for _, match := range keys {
	match = strings.TrimSpace(match)
	if match == "" {
		continue
	}
	k := key{tool, match}
```

同时在 `seenKeys` 和 `byKey` 使用时也需要调整:
- 原 `seenKeys[k]` 改为 `seenKeys[k] = struct{}{}` (不变,还是按 key 去重)
- 原 `byKey[k]` 改为 `byKey[k]` (不变)

但在最后的删除循环中,需要考虑同一 template ID 对应多个 key 的情况:原逻辑是 `key{tool, match}` 唯一,现在一个 template ID 会对应多个 key(在迭代中被拆成多行)。

这需要更复杂的调整。为简化,本次 MVP 维持「seed 里的一行对应一个 DB 行,match_key 单值仍存在」。这样同步逻辑不用大改,只是 `MatchKeys` 在读取时从 `MatchKey` 构造数组。

**修正策略:** 保持 `SyncFindingTemplatesFromFile` 的「一行一记录」逻辑不变。在 `scanFindingTemplate` 的 fallback 路径里,`MatchKey` 单值会被构造为单元素数组 `MatchKeys`。在同步时不处理 `MatchKeys` 字段,只处理 `MatchKey`。等后续 JSON 全部升级为新形态后,再移除 `MatchKey`。

所以 Task 6 实际上只需要修改结构体字段定义,不动同步逻辑。

**修正步骤:**

- [ ] **Step 1: 仅修改 SeedFindingTemplate 结构体**

如上:

```go
type SeedFindingTemplate struct {
	SourceTool  string   `json:"source_tool"`
	MatchKey    string   `json:"match_key,omitempty"`    // 兼容老文件
	MatchKeys   []string `json:"match_keys,omitempty"`   // 新形态
	Title       string   `json:"title,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./internal/db/... -run Seed -v`
Expected: 测试通过

- [ ] **Step 3: 提交**

```bash
git add internal/db/seed_finding_templates.go
git commit -m "feat(db): add MatchKeys field to SeedFindingTemplate

- 添加 MatchKeys []string,保留 MatchKey 兼容老文件
- 同步逻辑暂不变,仍在 MatchKey 上运作
- scanFindingTemplate fallback 会在读取时把单值构造为数组

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: 扩展 ReportData — 添加 Sections 与分桶

**目标:** 在 `report.go` 中扩展 `ReportData` 结构,新增 `Sections []*ReportSection` 字段,并在 `Aggregate` / `AggregateByRunWithBatchEvidence` 末尾实现分桶逻辑。

**文件:**
- Modify: `internal/report/report.go:12-35` (ReportData / ReportFinding 结构体)
- Modify: `internal/report/report.go:36-130` (Aggregate 函数末尾)
- Modify: `internal/report/aggregate_run.go:20-121` (AggregateByRunWithBatchEvidence 函数末尾)

**步骤:**

- [ ] **Step 1: 在 report.go 新增 ReportSection 结构体**

在 `ReportFinding` 结构体定义**之后**插入:

```go
// ReportSection 是「漏洞详情」章节单元。
//   - 命中词条: Template 非 nil,Findings 是同词条下的多个 finding
//   - 未命中:    Template 为 nil,Findings 切片长度恰好为 1
type ReportSection struct {
	Template *models.FindingTemplate // nil = 未命中(同位混排原始块)
	Severity models.FindingSeverity  // 用于排序,命中时 = template.severity,未命中 = finding.severity
	Findings []*ReportFinding
}
```

- [ ] **Step 2: 在 ReportData 添加 Sections 字段**

在 `ReportData` 结构体中(第 13-22 行左右)的字段列表末尾添加:

```go
// Sections 是按词条聚合后的章节列表,已按 severity 倒序。
// 用于 Markdown 渲染的「三、漏洞详情」段。
Sections []*ReportSection
```

- [ ] **Step 3: 在 Aggregate 函数末尾添加分桶逻辑**

在 `Aggregate` 函数的 `return data, nil` 之前(第 129 行左右)插入:

```go
	// --- 分桶:把 findings 按词条聚合 ---
	buckets := make(map[string][]*ReportFinding)
	var unmatched []*ReportFinding
	for _, rf := range data.Findings {
		if rf.Template != nil {
			buckets[rf.Template.ID] = append(buckets[rf.Template.ID], rf)
		} else {
			unmatched = append(unmatched, rf)
		}
	}

	sections := make([]*ReportSection, 0, len(buckets)+len(unmatched))
	for _, findings := range buckets {
		if len(findings) == 0 {
			continue
		}
		t := findings[0].Template
		sev := models.FindingSeverity(t.Severity)
		if sev == "" {
			sev = findings[0].Finding.Severity
		}
		sections = append(sections, &ReportSection{Template: t, Severity: sev, Findings: findings})
	}
	for _, rf := range unmatched {
		sections = append(sections, &ReportSection{Template: nil, Severity: rf.Finding.Severity, Findings: []*ReportFinding{rf}})
	}

	// 按 severity 倒序排序;同级时命中排在未命中前
	severityRank := map[models.FindingSeverity]int{
		models.SeverityCritical: 5,
		models.SeverityHigh:     4,
		models.SeverityMedium:   3,
		models.SeverityLow:      2,
		models.SeverityInfo:     1,
	}
	sort.SliceStable(sections, func(i, j int) bool {
		if severityRank[sections[i].Severity] != severityRank[sections[j].Severity] {
			return severityRank[sections[i].Severity] > severityRank[sections[j].Severity]
		}
		// 同级:命中(Template != nil)排在未命中前面
		return (sections[i].Template != nil) && (sections[j].Template == nil)
	})

	data.Sections = sections
```

同时在文件顶部 import 部分确保有 `"sort"`。

- [ ] **Step 4: 在 AggregateByRunWithBatchEvidence 同步添加分桶逻辑**

在 `internal/report/aggregate_run.go` 的 `return data, run, nil` 之前(第 120 行左右)插入**相同的分桶代码**(复制 Step 3 的逻辑,因为两个 Aggregate 函数目前各自实现)。

或者,更好的做法:把分桶逻辑抽取成独立函数 `bucketFindings(data *ReportData) []*ReportSection`,两个 Aggregate 都调用它。但为减少本次改动的 blast radius,先复制一遍;后续重构再抽取。

- [ ] **Step 5: 编译验证**

Run: `go build ./internal/report/...`
Expected: 编译通过

- [ ] **Step 6: 提交**

```bash
git add internal/report/report.go internal/report/aggregate_run.go
git commit -m "feat(report): add Sections bucketing to ReportData

- 新增 ReportSection 结构体(模板/严重级/finding 列表)
- ReportData 新增 Sections 字段
- Aggregate/AggregateByRunWithBatchEvidence 实现分桶逻辑
- 按 severity 倒序,同级时命中优先于未命中

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: 重写 GenerateMarkdown 的「漏洞详情」段 — 使用 Sections 聚合渲染

**目标:** 把 `GenerateMarkdown` 中的「三、漏洞详情」段改为遍历 `data.Sections`,命中词条合并一节(表格列出多个 finding),未命中单条一节同位混排。

**文件:**
- Modify: `internal/report/markdown.go:136-159` (GenerateMarkdown 中的漏洞详情段)

**步骤:**

- [ ] **Step 1: 重写「三、漏洞详情」段**

把第 136-159 行左右的内容替换为:

```go
	// --- 4. 漏洞详情(按 severity 倒序,使用 Sections 聚合渲染) ---
	sb.WriteString("## 三、漏洞详情\n\n")

	if len(data.Sections) == 0 {
		sb.WriteString("*无漏洞记录。*\n\n")
	} else {
		num := 0
		for _, section := range data.Sections {
			for _, rf := range section.Findings {
				num++
			}
			writeSectionCN(&sb, num, section)
		}
	}
```

- [ ] **Step 2: 实现 writeSectionCN 渲染函数**

在 `writeFindingDetailCN` 函数**之后**插入新函数:

```go
// writeSectionCN 渲染一个章节:命中词条合并一节,或未命中单条一节。
func writeSectionCN(sb *strings.Builder,起始Num int, section *ReportSection) {
	// 写标题: emoji + severity + title (命中用 template.title,未命中用 finding.title)
	var title string
	var severity models.FindingSeverity
	if section.Template != nil {
		title = section.Template.Title
		severity = models.FindingSeverity(section.Template.Severity)
	} else {
		// 未命中:单条 finding
		rf := section.Findings[0]
		title = rf.Finding.Title
		severity = rf.Finding.Severity
	}

	fmt.Fprintf(sb, "### %d. %s %s · %s\n\n",
		起始Num, severityEmojiOf(severity), severityLabelCN(severity), escapeMDTable(title))

	// 漏洞描述
	sb.WriteString("**漏洞描述**\n\n")
	var summary string
	if section.Template != nil {
		summary = section.Template.Summary
	} else {
		summary = section.Findings[0].Finding.Summary
	}
	if strings.TrimSpace(summary) != "" {
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*暂无描述。*\n\n")
	}

	// 受影响资产
	sb.WriteString("**受影响资产**\n\n")
	writeAffectedAssetsTable(sb, section)

	// 修复建议
	sb.WriteString("**修复建议**\n\n")
	var remediation string
	if section.Template != nil {
		remediation = section.Template.Remediation
	}
	if strings.TrimSpace(remediation) != "" {
		sb.WriteString(remediation)
		sb.WriteString("\n\n")
	} else {
		if section.Template != nil {
			sb.WriteString("*暂无修复建议。*\n\n")
		} else {
			sb.WriteString("*暂无修复建议 — 该漏洞类型尚未在辞典中维护。可在「漏洞模板」页补充。*\n\n")
		}
	}

	// 底部元数据
	var meta []string
	if section.Template != nil {
		meta = append(meta, "套用词条")
	}
	meta = append(meta, fmt.Sprintf("检测来源:%s", section.Findings[0].Finding.SourceTool))
	if len(section.Findings) > 1 && section.Template != nil {
		meta = append(meta, fmt.Sprintf("命中 %d 项", len(section.Findings)))
	}
	if section.Template == nil {
		meta = append(meta, "未套词条")
		rf := section.Findings[0]
		if rf.Finding.SourceRuleID != "" {
			meta = append(meta, fmt.Sprintf("规则:%s", escapeMDTable(rf.Finding.SourceRuleID)))
		}
	}
	meta = append(meta, "状态:"+statusLabelCN(section.Findings[0].Finding.Status))
	fmt.Fprintf(sb, "> %s\n\n", strings.Join(meta, " · "))

	sb.WriteString("---\n\n")
}
```

- [ ] **Step 3: 实现 writeAffectedAssetsTable**

在 `writeSectionCN` 之后插入:

```go
// writeAffectedAssetsTable 渲染受影响资产表格。
// 命中词条:多行表格;未命中:单行表格。
func writeAffectedAssetsTable(sb *strings.Builder, section *ReportSection) {
	sb.WriteString("| 资产 | 端口 | 访问地址 | 工具规则 |\n")
	sb.WriteString("|---|---|---|---|\n")

	for _, rf := range section.Findings {
		assetVal := "—"
		if rf.Asset != nil {
			assetVal = escapeMDTable(rf.Asset.Value)
		}

		portVal := "—"
		urlVal := "—"
		if rf.WebEndpoint != nil {
			if rf.WebEndpoint.Port != nil {
				portVal = fmt.Sprintf("%d", *rf.WebEndpoint.Port)
			} else if rf.WebEndpoint.Scheme == "https" {
				portVal = "443"
			} else if rf.WebEndpoint.Scheme == "http" {
				portVal = "80"
			}
			if rf.WebEndpoint.URL != "" {
				urlVal = escapeMDTable(rf.WebEndpoint.URL)
			}
		}

		ruleVal := "—"
		if rf.Finding.SourceRuleID != "" {
			ruleVal = escapeMDTable(rf.Finding.SourceRuleID)
		}

		fmt.Fprintf(sb, "| %s | %s | %s | %s |\n", assetVal, portVal, urlVal, ruleVal)
	}
}
```

- [ ] **Step 4: 删除旧的 writeFindingDetailCN 函数**

找到原有的 `writeFindingDetailCN` 函数(约第 206-265 行),整段删除。因为新逻辑由 `writeSectionCN` 替代。

**注意:** 别删 `writeAffectedTargets`,新函数叫 `writeAffectedAssetsTable`,但可能跟老函数有细微差异。确保删的是 `writeFindingDetailCN`。

- [ ] **Step 5: 删除 writeAffectedTargets 函数(已被 writeAffectedAssetsTable 替代)**

找到原有的 `writeAffectedTargets` 函数(约第 278-310 行),整段删除。

- [ ] **Step 6: 运行测试**

Run: `go test ./internal/report/... -v`
Expected: 现有测试需要更新(report_test.go 可能还在用旧 API)

- [ ] **Step 7: 提交**

```bash
git add internal/report/markdown.go
git commit -m "feat(report): rewrite GenerateMarkdown with section-based rendering

- 使用 data.Sections 聚合渲染
- 命中词条:合并一节,受影响资产表格列出多个 finding
- 未命中:单条一节同位混排,提示「该漏洞类型尚未在辞典中维护」
- 删除旧的 writeFindingDetailCN/writeAffectedTargets
- 新增 writeSectionCN/writeAffectedAssetsTable

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: 删除 internal/report/html.go 和 json.go

**目标:** 删除 HTML 和 JSON 报告生成文件,因为报告导出统一只用 Markdown。

**文件:**
- Delete: `internal/report/html.go`
- Delete: `internal/report/json.go`

**步骤:**

- [ ] **Step 1: 删除 html.go**

Run: `rm /Users/kun/DEV/p0m32kun/internal/report/html.go`

- [ ] **Step 2: 删除 json.go**

Run: `rm /Users/kun/DEV/p0m32kun/internal/report/json.go`

- [ ] **Step 3: 编译验证**

Run: `go build ./internal/report/...`
Expected: 编译通过,无 undefined references(如果有,说明有代码还在调用 `GenerateHTML/GenerateJSON`,需要清理)

- [ ] **Step 4: 搜索残留引用(可选)**

Run: `grep -rn "GenerateHTML\|GenerateJSON" /Users/kun/DEV/p0m32kun/internal --include="*.go" | grep -v ".git"`
Expected: 应该没有输出(如果有,说明还有代码在用,需要回到 Task 14 处理)

- [ ] **Step 5: 提交**

```bash
git add -A internal/report/
git commit -m "refactor(report): delete html.go and json.go

- 报告导出统一为 Markdown,删除 HTML/JSON 生成
- 删除 internal/report/html.go
- 删除 internal/report/json.go

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: 修改 finding_template_handlers.go — 创建请求 payload 改造

**目标:** `handleCreateFindingTemplate` 的 payload 从 `match_key *string` 改为 `match_keys *[]string`,并添加应用层唯一性检查(同 source_tool 下不允许重复 match_key)。

**文件:**
- Modify: `internal/api/finding_template_handlers.go:15-23` (findingTemplatePayload 结构体)
- Modify: `internal/api/finding_template_handlers.go:43-84` (handleCreateFindingTemplate 函数)

**步骤:**

- [ ] **Step 1: 修改 findingTemplatePayload 结构体**

把第 15-23 行替换为:

```go
type findingTemplatePayload struct {
	SourceTool  *string    `json:"source_tool,omitempty"`
	MatchKeys   *[]string  `json:"match_keys,omitempty"`
	Title       *string    `json:"title,omitempty"`
	Severity    *string    `json:"severity,omitempty"`
	Summary     *string    `json:"summary,omitempty"`
	Remediation *string    `json:"remediation,omitempty"`
	Enabled     *bool      `json:"enabled,omitempty"`
}
```

- [ ] **Step 2: 重写 handleCreateFindingTemplate**

把整个 `handleCreateFindingTemplate` 函数(第 43-84 行)替换为:

```go
func (s *Server) handleCreateFindingTemplate(w http.ResponseWriter, r *http.Request) {
	var req findingTemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	sourceTool := strings.TrimSpace(deref(req.SourceTool))
	if sourceTool == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "source_tool is required"))
		return
	}

	if req.MatchKeys == nil || len(*req.MatchKeys) == 0 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_keys is required and must not be empty"))
		return
	}

	// 应用层唯一性检查:同 source_tool 下,任一 match_key 不能与其他词条重复
	keys := *req.MatchKeys
	var cleanedKeys []string
	keySet := make(map[string]bool)
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if keySet[k] {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_keys contains duplicate values"))
			return
		}
		// 检查是否与已有词条冲突
		existing, err := s.queries.ListFindingTemplates(sourceTool)
		if err == nil {
			for _, ex := range existing {
				for _, ek := range ex.MatchKeys {
					if ek == k {
						writeError(w, http.StatusConflict, errors.Newf(errors.ErrConflict, "match_key %q is already used by template %s", k, ex.ID))
						return
					}
				}
			}
		}
		keySet[k] = true
		cleanedKeys = append(cleanedKeys, k)
	}
	if len(cleanedKeys) == 0 {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_keys must contain at least one non-empty value"))
		return
	}

	severity := strings.TrimSpace(deref(req.Severity))
	if !validSeverity(severity) {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "severity must be info/low/medium/high/critical or empty"))
		return
	}

	now := time.Now().UTC()
	t := &models.FindingTemplate{
		ID:         util.GenerateID(),
		SourceTool: sourceTool,
		MatchKeys:   cleanedKeys,
		Title:       strings.TrimSpace(deref(req.Title)),
		Severity:    severity,
		Summary:     deref(req.Summary),
		Remediation: deref(req.Remediation),
		Enabled:     derefBool(req.Enabled, true),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.queries.CreateFindingTemplate(t); err != nil {
		writeError(w, http.StatusConflict, errors.Newf(errors.ErrInternal, "create finding template: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, t)
}
```

同时确保 `import` 里有 `"strings"`。

- [ ] **Step 3: 运行测试(如果有)**

Run: `go test ./internal/api/... -run FindingTemplate -v`
Expected: 通过(如果有 handler 测试的话)

- [ ] **Step 4: 提交**

```bash
git add internal/api/finding_template_handlers.go
git commit -m "feat(api): create finding template with match_keys array

- payload 改为 match_keys *[]string
- 应用层唯一性检查:同 source_tool 下任一 key 不能重复
- 清理空 key/去重/非空校验

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: 修改 finding_template_handlers.go — PATCH 请求 payload 改造

**目标:** `handlePatchFindingTemplate` 支持 `match_keys` 的部分更新,同样添加唯一性检查。

**文件:**
- Modify: `internal/api/finding_template_handlers.go:100-187` (handlePatchFindingTemplate 函数)

**步骤:**

- [ ] **Step 1: 在 handlePatchFindingTemplate 前添加辅助函数**

在 `handlePatchFindingTemplate` 函数**之前**插入:

```go
// checkMatchKeysConflict 检查给定的 match_keys 是否与已有词条(排除当前模板)冲突。
func (s *Server) checkMatchKeysConflict(sourceTool, templateID string, keys []string) *string {
	existing, err := s.queries.ListFindingTemplates(sourceTool)
	if err != nil {
		return nil // DB 错误,让后续处理
	}
	for _, ex := range existing {
		if ex.ID == templateID {
			continue
		}
		for _, ek := range ex.MatchKeys {
			for _, k := range keys {
				if ek == strings.TrimSpace(k) {
					return &ex.ID
				}
			}
		}
	}
	return nil
}
```

- [ ] **Step 2: 重写 handlePatchFindingTemplate**

把 `handlePatchFindingTemplate` 函数(第 100-187 行)替换为:

```go
func (s *Server) handlePatchFindingTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.queries.GetFindingTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get finding template: %v", err))
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, errors.Newf(errors.ErrNotFound, "finding template %s not found", id))
		return
	}

	var req findingTemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	contentChanged := false
	keysChanged := false
	var newKeys []string

	if req.SourceTool != nil {
		v := strings.TrimSpace(*req.SourceTool)
		if v == "" {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "source_tool cannot be empty"))
			return
		}
		if v != t.SourceTool {
			contentChanged = true
		}
		t.SourceTool = v
	}

	if req.MatchKeys != nil {
		keys := *req.MatchKeys
		var cleanedKeys []string
		keySet := make(map[string]bool)
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if keySet[k] {
				writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_keys contains duplicate values"))
				return
			}
			keySet[k] = true
			cleanedKeys = append(cleanedKeys, k)
		}
		if len(cleanedKeys) == 0 {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "match_keys must contain at least one non-empty value"))
			return
		}
		// 唯一性检查
		if conflictID := s.checkMatchKeysConflict(t.SourceTool, t.ID, cleanedKeys); conflictID != nil {
			writeError(w, http.StatusConflict, errors.Newf(errors.ErrConflict, "some match_keys already used by template %s", *conflictID))
			return
		}
		newKeys = cleanedKeys
		if len(newKeys) != len(t.MatchKeys) || !slices.Equal(newKeys, t.MatchKeys) {
			keysChanged = true
			contentChanged = true
		}
		t.MatchKeys = newKeys
	}

	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		if v != t.Title {
			contentChanged = true
		}
		t.Title = v
	}
	if req.Severity != nil {
		v := strings.TrimSpace(*req.Severity)
		if !validSeverity(v) {
			writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "severity must be info/low/medium/high/critical or empty"))
			return
		}
		if v != t.Severity {
			contentChanged = true
		}
		t.Severity = v
	}
	if req.Summary != nil {
		if *req.Summary != t.Summary {
			contentChanged = true
		}
		t.Summary = *req.Summary
	}
	if req.Remediation != nil {
		if *req.Remediation != t.Remediation {
			contentChanged = true
		}
		t.Remediation = *req.Remediation
	}
	if req.Enabled != nil {
		if *req.Enabled != t.Enabled {
			contentChanged = true
		}
		t.Enabled = *req.Enabled
	}

	if t.IsBuiltin && contentChanged {
		t.UserModified = true
	}
	t.UpdatedAt = time.Now().UTC()

	if err := s.queries.UpdateFindingTemplate(t); err != nil {
		writeError(w, http.StatusConflict, errors.Newf(errors.ErrInternal, "update finding template (duplicate match key?): %v", err))
		return
	}
	writeJSON(w, http.StatusOK, t)
}
```

同时在文件顶部 import 部分确保有 `"slices"`。

- [ ] **Step 3: 提交**

```bash
git add internal/api/finding_template_handlers.go
git commit -m "feat(api): patch finding template with match_keys validation

- 支持 match_keys 部分更新
- 应用层唯一性检查,排除当前模板自身
- 使用 slices.Equal 比较

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: 修改 handleExportFindingTemplates — 输出形态改为 match_keys 数组

**目标:** 导出的 JSON 每条记录从 `match_key` 改为 `match_keys []string`。

**文件:**
- Modify: `internal/api/finding_template_handlers.go:242-270` (handleExportFindingTemplates 函数)

**步骤:**

- [ ] **Step 1: 重写 handleExportFindingTemplates**

把第 242-270 行替换为:

```go
func (s *Server) handleExportFindingTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := s.queries.ListFindingTemplates("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list finding templates: %v", err))
		return
	}
	seeds := make([]db.SeedFindingTemplate, 0, len(list))
	for _, t := range list {
		enabled := t.Enabled
		seeds = append(seeds, db.SeedFindingTemplate{
			SourceTool:  t.SourceTool,
			MatchKeys:   t.MatchKeys, // 新形态:直接用数组
			Title:       t.Title,
			Severity:    t.Severity,
			Summary:     t.Summary,
			Remediation: t.Remediation,
			Enabled:     &enabled,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="vuln-templates.json"`)
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(seeds)
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/api/finding_template_handlers.go
git commit -m "feat(api): export finding templates as match_keys array

- handleExportFindingTemplates 输出 match_keys []string
- 符合新 JSON 形态

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: 删除 report_handlers.go 中的异步报告 handler

**目标:** 删除 `handleExportReportJSON` / `handleCreateReport` / `generateReport` / `broadcastReportProgress` / `handleGetReport` / `handleGetReportByRun` / `handleDownloadReport` / `handleDeleteReport` / `handleListReports` / `checkDiskSpace`。只保留 `handleExportReportMD`。

**文件:**
- Modify: `internal/api/report_handlers.go`

**步骤:**

- [ ] **Step 1: 删除 handleExportReportJSON 函数**

找到 `handleExportReportJSON` 函数(第 46-75 行),整段删除。

- [ ] **Step 2: 删除 handleCreateReport 函数**

找到 `handleCreateReport` 函数(第 79-161 行),整段删除。

- [ ] **Step 3: 删除 generateReport 函数**

找到 `generateReport` 函数(第 163-231 行),整段删除。

- [ ] **Step 4: 删除 broadcastReportProgress 函数**

找到 `broadcastReportProgress` 函数(第 233-257 行),整段删除。

- [ ] **Step 5: 删除 handleGetReportByRun 函数**

找到 `handleGetReportByRun` 函数(第 259-271 行),整段删除。

- [ ] **Step 6: 删除 handleGetReport 函数**

找到 `handleGetReport` 函数(第 273-287 行),整段删除。

- [ ] **Step 7: 删除 handleDownloadReport 函数**

找到 `handleDownloadReport` 函数(第 289-321 行),整段删除。

- [ ] **Step 8: 删除 handleDeleteReport 函数**

找到 `handleDeleteReport` 函数(第 323-347 行),整段删除。

- [ ] **Step 9: 删除 handleListReports 函数**

找到 `handleListReports` 函数(第 349-363 行),整段删除。

- [ ] **Step 10: 删除 checkDiskSpace 函数**

找到 `checkDiskSpace` 函数(第 366-374 行),整段删除。

- [ ] **Step 11: 清理不需要的 import**

删除文件顶部以下 import:
- `"context"` (如果只有 `generateReport` 在用)
- `"filepath"` (如果只有 `generateReport` / `handleDownloadReport` 在用)
- `"syscall"` (如果只有 `checkDiskSpace` 在用)

保留 `"encoding/json"`, `"fmt"`, `"log"`, `"net/http"`, `"os"`, `"time"` (handleExportReportMD 还在用)

- [ ] **Step 12: 编译验证**

Run: `go build ./internal/api/...`
Expected: 编译通过

- [ ] **Step 13: 提交**

```bash
git add internal/api/report_handlers.go
git commit -m "refactor(api): remove async report handlers from report_handlers.go

- 删除 handleExportReportJSON
- 删除 handleCreateReport
- 删除 generateReport (goroutine)
- 删除 broadcastReportProgress
- 删除 handleGetReport/handleGetReportByRun
- 删除 handleDownloadReport/handleDeleteReport/handleListReports
- 删除 checkDiskSpace
- 清理不需要的 import
- 只保留 handleExportReportMD

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 14: 删除报告异步 endpoint 的路由注册

**目标:** 在路由注册处删除 `POST /runs/:runId/reports` / `GET /reports/:reportId` / `GET /reports/:reportId/download` / `DELETE /reports/:reportId` / `GET /reports` 等路由。

**文件:**
- Modify: `internal/api/handlers.go` 或 `internal/api/server.go` (取决于路由注册位置)

**步骤:**

- [ ] **Step 1: 定位路由注册**

Run: `grep -n "reports\|handleCreateReport\|handleGetReport" /Users/kun/DEV/p0m32kun/internal/api/handlers.go /Users/kun/DEV/p0m32kun/internal/api/server.go`
Expected: 输出包含 `Register` 或路由注册的行号

- [ ] **Step 2: 删除异步报告路由注册**

在找到的路由注册位置,删除类似以下的行:
- `"GET /reports/{reportId}"` → handleGetReport
- `"POST /runs/{runId}/reports"` → handleCreateReport
- `"GET /runs/{runId}/reports"` → handleGetReportByRun
- `"GET /reports/{reportId}/download"` → handleDownloadReport
- `"DELETE /reports/{reportId}"` → handleDeleteReport
- `"GET /reports"` → handleListReports

**注意:** 保留 `"GET /projects/{id}/reports/export.md"` → handleExportReportMD

- [ ] **Step 3: 编译验证**

Run: `go build ./internal/api/...`
Expected: 编译通过

- [ ] **Step 4: 提交**

```bash
git add internal/api/handlers.go
git commit -m "refactor(api): remove async report endpoint routes

- 删除 POST /runs/:runId/reports
- 删除 GET /reports/:reportId
- 删除 GET /runs/:runId/reports
- 删除 GET /reports/:reportId/download
- 删除 DELETE /reports/:reportId
- 删除 GET /reports
- 保留 GET /projects/:id/reports/export.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 15: 删除 internal/db/queries_report.go

**目标:** 删除 `reports` 表的所有查询函数,因为 `reports` 表不再使用(成为孤儿表)。

**文件:**
- Delete: `internal/db/queries_report.go`

**步骤:**

- [ ] **Step 1: 删除文件**

Run: `rm /Users/kun/DEV/p0m32kun/internal/db/queries_report.go`

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/db/...`
Expected: 编译通过,无 undefined references(如果有,说明有代码还在调用 reports 相关函数,需要清理)

- [ ] **Step 3: 搜索残留引用(可选)**

Run: `grep -rn "CreateReport\|GetReport\|ListReports\|DeleteReport\|UpdateReport\|GetReportByRunID\|GetReportByRun" /Users/kun/DEV/p0m32kun/internal --include="*.go" | grep -v ".git" | grep -v "_test.go"`
Expected: 应该没有输出(如果有,需要回到 Task 13/14 检查是否有遗漏的调用点)

- [ ] **Step 4: 提交**

```bash
git add -A internal/db/
git commit -m "refactor(db): delete queries_report.go

- 删除 reports 表的查询函数
- reports 表成为孤儿表,保留 schema 供后续清理

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 16: 删除 internal/models/scan.go 中的 Report 相关结构

**目标:** 删除 `Report` 结构体、`ReportStatus` 类型及相关常量(`ReportGenerating` / `ReportComplete` / `ReportFailed` / `ReportPartial`)。

**文件:**
- Modify: `internal/models/scan.go`

**步骤:**

- [ ] **Step 1: 定位 Report 相关定义**

Run: `grep -n "type Report\|type ReportStatus\|ReportGenerating\|ReportComplete\|ReportFailed\|ReportPartial" /Users/kun/DEV/p0m32kun/internal/models/scan.go`
Expected: 输出行号

- [ ] **Step 2: 删除 ReportStatus 类型和常量**

删除类似以下的行:
```go
type ReportStatus string

const (
	ReportGenerating ReportStatus = "generating"
	ReportComplete   ReportStatus = "complete"
	ReportFailed     ReportStatus = "failed"
	ReportPartial    ReportStatus = "partial"
)
```

- [ ] **Step 3: 删除 Report 结构体**

删除类似以下的行:
```go
type Report struct {
	ID            string       `json:"id" db:"id"`
	RunID         string       `json:"run_id" db:"run_id"`
	Status        ReportStatus `json:"status" db:"status"`
	...
}
```

- [ ] **Step 4: 删除 ReportStatus 的 Value/Scan 方法(如果有)**

如果有 `func (r ReportStatus) Value()` 和 `func (r *ReportStatus) Scan()` 之类,也删除。

- [ ] **Step 5: 编译验证**

Run: `go build ./internal/models/...`
Expected: 编译通过

- [ ] **Step 6: 搜索残留引用(可选)**

Run: `grep -rn "models.Report\|ReportStatus\|ReportGenerating\|ReportComplete" /Users/kun/DEV/p0m32kun/internal --include="*.go" | grep -v ".git" | grep -v "_test.go"`
Expected: 应该没有输出

- [ ] **Step 7: 提交**

```bash
git add internal/models/scan.go
git commit -m "refactor(models): delete Report model and ReportStatus

- 删除 Report 结构体
- 删除 ReportStatus 类型
- 删除 ReportGenerating/ReportComplete/ReportFailed/ReportPartial 常量

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 17: 前端 lib/api.ts — 修改 FindingTemplate 类型

**目标:** `FindingTemplate` 类型的 `match_key: string` 改为 `match_keys: string[]`。

**文件:**
- Modify: `frontend/src/lib/api.ts`

**步骤:**

- [ ] **Step 1: 定位 FindingTemplate 类型定义**

Run: `grep -n "export type FindingTemplate\|interface FindingTemplate" /Users/kun/DEV/p0m32kun/frontend/src/lib/api.ts`
Expected: 输出行号

- [ ] **Step 2: 修改 match_key 字段为 match_keys**

在 `FindingTemplate` 类型中找到:
```typescript
match_key: string;
```

改为:
```typescript
match_keys: string[];
```

- [ ] **Step 3: 运行 typecheck**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 4: 提交**

```bash
git add frontend/src/lib/api.ts
git commit -m "feat(frontend): FindingTemplate.match_key → match_keys array

- 类型从 string 改为 string[]

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 18: 前端 lib/api.ts — 删除 Report 类型和所有异步报告 API 方法

**目标:** 删除 `Report` 类型、`createReport` / `getReport` / `getReportByRun` / `deleteReport` / `listReports` / `downloadReport` / `exportReportJSON`。

**文件:**
- Modify: `frontend/src/lib/api.ts`

**步骤:**

- [ ] **Step 1: 定位 Report 类型定义和异步报告 API**

Run: `grep -n "export type Report\|export interface Report\|createReport:\|getReport:\|getReportByRun:\|deleteReport:\|listReports:\|downloadReport:\|exportReportJSON:" /Users/kun/DEV/p0m32kun/frontend/src/lib/api.ts`
Expected: 输出行号

- [ ] **Step 2: 删除 Report 类型**

删除类似:
```typescript
export interface Report {
  id: string;
  run_id: string;
  status: string;
  ...
}
```

- [ ] **Step 3: 删除异步报告 API 方法**

删除以下方法:
```typescript
createReport: (runId: string, title?: string, signal?: AbortSignal) => ...
getReport: (reportId: string, signal?: AbortSignal) => ...
getReportByRun: (runId: string, signal?: AbortSignal, skipGlobalError?: boolean) => ...
deleteReport: (reportId: string, signal?: AbortSignal) => ...
listReports: (cursor?: string, signal?: AbortSignal) => ...
downloadReport: (reportId: string) => ...
```

保留 `exportReportMD:` (不变)。

- [ ] **Step 4: 删除 exportReportJSON**

找到:
```typescript
exportReportJSON: (projectId: string, signal?: AbortSignal) => ...
```

删除。

- [ ] **Step 5: 运行 typecheck**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 6: 提交**

```bash
git add frontend/src/lib/api.ts
git commit -m "refactor(frontend): remove Report type and async report APIs

- 删除 Report 类型
- 删除 createReport/getReport/getReportByRun/deleteReport/listReports/downloadReport
- 删除 exportReportJSON
- 保留 exportReportMD

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 19: 创建共享组件 TemplateEditor.tsx

**目标:** 创建可复用的词条编辑器组件,支持 chip 输入管理 `match_keys`,被 VulnTemplatesPage 和 ReportsPage 复用。

**文件:**
- Create: `frontend/src/components/TemplateEditor.tsx`

**步骤:**

- [ ] **Step 1: 创建 TemplateEditor.tsx**

创建文件 `frontend/src/components/TemplateEditor.tsx`,内容:

```typescript
import { useState } from "react";
import { Button, Input, Badge } from "../components";

const SOURCE_TOOL_OPTIONS = [
  "nuclei",
  "sqlmap",
  "hydra",
  "httpx",
  "dnsx",
  "ffuf",
  "其他",
];

const SEVERITY_OPTIONS: { value: string; label: string }[] = [
  { value: "", label: "保留 finding 原值" },
  { value: "critical", label: "严重 critical" },
  { value: "high", label: "高危 high" },
  { value: "medium", label: "中危 medium" },
  { value: "low", label: "低危 low" },
  { value: "info", label: "信息 info" },
];

type TemplateForm = {
  source_tool: string;
  match_keys: string[];
  title: string;
  severity: string;
  summary: string;
  remediation: string;
  enabled: boolean;
};

interface Props {
  initialValue?: Partial<TemplateForm>;
  mode: "create" | "edit";
  onSaved: (form: TemplateForm) => Promise<void>;
  onCancel?: () => void;
}

export default function TemplateEditor({ initialValue, mode, onSaved, onCancel }: Props) {
  const [form, setForm] = useState<TemplateForm>({
    source_tool: initialValue?.source_tool ?? "nuclei",
    match_keys: initialValue?.match_keys ?? [],
    title: initialValue?.title ?? "",
    severity: initialValue?.severity ?? "",
    summary: initialValue?.summary ?? "",
    remediation: initialValue?.remediation ?? "",
    enabled: initialValue?.enabled ?? true,
  });
  const [keyInput, setKeyInput] = useState("");
  const [saving, setSaving] = useState(false);

  const handleAddKey = () => {
    const k = keyInput.trim();
    if (k && !form.match_keys.includes(k)) {
      setForm((p) => ({ ...p, match_keys: [...p.match_keys, k] }));
      setKeyInput("");
    }
  };

  const handleRemoveKey = (k: string) => {
    setForm((p) => ({ ...p, match_keys: p.match_keys.filter((x) => x !== k) }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.source_tool.trim() || form.match_keys.length === 0) {
      return; // TODO: 提示错误
    }
    setSaving(true);
    try {
      await onSaved(form);
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-sm font-medium mb-1.5 block">检测工具 *</label>
          <select
            value={form.source_tool}
            onChange={(e) => setForm((p) => ({ ...p, source_tool: e.target.value }))}
            className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
          >
            {SOURCE_TOOL_OPTIONS.map((v) => (
              <option key={v} value={v}>{v}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="text-sm font-medium mb-1.5 block">严重等级</label>
          <select
            value={form.severity}
            onChange={(e) => setForm((p) => ({ ...p, severity: e.target.value }))}
            className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
          >
            {SEVERITY_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>
      </div>

      <div>
        <label className="text-sm font-medium mb-1.5 block">匹配键 * (至少一个)</label>
        <div className="flex gap-2 mb-2">
          <Input
            value={keyInput}
            onChange={(e) => setKeyInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), handleAddKey())}
            placeholder="例如: sqli-blind 或 MS17-010"
            className="flex-1"
          />
          <Button type="button" onClick={handleAddKey} variant="secondary" size="sm">
            添加
          </Button>
        </div>
        <div className="flex flex-wrap gap-2">
          {form.match_keys.map((k) => (
            <Badge key={k} className="bg-primary/10 text-primary border border-primary/20 px-2 py-1 text-xs">
              {k}
              <button
                type="button"
                onClick={() => handleRemoveKey(k)}
                className="ml-1 hover:text-red-400"
              >
                ×
              </button>
            </Badge>
          ))}
        </div>
      </div>

      <div>
        <label className="text-sm font-medium mb-1.5 block">模板标题</label>
        <Input
          value={form.title}
          onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))}
          placeholder="留空则保留 finding 原标题"
        />
      </div>

      <div>
        <label className="text-sm font-medium mb-1.5 block">漏洞描述</label>
        <textarea
          value={form.summary}
          onChange={(e) => setForm((p) => ({ ...p, summary: e.target.value }))}
          placeholder="留空则保留 finding 原描述"
          rows={4}
          className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50 resize-y"
        />
      </div>

      <div>
        <label className="text-sm font-medium mb-1.5 block">修复建议</label>
        <textarea
          value={form.remediation}
          onChange={(e) => setForm((p) => ({ ...p, remediation: e.target.value }))}
          placeholder="留空则保留 finding 原修复建议"
          rows={4}
          className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50 resize-y"
        />
      </div>

      <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
        <input
          type="checkbox"
          checked={form.enabled}
          onChange={(e) => setForm((p) => ({ ...p, enabled: e.target.checked }))}
          className="h-4 w-4"
        />
        启用该模板
      </label>

      <div className="flex gap-2 justify-end pt-2">
        {onCancel && (
          <Button type="button" onClick={onCancel} variant="secondary" disabled={saving}>
            取消
          </Button>
        )}
        <Button type="submit" loading={saving}>
          {mode === "create" ? "创建" : "保存"}
        </Button>
      </div>
    </form>
  );
}
```

**注意:** 这里假设 `Button` / `Input` / `Badge` 组件已经在 `components/index.ts` 导出。如果没有,需要相应调整。

- [ ] **Step 2: 运行 typecheck**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/TemplateEditor.tsx
git commit -m "feat(frontend): add TemplateEditor shared component

- chip 输入管理 match_keys
- 支持 create/edit 模式
- 被 VulnTemplatesPage 和 ReportsPage 复用

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 20: 前端 VulnTemplatesPage — 使用 TemplateEditor + chip 输入

**目标:** 把内联编辑器改为使用共享的 `TemplateEditor` 组件,`match_key` 输入改为 chip 输入。

**文件:**
- Modify: `frontend/src/pages/VulnTemplatesPage.tsx`

**步骤:**

- [ ] **Step 1: 导入 TemplateEditor**

在文件顶部 import 区域添加:
```typescript
import TemplateEditor from "../components/TemplateEditor";
```

- [ ] **Step 2: 删除旧的编辑表单状态和 UI**

删除原有的 `SOURCE_TOOL_OPTIONS` / `SEVERITY_OPTIONS` / `TemplateForm` / `EMPTY_FORM` 定义(第 22-59 行左右)。

删除 Modal 里的内联表单(第 369-464 行左右),改为用 `<TemplateEditor>`。

- [ ] **Step 3: 修改 handleSave 使用新结构**

找到 `handleSave` 函数,把 payload 构造改为:

```typescript
const payload: {
  source_tool: string;
  match_keys: string[];
  title: string;
  severity: string;
  summary: string;
  remediation: string;
  enabled: boolean;
} = {
  source_tool: form.source_tool.trim(),
  match_keys: form.match_keys,
  title: form.title.trim(),
  severity: form.severity,
  summary: form.summary,
  remediation: form.remediation,
  enabled: form.enabled,
};
```

- [ ] **Step 4: 替换 Modal 内容为 TemplateEditor**

把 Modal 的 `footer` 之上的 children 内容替换为:

```tsx
<Modal
  open={editorOpen}
  onClose={() => setEditorOpen(false)}
  title={editingId ? "编辑模板" : "新增模板"}
  description="匹配键可填:nuclei 的 template-id、其他工具的规则 ID,或漏洞标题精确文本。一个词条可挂多个匹配键。"
  footer={
    <>
      <Button variant="secondary" onClick={() => setEditorOpen(false)} disabled={saving}>
        取消
      </Button>
      <TemplateEditor
        initialValue={editingId ? undefined : {
          source_tool: "nuclei",
          match_keys: [],
          enabled: true,
        }}
        mode={editingId ? "edit" : "create"}
        onSaved={async (form) => {
          if (editingId) {
            await api.patchFindingTemplate(editingId, form);
            toast("模板已更新", "success");
          } else {
            await api.createFindingTemplate(form);
            toast("模板已创建", "success");
          }
          setEditorOpen(false);
          load();
        }}
      />
    </>
  }
>
  {/* TemplateEditor 自己渲染表单 */}
</Modal>
```

- [ ] **Step 5: 运行 typecheck**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 6: 提交**

```bash
git add frontend/src/pages/VulnTemplatesPage.tsx
git commit -m "refactor(frontend): VulnTemplatesPage use TemplateEditor component

- 删除内联编辑器,改用共享 TemplateEditor
- match_key 改为 match_keys chip 输入
- 简化 handleSave payload 构造

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 21: VulnTemplatesPage 表格列 — chip 显示 match_keys

**目标:** 表格中「匹配键」列改为 chip 形式,显示前 3 个 + 折叠「+N」。

**文件:**
- Modify: `frontend/src/pages/VulnTemplatesPage.tsx`

**步骤:**

- [ ] **Step 1: 修改表格列定义**

在 TableHeader 的「匹配键」列之后,改为 chip 渲染。找到 `<TableCell>{t.match_key}</TableCell>` 部分,替换为:

```tsx
<TableCell>
  <div className="flex flex-wrap gap-1">
    {t.match_keys.slice(0, 3).map((k) => (
      <Badge key={k} className="bg-primary/10 text-primary border border-primary/20 px-2 py-0.5 text-xs">
        {k}
      </Badge>
    ))}
    {t.match_keys.length > 3 && (
      <span className="text-xs text-muted-foreground">+{t.match_keys.length - 3}</span>
    )}
  </div>
</TableCell>
```

- [ ] **Step 2: 在文件顶部确保 Badge 已导入**

检查 `import` 里是否有 `Badge`,如果没有则添加:
```typescript
import { Badge } from "../components";
```

- [ ] **Step 3: 运行 typecheck**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 4: 提交**

```bash
git add frontend/src/pages/VulnTemplatesPage.tsx
git commit -m "feat(frontend): VulnTemplatesPage table show match_keys as chips

- 表格「匹配键」列改为 chip 显示
- 显示前 3 个 + 折叠 +N

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 22: VulnTemplatesPage 「导出 JSON」按钮文案修改

**目标:** 改按钮文案为「导出 JSON · 贴回仓库」,tooltip 解释用途。

**文件:**
- Modify: `frontend/src/pages/VulnTemplatesPage.tsx`

**步骤:**

- [ ] **Step 1: 修改导出按钮**

找到「导出 JSON」按钮,改为:

```tsx
<Button variant="outline" onClick={handleExportAll}>
  <Download className="mr-2 h-4 w-4" />
  导出 JSON · 贴回仓库
</Button>
```

- [ ] **Step 2: 添加 tooltip 或文案说明**

在按钮上方或附近添加说明文本,或在 handleExportAll 函数里的 toast 提示修改为「已下载 JSON,可粘贴到仓库 `docs/templates/vuln-templates.json` 共享给团队」

- [ ] **Step 3: 提交**

```bash
git add frontend/src/pages/VulnTemplatesPage.tsx
git commit -m "feat(frontend): clarify finding template export usage

- 按钮文案改为「导出 JSON · 贴回仓库」
- 提示用途:粘贴到仓库 docs/templates/vuln-templates.json

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 23: ReportsPage — 添加「词条覆盖状态」徽章

**目标:** 在 finding 列表每条上显示「已套词条:XXX」或「未套词条 · + 加入辞典」徽章。

**文件:**
- Modify: `frontend/src/pages/ReportsPage.tsx`

**步骤:**

- [ ] **Step 1: 导入 TemplateEditor**

在文件顶部 import:
```typescript
import TemplateEditor from "../components/TemplateEditor";
```

- [ ] **Step 2: 添加「加入辞典」Modal 状态**

在组件内部 useState 区域添加:
```typescript
const [dictModalOpen, setDictModalOpen] = useState(false);
const [dictFinding, setDictFinding] = useState<Finding | null>(null);
```

- [ ] **Step 3: 修改 finding 卡片,添加状态徽章**

在 finding 卡片的 `CardContent` 里,在 `<StatusBadge status={fd.finding.status} />` 旁边添加徽章逻辑。找到 status badge 行,添加:

```tsx
{fd.finding.template_id ? (
  <Badge className="bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-xs">
    已套词条
  </Badge>
) : (
  <Badge className="bg-amber-500/10 text-amber-400 border border-amber-500/20 text-xs">
    未套词条 · + 加入辞典
  </Badge>
)}
```

**注意:** 这里假设 finding 有 `template_id` 字段。如果没有,需要用其他方式判断(例如通过遍历 templates 并比较 match_keys)。暂时用假设实现,实际需要根据 API 返回的 finding 结构调整。

- [ ] **Step 4: 实现点击「未套词条」弹出的编辑器**

在「未套词条」Badge 上添加点击事件:
```tsx
{!fd.finding.template_id && (
  <Badge
    className="bg-amber-500/10 text-amber-400 border border-amber-500/20 text-xs cursor-pointer hover:bg-amber-500/20"
    onClick={() => {
      setDictFinding(fd.finding);
      setDictModalOpen(true);
    }}
  >
    未套词条 · + 加入辞典
  </Badge>
)}
```

- [ ] **Step 5: 添加 Modal 组件**

在页面末尾 return 的最外层 `</div>` 之前添加:

```tsx
<Modal
  open={dictModalOpen}
  onClose={() => setDictModalOpen(false)}
  title="加入漏洞辞典"
  description="为该扫描结果创建词条,以后扫描出相同漏洞会自动套用。"
  footer={
    <>
      <Button variant="secondary" onClick={() => setDictModalOpen(false)} disabled={saving}>
        取消
      </Button>
      <Button
        onClick={async () => {
          if (!dictFinding) return;
          const payload = {
            source_tool: dictFinding.source_tool,
            match_keys: [dictFinding.source_rule_id || dictFinding.title],
            title: "",
            severity: dictFinding.severity,
            summary: "",
            remediation: "",
            enabled: true,
          };
          await api.createFindingTemplate(payload);
          toast("词条已创建,刷新报告查看效果", "success");
          setDictModalOpen(false);
          loadData(); // 重新加载 finding
        }}
        loading={saving}
      >
        创建词条
      </Button>
    </>
  }
>
  {/* 暂时简化,不使用 TemplateEditor,直接 Modal 内表单 */}
  <div className="space-y-4">
    <div>
      <label className="text-sm font-medium">检测工具</label>
      <div className="text-sm text-muted-foreground">{dictFinding?.source_tool}</div>
    </div>
    <div>
      <label className="text-sm font-medium">匹配键</label>
      <div className="text-sm text-muted-foreground">
        {dictFinding?.source_rule_id || dictFinding?.title}
      </div>
    </div>
  </div>
</Modal>
```

**注意:** 这里使用了简化的 Modal 内容,没有用 TemplateEditor。实际实现时可以考虑用 TemplateEditor 但需要调整 initialValue 逻辑。

- [ ] **Step 6: 运行 typecheck**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 7: 提交**

```bash
git add frontend/src/pages/ReportsPage.tsx
git commit -m "feat(frontend): ReportsPage show template coverage status

- 显示「已套词条 / 未套词条」徽章
- 未套词条可点击「加入辞典」弹出创建词条弹窗
- 创建后刷新报告看效果

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 24: ReportsPage — 删除 JSON 导出按钮

**目标:** 删除「JSON Data (.json)」导出按钮和 `handleExport("json")` 分支。

**文件:**
- Modify: `frontend/src/pages/ReportsPage.tsx`

**步骤:**

- [ ] **Step 1: 删除 JSON 导出按钮**

找到导出区卡片中的「JSON Data (.json)」按钮,整行删除。

- [ ] **Step 2: 删除 handleExport 中的 "json" 分支**

找到 `handleExport` 函数,删除 `format === "json"` 的分支。只保留 `"md"` 分支。

- [ ] **Step 3: 提交**

```bash
git add frontend/src/pages/ReportsPage.tsx
git commit -m "refactor(frontend): ReportsPage remove JSON export button

- 删除 JSON Data (.json) 按钮
- 删除 handleExport("json") 分支

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 25: ReportsPage — 添加覆盖率统计

**目标:** 在「严重等级分布」上方添加「已套词条 X / 未套词条 Y」统计。

**文件:**
- Modify: `frontend/src/pages/ReportsPage.tsx`

**步骤:**

- [ ] **Step 1: 计算覆盖率**

在 `severityCounts` 定义后添加:

```typescript
const coverageCount = useMemo(() => {
  let covered = 0, uncovered = 0;
  for (const fd of findings) {
    if (fd.finding.template_id) {
      covered++;
    } else {
      uncovered++;
    }
  }
  return { covered, uncovered };
}, [findings]);
```

- [ ] **Step 2: 在「严重等级分布」上方显示统计**

在 severity 分布卡片网格之前添加:

```tsx
{findings.length > 0 && (
  <div className="text-sm text-muted-foreground">
    已套词条 <span className="text-emerald-400 font-semibold">{coverageCount.covered}</span> 项 ·
    未套词条 <span className="text-amber-400 font-semibold">{coverageCount.uncovered}</span> 项
  </div>
)}
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/pages/ReportsPage.tsx
git commit -m "feat(frontend): ReportsPage show template coverage stats

- 显示「已套词条 X / 未套词条 Y」统计
- 帮助用户判断辞典覆盖度

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 26: RunsPage — 删除报告流程相关状态和 UI

**目标:** 删除 `reports` / `generatingReports` / `checkedReportRuns` 状态,删除 SSE 监听里的 `report_progress` 处理,删除「生成报告」按钮和进度气泡,改为「查看报告」按钮导航到 ReportsPage。

**文件:**
- Modify: `frontend/src/pages/RunsPage.tsx`

**步骤:**

- [ ] **Step 1: 删除 useState**

删除以下行:
```typescript
const [reports, setReports] = useState<Map<string, Report>>(new Map());
const [generatingReports, setGeneratingReports] = useState<Set<string>>(new Set());
const [checkedReportRuns, setCheckedReportRuns] = useState<Set<string>>(new Set());
```

- [ ] **Step 2: 删除 useEffect 报告轮询逻辑**

删除第 128-149 行左右的 useEffect(`useEffect(() => { if (!runs.length) return; ...`), ...`)

- [ ] **Step 3: 删除 SSE report_progress 监听分支**

在 SSE 监听逻辑中,删除处理 `report_progress` 事件的分支(如果有)。

- [ ] **Step 4: 删除「生成报告」按钮和 handleGenerateReport**

删除调用 `api.createReport` 的按钮和 `handleGenerateReport` 函数。

- [ ] **Step 5: 替换为「查看报告」按钮**

在原来的「报告」按钮位置,替换为:

```tsx
<Button
  variant="ghost"
  size="sm"
  onClick={() => (window.location.href = `/projects/${projectId}/reports`)}
>
  查看报告
</Button>
```

- [ ] **Step 6: 删除 api.createReport/getReport 等调用**

确保没有残留的 `api.getReportByRun` / `api.createReport` / `api.getReport` / `api.downloadReport` 调用。

- [ ] **Step 7: 运行 typecheck**

Run: `cd /Users/kun/DEV/p0m32kun/frontend && npm run typecheck`
Expected: 无类型错误

- [ ] **Step 8: 提交**

```bash
git add frontend/src/pages/RunsPage.tsx
git commit -m "refactor(frontend): RunsPage remove async report flow

- 删除 reports/generatingReports/checkedReportRuns 状态
- 删除 useEffect 报告轮询
- 删除 SSE report_progress 监听
- 删除「生成报告」按钮和进度气泡
- 改为「查看报告」按钮,导航到 /projects/:id/reports

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 27: 文档同步 — internal/api/README.md

**目标:** 更新 API README 的 handler 总览和字段反向索引,反映 finding_template 改动和 report 异步 endpoint 删除。

**文件:**
- Modify: `internal/api/README.md`

**步骤:**

- [ ] **Step 1: 更新 finding_template handler 行描述**

在「Handler 文件总览」表中找到 `finding_template_handlers.go` 行,更新「路径前缀」列和描述:
- 路径前缀改为 `/finding-templates` (如果变了)
- 描述改为:「管理漏洞模板词条(辞典),支持 match_keys 数组,应用层唯一性检查」

- [ ] **Step 2: 更新字段反向索引**

如果 README 有「字段反向索引」表:
- 删除 `match_key` 字段行
- 新增 `match_keys` 字段行,标注「消费 handler 文件: finding_template_handlers.go」

- [ ] **Step 3: 更新 report handler 行描述**

在「Handler 文件总览」表中找到 `report_handlers.go` 行,更新:
- 路径前缀改为 `/projects/:id/reports` (只剩 export.md)
- 描述改为:「报告导出(仅 Markdown 同步下载),已删除异步报告持久化流程」

- [ ] **Step 4: 删除异步报告 endpoint 描述**

如果有「API 路由总览」表,删除以下行:
- `POST /runs/:runId/reports`
- `GET /reports/:reportId`
- `GET /runs/:runId/reports`
- `GET /reports/:reportId/download`
- `DELETE /reports/:reportId`
- `GET /reports`

- [ ] **Step 5: 提交**

```bash
git add internal/api/README.md
git commit -d "docs(api): sync README after vulnerability template redesign

- 更新 finding_template handler 描述(match_keys 数组)
- 更新 report handler 描述(仅 Markdown 导出)
- 删除异步报告 endpoint 文档
- 更新字段反向索引

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 28: 文档同步 — docs/current/architecture.md

**目标:** 更新架构文档中关于漏洞模板 / 报告导出 / RunsPage 报告流程的描述,与当前实现一致。

**文件:**
- Modify: `docs/current/architecture.md`

**步骤:**

- [ ] **Step 1: 定位相关章节**

Run: `grep -n "模板\|template\|report\|Report\|RunsPage" /Users/kun/DEV/p0m32kun/docs/current/architecture.md`
Expected: 输出行号

- [ ] **Step 2: 更新漏洞模板描述**

把模板描述从「字段覆盖工具」改为「漏洞辞典」,强调 match_keys 数组、聚合渲染、未命中同位混排。

- [ ] **Step 3: 更新报告导出描述**

删除 HTML/JSON 导出的描述,强调只保留 Markdown 同步下载。删除异步报告持久化流程的描述。

- [ ] **Step 4: 更新或删除 RunsPage 报告流程描述**

删除 RunsPage 中「生成报告按钮 → 异步生成 → SSE 推进度 → 下载」的描述,改为「查看报告按钮 → 跳转到 ReportsPage」。

- [ ] **Step 5: 提交**

```bash
git add docs/current/architecture.md
git commit -d "docs(architecture): sync after vulnerability dictionary redesign

- 更新漏洞模板为「辞典」定位
- 更新报告导出:仅 Markdown,删除异步流程
- 更新 RunsPage:查看报告按钮

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 29: E2E 验证

**目标:** 跑通完整的端到端流程:扫描 → 词条匹配 → 报告生成 → 前端展示。

**文件:** 无(执行命令)

**步骤:**

- [ ] **Step 1: 起 server + worker**

```bash
cd /Users/kun/DEV/p0m32kun
go run main.go
# 或 docker-compose 方式启动
```

- [ ] **Step 2: 跑一次 nuclei 扫描(任意目标)**

```bash
# 使用 CLI 或前端触发
```

- [ ] **Step 3: 在 VulnTemplatesPage 创建词条**

- 打开 `http://localhost:8080/vuln-templates`
- 点击「新增模板」
- 填 source_tool, match_keys(至少一个), title, severity, summary, remediation
- 保存

- [ ] **Step 4: 在 ReportsPage 刷新,验证词条覆盖状态**

- 打开 `http://localhost:8080/projects/:id/reports`
- 确认看到「已套词条:XXX」绿色徽章
- 确认看到「未套词条」橙色徽章

- [ ] **Step 5: 点击「+ 加入辞典」弹窗**

- 点击未套词条徽章
- 弹窗打开,预填 source_tool / match_keys / title / severity
- 填中文标题/summary/remediation
- 保存

- [ ] **Step 6: 刷新报告,验证新词条命中**

- 刷新页面
- 确认该 finding 变成绿色「已套词条」

- [ ] **Step 7: 下载 Markdown 报告**

- 点击「导出 MD」
- 打开下载的 md 文件,验证章节结构:
  - 命中词条合并一节,表格列出多个 finding
  - 未命中单条一节
  - 章节编号连续
  - 底部脚注区分「套用词条 / 未套词条」

- [ ] **Step 8: VulnTemplatesPage 导出 JSON**

- 点击「导出 JSON · 贴回仓库」
- 打开下载的 json,验证是 `match_keys` 数组形态

- [ ] **Step 9: RunsPage 验证**

- 打开 `http://localhost:8080/projects/:id/runs`
- 确认只有「查看报告」按钮
- 点击后跳转到 `/projects/:id/reports`
- 确认没有「生成报告」按钮/进度气泡

- [ ] **Step 10: 测试回滚(可选)**

停止 server,切回旧分支,重新启动,确认没有死链接/错误。

- [ ] **Step 11: 提交 E2E 结果**

```bash
git add docs/design/vuln-template-redesign.md
git commit -m "docs(design): move vuln-template-redesign to current, mark active

- 实施完成,spec 从 design 移到 current
- frontmatter 改为 status: active
- source_of_truth: true

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## 验收标准

所有任务完成后:

- ✅ **后端单元测试:** `go test ./internal/db/... ./internal/report/... ./internal/api/...` 全绿
- ✅ **前端 typecheck:** `cd frontend && npm run typecheck` 通过
- ✅ **E2E:** Task 29 的 11 步全部通过
- ✅ **文档同步:** `internal/api/README.md` 和 `docs/current/architecture.md` 已更新
- ✅ **提案归档:** `docs/design/vuln-template-redesign.md` 移到 `docs/current/` 并标记 active

---

**Plan written and saved to `docs/active/plan/2026-05-19-vuln-template-redesign.md`.**

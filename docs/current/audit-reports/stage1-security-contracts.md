# 阶段一审计报告：安全性 & 契约一致性

> 审计日期：2026-05-13
> 审计范围：internal/（Go 后端）、frontend/src/lib/api.ts（前端类型）
> 工具链状态：go vet ./internal/... 通过，npx tsc --noEmit 通过

---

## 1. 契约差异报告（1.1）

### 1.1.1 后端模型 → 前端类型映射总览

| 后端模型 | 前端类型 | 映射状态 |
|---------|---------|---------|
| `models.Project` | `Project` | 基本对齐，字段差异见下 |
| `models.Target` | `Target` | 基本对齐，缺失 `source`/`status` |
| `models.Asset` | `Asset` | 基本对齐 |
| `models.WebEndpoint` | `WebEndpoint` | 基本对齐 |
| `models.Port` | `Port` | 基本对齐 |
| `models.Service` | `Service` | 基本对齐 |
| `models.ServicePort` | `ServicePort` | 基本对齐 |
| `models.Finding` | `Finding` | 严重差异：后端 `source_rule_id` 非指针，前端可选 |
| `models.Evidence` | `Evidence` | 差异：`created_by` 后端非指针，前端可选 |
| `models.WorkerNode` | `WorkerNode` | 严重差异：后端多 7 个字段，前端缺失 |
| `models.ScopeRule` | `ScopeRule` | 基本对齐，缺失 `updated_at` |
| `models.ScanTask` | `ScanTask` | 严重差异：后端 15+ 字段，前端仅 8 个 |
| `models.Run` | `Run` | 基本对齐，后端多 `tool_template_id` |
| `models.Report` | `Report` | 基本对齐 |
| `models.PipelineRun` | `PipelineRun` | 差异：`started_at` 后端非指针，前端非可选 |
| `models.PipelineRunStage` | `PipelineRunStage` | 基本对齐 |
| `models.ToolTemplate` | `ToolTemplate` | 差异：后端多 `profile_type` 等字段 |
| `models.Dictionary` | `Dictionary` | 基本对齐 |
| `models.HttpxFingerprint` | `HttpxFingerprint` | 基本对齐 |
| `models.FindingTemplate` | `FindingTemplate` | 基本对齐 |
| `models.NucleiCustomSource` | `NucleiCustomSource` | 基本对齐 |
| `models.NucleiCustomFileEntry` | `NucleiCustomFileEntry` | 基本对齐 |
| `models.NucleiCustomValidationResult` | `NucleiCustomValidationResult` | 基本对齐 |
| `models.NucleiCustomManifest` | `NucleiCustomManifest` | 差异：后端 `ManifestJSON` vs 前端 `version`+`sources` |
| `models.EngineCredential` | `EngineCredential` | 差异：后端多 `updated_at` |
| `models.DashboardStats` | `DashboardStats` | 基本对齐 |

### 1.1.2 具体字段差异清单

| 后端字段 | 文件:行 | 前端字段 | 差异描述 | 风险 |
|---------|--------|---------|---------|------|
| `Target.Source` | target.go:26 | 缺失 | 前端 `Target` 无 `source` 字段 | Medium |
| `Target.Status` | target.go:27 | 缺失 | 前端 `Target` 无 `status` 字段 | Medium |
| `Finding.SourceRuleID` | finding.go:40 | `source_rule_id?: string` | 后端 `string`（非空），前端 `?`（可选） | High |
| `Evidence.CreatedBy` | finding.go:75 | `created_by?: string` | 后端 `string`（非空），前端 `?`（可选） | Medium |
| `WorkerNode.TrustLevel` | worker.go:32 | 缺失 | 前端无此字段 | Low |
| `WorkerNode.NetworkProfile` | worker.go:33 | 缺失 | 前端无此字段 | Low |
| `WorkerNode.Capabilities` | worker.go:34 | 缺失 | 前端无此字段 | Low |
| `WorkerNode.ToolVersions` | worker.go:35 | 缺失 | 前端无此字段 | Low |
| `WorkerNode.TemplateVersions` | worker.go:36 | 缺失 | 前端无此字段 | Low |
| `WorkerNode.MaxConcurrency` | worker.go:37 | 缺失 | 前端无此字段 | Low |
| `WorkerNode.RevokedAt` | worker.go:40 | 缺失 | 前端无此字段 | Low |
| `ScanTask.PlanID` | scan.go:48 | 缺失 | 前端 `ScanTask` 无 `plan_id` | Medium |
| `ScanTask.RunID` | scan.go:49 | 缺失 | 前端 `ScanTask` 无 `run_id` | Medium |
| `ScanTask.CommandTemplate` | scan.go:53 | 缺失 | 前端 `ScanTask` 无 `command_template` | Medium |
| `ScanTask.ArgumentsRedacted` | scan.go:54 | 缺失 | 前端 `ScanTask` 无 `arguments_redacted` | Medium |
| `ScanTask.WorkerID` | scan.go:60 | 缺失 | 前端 `ScanTask` 无 `worker_id` | Medium |
| `PipelineRun.StartedAt` | scan.go:176 | `started_at: string` | 后端 `time.Time`（非指针），前端非可选但后端可能零值 | Low |
| `ScopeRule.UpdatedAt` | scope.go:26 | 缺失 | 前端 `ScopeRule` 无 `updated_at` | Low |
| `Project.StartTime` | project.go 无此字段 | `start_time?: string` | 前端有 `start_time`/`end_time`，后端 `Project` 模型无此字段 | **Critical** |
| `Project.EndTime` | project.go 无此字段 | `end_time?: string` | 前端有 `end_time`，后端 `Project` 模型无此字段 | **Critical** |
| `Project.RateLimit` | project.go:12 | `rate_limit?: number` | 后端 `int`（非指针），前端 `?`（可选） | Medium |
| `Project.DefaultProfile` | project.go:14 | `default_profile?: string` | 后端 `string`（非空），前端 `?`（可选） | Medium |
| `Project.PortRange` | project.go:13 | `port_range?: string` | 后端 `*string`（指针），前端 `?`（可选）— 语义对齐 | Low |
| `Project.PipelineConfig` | project.go:15 | `pipeline_config?: string` | 后端 `*string`，前端 `?` — 语义对齐 | Low |
| `NucleiCustomManifest.ManifestJSON` | nuclei_custom.go:44 | 缺失 | 后端 `manifest_json` 字符串，前端解构为 `version`+`sources` | Medium |

### 1.1.3 前后端未对齐风险字段

| 风险类型 | 涉及字段 | 说明 |
|---------|---------|------|
| `*string` vs `string` | `Project.Organization`, `Project.Purpose` | 后端 `string`（非空），前端 `?`（可选）。若后端返回 `""`，前端无问题；若前端传 `undefined`，后端接收 `""` |
| `*int` vs `number \| undefined` | `WebEndpoint.Port`, `WebEndpoint.StatusCode` | 语义对齐，但后端 JSON 序列化时 `*int` 为 `null` 或数字，前端 `?` 接收时无 `null` 类型 |
| enum string vs union type | `Finding.Severity` | 后端 `FindingSeverity`（typed string），前端 `string` — 类型安全丢失 |
| `time.Time` vs `string` | 所有时间字段 | 后端序列化为 RFC3339 字符串，前端接收为 `string` — 需运行时验证格式 |

---

## 2. Nil-Safety 风险清单（1.2）

### 1.2.1 数据库 NULL 列扫描审计

| 文件 | 行号 | 列名 | 模型类型 | 扫描方式 | 状态 |
|------|------|------|---------|---------|------|
| queries_finding.go | 139 | `asset_id` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_finding.go | 139 | `service_id` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_finding.go | 139 | `web_endpoint_id` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_finding.go | 196 | `artifact_id` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_finding.go | 196 | `created_by` | `string` | `sql.NullString` + `.Valid` 判断 | **安全（已修复）** |
| queries_finding.go | 234 | `artifact_id` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_finding.go | 234 | `created_by` | `string` | `sql.NullString` + `.Valid` 判断 | **安全（已修复）** |
| queries_asset.go | 187 | `port_id` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_asset.go | 277 | `port` | `*int` | `sql.NullInt64` + 指针赋值 | 安全 |
| queries_asset.go | 278 | `status_code` | `*int` | `sql.NullInt64` + 指针赋值 | 安全 |
| queries_asset.go | 279 | `screenshot_artifact_id` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_scan.go | 37-40 | 多个 NULL 列 | 混合 | `sql.NullString`/`sql.NullInt64`/`sql.NullTime` | 安全 |
| queries_worker.go | 25 | `last_seen` | `*time.Time` | `sql.NullTime` + 指针赋值 | 安全 |
| queries_worker.go | 25 | `revoked_at` | `*time.Time` | `sql.NullTime` + 指针赋值 | 安全 |
| queries_health.go | 38 | `proxy_reachable` | `*bool` | `sql.NullBool` + `nullableBool()` | 安全 |
| queries_health.go | 39 | `template_path` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_nuclei.go | 78 | `uri`/`branch`/`last_error` | `*string` | `sql.NullString` + `nullableString()` | 安全 |
| queries_nuclei.go | 79 | `last_sync_at`/`last_validate_at` | `*time.Time` | `sql.NullTime` + 指针赋值 | 安全 |
| queries_report.go | 82-83 | `title`/`file_path`/`error_msg`/`completed_at` | 混合 | `sql.NullString`/`sql.NullTime` | 安全 |
| queries_target.go | 199 | `hostname` | `*string` | `sql.NullString` + `nullableString()` | 安全 |

### 1.2.2 指针反引用高风险点

| 文件 | 行号 | 变量 | nil 可能性 | 风险 | 修复建议 |
|------|------|------|-----------|------|---------|
| workflow/discovery.go | 173 | `port` | `parseInt` 成功后才赋值，但 `hr.Port` 为空字符串时 `port` 为 nil | Low | 已在后续使用处做空值处理 |
| workflow/discovery.go | 180 | `statusCode` | 同上，`hr.StatusCode > 0` 时才赋值 | Low | 安全 |
| workflow/discovery.go | 377 | `port` | 同上 | Low | 安全 |
| workflow/discovery.go | 384 | `statusCode` | 同上 | Low | 安全 |
| api/worker_handlers.go | 64 | `worker.LastSeen` | 新注册 worker 初始化时 `&now`，非 nil | Low | 安全 |
| api/server.go | 140 | `w.LastSeen` | 可能 nil（从未心跳的 worker） | Medium | 已在 159 行检查 `w.LastSeen == nil` |
| scope/scope.go | 119 | `latest.TaskID` | 可能 nil（新决策无 task） | Low | 已在 118 行检查 |

### 1.2.3 Pitfall 修复验证

| Pitfall | 文件 | 验证结果 | 状态 |
|---------|------|---------|------|
| 20260426-artifact-type-mismatch | worker/server.go:280-304 | Worker 现在扫描 workdir 文件并标记 `jsonl` 类型；workflow 有 stdout fallback | **已修复** |
| 20260426-raw-artifact-redaction-loss | workflow/screenshot.go:392-433 | `saveEvidenceArtifact` 先保存原始数据（`RedactionStatus: "raw"`），`Excerpt` 使用脱敏数据 | **已修复** |
| 20260427-null-scan-crash | queries_finding.go:196-203 | `ListEvidenceByFinding` 使用 `sql.NullString` 接收 `created_by`，通过 `.Valid` 判断 | **已修复** |
| 20260427-markdown-pipe-corruption | report/markdown.go:12-16 | `escapeMDTable` 转义 `\|` 和 `\n`，所有表格值均经过此函数 | **已修复** |

---

## 3. Scope Check 调用图 & 绕过风险（1.3）

### 1.3.1 Scope Check 决策树

```
Check(projectID, target)
  ├── GetProject(projectID)
  │   └── RateLimit < 0 ? → Deny
  ├── ListScopeRulesByProject(projectID)
  └── evaluate(target, rules)
        ├── 遍历规则，matchRule(target, rule)
        │   ├── Domain: matchDomainRule (精确/通配符/子域)
        │   ├── URL: matchURL (前缀/域名/IP/CIDR)
        │   ├── IP/CIDR: matchIP (精确/CIDR 包含)
        │   └── Company: 不直接匹配
        ├── Exclude 优先于 Include
        └── 无匹配 → Deny（白名单模式）
```

### 1.3.2 Scope Check 调用点追踪

| 调用文件 | 行号 | 调用函数 | 被检查对象 | deny 后行为 |
|---------|------|---------|-----------|------------|
| workflow/discovery.go | 107 | `w.scope.Check` | 初始 domain target | `continue`（跳过） |
| workflow/discovery.go | 128 | `w.scope.Check` | subfinder 发现的子域 | `continue`（跳过） |
| workflow/discovery.go | 160 | `w.scope.Check` | httpx 发现的 URL | `continue`（跳过） |
| workflow/discovery.go | 224 | `w.scope.Check` | 初始 IP target | `continue`（跳过） |
| workflow/discovery.go | 260 | `w.scope.Check` | 初始 CIDR target | `continue`（跳过） |
| workflow/discovery.go | 309 | `w.scope.CheckIP` | naabu 发现的 IP | `continue`（跳过） |
| workflow/discovery.go | 364 | `w.scope.Check` | post-discovery 的 URL | `continue`（跳过） |
| workflow/screenshot.go | 60 | `w.scope.Check` | WebEndpoint URL | `continue`（跳过） |
| workflow/screenshot.go | 205 | `w.scope.CheckIP` | 网络扫描资产 IP | `continue`（跳过） |
| workflow/pipeline.go | 151 | `p.scope.FilterTargets` | 所有 pipeline targets | 过滤后返回允许列表 |
| workflow/pipeline_tool.go | 25 | `p.scope.ValidateBeforeRun` | 任务运行前 target | 错误返回，阻止任务 |

### 1.3.3 资产入库路径 Scope Gate 验证

| 资产来源 | 入库路径 | Scope Gate | 状态 |
|---------|---------|-----------|------|
| FOFA 结果 | workflow/pipeline_flow.go → runCompanyFlow | 通过 `FilterTargets` + `Check` | 有 gate |
| Subfinder 子域 | workflow/discovery.go:128 | `w.scope.Check` | 有 gate |
| httpx URL | workflow/discovery.go:160 | `w.scope.Check` | 有 gate |
| Naabu IP | workflow/discovery.go:309 | `w.scope.CheckIP` | 有 gate |
| Naabu Port | workflow/discovery.go:318 | 通过 IP asset 间接 | **Port 本身不过 scope** |
| CDN 结果 | workflow/discovery.go 无 CDN 步骤 | CDN 在 pipeline_flow 中 | 需单独验证 |

### 1.3.4 边界条件 & 绕过风险

| # | 风险场景 | 风险等级 | 说明 |
|---|---------|---------|------|
| 1 | **CDN IP 未过 scope** | Medium | CDN 检测在 pipeline_flow 中，检测到的 CDN IP 写入 `cdn` 表时未调用 scope check。CDN IP 可能不属于目标组织。 |
| 2 | **CIDR 展开后每个 IP 单独过 scope** | Low | `runCIDRChain` 先对 CIDR 本身 check，展开后 `runPostDiscovery` 对每个 IP 调用 `CheckIP`。正确。 |
| 3 | **URL 重定向后的域名未重新过 scope** | Medium | httpx 可能跟随重定向，重定向后的域名未再次 scope check。当前仅对原始 URL check。 |
| 4 | **Company 展开的资产过 scope** | Low | `FilterTargets` 对 company 类型 pass through，由 `runCompanyFlow` 通过 FOFA 展开后逐个 check。正确。 |
| 5 | **Port 资产无独立 scope** | Low | Port 是 IP 的子资源，通过 IP 间接受 scope 约束。但 Port 值本身（如 22, 3389）不涉及 scope。可接受。 |
| 6 | **WebEndpoint 创建时 host asset 未单独 check** | Low | `runDomainChain` 创建 host asset 时（domain 类型）与 URL 同时 check，但 host 提取自 URL，已隐含在 URL check 中。 |

### 1.3.5 应测边界场景（不写代码）

1. **CDN IP scope 绕过**：配置 `exclude 1.1.1.1`，FOFA 返回的 CDN 结果含 `1.1.1.1`，验证是否被写入 `cdn` 表
2. **URL 重定向 scope 绕过**：httpx 发现 `http://evil.com` 重定向到 `http://target.com`，验证重定向后的域名是否过 scope
3. **CIDR 部分 IP 被排除**：配置 `exclude 192.168.1.5`，扫描 `192.168.1.0/24`，验证 `.5` 是否被排除
4. **Company 展开后 exclude**：配置 `exclude sub.example.com`，FOFA 展开 `example.com` 时发现 `sub.example.com`，验证是否被排除
5. **Scope rule 修改后 TOCTOU**：先 allow 后修改 rule 为 exclude，验证 `ValidateBeforeRun` 是否重新 check

---

## 4. Ghost Worker 风险清单（1.4）

### 1.4.1 Worker 注册逻辑审查

| 检查项 | 文件:行 | 结果 | 风险 |
|--------|--------|------|------|
| 同名 worker 重复注册检查 | worker_handlers.go:43-82 | **无去重检查**。每次注册生成新 ID 直接插入，同一 endpoint 可注册多次 | **High** |
| 心跳时间戳初始化 | worker_handlers.go:64 | `LastSeen: &now` 正确初始化 | 安全 |
| Token 生成 | worker_handlers.go:37-41 | `crypto/rand` 生成 32 字节，安全 | 安全 |

### 1.4.2 心跳超时处理审查

| 检查项 | 文件:行 | 结果 | 风险 |
|--------|--------|------|------|
| 心跳超时检测 | server.go:128-174 | `cleanupStaleWorkers()` goroutine，每 60s 检查，120s 无心跳标记 offline | 安全 |
| 超时 worker 任务重新分配 | server.go:167-169 | 仅关闭 taskQueue channel，**未重新分配未完成任务** | **High** |
| 离线 worker 清理 | server.go:139-153 | 离线 168 小时（7 天）后删除 | 安全 |
| Server 启动时清理 | server.go:102-126 | `markAllWorkersOffline()` 将所有非 offline worker 标记 offline | 安全 |

### 1.4.3 Worker 注销与清理

| 检查项 | 文件:行 | 结果 | 风险 |
|--------|--------|------|------|
| 注销 API | worker_handlers.go:253-261 | `handleRevokeWorker` 设置 `revoked_at`，但 worker 仍存在于 DB | Medium |
| 删除限制 | worker_handlers.go:263-294 | 仅允许删除 `status == offline` 的 worker | 安全 |
| 任务队列清理 | worker_handlers.go:280-285 | 删除时关闭 channel 并从 map 移除 | 安全 |
| Docker 重启场景 | server.go:97-99 | 启动时 `markAllWorkersOffline()` 处理 | 安全 |

### 1.4.4 Pitfall 修复验证（20260428-ghost-worker-cleanup）

| 修复项 | 预期 | 实际 | 状态 |
|--------|------|------|------|
| 心跳清理 goroutine | 有 | `cleanupStaleWorkers()` 存在 | 已修复 |
| 列表不过滤（保留历史） | API 返回全部 | `handleListWorkers` 返回全部 | 已修复 |
| Endpoint 去重 | 应有 | **缺失** | **未修复** |
| 超时任务重分配 | 应有 | **缺失** | **未修复** |

---

## 5. 契约回归测试骨架（1.5）

### 1.5.1 null-scan-crash 回归测试

**建议文件**：`internal/db/queries_finding_null_test.go`（新建）

**测试用例**：

1. `TestListEvidenceByFinding_NullCreatedBy`
   - 准备：插入一条 `evidence` 记录，`created_by = NULL`
   - 调用：`queries.ListEvidenceByFinding(findingID)`
   - 断言：返回 `[]*models.Evidence`，长度 1，`CreatedBy == ""`，无 panic

2. `TestListEvidenceByFindingIDs_NullCreatedBy`
   - 准备：插入多条 evidence，部分 `created_by = NULL`
   - 调用：`queries.ListEvidenceByFindingIDs([]string{findingID1, findingID2})`
   - 断言：返回 map，每个 finding 下的 evidence `CreatedBy` 正确（空字符串或实际值），无 panic

3. `TestListEvidenceByFinding_MixedCreatedBy`
   - 准备：同一 finding 下插入 3 条 evidence：`created_by = NULL`、`created_by = 'admin'`、`created_by = ''`
   - 断言：全部正确扫描，`NULL` → `""`，非 NULL → 原值

### 1.5.2 markdown-pipe-corruption 回归测试

**建议文件**：`internal/report/markdown_test.go`（新建，或补充到现有 report_test.go）

**测试用例**：

1. `TestEscapeMDTable_PipeInTitle`
   - 输入：`Finding{Title: "SQL Injection | XSS | CSRF"}`
   - 调用：`GenerateMarkdown(reportData)`
   - 断言：输出中包含 `SQL Injection \| XSS \| CSRF`，Markdown 表格结构不被破坏

2. `TestEscapeMDTable_PipeInAssetValue`
   - 输入：`Asset{Value: "http://example.com?a=1|b=2"}`
   - 调用：`writeAffectedTargets(sb, rf)`
   - 断言：表格列数正确（3 列），管道符被转义

3. `TestEscapeMDTable_NewlineInExcerpt`
   - 输入：`Evidence{Excerpt: "line1\nline2"}`
   - 断言：输出中换行符被替换为空格

4. `TestEscapeMDTable_NoDoubleEscape`
   - 输入：`"already \| escaped"`
   - 断言：输出为 `"already \\| escaped"`（验证不重复转义）

### 1.5.3 artifact-type-mismatch 回归测试

**建议文件**：`internal/workflow/discovery_test.go`（新建）

**测试用例**：

1. `TestParseSubfinderOutput_FallbackToStdout`
   - 准备：创建 task，写入 stdout artifact（JSONL 格式），无 jsonl artifact
   - 调用：`workflow.parseSubfinderOutput(taskID)`
   - 断言：成功解析，结果非空，无 error

2. `TestParseHttpxOutput_FallbackToStdout`
   - 准备：同上，stdout 含 httpx JSONL 输出
   - 调用：`workflow.parseHttpxOutput(taskID)`
   - 断言：成功解析

3. `TestParseNaabuOutput_FallbackToStdout`
   - 准备：同上，stdout 含 naabu JSONL 输出
   - 调用：`workflow.parseNaabuOutput(taskID)`
   - 断言：成功解析

4. `TestWorkerArtifactType_JSONLFile`
   - 准备：模拟 worker 执行，workdir 产生 `.json` 文件
   - 调用：检查 worker 上报的 artifact type
   - 断言：`artifactType == "jsonl"`

---

## 6. 汇总

### 风险统计

| 等级 | 数量 | 问题简述 |
|------|------|---------|
| **Critical** | 2 | `Project` 后端缺失 `start_time`/`end_time` 字段；前端有而后端无 |
| **High** | 4 | Worker 注册无去重；超时任务未重分配；`Finding.SourceRuleID` 前后端可选性不一致；`ScanTask` 前后端字段严重不匹配 |
| **Medium** | 10 | CDN IP 未过 scope；URL 重定向后未重新 scope；多个前后端字段差异；Target 缺失 source/status |
| **Low** | 15 | WorkerNode 字段缺失；ScopeRule 缺失 updated_at；其他次要字段差异 |

### 最紧迫的 3 个问题

1. **Critical: Project 模型缺失时间窗口字段** — 后端 `Project` 无 `start_time`/`end_time`，但前端有。`scope.Check` 中引用了时间窗口检查（`checkTimeWindow` 函数存在但为空实现），实际未生效。这导致 scope 的时间窗口控制完全失效。

2. **High: Worker 注册无去重** — 同一 endpoint 可无限注册，虽然 cleanup 会在 7 天后删除离线 worker，但短期内仍会积累大量重复记录，影响调度器性能。

3. **High: 超时任务未重新分配** — `cleanupStaleWorkers` 将超时 worker 标记 offline 并关闭 channel，但已分配给该 worker 的 `running` 状态任务不会被重新入队，导致任务永久挂起。

### 已验证的 Pitfall 修复

| Pitfall | 状态 |
|---------|------|
| 20260426-artifact-type-mismatch | 已修复（stdout fallback + jsonl 文件扫描） |
| 20260426-raw-artifact-redaction-loss | 已修复（原始数据保存为 raw，excerpt 用脱敏数据） |
| 20260427-null-scan-crash | 已修复（sql.NullString + .Valid 判断） |
| 20260427-markdown-pipe-corruption | 已修复（escapeMDTable 全局转义） |
| 20260428-ghost-worker-cleanup | 部分修复（心跳清理 + 启动清理已实现，去重 + 任务重分配缺失） |

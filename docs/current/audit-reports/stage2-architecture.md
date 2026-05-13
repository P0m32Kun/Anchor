# 阶段二：架构分层 & 耦合度审计报告

> 审计日期：2026-05-13
> 审计范围：internal/ 下全部 Go 代码
> 工具链：go vet (pass), tsc --noEmit (pass)

---

## 2.1 分层违规清单

### 2.1.1 api -> db 直接依赖（3 处，非 4 处）

| 文件 | 行号 | 使用 db 的场景 | 风险等级 | 修复建议 |
|------|------|---------------|----------|----------|
| `internal/api/server.go` | 14 | Server 结构体持有 `*db.Queries` 和 `*sql.DB`；NewServer 直接接收 db 依赖并分发给各子模块 | Medium | 保持现状 — Server 作为依赖注入根容器，需持有底层连接。但应逐步将 `queries` 字段从 Server 中移除，改为仅通过 Service 访问 |
| `internal/api/finding_template_handlers.go` | 9 | 直接调用 `s.queries.ListFindingTemplates` / `CreateFindingTemplate` / `GetFindingTemplate` / `UpdateFindingTemplate` / `DeleteFindingTemplate` | Medium | 应下沉到 `service.FindingTemplateService`，handler 只负责 HTTP 层 |
| `internal/api/retest_handlers.go` | 8 | `handleBatchUpdateFindingStatus` 中直接开启事务 `s.rawDB.Begin()` 并创建 `db.New(tx)` | High | 事务管理应下沉到 Service 层；handler 不应直接操作 `*sql.DB` |

**注**：审计计划基线声称 4 处（含 `handlers_test.go`），但 `handlers_test.go:15` 的 `internal/db` import 属于测试辅助代码（setupTestServer 需要内存数据库迁移），**测试中的直接依赖可接受**。

### 2.1.2 api -> models 直接依赖（15 处，非 17 处）

实际扫描 `internal/api/*.go`（排除 _test.go），发现 15 个文件 import models：

```
dashboard_handlers.go, asset_handlers.go, engine_handlers.go,
finding_template_handlers.go, handlers.go, httpx_fingerprint_handlers.go,
dictionary_handlers.go, pipeline_handlers.go, report_handlers.go,
run_handlers.go, retest_handlers.go, server.go, task_handlers.go,
worker_handlers.go, scope_handlers.go
```

**分类**：

| 类别 | 文件 | 使用 models 的方式 | 是否可接受 |
|------|------|-------------------|-----------|
| 纯 JSON 序列化/反序列化 | 全部 15 个 | 用 models 结构体做 HTTP 响应体 | 可接受 — models 作为 DTO 是合理的 |
| 业务判断（应下沉） | `retest_handlers.go` | `validStatuses` map 检查 + `models.FindingStatus(req.Status)` | 应下沉到 FindingService |
| 业务判断（应下沉） | `finding_template_handlers.go` | 直接构造 `models.FindingTemplate` 并写 DB | 应下沉到 Service |
| 业务判断（应下沉） | `pipeline_handlers.go` | `buildConfigForMode` 含大量业务逻辑（mode→config 映射） | 应下沉到 PipelineService |
| 业务判断（应下沉） | `asset_handlers.go` | `handleListServicePorts` 中大量聚合逻辑（200+行） | 应下沉到 AssetService |
| 业务判断（应下沉） | `report_handlers.go` | `generateReport` 含异步报告生成全流程 | 应下沉到 ReportService |

### 2.1.3 其他分层违规

| 检查项 | 结果 | 说明 |
|--------|------|------|
| `internal/db/` import `internal/api/` | 无违规 | 已验证 |
| `internal/models/` import `internal/db/` | 无违规 | 已验证 |
| `internal/workflow/` import `internal/api/` | 无违规 | 已验证 |
| `internal/workflow/` import `internal/worker/` | **存在** | `workflow/discovery.go:20` import `internal/worker` — workflow 调用 worker.Runner 是设计意图，但 Runner 同时被 api.Server 持有，形成共享依赖 |

**逆依赖风险**：无循环依赖，但 `workflow -> worker` 的依赖意味着 workflow 层与执行层耦合。理想情况下 workflow 应通过接口调用执行层。

---

## 2.2 Service 层缺口清单

### 2.2.1 当前 Service 覆盖范围

| Service | 文件 | 覆盖业务域 | 接口方法数 |
|---------|------|-----------|----------|
| ProjectService | `internal/service/project.go` | 项目 CRUD | 5 |
| TargetService | `internal/service/target.go` | 目标 CRUD + 导入 + Scope 检查 | 4 |
| FindingService | `internal/service/finding.go` | Finding CRUD + 状态更新 + Evidence | 7 |

**已覆盖**：Project、Target、Finding（基础 CRUD）

### 2.2.2 Handler 中超过 30 行的函数（应迁移到 Service）

| 文件 | 函数 | 行数 | 业务逻辑内容 | 建议 Service |
|------|------|------|-------------|-------------|
| `asset_handlers.go:131` | `handleListServicePorts` | ~186L | IP→Asset 映射、指纹合并、WebEndpoint 合并、Port 合并、排序 | `AssetService.AggregateServicePorts()` |
| `pipeline_handlers.go:16` | `handleRunPipeline` | ~68L | 创建 PipelineRun、启动 goroutine、配置转换、SSE 广播 | `PipelineService.Start()` |
| `pipeline_handlers.go:295` | `handleCreateScan` | ~85L | mode→config 映射、创建 PipelineRun、启动 goroutine | `PipelineService.StartScan()` |
| `pipeline_handlers.go:216` | `buildConfigForMode` | ~78L | 大量默认值填充 + mode 分支逻辑 | `PipelineService.BuildConfig()` |
| `report_handlers.go:79` | `handleCreateReport` | ~62L | 磁盘检查、查重、创建 Report 记录、启动异步生成 | `ReportService.Create()` |
| `report_handlers.go:142` | `generateReport` | ~68L | 数据聚合、HTML 生成、文件写入、状态更新、SSE 广播 | `ReportService.Generate()` |
| `worker_handlers.go:159` | `handleTaskResult` | ~92L | 工件保存、base64 解码、SHA256 计算、RawArtifact 创建、任务状态更新、Run 完成检查 | `WorkerService.ProcessTaskResult()` |
| `finding_template_handlers.go:43` | `handleCreateFindingTemplate` | ~42L | 字段校验、构造模型、写 DB | `FindingTemplateService.Create()` |
| `finding_template_handlers.go:100` | `handlePatchFindingTemplate` | ~88L | 逐字段 diff、builtin 锁定逻辑、写 DB | `FindingTemplateService.Update()` |
| `retest_handlers.go:57` | `handleBatchUpdateFindingStatus` | ~55L | 状态校验、事务管理、批量更新 | `FindingService.BatchUpdateStatus()` |

### 2.2.3 Service 层扩展工作量评估

| 建议新增 Service | 预估方法数 | 参考代码量 | 优先级 |
|-----------------|-----------|-----------|--------|
| `AssetService` | 4-5 | ~200L | P1 |
| `PipelineService` | 3-4 | ~150L | P1 |
| `ReportService` | 3-4 | ~150L | P1 |
| `FindingTemplateService` | 5-6 | ~150L | P2 |
| `WorkerService` | 3-4 | ~120L | P2 |
| `RetestService` | 2-3 | ~80L | P3 |

**总计**：约 6 个新 Service，预估 850L 代码，2-3 工日。

---

## 2.3 上帝文件拆分方案

### 2.3.1 逐个分析

#### A. `internal/worker/server.go`（730L）— P0 拆分

**公开函数/方法清单**：

| 职责组 | 函数 | 说明 |
|--------|------|------|
| HTTP Handler | `Register`, `handleTask`, `handleProgress`, `handleResult`, `handleHealth`, `handleFile` | Worker HTTP API |
| 任务执行 | `executeTask` | 核心异步任务执行（~240L） |
| 结果报告 | `reportResult` | 写结果文件 + 上报 core |
| Nuclei 注入 | `injectCustomNucleiTemplates` | 命令行注入 |
| 超时配置 | `defaultToolTimeout`, `estimateScanScale`, `dynamicRunningTimeout`, `resolveTimeoutConfig`, `detectScanStrategy` | 工具超时策略 |
| 进程监控 | `readProcessCPUTime`, `readProcessState`, `isProcessStateHung` | /proc 读取 |
| IO 包装 | `idleWatchedWriter` (Write/Bytes/Last) | 带时间戳的 buffer |

**拆分方案**：

| 新文件 | 内容 | 预估行数 |
|--------|------|----------|
| `worker/server.go` | WorkerServer 结构体 + Register + HTTP handlers | ~120L |
| `worker/executor.go` | executeTask + reportResult | ~280L |
| `worker/timeout.go` | 全部超时相关函数 + TaskTimeoutConfig | ~150L |
| `worker/procmon.go` | 进程监控相关（readProcessCPUTime 等） | ~80L |
| `worker/idlewriter.go` | idleWatchedWriter | ~40L |

**拆分风险**：`executeTask` 与 `reportResult` 耦合紧密（共享 ws.dataDir, ws.coreURL, ws.httpClient），需提取为依赖接口。

#### B. `internal/db/queries_scan.go`（721L）— P1 拆分

**公开函数清单（46 个）**：

| 职责组 | 函数数量 | 函数列表 |
|--------|----------|----------|
| ScanPlan | 1 | CreateScanPlan |
| ScanTask | 7 | CreateScanTask, GetScanTask, UpdateScanTaskStatus, UpdateScanTaskErrorMessage, SetScanTaskRunning, ResetScanTaskForRetry, ListScanTasksByPlan, ListScanTasksByRun |
| ScanStep | 3 | CreateScanStep, UpdateScanStepStatus, ListScanStepsByTask |
| Run | 6 | CreateRun, GetRun, ListRunsByProject, CountRunsByProject, ListRunsByProjectPaginated, UpdateRunStatus |
| RawArtifact | 2 | CreateRawArtifact, ListRawArtifactsByTask |
| Screenshot | 2 | CreateScreenshot, ListScreenshotsByProject |
| Pipeline v0.4 | 11 | SaveDNSRecord, ListDNSRecordsByProject, SaveCDNResult, ListCDNResultsByProject, SaveServiceFingerprint, ListServiceFingerprintsByProject, CreatePipelineRun, UpdatePipelineRunStatus, UpdatePipelineRunStage, UpdatePipelineRunError, UpdatePipelineRunCompleted, GetPipelineRun, ListPipelineRunsByProject, CountPipelineRunsByProject, ListPipelineRunsByProjectPaginated, CreatePipelineRunStage, UpdatePipelineRunStageRecord, GetPipelineRunStage, ListPipelineRunStages |
| Dashboard | 4 | CountActiveRuns, CountPendingFindings, CountOnlineWorkers, ListRecentRuns, ListRecentFindingsByStatus |

**拆分方案**：

| 新文件 | 内容 | 预估行数 |
|--------|------|----------|
| `queries_scan_task.go` | ScanPlan + ScanTask + ScanStep | ~180L |
| `queries_run.go` | Run + RawArtifact + Screenshot | ~140L |
| `queries_pipeline.go` | PipelineRun + PipelineRunStage + DNS/CDN/ServiceFingerprint | ~220L |
| `queries_dashboard.go` | Dashboard 统计查询 | ~80L |

**拆分风险**：低 — 纯数据访问层，无交叉依赖。

#### C. `internal/workflow/discovery.go`（646L）— P1 拆分

**公开函数/方法清单**：

| 职责组 | 函数 | 说明 |
|--------|------|------|
| 工作流入口 | `AssetDiscoveryWorkflow.Run` | 主流程编排 |
| 链执行器 | `runDomainChain`, `runIPChain`, `runCIDRChain` | 三种目标类型的执行链 |
| 后置处理 | `runPostDiscovery` | Naabu 结果处理 + httpx 二次探测 |
| 工具输出解析 | `parseSubfinderOutput`, `parseHttpxOutput`, `parseNaabuOutput` | 解析工件文件 |
| 辅助 | `createAndRunTask`, `findArtifactPath`, `getPortRange`, `buildNaabuArgsWithPortRange`, `writeHostsFile` | 工具函数 |
| 纯工具 | `extractValues`, `joinArgs`, `parseInt` | 无状态函数 |

**拆分方案**：

| 新文件 | 内容 | 预估行数 |
|--------|------|----------|
| `workflow/discovery.go` | AssetDiscoveryWorkflow + Run + 链入口 | ~120L |
| `workflow/discovery_domain.go` | runDomainChain | ~120L |
| `workflow/discovery_ip.go` | runIPChain + runCIDRChain + runPostDiscovery | ~150L |
| `workflow/discovery_parser.go` | 三个 parse* 函数 + findArtifactPath | ~100L |
| `workflow/discovery_util.go` | buildNaabuArgs, writeHostsFile, extractValues, joinArgs, parseInt | ~80L |

**拆分风险**：`runPostDiscovery` 被 domain 和 cidr 链共享，提取后需确保接口兼容。

#### D. `internal/nuclei/custom/manager.go`（628L）— P2 拆分

**公开函数/方法清单**：

| 职责组 | 函数 | 说明 |
|--------|------|------|
| CRUD | `List`, `GetByID`, `CreateFromGit`, `CreateFromUpload`, `Refresh`, `Patch`, `Delete` | 数据源生命周期 |
| 文件 CRUD | `ListFiles`, `ReadFile`, `WriteFile`, `DeleteFile` | 源内文件操作 |
| 验证发布 | `ValidateSource`, `ValidateAll`, `Publish` | Phase 2 功能 |
| 辅助 | `EnsureLayout`, `Layout`, `lockFor` | 基础设施 |
| 校验函数 | `validateAllowed`, `validateName`, `validateRoutingPolicy`, `validateSourceType` | 输入校验 |
| 工具 | `strPtr`, `nullableStr`, `isYAMLFile`, `isNucleiTemplatePath` | 小函数 |

**拆分方案**：

| 新文件 | 内容 | 预估行数 |
|--------|------|----------|
| `manager.go` | 结构体 + CRUD + 辅助 | ~280L |
| `manager_file.go` | 文件 CRUD | ~80L |
| `manager_validate.go` | ValidateSource + ValidateAll + Publish | ~130L |
| `manager_validate_helpers.go` | isYAMLFile + isNucleiTemplatePath | ~50L |

#### E. `internal/scope/scope.go`（415L）— P3

**公开函数/方法清单**：

| 职责组 | 函数 | 说明 |
|--------|------|------|
| 核心引擎 | `Check`, `ValidateBeforeRun` | Scope 决策入口 |
| 评估 | `evaluate`, `matchRule`, `matchDomain`, `matchURL`, `matchIP` | 规则匹配 |
| 过滤 | `FilterTargets`, `filterTargetsWithRules` | 批量过滤 |
| CIDR | `ExpandCIDR`, `inc` | CIDR 展开 |
| 工具 | `CheckIP`, `matchDomainRule` | 便利函数 |

**内聚性良好** — 全部围绕 scope 规则匹配。415L 在可接受范围内，**暂不拆分**，建议保持在 500L 警戒线以下。

#### F. `internal/scope/import.go`（536L 含 test）— P3

**公开函数清单**：

| 职责组 | 函数 | 说明 |
|--------|------|------|
| 导入解析 | `ParseTXT`, `ParseCSV` | 批量导入入口 |
| 行解析 | `parseLine`, `expandCommas` | 单行解析 |
| 类型检测 | `DetectType`, `isLikelyCompanyName`, `looksLikeIPv4` | 自动类型推断 |
| IP 展开 | `expandIPRange` | 连字符范围展开 |
| 校验 | `isValidTargetType` | 类型校验 |

**内聚性良好** — 全部围绕目标导入。实际代码约 400L（含注释），**暂不拆分**。

#### G. `internal/worker/worker.go`（405L）— P3

**公开函数/方法清单**：

| 职责组 | 函数 | 说明 |
|--------|------|------|
| 执行 | `Run` | 核心本地执行（~210L） |
| 生命周期 | `saveArtifact`, `Cancel` | 工件保存 + 取消 |
| 注入 | `injectCustomNucleiTemplates` | Nuclei 模板注入 |
| IO | `limitedBuffer` | 带限制的 buffer |
| 错误 | `isUnreachableError` | 不可达错误判断 |

**内聚性良好** — 全部围绕 Runner 执行。405L 在可接受范围内，**暂不拆分**。

### 2.3.2 拆分优先级排序

| 优先级 | 文件 | 当前行数 | 拆分后文件数 | 风险 | 改动频率 |
|--------|------|----------|-------------|------|----------|
| **P0** | `worker/server.go` | 730L | 5 | 中（executeTask 与 reportResult 耦合） | 高 |
| **P1** | `db/queries_scan.go` | 721L | 4 | 低 | 高 |
| **P1** | `workflow/discovery.go` | 646L | 5 | 低 | 高 |
| **P2** | `nuclei/custom/manager.go` | 628L | 4 | 低 | 中 |
| **P3** | `scope/scope.go` | 415L | 1（不拆） | — | 中 |
| **P3** | `worker/worker.go` | 405L | 1（不拆） | — | 高 |
| **P3** | `scope/import.go` | ~400L | 1（不拆） | — | 低 |

---

## 2.4 DB 迁移健康报告

### 2.4.1 迁移文件时间线

| 版本 | 行数 | DDL 操作数 | 内容摘要 |
|------|------|-----------|----------|
| v1 | 352 | 24 | 初始 Schema（22 张表 + 索引） |
| v2 | 24 | 2 | 添加 `projects.rate_limit` |
| v3 | 97 | 3 | 添加 `targets.status`, `scope_rules.reason`, `findings.dedup_key` |
| v4 | 126 | 16 | 重构 targets 表（加 type/url/ip/cidr 字段，数据迁移） |
| v5 | 60 | 5 | 重构 projects 表（加 organization/purpose，数据迁移） |
| v6 | 26 | 3 | 添加 `findings.confidence`, `findings.priority` |
| v7 | 27 | 4 | 添加 `assets.tags`, `web_endpoints.status_code` |
| v8 | 16 | 1 | 添加 `findings.source_rule_id` |
| v9 | 50 | 3 | 添加 `engine_credentials` 表 |
| v10 | 64 | 4 | 添加 `worker_nodes` 扩展字段（mode/trust_level 等） |
| v11 | 93 | 8 | **破坏性**：重建 projects/targets/scope_rules/scan_plans/scan_tasks/engine_credentials（加 ON DELETE CASCADE） |
| v12 | 101 | 12 | **破坏性**：重建 scan_tasks（加 error_message/arguments_redacted），重建 runs（加 tool_template_id） |
| v13 | 20 | 1 | ALTER TABLE scan_tasks ADD error_message |
| v14 | 39 | 1 | CREATE TABLE reports |
| v15 | 56 | 1 | 添加 `nuclei_custom_sources` 表 |
| v16 | 44 | 5 | 添加 `dictionaries` + `httpx_fingerprints` 表 |
| v17 | 34 | 7 | 添加 `slow_scan_tasks` 表 |
| v18 | 29 | 1 | CREATE TABLE finding_templates |
| v19 | 27 | 4 | ALTER TABLE finding_templates（加 is_builtin/user_modified/builtin_payload） |
| v20 | 19 | 2 | ALTER TABLE dictionaries（加 builtin） |

### 2.4.2 Schema Drift 检测

对比 v1 初始 schema 与当前 models 定义：

| 表 | v1 字段 | models 结构体 | 状态 |
|----|---------|--------------|------|
| projects | v1 无 `rate_limit`, `organization`, `purpose`, `default_profile`, `port_range`, `fofa_email`, `fofa_api_key`, `pipeline_config` | `Project` 有全部 | 由 v2-v5 迁移补齐 |
| targets | v1 有 `type`, `value`, `source`, `status` | `Target` 一致 | 一致 |
| scan_tasks | v1 无 `error_message`, `arguments_redacted`, `run_id`, `depends_on_task_id`, `target_id`, `worker_id`, `nuclei_custom_bundle_version` | `ScanTask` 有全部 | 由 v11-v13 迁移补齐 |
| findings | v1 有 `confidence`, `priority`, `source_rule_id` | `Finding` 一致 | 一致 |
| assets/ports/services/web_endpoints | v1 定义完整 | 模型一致 | 一致 |
| worker_nodes | v1 无（v10 添加） | `WorkerNode` 有 mode/trust_level 等 | 由 v10 迁移添加 |
| reports | v1 无（v14 添加） | `Report` 一致 | 由 v14 迁移添加 |
| nuclei_custom_sources | v1 无（v15 添加） | `NucleiCustomSource` 一致 | 由 v15 迁移添加 |
| dictionaries | v1 无（v16 添加） | `Dictionary` 一致 | 由 v16/v20 迁移添加 |
| finding_templates | v1 无（v18 添加） | `FindingTemplate` 一致 | 由 v18/v19 迁移添加 |
| slow_scan_tasks | v1 无（v17 添加） | `SlowScanTask` 一致 | 由 v17 迁移添加 |

**结论**：无 schema drift。所有 models 字段均有对应的迁移版本。

### 2.4.3 迁移合并机会

**可合并的微型迁移**（全部 < 40 行，纯 ALTER TABLE / CREATE TABLE）：

| 版本 | 操作 | 是否可逆 |
|------|------|----------|
| v13 | ALTER scan_tasks ADD error_message | 不可逆（SQLite 不支持 DROP COLUMN） |
| v14 | CREATE TABLE reports | 可逆（DROP TABLE） |
| v18 | CREATE TABLE finding_templates | 可逆 |
| v19 | ALTER finding_templates ADD 3 列 | 不可逆 |
| v20 | ALTER dictionaries ADD builtin + index | 不可逆 |

**建议**：合并为 `v21.go` "v0.4 schema consolidation"，包含：
- v13: error_message 列（若不存在）
- v14: reports 表（若不存在）
- v18: finding_templates 表（若不存在）
- v19: finding_templates 扩展列（若不存在）
- v20: dictionaries builtin 列（若不存在）

**风险**：合并后无法单独回滚某一步。但由于全部是 `IF NOT EXISTS` 保护，实际风险极低。

### 2.4.4 回滚安全性

| 包含 DROP TABLE 的迁移 | 影响 |
|----------------------|------|
| v4 | DROP TABLE targets（重建） |
| v5 | DROP TABLE projects（重建） |
| v11 | DROP TABLE projects, targets, scope_rules, scan_plans, scan_tasks, engine_credentials（全部重建） |
| v12 | DROP TABLE scan_tasks（重建） |

**风险**：v11 是最大风险点 — 一次迁移中 DROP 6 张表。虽然 SQLite 中数据通过 INSERT SELECT 迁移，但在生产环境中若迁移中断，数据可能处于不一致状态。

**建议**：未来迁移禁止在单版本中使用 DROP TABLE + 重建模式。改用 ALTER TABLE（或 SQLite 的表重命名+列添加模式）。

---

## 2.5 SQL 治理选型建议

### 2.5.1 SQL 分布统计

| 文件 | SQL 语句数 | 分类 |
|------|-----------|------|
| `queries_scan.go` | 66 | CRUD + JOIN + 聚合（Dashboard） |
| `queries_finding.go` | 30 | CRUD + 复杂 JOIN（finding+evidence） |
| `queries_asset.go` | 27 | CRUD + 关联查询 |
| `queries_target.go` | 21 | CRUD |
| `queries_nuclei.go` | 17 | CRUD |
| `queries_slow_scan.go` | 15 | CRUD |
| `queries_finding_template.go` | 15 | CRUD |
| `queries_httpx_fingerprint.go` | 15 | CRUD |
| `queries_dictionary.go` | 13 | CRUD |
| `queries_report.go` | 14 | CRUD + 游标分页 |
| `queries_health.go` | 11 | 简单查询 |
| `queries_project.go` | 8 | CRUD |
| `queries_worker.go` | 14 | CRUD |
| **总计** | **~286** | — |

**注**：审计计划基线声称 182 条裸 SQL，实际统计约 286 条（按反引号计数）。差异可能来自基线未计入多行字符串或测试文件。

### 2.5.2 SQL 注入风险评估

| 检查项 | 结果 | 说明 |
|--------|------|------|
| `fmt.Sprintf` 拼接 SQL | **0 处** | 已验证：无 `fmt.Sprintf.*SELECT/INSERT/UPDATE/DELETE` 模式 |
| 字符串拼接 WHERE | **未发现** | 所有 WHERE 条件均使用 `?` 占位符 |
| 参数化查询占比 | **~100%** | 所有 queries_*.go 中的 SQL 均使用 `?` 占位符 |
| 动态表名/列名 | **未发现** | 无用户输入用于表名或列名 |

**结论**：当前裸 SQL 的注入风险为 **Low**。所有查询均使用参数化占位符。

### 2.5.3 治理方案对比

| 维度 | 方案 A: sqlc | 方案 B: ent | 方案 C: 手工 Repository |
|------|-------------|-------------|----------------------|
| **学习成本** | 中（需学 SQL→Go 生成流程） | 高（需学 ent 的 schema DSL 和查询 DSL） | 低（当前模式） |
| **迁移工作量** | 高（需将 286 条 SQL 改写为 sqlc 格式，生成代码后替换调用点） | 极高（需重写全部 schema 定义，重构所有查询） | 低（渐进式，按需提取） |
| **运行时开销** | 零（生成纯 Go 代码） | 中（ORM 反射+缓存开销） | 零 |
| **性能** | 与手写相同 | 略低于手写（复杂查询需原生 SQL 逃逸） | 最优 |
| **AI 友好度** | 高（SQL 是通用语言，AI 易理解） | 中（需理解 ent DSL） | 高 |
| **类型安全** | 高（编译时检查） | 高（编译时检查） | 中（运行时 scan 错误） |
| **与现有代码兼容** | 中（需重构调用层） | 低（需大规模重构） | 完全兼容 |

**推荐：方案 C（手工 Repository 模式）渐进优化**

理由：
1. 当前 SQL 注入风险已可控（100% 参数化）
2. 项目处于 v0.x 快速迭代期，引入 sqlc/ent 的迁移成本过高
3. 手工 Repository 可通过以下步骤渐进改善：
   - 为每个 domain 提取 Repository 接口（参考 `service/project.go` 的 `ProjectRepository`）
   - 将 `queries_*.go` 中的函数按 domain 分组到对应 Repository 实现
   - 引入 `sqlc` 的时机：v1.0 后 schema 稳定时

**短期行动（v0.5 前）**：
- 为 `FindingService`, `TargetService` 补充 Repository 接口（参考 ProjectService 模式）
- 将 Dashboard 查询从 `queries_scan.go` 中提取到独立文件

**长期规划（v1.0 后）**：
- 评估 sqlc 引入，将稳定 domain 的 SQL 转为生成代码
- 保留复杂动态 SQL 为手写 Repository

---

## 附录：检查项完成状态

| 检查项 | 状态 | 发现概要 |
|--------|------|----------|
| 2.1.1 api->db 直接依赖 | 完成 | 3 处（server.go, finding_template_handlers.go, retest_handlers.go） |
| 2.1.2 api->models 直接依赖 | 完成 | 15 处，其中 5 处含业务判断应下沉 |
| 2.1.3 其他分层违规 | 完成 | 无逆依赖，workflow->worker 为设计意图 |
| 2.2.1 Service 层覆盖范围 | 完成 | 3 个 Service（Project/Target/Finding） |
| 2.2.2 Handler 大函数扫描 | 完成 | 10 个函数应迁移到 Service |
| 2.2.3 Service 扩展评估 | 完成 | 需新增 6 个 Service，~850L |
| 2.3.1 上帝文件职责分析 | 完成 | 7 个文件全部分析，按职责分组 |
| 2.3.2 拆分优先级 | 完成 | P0: worker/server.go; P1: queries_scan.go, discovery.go |
| 2.4.1 迁移文件概览 | 完成 | 20 个迁移，v1 初始 schema 352L |
| 2.4.2 Schema drift | 完成 | 无 drift，所有 models 字段均有对应迁移 |
| 2.4.3 迁移合并 | 完成 | v13/v14/v18/v19/v20 可合并为 v21 |
| 2.4.4 回滚安全性 | 完成 | v11 风险最高（DROP 6 表），v4/v5/v12 含 DROP |
| 2.5.1 SQL 分布 | 完成 | ~286 条 SQL，queries_scan.go 最多（66） |
| 2.5.2 SQL 注入 | 完成 | 0 处 fmt.Sprintf 拼接，100% 参数化 |
| 2.5.3 治理方案 | 完成 | 推荐手工 Repository 渐进优化，v1.0 后评估 sqlc |

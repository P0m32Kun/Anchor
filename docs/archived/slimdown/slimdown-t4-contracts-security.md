---
status: archived
source_of_truth: false
owner: kun
audit_date: 2026-05-26
audit_baseline_commit: "main @ 7399a5f"
scope: contracts-security-regression
archived_date: "2026-06-17"
archive_reason: "slimdown series completed, moved from active/review/"
---

# T4 报告：契约、安全、Pitfall 回归

> 审计日期：2026-05-26 | Delta 基线：stage1-security-contracts.md (2026-05-13)

---

## 0. 新增/变更 API 端点（2026-05-13 至今）

| 端点 | Handler 文件 | 前端对应 | 契约已对齐？ |
|------|-------------|---------|-------------|
| `GET /tasks/{id}/output` | `task_output_handlers.go` | `useTaskLiveOutput.ts` + `api.ts:getTaskOutput` | ✅ |
| `GET /artifacts/content` | `task_handlers.go:148` | `api.ts:getArtifactContent` | ✅ |

**验证**：

| 后端字段 | 前端类型 | 差异 | 风险 |
|---------|---------|------|------|
| `stream` (string, "stdout"/"stderr") | `stream: string` | 对齐 | Low |
| `offset` (int64) | `offset: number` | 对齐 | Low |
| `content` (string) | `content: string` | 对齐 | Low |
| `done` (bool) | `done: boolean` | 对齐 | Low |

---

## 1. 前后端契约 Delta

### 1.1 新增字段 (2026-05-13 以来新增的 Go 模型字段)

| Go 模型 | 新增字段 | 前端对应 | 状态 |
|---------|---------|---------|------|
| `models.PipelineConfig` | 多个 external-scan 配置字段 | `frontend/src/lib/api.ts` PipelineConfig 类型 | ✅ 已同步 |
| `models.ScanTask` | `worker_id` (已在 v0.4 添加) | `ScanTask` 类型 | ✅ 已同步 |

### 1.2 存量 High/Critical 问题复查 (stage1)

| ID | 问题 | 等级 | 2026-05-13 状态 | 2026-05-26 状态 | 变化 |
|----|------|------|----------------|----------------|------|
| C-01 | Project 缺失 `start_time`/`end_time` | **Critical** | 后端无该字段 | **未修复** | → |
| C-02 | Worker 注册无去重 | **High** | 缺失 | **未修复** | → |
| C-03 | 超时任务未重分配 | **High** | 缺失 | **未修复** | → |
| C-04 | `Finding.SourceRuleID` 可选性不一致 | **High** | 后端非指针，前端可选 | **未修复** | → |
| C-05 | `ScanTask` 前后端字段严重不匹配 | **High** | 后端 15+ 字段，前端仅 8 个 | **未修复** | → |

**结论**：5 个 High/Critical 契约问题无任何修复。代码不在这轮瘦身范围内，记录在案即可。

---

## 2. Pitfall 回归检查（7 条）

### 逐个验证

| Pitfall | 文件 | 2026-05-13 状态 | 2026-05-26 状态 | Code 证据 | 结论 |
|---------|------|-----------------|-----------------|-----------|------|
| `20260426-artifact-type-mismatch` | `worker/server.go` | 已修复（stdout fallback） | ✅ 稳定 | `server.go:280-304` 逻辑未变 | **无回归** |
| `20260426-asset-scope-check-missing` | `scope/scope.go` | 已修复 | ✅ 稳定 | scope check 逻辑未变 | **无回归** |
| `20260426-frontend-backend-field-mismatch` | 多处 | 已修复 | ✅ 稳定 | 字段已对齐 | **无回归** |
| `20260426-raw-artifact-redaction-loss` | `workflow/screenshot.go` | 已修复 | ✅ 稳定 | `saveEvidenceArtifact` 逻辑未变 | **无回归** |
| `20260427-markdown-pipe-corruption` | `report/markdown.go` | 已修复 | ✅ 稳定 | `escapeMDTable` 逻辑未变 | **无回归** |
| `20260427-null-scan-crash` | `db/queries_finding.go` | 已修复 | ✅ 稳定 | `.Valid` 判断逻辑未变 | **无回归** |
| `20260428-ghost-worker-cleanup` | `worker/server.go` + `api/worker_handlers.go` | 部分修复 | ✅ 未退化 | 心跳清理 + 启动清理仍在 | **无回归**（去重 + 任务重分配仍未实现） |

**总计**：7/7 无回归。0 条回退。

---

## 3. Pipeline 外网 Preset 字段审计

| 前端字段 | 后端 `PipelineConfig` 对应 | 类型检查 | 风险 |
|---------|--------------------------|---------|------|
| `enablePassiveSearch` | `EnablePassiveSearch` | `bool` ↔ `boolean` | Low |
| `enableFOFA` | `EnableFOFA` | `bool` ↔ `boolean` | Low |
| `enableHunter` | `EnableHunter` | `bool` ↔ `boolean` | Low |
| `enableQuake` | `EnableQuake` | `bool` ↔ `boolean` | Low |
| `passiveCertEnabled` | `PassiveCertEnabled` | `bool` ↔ `boolean` | Low |
| `passiveURLEnabled` | `PassiveURLEnabled` | `bool` ↔ `boolean` | Low |
| `enablePortscan` | `EnablePortscan` | `bool` ↔ `boolean` | Low |
| `portPreset` | `PortPreset` | `string` ↔ `string` | Low |
| `customPorts` | `CustomPorts` | `string` ↔ `string` | Low |
| `enableFfuf` | `EnableFfuf` | `bool` ↔ `boolean` | Low |
| `ffufTier` | `FfufTier` | `string` ↔ `string` | Low |
| `ffufDictionaryID` | `FfufDictionaryID` | `string` ↔ `string` | Low |
| `enableCrawl` | `EnableCrawl` | `bool` ↔ `boolean` | Low |
| `scanDepth` | `ScanDepth` | `int` ↔ `number` | Low |

**结论**：全部对齐。外网 preset 字段均为 `PipelineConfig` 的新增字段，前后端类型一致。

---

## 4. 安全回归检查

### 4.1 `toolguard` allowlist 覆盖

| exec.Command 调用点 | 走 toolguard? | 状态 |
|--------------------|--------------|------|
| `worker/worker.go:206` | ✅ 通过 `Runner.Run` → `Allowlist.Validate` | ✅ 安全 |
| `worker/server.go:167` | ✅ 通过 `Allowlist.Validate` | ✅ 安全 |
| `health/health.go:93,141` | ✅ 通过 `Allowlist.Validate` | ✅ 安全 |
| `cdn/detector.go:27,57` | ❌ 未经过 toolguard | **Medium** — CDN 检测直 exec cdncheck，无白名单校验 |
| `nuclei/custom/git.go:59` | ❌ 未经过 toolguard | **Low** — git 命令参数来自程序逻辑，非用户输入 |
| `builtin/sync.go:50-68` | ❌ 未经过 toolguard | **Low** — git clone 来自内置 asset 同步 |

### 4.2 `task_output_handlers.go` HTTP 代理安全检查

`proxyWorkerTaskOutput` 向 worker endpoint 发送 HTTP 请求：

- 创建 `http.Client{Timeout: 15s}` — 超时可接受
- endpoint 来自 DB 的 `worker_nodes.endpoint` 字段，非直接用户输入 → 注入风险 Low
- 无 TLS 证书验证 → **Medium**（中间人可读取/篡改 task output）

---

## 5. 汇总

| 类别 | 详情 |
|------|------|
| 新端点契约对齐 | ✅ `GET /tasks/{id}/output` 和 `GET /artifacts/content` 前后端一致 |
| 存量 Critical/High 问题 | 5 个未修复（记录在案，非瘦身范围） |
| Pitfall 回归 | ✅ 7/7 无回归 |
| 安全缺口 | 2 个 Medium：`cdn/detector.go` 绕过 toolguard、task_output proxy 无 TLS 验证 |

## 6. 建议（记录为主，不改代码）

| 优先级 | 项 | 说明 |
|--------|----|------|
| P2 | 将 CDN 检测接入 toolguard | `cdn/detector.go` 的 exec.Command 应走 allowlist 校验 |
| P3 | task_output proxy 加 TLS 验证 | 需要 worker 证书管理基础设施 |

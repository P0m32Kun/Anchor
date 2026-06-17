---
status: archived
source_of_truth: false
owner: kun
audit_date: 2026-05-26
audit_baseline_commit: "main @ 7399a5f"
scope: dead-code-migration
archived_date: "2026-06-17"
archive_reason: "slimdown series completed, moved from active/review/"
---

# T2 报告：死代码与迁移收口

> 审计日期：2026-05-26 | 基线 commit：`7399a5f`

---

## 0. 环境检查结果

| 检查项 | 结果 |
|--------|------|
| `go vet ./internal/...` | ✅ 通过 |
| `npx tsc --noEmit` | ✅ 通过 |
| `go test ./internal/...` | ✅ **全绿**（29 个包全部 ok） |

---

## 1. 工具调用迁移状态

### 1.1 `toolrun.Invoke` 调用点（已迁移，新路径）

| 文件 | 工具 | 行号 |
|------|------|------|
| `pipeline.go` | httpx, nuclei | 399, 476 |
| `pipeline_crawl.go` | katana | 39 |
| `pipeline_passive.go` | gau | 312 |
| `pipeline_tool.go` | subfinder, dnsx, httpx, cdncheck, nmap_alive, nmap_service, naabu, nuclei(×2), katana, ffuf, gau | 52–682 (12 处) |

**合计**：15 处 `toolrun.Invoke` 调用，覆盖全部流水线工具

### 1.2 `worker.Build*` 调用点（旧路径）

| 文件 | 函数 | 行号 | 状态 |
|------|------|------|------|
| `workflow/discovery.go` | `BuildSubfinderCommand` | 115 | ✅ 旧版工作流，维护模式 |
| `workflow/discovery.go` | `BuildHttpxCommand` | 148 | ✅ 同上 |
| `workflow/discovery.go` | `BuildHttpxCommand` | 351 | ✅ 同上 |
| `workflow/screenshot.go` | `BuildNucleiCommand` | 104 | ✅ 同上 |
| `workflow/screenshot.go` | `BuildNucleiCommand` | 248 | ✅ 同上 |
| `toolregistry/registry_test.go` | 全部 14 个 `Build*` | 34–201 | ✅ golden test 对照（项目政策允许） |

### 1.3 已验证的迁移覆盖

> 验证命令：`grep -rn "worker\.Build" ./internal/workflow/ | grep -v "discovery\.go\|screenshot\.go"`

```
输出为空 ✅ — pipeline 已全量切换到 toolrun.Invoke
```

---

## 2. Allowlist vs `tools/*.yaml` vs 实际调用

### 2.1 三方对照表

| 工具 | allowlist | `tools/*.yaml` | `toolrun.Invoke` 调用 | 状态 |
|------|-----------|----------------|----------------------|------|
| subfinder | ✅ | `subfinder.yaml` | ✅ pipeline_tool.go:52 | 对齐 |
| dnsx | ✅ | `dnsx.yaml` | ✅ pipeline_tool.go:89 | 对齐 |
| httpx | ✅ | `httpx.yaml` | ✅ pipeline_tool.go:123, pipeline.go:399 | 对齐 |
| naabu | ✅ | `naabu.yaml` | ✅ pipeline_tool.go:191, 226 | 对齐 |
| nmap | ✅ (nmap) | `nmap_alive.yaml` + `nmap_service.yaml` | ✅ 两个子命令 | **对应关系模糊** — allowlist 仅 `nmap`，tools 分两个文件 |
| nuclei | ✅ | `nuclei.yaml` | ✅ pipeline_tool.go:261,298, pipeline.go:476 | 对齐 |
| cdncheck | ✅ | `cdncheck.yaml` | ✅ pipeline_tool.go:448 | 对齐 |
| ffuf | ✅ | `ffuf.yaml` | ✅ pipeline_tool.go:540, pipeline_crawl.go:39 | 对齐 |
| gau | ✅ | `gau.yaml` | ✅ pipeline_passive.go:312, pipeline_tool.go:682 | 对齐 |
| katana | ✅ | `katana.yaml` | ✅ pipeline_tool.go:573 | 对齐 |
| git | ✅ | ❌ 无 | ❌ 无 | ✅ 系统工具，不应在 tools/ |
| git.exe | ✅ | ❌ 无 | ❌ 无 | ✅ 同上 |
| sh/bash | ✅ | ❌ 无 | ❌ 无 | ✅ 系统工具 |

### 2.2 孤儿/不一致项

| 问题 | 等级 | 说明 |
|------|------|------|
| `tools/*.yaml` 中 `nmap_alive`/`nmap_service` 分离但 allowlist 仅 `nmap` | Low | 二进制是同一个 `nmap`，YAML 分两文件是为了不同参数模板，allowlist 正确 |
| `tools/tools.go` | Low | 空 Go 文件（仅 package 声明），用于 go mod 保留 tools 依赖？建议检查是否必要 |

---

## 3. 迁移四角表

### 工具删除/迁移情况

| 工具 | 代码 | DB enum | 前端类型 | 文档 | 状态 |
|------|------|---------|----------|------|------|
| **urlfinder** (已删) | `internal/parser/urlfinder.go` ❌ 已删 | `slow_scan_tasks.tool` CHECK 约束含 `'urlfinder'` ✅ 需保留历史数据 | 前端无引用 ✅ | `stageemitter.go` 注释提及 | **代码已删**，DB 约束和 `SlowScanToolURLFinder` 模型常量保留（历史数据兼容） |
| **gau** (已迁移) | `internal/passive/gau.go` ❌ 已删 → `internal/parser/gau.go` ✅ 新建 | `engine_credentials` 表含 Gau 引擎类型 ✅ | `api.ts` 有 Gau 相关方法 ✅ | 无过时引用 | **已迁移**，parser 层接管 |
| **katana** (新增) | `internal/parser/katana.go` ✅ | 无独立 enum（通过 ToolTemplate 管理） ✅ | 前端通过 `PipelineConfig.tools` 控制 ✅ | `architecture.md` 已更新 ✅ | **已完成** |
| **passive/** | `internal/passive/gau.go` 已删，`crt.go` 保留 ✅ | 无变化 | — | — | **crt.sh 独立工作**，gau 已迁移 |

### 需要关注的点

| 项 | 等级 | 建议 |
|----|------|------|
| `SlowScanToolURLFinder` 常量在 `models/slow_scan.go:9` | Low | 建议标记 `// Deprecated: urlfinder removed; kept for DB compatibility` |
| `StageURLFinder` 常量在 `pipeline_stage.go:19` | Low | 已标注 `legacy runs only`，无执行路径引用；可保留或删除 |

---

## 4. 删除候选 — Callers=0 证据

### 4.1 可删除（安全）

| 候选 | 证据 | 理由 |
|------|------|------|
| `StageURLFinder` 常量 + 执行路径 | `search_content "StageURLFinder"` — 仅 `pipeline_stage.go` 自身定义 + `stageemitter.go:15` 注释 | 无运行时引用。`pipeline_flow.go` 中无对应分支 |
| 注释中的 `urlfinder` 提及 | 仅 comment，不影响编译 | 无影响 |

### 4.2 须保留

| 候选 | 原因 |
|------|------|
| `models/slow_scan.go:9` `SlowScanToolURLFinder` | 已有历史数据库行含 `tool='urlfinder'`，删除常量会破坏反序列化。建议仅加 Deprecated 注释 |
| `db/v17.go` CHECK 约束 `IN ('urlfinder','ffuf')` | 迁移文件不可改（已 apply），不影响新数据写入 |

### 4.3 阻塞于 WIP

| 候选 | 阻塞原因 |
|------|----------|
| `workflow/discovery.go` 中的 `worker.BuildSubfinderCommand` / `BuildHttpxCommand` | 旧版工作流，维护模式，按项目政策暂不迁移 |
| `workflow/screenshot.go` 中的 `worker.BuildNucleiCommand` | 同上 |

---

## 5. Dead Code 无 callers=0 证据的禁止删除

| 路径 | callers=0 证据 | 结论 |
|------|---------------|------|
| `StageURLFinder` | `rg "StageURLFinder"` → 仅定义 + 注释 | ✅ 可删 |
| `SlowScanToolURLFinder` | `rg "SlowScanToolURLFinder"` → 定义 + v17 迁移注释 | ❌ 保留（DB 兼容） |

---

## 6. 总结

| 类别 | 数量 | 说明 |
|------|------|------|
| 生产代码可安全删除 | 0 | 无——urlfinder 代码已在之前提交中删除 |
| 常量/注释可清理 | 1 | `StageURLFinder` 死常量 |
| DB 兼容保留 | 2 | `SlowScanToolURLFinder` 模型常量、v17 CHECK 约束 |
| 需标注 Deprecated | 1 | `SlowScanToolURLFinder` |
| WIP 阻塞 | 2 | discovery.go + screenshot.go 的旧版 Build* 调用 |

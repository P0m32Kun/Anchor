---
status: active
source_of_truth: false
owner: kun
last_updated: 2026-05-26
scope: tool-registry-implementation-handoff
audience: external-implementer (e.g. DeepSeek)
---

# 工具注册表 — 实现者交接说明

> 给接手实现的 Agent（DeepSeek 等）用。**先读完本文 + 主设计，再写代码。**

## 1. 任务一句话

把 Anchor 的扫描 CLI 从分散的 `internal/worker/commands.go` + `toolguard/allowlist.go` 双份维护，重构为 **`tools/*.yaml` 注册表 + `toolrun.Invoke`**；管线只传 `tool_id` + `params`；大 stdout 流式/分页；**废弃任意 command 的 ad-hoc API**；**gau 从 Server exec 迁到 Worker**。

## 2. 必读文档（按顺序）

| 顺序 | 路径 | 用途 |
|------|------|------|
| 1 | [`tool-registry-and-artifact-store.md`](tool-registry-and-artifact-store.md) | **主设计**（架构、YAML schema、Phase、已决事项 §2.1） |
| 2 | [`../current/architecture.md`](../current/architecture.md) | 运行时基线、外网 P1–P5 |
| 3 | [`../../CLAUDE.md`](../../CLAUDE.md) | 文档同步、禁止单位换算、改 API 要改 README |
| 4 | [`../../AGENTS.md`](../../AGENTS.md) | Go 风格、测试要求 |
| 5 | [`../active/review/external-scan-e0-e5.md`](../active/review/external-scan-e0-e5.md) | 外网管线验收参考 |
| 6 | [`../../internal/api/README.md`](../../internal/api/README.md) | 若动 `task_handlers` / 路由 |

## 3. 已关闭的产品决定（禁止擅自改）

| 议题 | 决定 |
|------|------|
| Nuclei | **全 YAML**；**不要** Go `builder`；**删除** `-stats -si 30`（无消费者） |
| 手调命令 | **无**；移除或 410 `POST /tasks/run` 的任意 `command` 字段 |
| 工具边界 | **要** registry（可点名 `tool_id`）+ Worker allowlist（exec 前） |
| 历史版本 | **不** 加 `tool_registry_version` 列；靠 `command_template`；数据 **≤30 天** |
| 执行位置 | **Server 仅 HTTP API**（FOFA/Hunter/Quake/crt.sh）；**所有 CLI 仅 Worker**（含 gau） |

## 4. 实施顺序（必须按 Phase，不要跳）

### Phase 1 — `toolregistry` only（不改 pipeline 行为）

- 新增 `tools/*.yaml`、`internal/toolregistry/`（embed + Load + Render + Validate）。
- 覆盖 §6.4 工具清单（含 `nuclei`、`gau` 的 yaml 定义，即使 gau 尚未切 pipeline）。
- **黄金测试**：`registry.Render(id, params)` 与现有 `worker.Build*` argv 一致；**nuclei 允许少** `-stats`、`-si`、`30` 三个 token。
- `toolguard.NewAllowlistFromRegistry(reg)` + 测试。
- **不要** 删 `Build*`；**不要** 改 `pipeline_*.go`。

**验收**：`go test ./internal/toolregistry/... ./internal/toolguard/...`

### Phase 2 — `toolrun` + pipeline + gau + API

- 新增 `internal/toolrun/`：`Invoke(ctx, runner, reg, InvokeInput)`；**禁止 Server 对 registry 内工具做本地 exec**。
- `pipeline.createAndRunTask` → `toolrun.Invoke`；删除对 `worker.Build*` 的调用。
- 删除 `internal/passive/gau.go` 的 Server `exec`；`runPassiveURL` 改 Worker 路径。
- 废弃 `handleRunTask` 任意 command（删路由或返回 410；前端 `api.runTask` 若无用则删）。
- Pipeline 注入 `*toolregistry.Registry`。

**验收**：`go test ./internal/workflow/...`；`go vet ./...`；外网 E2E 若环境具备则跑 `frontend/e2e/tests/external-scan-flow.spec.ts`。

### Phase 3 — 大结果

- 废弃 `readTaskStdout` 全量读大文件；parser 用 `io.Reader` / 文件路径。
- 分页 API（设计 §7.3）；小文件保持兼容。

### Phase 4 — 可选（未要求则不做）

- `ANCHOR_TOOLS_DIR`、port_scan 可配置、MCP、30 天清理 job。

## 5. 关键代码地图（迁移前）

| 现状 | 路径 |
|------|------|
| CLI 拼装 | `internal/worker/commands.go` |
| Allowlist | `internal/toolguard/allowlist.go` |
| 管线调工具 | `internal/workflow/pipeline_tool.go`（`createAndRunTask`） |
| gau Server exec | `internal/passive/gau.go` → `pipeline_passive.go` `runPassiveURL` |
| Ad-hoc API | `internal/api/task_handlers.go` `handleRunTask`；路由 `POST /tasks/run` in `server.go` |
| Worker 执行 | `internal/worker/worker.go`（`CommandTemplate` → exec） |
| 解析 | `internal/parser/*` |

## 6. 项目硬约束（违反 = 未完成）

1. **禁止** 在代码里对 CLI 参数做单位换算（如 `*1000`）；YAML 字段与工具 CLI 一致。
2. **禁止** 用 YAML 描述整条 pipeline DAG；P1–P5、CDN 跳过、Nuclei 多轮循环留在 Go。
3. 改 `internal/api/server.go` 路由或字段 → 同步 `internal/api/README.md` + 字段注释（见 `CLAUDE.md`）。
4. 架构级变更 → 更新 `docs/current/architecture.md`（Phase 2 完成后）。
5. **修复后必须跑相关测试**；仅 `go build` 不够。
6. **不要** 引入 CyberStrikeAI 的 AI 编排、C2、100+ 工具。
7. **不要** 擅自 `git commit`，除非用户明确要求。
8. 对话/注释：项目约定中文；代码与错误信息保持现有风格（英文小写 error）。

## 7. Definition of Done（全部满足才算完）

- [ ] 扫描阶段 CLI 仅经 `toolregistry` + `toolrun.Invoke`
- [ ] allowlist 与 registry 二进制一致（+ git/sh/bash 系统扩展）
- [ ] 无 Server 侧扫描 CLI `exec`（含 gau）
- [ ] `POST /tasks/run` 无任意 command 入口
- [ ] `docs/active/review/` 简短验收记录 + architecture 更新（若 Phase 2 完成）
- [ ] 相关 `go test` 通过

## 8. 建议首轮交付物（PR 粒度）

**PR1（仅 Phase 1）**：`toolregistry` + `tools/*.yaml` + golden tests + allowlist from registry  
**PR2（Phase 2）**：`toolrun` + pipeline 切换 + gau + API 废弃  
**PR3（Phase 3）**：大结果流式 + 分页 API  

每个 PR 独立可 review、可回滚。

## 9. 常见陷阱

- 在 Server `Run()` 本地 fallback 路径执行 naabu/nuclei — 与「CLI 仅 Worker」冲突；registry 工具应 **始终 dispatch Worker**（无 Worker 时明确失败或文档说明 dev 模式例外需与用户确认）。
- Nuclei 多轮 Invoke 写在 YAML — 应留在 `pipeline_tool.go`。
- 只改 allowlist 不加 `tools/foo.yaml` — 禁止。
- 迁移时改变 portRange preset 语义 — 黄金测试必须锁住行为。

---

## 10. 复制给 DeepSeek 的启动 Prompt（可直接粘贴）

```markdown
你在仓库 Anchor（/Users/kun/DEV/p0m32kun）实现「工具注册表」重构。

先只读、不要改代码，按顺序阅读：
1. docs/design/tool-registry-handoff-for-implementer.md
2. docs/design/tool-registry-and-artifact-store.md（尤其 §2.1 已决事项）
3. CLAUDE.md、AGENTS.md

然后 **只做 Phase 1**：
- 新增 internal/toolregistry 与 tools/*.yaml
- 黄金测试对齐 internal/worker/commands.go（nuclei 不含 -stats -si 30）
- toolguard 从 registry 派生 allowlist

约束：不改 pipeline 行为；不 commit；中文回复进度；禁止 CLI 单位换算；不把 pipeline 写成 YAML DAG。

完成后列出：改动文件、如何跑测试、Phase 2 剩余项。若 Phase 1 测试未全绿，不要开始 Phase 2。
```

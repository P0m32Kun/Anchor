# Anchor 项目约定

## 工作语言

- **所有对话使用中文进行**

## 验证要求

- **修复后只跑 build/typecheck 不算修完** — 必须起服务跑通真实场景（E2E）

## 配置单位

- **包装外部 CLI 工具时禁止在代码里做单位转换**（如 `*1000`），字段单位与工具自身一致

## 文档同步约束（修改代码必须同时改文档,否则视为未完成）

修改下列任一处时,**必须在同一个提交里**同步更新对应文档,不一致即视为未完成:

| 修改的代码 | 必须同步更新的文档 |
|---|---|
| `internal/api/server.go` 中 `Server` 结构体字段（增删/重命名/换类型） | 1. `server.go` 中该字段上方的中文注释（分类 + 消费者列表）<br>2. `internal/api/README.md` 的「字段反向索引」表 |
| `internal/api/server.go` 中 `Register` 函数（新增/移除路由） | `internal/api/README.md` 「Handler 文件总览」中对应行的「路径前缀」列 |
| 任一 `internal/api/*_handlers.go` 文件中新增/移除对 `s.xxx` 字段的引用 | 1. `internal/api/README.md` 中该 handler 行的「依赖 Server 字段」列<br>2. `internal/api/README.md` 「字段反向索引」表中对应字段的「消费 handler 文件」列<br>3. `server.go` 中该字段注释里的「消费者」列表 |
| 新增 `internal/api/*_handlers.go` 文件 | `internal/api/README.md` 「Handler 文件总览」表新增一行 |
| 删除 `internal/api/*_handlers.go` 文件 | 同时清理上述两表里对它的引用 |
| 架构层级的代码变化（包拆分、命名变更、新增包） | `docs/current/architecture.md` 新增章节或更新现有描述 |

执行流程:

1. 修改代码前,先看 `internal/api/README.md` 的反向索引判断 blast radius。
2. 修改完成后,先跑 `go vet ./internal/api/` 确保编译通过。
3. 提交前用 `git diff --stat` 确认改动只波及预期范围。
4. PR 描述中**显式列出**这次修改了哪些文档,reviewer 对照本表逐项核对。

reviewer 发现任一文档未同步,直接 block 合并并要求补齐,不商量。

## 文档入口

- `docs/current/plan.md` — 当前唯一有效的实施计划
- `docs/current/architecture.md` — 当前运行时架构基线
- `docs/current/code-health-audit.md` — 多阶段代码健康审计计划（2026-05-13 激活）
- `docs/conventions/testing-workflow.md` — SDD→BDD→TDD 开发与验收流程（功能开发前必读）
- `docs/conventions/testing.md` — 测试金字塔与 E2E 硬性规则
- `docs/functional-test.md` — BDD 手工验收与场景注册表
- `~/.p-skills/skills/develop-feature/SKILL.md` — 通用开发与测试编排（SDD→BDD→TDD）
- `.cursor/skills/anchor-dev-test/SKILL.md` — Anchor 测试约定适配（文档路径与范例）
- `internal/api/README.md` — API handler 地图与字段反向索引（改 handler 之前必读）

## 工作流约束（2026-05-26 复盘落地）

### 每次编辑后立即 build

不要累积多个编辑后一起 build。单文件编辑后 `go build ./that/pkg/`，批量编辑后 `go build ./pkg1/ ./pkg2/`。`imported and not used` 在秒级修的成本远低于在十几个编辑后排查。

### Phase 完成用 grep 验证

每个 Phase 结束时跑验证 grep，输出为空才算完成：
```bash
grep -rn "旧函数名\|应删除的模式" ./internal/workflow/
```
当前适用：全量 pipeline 工具调用已切换到 `toolrun.Invoke`，验证命令：
```bash
grep -rn "worker\.Build" ./internal/workflow/ | grep -v "discovery\.go\|screenshot\.go"
```
输出应仅剩下 `discovery.go` 和 `screenshot.go`（两个旧版工作流，维护模式）。

### 架构决策前先确认

遇到"代码做了 X，但不确定设计意图是 X 还是 Y"时：
- 先说"代码现状是这样的，你确认应该怎么做？"
- 尤其涉及 **Server 执行 vs Worker 调度** 的选择时

### 代码搜索工具优先级

禁止直接用 Grep/Glob 做代码探索。按问题类型选工具：

| 问题类型 | 工具 | 示例 |
|---|---|---|
| 符号定义/签名 | `codegraph_search` + `codegraph_node` | "X 定义在哪？" |
| 调用关系 | `codegraph_callers` / `codegraph_callees` | "谁调用了 X？" |
| 改动影响面 | `codegraph_impact` | "改 X 会波及什么？" |
| 模块/功能理解 | `semble-search` agent | "这个模块怎么工作的？" |
| 纯字面量搜索 | Grep | 找特定字符串、日志、注释 |

### 编辑策略优先级

1. 短文件（<200 行）→ `write_file` 完整覆写
2. 大文件局部修改 → `edit_file`，但先用 `search_content` 确认 SEARCH 文本精确形状
3. 每次 `edit_file` 后立即 `go build ./target-pkg/`

### Import cycle 预防

`internal/toolregistry` 是底层声明层，不应被同级包（如 `toolguard`）的**生产代码**反向依赖。新增跨包 import 前画导入链：
```
A_test → B → C → A？
```
需要共享类型时优先用接口或简单类型（`[]string` 而非 `*Registry`）。

### 已知状态

- `discovery.go` / `screenshot.go` — 旧版工作流，维护模式，**暂不迁移** toolrun
- `worker.Build*` — 迁移后保留，**仅供 golden test 对照**，禁止 pipeline 新调用
- `POST /tasks/run` — 路由已移除，handler 保留为 dead code

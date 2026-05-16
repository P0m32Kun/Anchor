<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **Anchor** (8472 symbols, 18153 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-groupes ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/Anchor/context` | Codebase overview, check index freshness |
| `gitnexus://repo/Anchor/clusters` | All functional areas |
| `gitnexus://repo/Anchor/processes` | All execution flows |
| `gitnexus://repo/Anchor/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

---

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
3. 提交前再跑一次 `gitnexus_detect_changes`,确认改动只波及预期范围。
4. PR 描述中**显式列出**这次修改了哪些文档,reviewer 对照本表逐项核对。

reviewer 发现任一文档未同步,直接 block 合并并要求补齐,不商量。

## 文档入口

- `docs/current/plan.md` — 当前唯一有效的实施计划
- `docs/current/architecture.md` — 当前运行时架构基线
- `docs/current/code-health-audit.md` — 多阶段代码健康审计计划（2026-05-13 激活）
- `internal/api/README.md` — API handler 地图与字段反向索引（改 handler 之前必读）

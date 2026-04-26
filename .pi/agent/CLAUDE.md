# SecBench 项目上下文

> 本项目 Agent 配置。所有在此项目目录下工作的 Agent 必须先读取 `wiki/SCHEMA.md`。

---

## 强制规则

1. **每次任务开始前**，读取 `wiki/SCHEMA.md` 了解项目架构、技术栈、编码约定和安全红线
2. **遇到架构问题时**，先查阅 `wiki/decisions/` 中的 ADR，避免重复决策
3. **编码时**，遵守 `wiki/conventions/` 中的前后端约定
4. **遇到已知坑时**，查阅 `wiki/pitfalls/` 并记录新发现
5. **完成关键决策后**，在 `wiki/decisions/` 添加新的 ADR
6. **完成里程碑后**，更新 `wiki/SCHEMA.md` 中的里程碑状态

---

## 项目速查

- **名称**: SecBench
- **类型**: 目标中心自动化安全测试工作台（桌面应用）
- **技术栈**: Tauri 2.x + React 18 + TypeScript + Tailwind + Go 1.22 + SQLite (WAL)
- **通信**: HTTP API (:8080) + SSE 实时推送
- **当前里程碑**: M0 已完成（工程骨架），M1 待开始
- **数据目录**: `~/.secbench`

---

## Agent 使用指南

| 任务类型 | 使用 Agent | Chain |
|----------|-----------|-------|
| 新功能开发 | `frontend-dev` + `backend-dev` | `/chain feature-dev` |
| Bug 修复 | `frontend-dev` 或 `backend-dev` | `/chain bug-fix` |
| 重构 | 视范围而定 | `/chain refactor` |
| 安全审计 | `qa-engineer` | `/chain security-audit` |
| 架构决策 | `tech-advisor` | `/chain arch-decision` |
| 代码审查 | `qa-engineer` | 单独调用 |

---

## 相关文件

- `wiki/SCHEMA.md` — 项目 AI 指令（必读）
- `wiki/index.md` — 知识库索引
- `设计.md` — PRD
- `plan.md` — 开发计划
- `docs/API.md` — API 参考
- `docs/ARCHITECTURE.md` — 架构说明

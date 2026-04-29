# 📚 Anchor 文档中心

> 最后更新：2026-04-29
> 当前状态：v0.2 已完成，准备开启新开发计划

---

## 🗂️ 文档索引

### 归档文档（v0.1 + v0.2）

所有 v0.1 和 v0.2 阶段的计划、设计、决策文档已归档至 [`archived/`](archived/)。

| 文档 | 版本 | 说明 |
|------|------|------|
| [v0.1 执行计划](archived/v0.1/plan.md) | v0.1 | M0-M4 开发计划，全部完成 |
| [v0.1 PRD](archived/v0.1/设计.md) | v0.1 | MVP 产品需求文档 |
| [v0.1 ADR](archived/v0.1/decisions/) | v0.1 | 8 个架构决策记录（ADR-001 ~ ADR-008） |
| [v0.2 PRD](archived/v0.2/v0.2-prd.md) | v0.2 | 实战可用化与体验提升 |
| [v0.2 执行计划](archived/v0.2/执行计划-v0.2.md) | v0.2 | M0-M5 开发计划，全部完成 |
| [v0.2 ADR](archived/v0.2/ADR-v0.2.md) | v0.2 | Worker 架构、通信协议、DB Schema |
| [系统架构](archived/v0.2/ARCHITECTURE.md) | v0.2 | 系统架构说明（含 ASCII 架构图） |
| [API 参考](archived/v0.2/API.md) | v0.2 | API 端点参考文档 |
| [部署指南](archived/v0.2/部署指南.md) | v0.2 | Docker Compose 部署指南 |
| [内网扫描设计](archived/v0.2/design/内网扫描与容器化架构设计.md) | v0.2 | 内网扫描与容器化详细设计 |
| [前端重构计划](archived/v0.2/design/DESIGN_REFACTOR_PLAN.md) | v0.2 | Apple 设计语言 UI 重构 |
| [v0.2 进度](archived/v0.2/progress.md) | v0.2 | 开发进度跟踪 |
| [v0.2 ADR](archived/v0.2/decisions/) | v0.2 | 2 个架构决策记录（ADR-009 ~ ADR-010） |

> 📖 详细归档说明见 [archived/README.md](archived/README.md)

---

### 活跃文档

> 🔨 新开发计划的文档将放置在此区域

| 文档 | 说明 | 状态 |
|------|------|------|
| _(待创建)_ | 新阶段 PRD | ⏳ 待启动 |
| _(待创建)_ | 新阶段执行计划 | ⏳ 待启动 |
| _(待创建)_ | 新阶段架构决策 | ⏳ 待启动 |

---

### 通用文档（持续维护）

| 位置 | 说明 |
|------|------|
| [`../wiki/conventions/`](../wiki/conventions/) | 编码规范（API、后端、前端） |
| [`../wiki/pitfalls/`](../wiki/pitfalls/) | 踩坑记录（7 篇） |
| [`../wiki/SCHEMA.md`](../wiki/SCHEMA.md) | 数据库 Schema 文档 |
| [`../wiki/log.md`](../wiki/log.md) | 开发日志 |
| [`../README.md`](../README.md) | 项目 README |
| [`../CHANGELOG.md`](../CHANGELOG.md) | 变更日志 |

---

## 📋 文档规范

### 新文档命名约定

- PRD: `prd-v{版本}.md`
- 执行计划: `plan-v{版本}.md`
- 架构决策: `adr-{编号}-{主题}.md`
- 设计文档: `design/{主题}.md`
- API 文档: `api-v{版本}.md`

### 归档规则

- 阶段结束后，所有计划/设计/决策文档移入 `archived/v{版本}/`
- 归档文档添加 YAML frontmatter（archived、version、status、reason）
- 原位置保留重定向文件，指向归档位置

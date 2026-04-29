# 📋 文档归档报告

> 归档日期：2026-04-29
> 归档人：doc-archivist
> 触发原因：v0.2 阶段结束，准备开启新开发计划

---

## 归档操作清单

### v0.1 文档（8 个）

| 原始位置 | 归档位置 | 操作类型 | 说明 |
|----------|----------|----------|------|
| `plan.md` | `docs/archived/v0.1/plan.md` | 移动 | v0.1 执行计划，M0-M4 全部完成 |
| `设计.md` | `docs/archived/v0.1/设计.md` | 移动 | v0.1 MVP PRD，已完全实现 |
| `wiki/decisions/001-tauri-go-communication.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-001 |
| `wiki/decisions/002-sse-over-websocket.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-002 |
| `wiki/decisions/003-zustand-state-management.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-003 |
| `wiki/decisions/004-sqlite-wal.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-004 |
| `wiki/decisions/005-worker-in-process.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-005 |
| `wiki/decisions/006-scope-check-gate.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-006 |
| `wiki/decisions/007-fingerprint-driven-nuclei-scanning.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-007 |
| `wiki/decisions/008-asset-normalization.md` | `docs/archived/v0.1/decisions/` | 移动 | ADR-008 |

### v0.2 文档（10 个）

| 原始位置 | 归档位置 | 操作类型 | 说明 |
|----------|----------|----------|------|
| `第二阶段设计.md` | `docs/archived/v0.2/v0.2-prd.md` | 移动 | v0.2 PRD |
| `progress.md` | `docs/archived/v0.2/progress.md` | 移动 | v0.2 开发进度 |
| `docs/执行计划-v0.2.md` | `docs/archived/v0.2/执行计划-v0.2.md` | 移动 | v0.2 执行计划，M0-M5 全部完成 |
| `docs/ADR-v0.2.md` | `docs/archived/v0.2/ADR-v0.2.md` | 移动 | v0.2 架构决策记录 |
| `docs/ARCHITECTURE.md` | `docs/archived/v0.2/ARCHITECTURE.md` | 移动 | 系统架构说明 |
| `docs/API.md` | `docs/archived/v0.2/API.md` | 移动 | API 参考文档 |
| `docs/部署指南.md` | `docs/archived/v0.2/部署指南.md` | 移动 | Docker 部署指南 |
| `docs/design/内网扫描与容器化架构设计.md` | `docs/archived/v0.2/design/` | 移动 | 详细设计文档 |
| `frontend/DESIGN_REFACTOR_PLAN.md` | `docs/archived/v0.2/design/` | 移动 | 前端重构计划 |
| `wiki/decisions/009-remote-worker-architecture.md` | `docs/archived/v0.2/decisions/` | 移动 | ADR-009 |
| `wiki/decisions/010-docker-containerization.md` | `docs/archived/v0.2/decisions/` | 移动 | ADR-010 |

---

## 重定向文件（原位置保留）

以下文件在原位置已替换为重定向文件，指向归档位置：

| 原位置 | 指向 |
|--------|------|
| `plan.md` | `docs/archived/v0.1/plan.md` |
| `设计.md` | `docs/archived/v0.1/设计.md` |
| `第二阶段设计.md` | `docs/archived/v0.2/v0.2-prd.md` |
| `progress.md` | `docs/archived/v0.2/progress.md` |
| `docs/执行计划-v0.2.md` | `archived/v0.2/执行计划-v0.2.md` |
| `docs/ADR-v0.2.md` | `archived/v0.2/ADR-v0.2.md` |
| `docs/ARCHITECTURE.md` | `archived/v0.2/ARCHITECTURE.md` |
| `docs/API.md` | `archived/v0.2/API.md` |
| `docs/部署指南.md` | `archived/v0.2/部署指南.md` |
| `docs/design/README.md` | `../archived/v0.2/design/` |
| `frontend/DESIGN_REFACTOR_PLAN.md` | `../docs/archived/v0.2/design/DESIGN_REFACTOR_PLAN.md` |
| `wiki/decisions/README.md` | `docs/archived/v0.1/decisions/` + `docs/archived/v0.2/decisions/` |

---

## 创建/更新的索引文件

| 文件 | 说明 |
|------|------|
| `docs/README.md` | 文档中心索引（新建） |
| `docs/archived/README.md` | 归档目录说明（新建） |

---

## 保留不动的文件

| 文件 | 原因 |
|------|------|
| `README.md` | 项目 README，标准文件 |
| `CHANGELOG.md` | 变更日志，标准文件 |
| `wiki/conventions/*` | 编码规范，持续有效 |
| `wiki/pitfalls/*` | 踩坑记录，持续有效 |
| `wiki/SCHEMA.md` | 数据库 Schema，持续有效 |
| `wiki/log.md` | 开发日志，持续有效 |
| `wiki/index.md` | Wiki 索引，持续有效 |
| `docker-rangefield/README.md` | 靶场环境说明，持续有效 |
| `frontend/e2e/*` | E2E 测试文档，持续有效 |

---

## 注意事项

1. 所有归档文档已添加 YAML frontmatter（archived、version、status、reason）
2. 归档文档完整保留原始内容，未做任何修改
3. 原位置替换为轻量重定向文件，便于快速定位归档位置
4. `docs/README.md` 为新阶段文档预留了位置

## 下一步建议

- [ ] 创建新阶段 PRD
- [ ] 创建新阶段执行计划
- [ ] 确认 `wiki/SCHEMA.md` 是否需要更新（反映最新数据库结构）
- [ ] 更新 `wiki/index.md` 索引

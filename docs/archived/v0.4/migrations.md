---
status: superseded
source_of_truth: false
owner: kun
last_updated: 2026-05-07
archived_on: 2026-05-12
scope: v0.4-migrations
verification: passed
---

# v0.4 数据库迁移记录

> **归档说明(2026-05-12)**:本文档是 v0.4 阶段的迁移记录。后续 v6+ 迁移和当前 schema 见 `internal/db/migrate_*.go` 及 `docs/schema-migrations.md`。
>
> v0.4.0 已发布，迁移已通过 E2E 验收（见 `docs/active/review/v0.4-acceptance.md`）

---

## 迁移总览

| 版本 | 说明                                                               | 状态      |
| ---- | ------------------------------------------------------------------ | --------- |
| v1   | 初始 Schema                                                        | ✅ 已完成 |
| v2   | 新增 rate_limit                                                    | ✅ 已完成 |
| v3   | v0.2 功能（tool_templates, tool_invocations, runs, scan_plans 等） | ✅ 已完成 |
| v4   | v0.4 Pipeline（多目标类型、FOFA、DNS、CDN、指纹）                  | ✅ 已完成 |
| v5   | 删除 projects.start_time / end_time                                | ✅ 已完成 |

---

## v1 — 初始 Schema

创建核心表：

- `projects` — 项目
- `targets` — 扫描目标（domain/url/ip/cidr）
- `scope_rules` — 范围规则
- `scan_plans` — 扫描计划
- `scan_tasks` — 扫描任务
- `raw_artifacts` — 原始证据
- `assets` / `ports` / `services` / `web_endpoints` — 资产发现结果
- `findings` / `evidence` — 漏洞发现
- `audit_logs` — 审计日志

---

## v2 — 速率限制

- `projects` 新增 `rate_limit INTEGER`

---

## v3 — v0.2 功能扩展

新增表：

- `tool_templates` — 工具模板
- `tool_invocations` — 工具调用记录
- `runs` — 扫描运行
- `workers` — Worker 节点
- `health_checks` — 健康检查
- `screenshots` — 截图
- `retest_runs` — 复测记录

修改：

- `scan_tasks` 新增 `run_id` 外键

---

## v4 — v0.4 Pipeline

### 4.1 targets 表扩展

- `type` CHECK 约束扩展为 `('domain','url','ip','cidr','company')`
- 支持 `company` 目标类型

### 4.2 projects 表扩展

- `fofa_email TEXT` — FOFA 邮箱
- `fofa_api_key TEXT` — FOFA API Key
- `pipeline_config TEXT` — Pipeline 配置 JSON

### 4.3 新增 dns_records 表

```sql
CREATE TABLE dns_records (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    domain TEXT NOT NULL,
    ips TEXT NOT NULL,
    cnames TEXT,
    ttl INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, domain)
);
```

### 4.4 新增 cdn_results 表

```sql
CREATE TABLE cdn_results (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    ip TEXT NOT NULL,
    is_cdn BOOLEAN DEFAULT FALSE,
    cdn_provider TEXT,
    cdn_type TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, ip)
);
```

### 4.5 新增 service_fingerprints 表

```sql
CREATE TABLE service_fingerprints (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    protocol TEXT DEFAULT 'tcp',
    is_web BOOLEAN DEFAULT FALSE,
    service TEXT NOT NULL,
    metadata TEXT,
    source TEXT DEFAULT 'nerva',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, ip, port)
);
```

---

## v5 — 删除时间窗口字段

### 背景

项目级别的 `start_time` / `end_time` 功能被判定为不必要，删除以简化模型。

### 变更

- `projects` 表删除 `start_time DATETIME`
- `projects` 表删除 `end_time DATETIME`

### 实现方式

SQLite 不支持 `ALTER TABLE DROP COLUMN`，因此通过重建表实现：

1. 创建 `projects_new`（不含时间字段）
2. 复制数据
3. 删除旧表
4. 重命名新表

---

## 迁移幂等性说明

v4 迁移中的 `ALTER TABLE ADD COLUMN` 操作已做幂等处理：

- 通过 `pragma_table_info` 检查列是否存在
- 已存在则跳过，避免重复执行报错

v5 迁移同样做幂等检查：

- 通过 `pragma_table_info` 检查 `start_time` 是否存在
- 不存在则跳过重建

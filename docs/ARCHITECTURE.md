# SecBench 架构说明

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Desktop Client (Tauri)                    │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌───────────────┐  │
│  │ Project │  │ Target  │  │  Plan   │  │     Runs      │  │
│  │  Page   │  │  Page   │  │  Page   │  │    Page       │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └───────┬───────┘  │
│       │            │            │                │          │
│       └────────────┴────────────┴────────────────┘          │
│                         │                                    │
│                    ┌────┴────┐                               │
│                    │ Zustand │                               │
│                    │  Store  │                               │
│                    └────┬────┘                               │
│                         │                                    │
│                    ┌────┴────┐                               │
│                    │ api.ts  │  ← HTTP + SSE                 │
│                    └────┬────┘                               │
└─────────────────────────┼────────────────────────────────────┘
                          │
                          ▼ HTTP API (:8080)
┌─────────────────────────────────────────────────────────────┐
│                  Local Control Plane (Go)                    │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌───────────────┐  │
│  │  API    │  │  Scope  │  │ Worker  │  │   Health      │  │
│  │ Handlers│  │ Engine  │  │ Runner  │  │   Checker     │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └───────┬───────┘  │
│       │            │            │                │          │
│       └────────────┴────────────┴────────────────┘          │
│                         │                                    │
│                    ┌────┴────┐                               │
│                    │   DB    │  ← SQLite                     │
│                    │ Queries │                               │
│                    └────┬────┘                               │
│                         │                                    │
│                    ┌────┴────┐                               │
│                    │ SQLite  │                               │
│                    │  (WAL)  │                               │
│                    └─────────┘                               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼ subprocess
┌─────────────────────────────────────────────────────────────┐
│              External CLI Tools (Subfinder, etc.)            │
└─────────────────────────────────────────────────────────────┘
```

## 数据流

### 1. 创建项目 → 添加目标 → Scope Check

```
用户输入 → TargetPage → POST /projects/:id/targets
                              → DB (targets 表)
                              → ScopeEngine.Check()
                              → DB (scope_decisions 表)
                              → 返回 allow/deny
```

### 2. 启动扫描任务

```
用户点击"运行" → TargetPage → POST /tasks/run
                              → DB (scan_tasks 表, status=queued)
                              → goroutine: Worker.Run()
                              → TOCTOU Scope Check
                              → exec.CommandContext()
                              → DB (tool_invocations 表)
                              → 输出 → RawArtifact
                              → DB (raw_artifacts 表)
                              → DB (scan_tasks 表, status=completed)
                              → SSE broadcast
```

### 3. 查看结果

```
用户打开 RunsPage → GET /scan-tasks/:id
                              → GET /tasks/:id/artifacts
                              → 展示任务状态 + RawArtifact 列表
```

## 模块职责

| 模块 | 文件 | 职责 |
|------|------|------|
| API | `internal/api/handlers.go` | HTTP 路由、请求解析、响应序列化、错误包装 |
| Scope | `internal/scope/scope.go` | 域名/URL/IP 匹配、排除优先、TOCTOU 校验 |
| Worker | `internal/worker/worker.go` | 子进程执行、workdir 隔离、超时、取消、Artifact 保存 |
| Health | `internal/health/health.go` | 工具 binary/version/DNS/network 检测 |
| DB | `internal/db/db.go` | SQLite 初始化、schema、迁移 |
| Queries | `internal/db/queries.go` | 所有 CRUD 操作 |
| Models | `internal/models/models.go` | 数据模型定义 + 枚举 |
| Errors | `internal/errors/errors.go` | 结构化错误码 + HTTP 状态映射 |
| Util | `internal/util/id.go` | 线程安全 ID 生成 |

## 安全设计

1. **Scope Check 强制门控**：所有扫描任务执行前必须通过 Scope Check，TOCTOU 防护防止 Scope 变更后的误扫
2. **操作分级**：L1-L5 分级，MVP 只实现 L1-L3
3. **日志脱敏**：Authorization、Cookie、API Key 等敏感信息不写入日志
4. **workdir 隔离**：每个任务独立 workdir，文件权限 0640/0750
5. **输出截断**：单个输出上限 100MB，防止磁盘耗尽

## 进程模型

MVP（v0.1）：
- Worker 作为 Control Plane 内的 goroutine 运行（同进程）
- 工具子进程通过 `os/exec` 启动
- v0.2 计划拆分为独立进程通过 HTTP 通信

## 数据库 Schema

核心表：
- `projects` — 项目
- `targets` — 目标
- `scope_rules` — Scope 规则（include/exclude）
- `scan_plans` — 扫描计划
- `scan_tasks` — 扫描任务
- `tool_invocations` — 工具调用记录
- `scope_decisions` — Scope Check 结果
- `raw_artifacts` — 原始工具输出
- `audit_logs` — 审计日志
- `tool_health` — 工具健康状态

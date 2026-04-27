# Anchor 架构说明

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Desktop Client (Tauri)                    │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌───────────────┐  │
│  │ Project │  │ Target  │  │  Asset  │  │   Finding     │  │
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
│  ┌────┴────┐  ┌────┴────┐  ┌────┴────┐  ┌───────┴───────┐  │
│  │ Report  │  │  Asset  │  │ Scoring │  │   Parser      │  │
│  │ Generator│  │Normalizer│  │ Engine  │  │ (JSONL)       │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └───────┬───────┘  │
│       │            │            │                │          │
│       └────────────┴────────────┴────────────────┘          │
│                         │                                    │
│                    ┌────┴────┐                               │
│                    │   DB    │  ← SQLite (WAL)               │
│                    │ Queries │                               │
│                    └─────────┘                               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼ subprocess
┌─────────────────────────────────────────────────────────────┐
│              External CLI Tools (Subfinder, etc.)            │
└─────────────────────────────────────────────────────────────┘
```

---

## 数据流

### 1. 创建项目 → 添加目标 → Scope Check

```
用户输入 → TargetPage → POST /projects/:id/targets
                              → DB (targets 表)
                              → ScopeEngine.Check()
                              → DB (scope_decisions 表)
                              → 返回 allow/deny
```

### 2. 资产发现工作流（M2）

```
用户点击"资产发现" → AssetPage → POST /projects/:id/workflows/asset-discovery
                                          → DB (scan_tasks 表)
                                          → goroutine: AssetDiscoveryWorkflow
                                          → TOCTOU Scope Check
                                          → Worker.Run(Subfinder)
                                          → 解析 JSONL → MergeOrCreateAsset(domain)
                                          → Worker.Run(httpx)
                                          → 解析 JSONL → CreateWebEndpoint + MergeOrCreateAsset(url)
                                          → Worker.Run(Naabu)
                                          → 解析 JSONL → MergeOrCreateAsset(ip) + CreatePort
                                          → DB (scan_tasks 表, status=completed)
                                          → SSE broadcast
```

### 3. Web 初筛工作流（M3）

```
用户点击"Web 初筛" → FindingsPage → POST /projects/:id/workflows/web-screening
                                          → DB (scan_tasks 表)
                                          → goroutine: WebScreeningWorkflow
                                          → 查询 WebEndpoint 列表
                                          → 按技术指纹分组 → BuildNucleiCommand(按 tag)
                                          → Worker.Run(Nuclei) 每 tag 一组
                                          → 解析 JSONL → dedup_key 去重
                                          → ScoreFinding(confidence/priority)
                                          → CreateFinding / UpdateFindingEvidence
                                          → saveEvidenceArtifact(原始数据 + 脱敏 excerpt)
                                          → DB (scan_tasks 表, status=completed)
                                          → SSE broadcast
```

### 4. Finding 验证 → 报告导出（M4）

```
用户打开 FindingsPage → GET /projects/:id/findings
                              → 筛选/确认/标记状态
                              → PATCH /findings/:id/status
                              → POST /findings/:id/evidence (添加备注)

用户打开 ReportsPage → GET /projects/:id/reports/export.md
                              → report.Aggregate() JOIN 全量数据
                              → Markdown 生成（8 章节模板）
                              → text/markdown 响应

                       → GET /projects/:id/reports/export.json
                              → JSON 结构化导出
                              → application/json 响应
```

---

## 模块职责

| 模块 | 文件 | 职责 |
|------|------|------|
| API | `internal/api/handlers.go` | HTTP 路由、请求解析、响应序列化、错误包装 |
| Scope | `internal/scope/scope.go` | 域名/URL/IP 匹配、排除优先、TOCTOU 校验 |
| Worker | `internal/worker/worker.go` | 子进程执行、workdir 隔离、超时、取消、Artifact 保存 |
| Health | `internal/health/health.go` | 工具 binary/version/DNS/network 检测 |
| DB | `internal/db/db.go` | SQLite 初始化、schema、迁移 |
| Queries | `internal/db/queries.go` | 所有 CRUD 操作 |
| Models | `internal/models/models.go` | 数据模型定义 + 枚举 + 自定义 Scan/Value |
| Errors | `internal/errors/errors.go` | 结构化错误码 + HTTP 状态映射 |
| Parser | `internal/parser/*.go` | Subfinder/httpx/Naabu/Nuclei JSONL 解析 |
| Asset | `internal/asset/*.go` | 资产归一化（domain/url/ip）+ Merger 去重 |
| Nuclei | `internal/nuclei/tagmapper.go` | httpx fingerprint → Nuclei tag 映射 |
| Scoring | `internal/scoring/scoring.go` | Finding confidence/priority 规则评分 |
| Report | `internal/report/*.go` | Markdown/JSON 报告生成、数据聚合 |
| Util | `internal/util/*.go` | 线程安全 ID 生成、HTTP 脱敏 |
| Workflow | `internal/workflow/*.go` | 资产发现、Web 初筛工作流编排 |

---

## 安全设计

1. **Scope Check 强制门控**：所有扫描任务执行前必须通过 Scope Check，TOCTOU 防护防止 Scope 变更后的误扫
2. **资产二次校验**：工具发现的新资产（子域名、URL、IP）在写入数据库前再次过 Scope Check
3. **操作分级**：L1-L5 分级，MVP 只实现 L1-L3
4. **日志脱敏**：Authorization、Cookie、API Key 等敏感信息不写入日志
5. **证据保留原始数据**：RawArtifact 保存原始输出，Evidence.Excerpt 使用脱敏版本
6. **workdir 隔离**：每个任务独立 workdir，文件权限 0640/0750
7. **输出截断**：单个输出上限 100MB，Evidence 上限 10MB

---

## 进程模型

MVP（v0.1）：
- Worker 作为 Control Plane 内的 goroutine 运行（同进程）
- 工具子进程通过 `os/exec` 启动
- v0.2 计划拆分为独立进程通过 HTTP 通信

---

## 数据库 Schema（核心表）

| 表 | 说明 | 关键约束 |
|----|------|----------|
| `projects` | 项目 | — |
| `targets` | 目标 | — |
| `scope_rules` | Scope 规则（include/exclude） | — |
| `scan_plans` | 扫描计划 | — |
| `scan_tasks` | 扫描任务 | — |
| `tool_invocations` | 工具调用记录 | — |
| `scope_decisions` | Scope Check 结果 | — |
| `raw_artifacts` | 原始工具输出 | — |
| `audit_logs` | 审计日志 | — |
| `tool_health` | 工具健康状态 | — |
| `assets` | 资产（domain/ip/url） | `UNIQUE(project_id, normalized_value)` |
| `ports` | 端口 | `UNIQUE(asset_id, port)` |
| `services` | 服务识别结果 | — |
| `web_endpoints` | Web 端点 | `UNIQUE(project_id, url)` |
| `findings` | 漏洞发现 | `UNIQUE(project_id, dedup_key)` |
| `evidence` | 证据 | — |

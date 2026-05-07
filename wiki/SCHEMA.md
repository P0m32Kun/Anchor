# Anchor Project Schema

> 这是 Anchor 项目的 AI 指令文件。所有 Agent 在处理此项目代码前，必须先读取此文件。
> 最后更新：2026-05-07

---

## 1. 项目概述

**Anchor** — 目标中心自动化安全测试工作台。

通过编排成熟开源工具（Subfinder、dnsx、httpx、Naabu、Nuclei、nerva、cdncheck），强制 Scope 校验，统一结果模型，减少安全人员在工具切换、数据整理、证据归档和报告交付上的重复劳动。

---

## 2. 技术栈

| 层级          | 技术                        | 版本      |
| ------------- | --------------------------- | --------- |
| 桌面客户端    | Tauri 2.x                   | v2        |
| 前端框架      | React 18 + TypeScript       | 18+       |
| 样式          | Tailwind CSS v3 + shadcn/ui | v3 (当前) |
| 状态管理      | Zustand                     | —         |
| 本地/远程服务 | Go                          | 1.26+     |
| 数据库        | SQLite (WAL 模式)           | —         |
| 实时推送      | SSE (Server-Sent Events)    | —         |
| 语法高亮      | Prism.js                    | —         |

**外部工具依赖**：Subfinder、dnsx、httpx、Naabu、Nuclei、nerva、cdncheck

---

## 3. 项目目录结构

```
.
├── main.go                     # Go 服务入口（单二进制，server/worker 双模式）
├── go.mod / go.sum            # Go 模块
├── Makefile                    # 构建脚本 + Docker 生命周期
├── docker-compose.yml          # Server + Worker 编排
├── Dockerfile.server           # Server 镜像
├── Dockerfile.worker           # Worker 镜像
├── plan.md                     # 历史计划跳转页（非当前真相）
├── README.md                   # 项目说明
├── docs/                       # 文档中心
│   ├── README.md              # 文档导航入口
│   ├── current/               # 当前唯一有效的计划/架构入口
│   ├── design/                # 候选设计稿
│   ├── archived/              # 历史归档
│   └── 部署指南.md             # 部署指南
├── internal/                    # Go 内部包
│   ├── api/                    # HTTP API handlers
│   ├── asset/                  # 资产归一化与去重
│   ├── db/                     # SQLite schema + queries
│   ├── errors/                 # 结构化错误模型
│   ├── health/                 # 工具健康检查
│   ├── models/                 # 数据模型
│   ├── nuclei/                 # Nuclei 指纹-Tag 映射
│   ├── cdn/                    # CDN 检测器（cdncheck 集成）
│   ├── fingerprint/            # 服务指纹识别（nerva 集成）
│   ├── parser/                 # 工具输出解析器
│   ├── resolve/                # DNS 解析器
│   ├── search/                 # 搜索引擎客户端（FOFA / Hunter / Quake）
│   ├── report/                 # Markdown / JSON 报告生成
│   ├── scoring/                # Finding confidence/priority 评分引擎
│   ├── scope/                  # Scope Check 引擎
│   ├── util/                   # 工具函数（脱敏、ID 生成等）
│   ├── worker/                 # Worker subprocess runner
│   └── workflow/               # 资产发现 / Web 初筛工作流编排
├── frontend/                    # Tauri + React 前端
│   ├── src/
│   │   ├── lib/               # API 客户端 + Zustand store
│   │   ├── components/        # 共享 UI 组件
│   │   ├── pages/             # 页面组件
│   │   └── App.tsx            # 路由与布局
│   └── package.json
├── src-tauri/                   # Tauri 配置
│   ├── Cargo.toml
│   └── tauri.conf.json
└── wiki/                        # 本项目知识库 (你在这里)
```

---

## 4. 关键架构决策（不可违背）

### ADR-001: Tauri ↔ Go 通信模式

- **决策**: MVP 使用 HTTP API (:8080)，后续迁移到 Tauri Command
- **理由**: HTTP 模式开发调试更快，MVP 阶段不需要 Native API
- **当前状态**: HTTP API 已实现，SSE 用于实时推送

### ADR-002: SSE 替代 WebSocket

- **决策**: 使用 SSE 替代 WebSocket 做实时推送
- **理由**: MVP 只需服务端→客户端单向推送，SSE 更简单且基于 HTTP
- **当前状态**: 已实现

### ADR-003: Zustand 状态管理

- **决策**: 使用 Zustand 替代 Redux Toolkit
- **理由**: MVP 状态不复杂，Zustand 更轻量，API 更简洁
- **当前状态**: 已实现

### ADR-004: SQLite WAL 模式

- **决策**: 使用 SQLite WAL 模式
- **理由**: 读写并发性能更好，适合桌面应用本地存储
- **当前状态**: 已实现，data_dir = ~/.anchor

### ADR-005: Worker 同进程模型 (MVP，已演进)

- **决策**: ~~Worker 作为 Control Plane 内的 goroutine 运行~~ → v0.2 演进为单二进制双模式
- **理由**: MVP 复杂度最低，v0.2 再考虑拆分为独立进程
- **当前状态**: 已演进为 ADR-009/010

### ADR-009: 远程 Worker 架构（出站连接）

- **决策**: Worker 主动发起 outbound HTTP 连接到 Server，通过长轮询拉取任务
- **理由**: NAT 友好，Worker 不需要公网 IP，符合分布式扫描/C2 行业标准
- **当前状态**: 已实现

### ADR-010: Docker 容器化部署

- **决策**: 单二进制通过 `--worker` 标志区分模式，Docker 镜像分别构建 Server 和 Worker
- **理由**: 减少维护负担，Docker Compose 简化本地开发和生产部署
- **当前状态**: 已实现

### ADR-006: Scope Check 强制门控

- **决策**: 所有扫描任务执行前必须通过 Scope Check，TOCTOU 防护
- **理由**: 安全合规要求，防止授权范围外的误扫
- **当前状态**: 已实现

### ADR-007: 指纹驱动 Nuclei 模板精确筛选

- **决策**: 按 httpx 识别的技术指纹分组，每组传入精确 `-tags` 跑 Nuclei
- **理由**: 避免全量加载模板，减少扫描时间和资源消耗
- **当前状态**: 已实现

### ADR-008: 资产归一化与去重

- **决策**: 基于 `normalized_value` 做资产去重，URL 统一去除 `www.` 前缀
- **理由**: 不同工具可能发现同一资产的不同表示形式
- **当前状态**: 已实现

---

## 5. 里程碑状态

| 里程碑                           | 状态      | Tag         | 说明                                                                                      |
| -------------------------------- | --------- | ----------- | ----------------------------------------------------------------------------------------- |
| M0 工程骨架                      | ✅ 已完成 | `v0.1.0-m0` | SQLite + Scope Check + Worker + SSE + Tauri 骨架                                          |
| M1 目标与 Scope 增强             | ✅ 已完成 | `v0.1.0-m1` | 批量导入 + 时间窗口 + 速率限制                                                            |
| M2 资产发现                      | ✅ 已完成 | `v0.1.0-m2` | Subfinder/httpx/Naabu + 资产归一                                                          |
| M3 Nuclei 初筛                   | ✅ 已完成 | `v0.1.0-m3` | 指纹驱动模板筛选 + Finding + 评分 + 证据                                                  |
| M4 报告导出                      | ✅ 已完成 | `v0.1.0-m4` | Markdown/JSON 报告 + 前端预览 + 端到端验收                                                |
| v0.2 Phase 1 容器化与远程 Worker | ✅ 已完成 | `v0.2.0-p1` | Docker + 远程 Worker + 心跳清理 + WorkersPage                                             |
| v0.2 Phase 2 项目管理与体验修复  | ✅ 已完成 | `v0.2.0-p2` | 项目创建/删除/级联清理 + 导航修复 + Dashboard 快捷入口                                    |
| v0.3 桌面可用性与可靠性          | ✅ 已完成 | `v0.3.0`    | 网络服务扫描 + CPE 指纹补充 + 路由统一 + 扫描入口统一 + httpx follow-redirects + 靶场修复 |
| v0.4 智能扫描管线                | ✅ 已完成 | `v0.4.0`    | Company 目标 + FOFA 自动展开 + CDN 过滤 + nerva 服务指纹 + dnsx DNS 解析 + 8 阶段 Pipeline + Hunter/Quake 搜索引擎 + Nuclei 分层扫描（workflow/tags/both）+ RBKD-SEC/templates + 速率防爆破（-rlm/-c）+ E2E 验收（v0.4-acceptance.md） |

---

## 6. 安全红线

1. **所有扫描任务执行前必须通过 Scope Check**（含 TOCTOU 防护）
2. ** NEVER 在日志中输出 Authorization、Cookie、API Key**
3. **原始证据（RawArtifact）必须保留**，脱敏仅用于展示（Evidence.Excerpt）
4. **用户输入必须在边界处校验**，禁止 shell 拼接，使用 `exec.Command` 数组形式
5. **每个任务独立 workdir**，权限 0640/0750
6. **输出截断**：单个输出上限 100MB，Evidence 上限 10MB

---

## 7. 编码约定速查

### Go

- 包职责分离：`api/` 不直接操作 DB，`db/` 集中所有数据库操作
- 错误处理：使用 `errors.Is` 判断，结构化错误码
- Context：所有涉及 I/O 的函数必须接收 `context.Context`
- 并发安全：涉及 map/shared state 必须使用 sync 原语

### 前端

- 所有后端通信通过 `api.ts` 唯一入口
- 按 domain 拆分 Zustand store
- 组件分展示组件和容器组件
- 不在前端代码中嵌入任何 secret/token

完整约定见：

- `wiki/conventions/backend-conventions.md`
- `wiki/conventions/frontend-conventions.md`
- `wiki/conventions/api-contracts.md`

---

## 8. 踩坑记录

已记录的坑：

- `wiki/pitfalls/20260426-frontend-backend-field-mismatch.md` — 前端字段名与后端不匹配
- `wiki/pitfalls/20260426-artifact-type-mismatch.md` — Worker Artifact 类型与工作流不匹配
- `wiki/pitfalls/20260426-asset-scope-check-missing.md` — 发现的资产未过 Scope Check
- `wiki/pitfalls/20260426-raw-artifact-redaction-loss.md` — 原始证据被脱敏覆盖
- `wiki/pitfalls/20260427-null-scan-crash.md` — NULL 列 Scan 到 string 崩溃
- `wiki/pitfalls/20260427-markdown-pipe-corruption.md` — Markdown 表格被 `|` 破坏
- `wiki/pitfalls/20260428-ghost-worker-cleanup.md` — Worker 重启产生幽灵记录（无心跳超时清理）

---

## 9. Agent 必读文件清单

执行任何任务前，按顺序阅读：

1. `README.md` — 项目入口
2. `docs/README.md` — 文档导航入口
3. `docs/current/plan.md` — 当前唯一仓库级计划
4. `docs/current/architecture.md` — 当前唯一架构基线
5. `wiki/SCHEMA.md`（本文件）— 补充约束与领域上下文
6. `wiki/log.md` — 最新变更
7. `wiki/decisions/` 中相关 ADR — 历史架构约束
8. `wiki/conventions/` 中相关约定 — 编码规范

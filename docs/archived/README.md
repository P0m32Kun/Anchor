# 📦 文档归档

> 归档日期：2026-05-01
> 归档人：doc-archivist

## 归档说明

本目录存放 v0.1、v0.2 和 v0.3 阶段的所有计划、设计、决策文档。这些文档在对应阶段结束后进行归档，保留完整历史记录。

## 目录结构

```
archived/
├── v0.1/                           # v0.1 阶段文档
│   ├── plan.md                     # v0.1 开发执行计划（M0-M4）
│   ├── 设计.md                     # v0.1 MVP PRD
│   └── decisions/                  # v0.1 架构决策记录
│       ├── 001-tauri-go-communication.md
│       ├── 002-sse-over-websocket.md
│       ├── 003-zustand-state-management.md
│       ├── 004-sqlite-wal.md
│       ├── 005-worker-in-process.md
│       ├── 006-scope-check-gate.md
│       ├── 007-fingerprint-driven-nuclei-scanning.md
│       └── 008-asset-normalization.md
│
└── v0.2/                           # v0.2 阶段文档
    ├── v0.2-prd.md                 # v0.2 PRD（实战可用化与体验提升）
    ├── progress.md                 # v0.2 开发进度跟踪
    ├── 执行计划-v0.2.md            # v0.2 执行计划（M0-M5）
    ├── ADR-v0.2.md                 # v0.2 架构决策记录
    ├── ARCHITECTURE.md             # 系统架构说明
    ├── API.md                      # API 参考文档
    ├── 部署指南.md                  # v0.2 部署指南
    ├── design/                     # v0.2 设计文档
    │   ├── 内网扫描与容器化架构设计.md
    │   └── DESIGN_REFACTOR_PLAN.md
    └── decisions/                  # v0.2 架构决策记录
        ├── 009-remote-worker-architecture.md
        └── 010-docker-containerization.md

├── v0.3/                           # v0.3 阶段文档
│   ├── v0.3-plan.md                # v0.3 执行计划（桌面可用性与可靠性）
│   └── progress.md                 # v0.3 开发进度跟踪
```

## v0.1 阶段摘要

| 里程碑 | 状态 | 内容 |
|--------|------|------|
| M0 | ✅ 完成 | Tauri-Go 通信 + Scope Check + Subfinder 最小闭环 |
| M1 | ✅ 完成 | 目标输入 + Scope Check + 执行计划预览 |
| M2 | ✅ 完成 | Subfinder/httpx/Naabu + 资产归一 + RawArtifact |
| M3 | ✅ 完成 | Nuclei + Finding + confidence/priority 评分 |
| M4 | ✅ 完成 | 验证队列 + Markdown/JSON 报告导出 |

## v0.2 阶段摘要

| 里程碑 | 状态 | 内容 |
|--------|------|------|
| M0 | ✅ 完成 | 基础设施准备 |
| M1 | ✅ 完成 | Docker 容器化 |
| M2 | ✅ 完成 | 远程 Worker 架构 |
| M3 | ✅ 完成 | 内网扫描增强 |
| M4 | ✅ 完成 | 前端 UI 优化 |
| M5 | ✅ 完成 | 部署验证 |

## v0.3 阶段摘要

| 里程碑 | 状态 | 内容 |
|--------|------|------|
| 网络服务扫描 | ✅ 完成 | 非 Web 端口（Redis/MySQL/PostgreSQL 等）→ Nuclei tag 映射 + 分组扫描 |
| CPE 指纹补充 | ✅ 完成 | httpx CPE 解析 → tech fallback，解决 404/302 跳过问题 |
| 扫描入口统一 | ✅ 完成 | TargetPage Subfinder 按钮 → Runs 导航，消除入口分裂 |
| 路由统一 | ✅ 完成 | `/projects/:projectId/*` 嵌套路由 |
| 靶场修复 | ✅ 完成 | Tomcat 靶场 Dockerfile 适配新版镜像 |
| httpx 增强 | ✅ 完成 | `-follow-redirects` 参数 |

## 关键架构决策

1. **Tauri ↔ Go 通信**：HTTP API (:8080)，后续考虑 Tauri Command（ADR-001）
2. **实时推送**：SSE 替代 WebSocket（ADR-002）
3. **状态管理**：Zustand（ADR-003）
4. **数据库**：SQLite WAL 模式（ADR-004）
5. **Worker 架构**：从同进程演进到远程 Worker 双模式（ADR-005 → ADR-009）
6. **部署**：Docker 容器化（ADR-010）

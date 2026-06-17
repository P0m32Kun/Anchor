# 📦 文档归档

> 归档日期：2026-05-01（初始），2026-06-17 更新
> 归档人：doc-archivist

## 归档说明

本目录存放历史版本的计划、设计、决策文档。这些文档在对应阶段结束后进行归档，保留完整历史记录。

## 目录结构

```
archived/
├── v0.1/                           # v0.1 阶段文档
│   ├── plan.md                     # v0.1 开发执行计划（M0-M4）
│   ├── 设计.md                     # v0.1 MVP PRD
│   └── decisions/                  # v0.1 架构决策记录（ADR-001–008）
│
├── v0.2/                           # v0.2 阶段文档
│   ├── v0.2-prd.md                 # v0.2 PRD
│   ├── progress.md                 # v0.2 开发进度跟踪
│   ├── 执行计划-v0.2.md            # v0.2 执行计划（M0-M5）
│   ├── ADR-v0.2.md                 # v0.2 架构决策记录
│   ├── ARCHITECTURE.md             # 系统架构说明
│   ├── API.md                      # API 参考文档
│   ├── 部署指南.md                  # v0.2 部署指南
│   ├── design/                     # v0.2 设计文档
│   └── decisions/                  # v0.2 架构决策记录（ADR-009–010）
│
├── v0.3/                           # v0.3 阶段文档
│   ├── v0.3-plan.md                # v0.3 执行计划
│   ├── progress.md                 # v0.3 开发进度跟踪
│   └── review/                     # v0.3 AI review 文档
│
├── v0.4/                           # v0.4 阶段文档
│   ├── acceptance.md               # v0.4 验收记录
│   ├── migrations.md               # v0.4 迁移说明
│   ├── scan-pipeline.md            # 扫描管线设计
│   ├── unified-scan-flow.md        # 统一扫描流程
│   ├── visionos-glassmorphism-redesign.md  # UI 重设计（未采用）
│   └── cyberos-ui-redesign.md      # UI 重设计（未采用）
│
├── slimdown/                       # Anchor Core Slimdown review 系列（2026-06-17 归档）
│   ├── slimdown-t0-workspace.md    # T0: 工作空间清理
│   ├── slimdown-t1-docs.md         # T1: 文档收敛
│   ├── slimdown-t2-dead-code.md    # T2: 死代码清理
│   ├── slimdown-t3-architecture.md # T3: 架构简化
│   ├── slimdown-t4-contracts-security.md  # T4: 合约与安全
│   ├── slimdown-t5-test-matrix.md  # T5: 测试矩阵
│   ├── slimdown-2026-summary.md    # 总结报告
│   └── slimdown-2026-handoff-for-implementer.md  # 实施交接
│
└── plans/                          # 归档的实施计划
    └── 2026-05-19-vuln-template-redesign.md  # 漏洞模板重设计（已归档）
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
| 网络服务扫描 | ✅ 完成 | 非 Web 端口 → Nuclei tag 映射 + 分组扫描 |
| CPE 指纹补充 | ✅ 完成 | httpx CPE 解析 → tech fallback |
| 扫描入口统一 | ✅ 完成 | TargetPage Subfinder 按钮 → Runs 导航 |
| 路由统一 | ✅ 完成 | `/projects/:projectId/*` 嵌套路由 |
| 靶场修复 | ✅ 完成 | Tomcat 靶场 Dockerfile 适配 |
| httpx 增强 | ✅ 完成 | `-follow-redirects` 参数 |

## v0.4 阶段摘要

v0.4 阶段的核心设计文档已归档。UI 重设计方案（visionos-glassmorphism、cyberos-ui）未被采用，scan-pipeline 和 unified-scan-flow 已被资产驱动扫描引擎取代。

## Slimdown 系列摘要

Anchor Core Slimdown 是 2026-06-17 执行的仓库瘦身评审，覆盖工作空间清理、文档收敛、死代码清理、架构简化、合约安全和测试矩阵六个方面。

## 关键架构决策

1. **Tauri ↔ Go 通信**：HTTP API (:8080)，后续考虑 Tauri Command（ADR-001）
2. **实时推送**：SSE 替代 WebSocket（ADR-002）
3. **状态管理**：Zustand（ADR-003）
4. **数据库**：SQLite WAL 模式（ADR-004）
5. **Worker 架构**：从同进程演进到远程 Worker 双模式（ADR-005 → ADR-009）
6. **部署**：Docker 容器化（ADR-010）

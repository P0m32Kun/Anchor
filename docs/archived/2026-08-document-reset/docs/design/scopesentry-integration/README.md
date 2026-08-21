---
status: draft
source_of_truth: false
owner: kun
last_updated: 2026-08-20
scope: distributed-asm-evolution
verification: pending_implementation
---

# ScopeSentry 整合与分布式演进 — 提案索引

> Status: Draft
> Audience: 实施者（coding agent / 开发者）
> 默认规则：提案，不是当前架构基线。当前基线仍以 `docs/current/architecture.md` 为准。

## 背景

Anchor 定位从「目标中心渗透测试工作台」升级为「企业级分布式攻击面测绘平台」。
基于对 ScopeSentry（AGPL-3.0 + 商用附加条款）的完整调研（2026-08-20），确定：
以 Anchor 为基座（~60% 已验收代码复用），改造分布式外壳（~20%），
按 Anchor 范式重新实现 ScopeSentry 的扫描广度与 UI 设计（~20%）。

## 已定决策（owner 2026-08-20 确认）

1. **不做插件系统** — 固定工具链，扩展走 toolregistry YAML + 新解析器
2. **不接 LLM** — POC 由独立维护库供给，经 builtin 同步机制对接
3. **纯代码 + 规则 + API + 工具** — 不引入解释器插件运行时
4. **UI 参考 ScopeSentry** 信息架构与视觉，前端保留 React 组件体系
5. **合规红线** — ScopeSentry 为 AGPL-3.0 且商用需授权：只参考公开思路与数据格式，**不复制其代码**

## 目录

| 文档 | 内容 |
| --- | --- |
| [`design.md`](design.md) | 主设计：背景调研、目标架构、工具链基线、借鉴模块设计、数据模型、风险 |
| [`roadmap.md`](roadmap.md) | 里程碑 M1–M4：任务拆解与验收标准 |

## 状态生命周期

遵循 `docs/mempalace/convention-design-status-lifecycle.md`：
draft → in_review → (提升实施) / superseded / cancelled。

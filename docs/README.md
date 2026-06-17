# Anchor 文档中心

> 最后更新：2026-06-17
> 默认原则：任何 agent 在阅读历史材料前，先读本页和 `docs/current/`

## 先读这些

按默认顺序阅读：

1. [`../README.md`](../README.md)
2. [`current/agent-guide.md`](current/agent-guide.md)
3. [`current/plan.md`](current/plan.md)
4. [`current/architecture.md`](current/architecture.md)
5. [`current/design/README.md`](current/design/README.md)
6. [`current/decisions/README.md`](current/decisions/README.md)

## 当前有效文档

| 文档 | 角色 | 是否为当前真相 |
| --- | --- | --- |
| [`current/agent-guide.md`](current/agent-guide.md) | Coding agent 迭代协议 | Yes |
| [`current/plan.md`](current/plan.md) | 当前唯一仓库级计划 | Yes |
| [`current/architecture.md`](current/architecture.md) | 当前唯一架构基线 | Yes |
| [`current/deployment.md`](current/deployment.md) | **客户部署**（install.sh / ACR 镜像） | Yes |
| [`current/e2e-testing.md`](current/e2e-testing.md) | **开发者 E2E**（fast 镜像 / Playwright） | Yes |
| [`current/design/README.md`](current/design/README.md) | 当前候选设计索引 | Yes |
| [`current/decisions/README.md`](current/decisions/README.md) | 当前决策索引 | Yes |
| [`api-error-contract.md`](api-error-contract.md) | API 错误约定 | Supporting |
| [`schema-migrations.md`](schema-migrations.md) | Schema 迁移策略与版本历史 | Supporting |
| [`CHANGELOG.md`](CHANGELOG.md) | 项目变更日志 | Supporting |

## 候选方案与评审材料

这些文档可以参考，但默认不应当作当前真相：

| 文档/目录 | 状态 | 说明 |
| --- | --- | --- |
| [`design/`](design/) | In review | 新设计提案区 |
| [`refactoring-plan.md`](refactoring-plan.md) | Backlog | 重构想法和拆分路线 |
| [`superpowers/`](superpowers/) | Mixed | 外网扫描 pipeline 等 superpowers 生成的设计文档；部分已归档到 archived |

## 归档

- 历史版本文档统一放在 [`archived/`](archived/)
- 根目录 [`../plan.md`](../plan.md) 和若干旧文档只保留跳转作用
- 详细归档说明见 [`archived/README.md`](archived/README.md)

## 通用参考

| 位置 | 说明 |
| --- | --- |
| [`conventions/`](conventions/) | 编码规范 |
| [`pitfalls/`](pitfalls/) | 踩坑记录 |

## 维护规则

1. 任意时刻只保留一份仓库级 plan 和一份仓库级 architecture。
2. 新设计先进入 `docs/design/`，只有被提升后才能改写 `docs/current/`。
3. 阶段结束后，把计划、设计和评审材料移入 `docs/archived/` 或明确标成 `archived` / `superseded`。
4. 新增文档时优先补状态头，而不是再新增一个平行入口。
5. 面向 agent 的流程、验证门槛和本地 artifact 规则统一维护在 [`current/agent-guide.md`](current/agent-guide.md)。

### 版本号规则

项目采用三位版本号 `vMAJOR.MINOR.PATCH`（当前 `v0.4.0`），递增规则如下：

| 位置 | 递增条件 | 示例 |
|------|---------|------|
| **PATCH**（最后一位） | 小更新：bug 修复、文档同步、测试补充、配置调整 | `v0.2.1` → `v0.2.2` |
| **MINOR**（中间一位） | 一般更新：新功能、新模块、API 变更、UI 改进 | `v0.2.x` → `v0.3.0` |
| **MAJOR**（第一位） | 较大重构和更新：架构变更、数据模型重构、破坏性变更 | `v0.x.x` → `v1.0.0` |

- MINOR 或 MAJOR 递增时，低位归零。
- 当前处于 `v0.x.x` 阶段，MAJOR=0 表示尚未正式稳定发布；首次稳定版为 `v1.0.0`。

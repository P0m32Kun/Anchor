# Anchor 文档中心

> 最后更新：2026-05-07
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
| [`current/design/README.md`](current/design/README.md) | 当前候选设计索引 | Yes |
| [`current/decisions/README.md`](current/decisions/README.md) | 当前决策索引 | Yes |
| [`acceptance-criteria.md`](acceptance-criteria.md) | 验收标准参考 | Supporting |
| [`api-error-contract.md`](api-error-contract.md) | API 错误约定 | Supporting |
| [`coverage-baseline.md`](coverage-baseline.md) | 测试覆盖率基线 | Supporting |
| [`engineering-health.md`](engineering-health.md) | 工程健康基线 | Supporting |

## 候选方案与评审材料

这些文档可以参考，但默认不应当作当前真相：

| 文档/目录 | 状态 | 说明 |
| --- | --- | --- |
| [`design/`](design/) | In review | 新设计提案区 |
| [`refactoring-plan.md`](refactoring-plan.md) | Backlog | 重构想法和拆分路线 |
| [`active/plan/`](active/plan/) | Review material | 历史实施稿和评审汇总 |
| [`active/review/`](active/review/) | Review material | 多模型评审记录 |

## 归档

- 历史版本文档统一放在 [`archived/`](archived/)
- 根目录 [`../plan.md`](../plan.md) 和若干旧文档只保留跳转作用
- 详细归档说明见 [`archived/README.md`](archived/README.md)

## 通用参考

| 位置 | 说明 |
| --- | --- |
| [`../wiki/conventions/`](../wiki/conventions/) | 编码规范 |
| [`../wiki/pitfalls/`](../wiki/pitfalls/) | 踩坑记录 |
| [`../wiki/SCHEMA.md`](../wiki/SCHEMA.md) | 数据库 Schema |
| [`../wiki/log.md`](../wiki/log.md) | 开发日志 |

## 维护规则

1. 任意时刻只保留一份仓库级 plan 和一份仓库级 architecture。
2. 新设计先进入 `docs/design/`，只有被提升后才能改写 `docs/current/`。
3. 阶段结束后，把计划、设计和评审材料移入 `docs/archived/` 或明确标成 `archived` / `superseded`。
4. 新增文档时优先补状态头，而不是再新增一个平行入口。
5. 面向 agent 的流程、验证门槛和本地 artifact 规则统一维护在 [`current/agent-guide.md`](current/agent-guide.md)。

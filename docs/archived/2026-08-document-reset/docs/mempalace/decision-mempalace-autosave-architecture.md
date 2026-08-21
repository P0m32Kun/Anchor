# Decision: pi-mempalace autosave 维持提醒模式，不实现自动 ingest

## 类型
decision（pi 生态工具链）

## 决策
pi-mempalace-extension 的 autosave 保持当前"每 15 条消息提醒一次"机制，不实现自动提取+存储。

## 评估的方案

| 方案 | 结论 |
|------|------|
| A. 扩展内调 LLM 提取 | ❌ 成本高、延迟大、配置复杂 |
| B. 关键词启发式 | ❌ 脆弱，噪音污染 palace |
| C. 扩展写原始文件我整理 | ❌ 还是需要我参与 |
| D. session_before_compact 写摘要 | ⚠️ 可做但延迟太长，收益有限 |

## 理由
- 提醒机制本身够用，问题在于我是否响应
- 无 LLM 的自动提取质量不可控
- 有 LLM 的方案成本不合理（每 session 多 2-3 次 API 调用）
- 改进方向：提醒文本更具体 + 我养成发现即存的习惯

## 日期
2026-06-16

# Anchor TODOS

> 延后的工作项，按优先级排序。来源：/plan-ceo-review 2026-05-11。

## P2 — 下一个迭代候选

### 持续资产管理
- **What**: 资产不是扫描时才存在，而是持续监控。子域名变了、端口开了、证书快过期了，自动告警。
- **Why**: 当前 Anchor 的资产只在扫描时存在，无法发现扫描间隔期间的变化。
- **Pros**: 从"一次性扫描工具"变成"活的资产图谱"，产品价值质变。
- **Cons**: 需要定时任务基础设施、资产版本管理、告警通道。
- **Effort**: M (human ~3 days / CC ~2 hours)
- **Depends on**: 无

### 截图归档
- **What**: 扫描期间自动访问 web endpoint 并截图，作为漏洞证据。
- **Why**: 截图是渗透测试报告的核心证据，当前需要手工截屏。
- **Pros**: 报告可信度大幅提升，安全人员省去大量手工工作。
- **Cons**: 需要浏览器爬虫模块（chromedp），范围炸弹级复杂度。
- **Effort**: L (human ~5 days / CC ~3 hours)
- **Depends on**: 报告引擎完成

### 扫描历史对比
- **What**: 两次扫描结果 diff，高亮新增/消失的资产和 findings。
- **Why**: 安全人员需要快速看到"这次扫描比上次多了什么"。
- **Pros**: 资产变化可视化，增量发现效率提升。
- **Cons**: 需要资产快照和 diff 算法。
- **Effort**: M (human ~2 days / CC ~1 hour)
- **Depends on**: 无

## P3 — 未来探索

### 增量扫描
- **What**: 不是每次都全量扫描，上次扫过的资产只扫变化的部分。
- **Why**: 全量扫描耗时长，对目标有不必要干扰。
- **Pros**: 扫描效率提升 10x，减少对目标的请求量。
- **Cons**: 需要资产版本管理、diff 逻辑、扫描状态追踪。复杂度高。
- **Effort**: L (human ~5 days / CC ~3 hours)
- **Depends on**: 持续资产管理

### 模板市场
- **What**: 自定义 Nuclei 模板可以共享和复用，安全团队之间分享检测逻辑。
- **Why**: 从"工具"到"生态"的跃迁，社区驱动模板增长。
- **Pros**: 网络效应，用户黏性，差异化竞争力。
- **Cons**: 需要服务端基础设施、模板版本管理、安全审查。
- **Effort**: XL (human ~10 days / CC ~5 hours)
- **Depends on**: Custom Nuclei Template Management（已 in_review）

# Design Tokens 映射表

> 本文档记录前端硬编码颜色向语义 Token 的迁移状态。

## 已映射 Token

### 品牌色 (Brand)

| Token | 色值 | 用途 |
|-------|------|------|
| `brand-primary` | `#00D4FF` | 主操作、链接、运行中状态、聚焦光 |
| `brand-success` | `#00E676` | 成功、完成、在线 |
| `brand-warning` | `#F5A623` | 警告、高严重级别 |
| `brand-danger` | `#FF4757` | 错误、失败、危险操作 |
| `brand-purple` | `#A371F7` | 辅助强调、报告/配置类信息 |
| `brand-info` | `#5AC8FA` | 信息提示、低风险状态 |

### 强调色 (Accent)

| Token | 色值 | 用途 |
|-------|------|------|
| `accent-yellow` | `#FFD60A` | 待处理、队列、忙碌状态 |
| `accent-teal` | `#5AC8FA` | 低严重级别、信息补充 |
| `accent-orange` | `#FF9F0A` | 高亮提示 |
| `accent-green` | `#32D74B` | 健康指示 |
| `accent-red` | `#FF453A` | 紧急告警 |

### 表面层级 (Surface)

| Token | 色值 | 用途 |
|-------|------|------|
| `surface` | `#0A1628` | 页面背景 |
| `surface-base` | `#081020` | 最深遮罩、全局渐变底部 |
| `surface-muted` | `#0D1E36` | 侧栏、顶部移动导航 |
| `surface-elevated` | `#111D32` | 卡片、面板背景 |
| `surface-elevated-2` | `#1A2A42` | 输入框、悬停项 |
| `surface-elevated-3` | `#223555` | 强边框、分割线 |

### 文字色 (Text)

| Token | 色值 | 用途 |
|-------|------|------|
| `text-primary` | `#FFFFFF` | 标题、主文本 |
| `text-secondary` | `#C9D8EF` | 正文、次要内容 |
| `text-tertiary` | `#8B9DC3` | 辅助说明、占位符 |
| `text-quaternary` | `#5A6E8A` | 禁用、最弱层级 |

## 状态标签颜色映射

状态标签统一使用半透明背景 + 同色文字，保持视觉一致性。

| 状态 | 旧写法 | 新 Token |
|------|--------|----------|
| 待处理 / pending | `bg-yellow-500/15 text-yellow-300` | `bg-accent-yellow/15 text-accent-yellow` |
| 运行中 / running | `bg-blue-500/15 text-blue-300` | `bg-brand-primary/15 text-brand-primary` |
| 已完成 / completed | `bg-green-500/15 text-green-300` | `bg-brand-success/15 text-brand-success` |
| 失败 / failed | `bg-red-500/15 text-red-300` | `bg-brand-danger/15 text-brand-danger` |
| 默认 / cancelled | `bg-zinc-800/60 text-zinc-400` | `bg-white/[0.04] text-text-tertiary` |
| 已过期 / expired | `bg-red-500/15 text-red-300` | `bg-brand-danger/15 text-brand-danger` |
| 进行中 / active | `bg-green-500/15 text-green-300` | `bg-brand-success/15 text-brand-success` |
| 已确认 / confirmed | `bg-green-500/15 text-green-300` | `bg-brand-success/15 text-brand-success` |
| 待审核 / pending_review | `bg-yellow-500/15 text-yellow-300` | `bg-accent-yellow/15 text-accent-yellow` |
| 已接受风险 / accepted_risk | `bg-blue-100 text-blue-800` | `bg-brand-primary/15 text-brand-primary` |

## 严重级别映射

| 级别 | 旧写法 | 新 Token |
|------|--------|----------|
| critical | `bg-red-600 text-white` | `bg-brand-danger text-white` |
| high | `bg-orange-500 text-white` | `bg-brand-warning text-white` |
| medium | `bg-yellow-400 text-black` | `bg-accent-yellow text-black` |
| low | `bg-blue-300 text-black` | `bg-accent-teal text-black` |
| info | `bg-gray-200 text-zinc-300` | `bg-white/[0.04] text-text-tertiary` |

## 组件类约定

以下组件类承载“克制版监控大屏”质感，页面代码优先复用这些类，而不是散落 `zinc/slate/sky` 等硬编码色：

- **面板**: `.panel`、`.cyber-glass` 使用深蓝渐变、青色弱边框和轻微内发光。
- **输入**: `.input-dark` 统一输入框、select、textarea 的暗底、青色聚焦光效。
- **按钮**: `.btn-cyber-primary|secondary|ghost|danger|purple|success|warning|info` 覆盖主要语义操作。
- **列表项/筛选**: `.surface-item`、`.filter-pill`、`.filter-pill-active` 用于卡片列表、过滤器、步骤入口。
- **链接**: `.link-cyber` 用于可点击文本链接，保持青色高亮和悬停微光。

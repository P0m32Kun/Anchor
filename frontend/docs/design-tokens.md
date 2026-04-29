# Design Tokens 映射表

> 本文档记录前端硬编码颜色向语义 Token 的迁移状态。

## 已映射 Token

### 品牌色 (Brand)

| Token | 色值 | 用途 |
|-------|------|------|
| `brand-primary` | `#2F81F7` | 主操作、链接、运行中状态 |
| `brand-success` | `#3FB950` | 成功、完成、在线 |
| `brand-warning` | `#D29922` | 警告、高严重级别 |
| `brand-danger` | `#F85149` | 错误、失败、危险操作 |
| `brand-purple` | `#A371F7` | 辅助强调 |

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
| `surface` | `#0B0E14` | 页面背景 |
| `surface-elevated` | `#161B22` | 卡片、面板背景 |
| `surface-elevated-2` | `#21262D` | 输入框、悬停项 |
| `surface-elevated-3` | `#30363D` | 边框、分割线 |

### 文字色 (Text)

| Token | 色值 | 用途 |
|-------|------|------|
| `text-primary` | `#F0F6FC` | 标题、主文本 |
| `text-secondary` | `#8B949E` | 正文、次要内容 |
| `text-tertiary` | `#6E7681` | 辅助说明、占位符 |
| `text-quaternary` | `#484F58` | 禁用、最弱层级 |

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

## 待迁移项（Sprint 1.x 后续）

以下模式在代码库中仍有大量硬编码，建议后续 Sprint 统一替换：

- **卡片容器**: `bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl` → 提取为 `.card-glass` 组件类
- **文字层级**: `text-zinc-100` → `text-text-primary`, `text-zinc-200` → `text-text-secondary`, `text-zinc-400` → `text-text-tertiary`, `text-zinc-500` → `text-text-quaternary`
- **输入框**: `bg-zinc-800 border-zinc-700 text-zinc-200` → 使用 `.input-dark` 组件类
- **浅色残留**: TargetPage 中存在 `bg-gray-50`, `text-red-700`, `text-green-600`, `bg-blue-50` 等浅色模式颜色，需按深色主题统一
- **按钮颜色**: `bg-blue-600`, `bg-green-600`, `bg-purple-600` 等硬编码按钮背景应使用语义化按钮组件

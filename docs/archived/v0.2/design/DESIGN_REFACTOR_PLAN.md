---
archived: true
archived_at: "2026-04-29"
archived_by: doc-archivist
version: "v0.2"
original_path: "frontend/DESIGN_REFACTOR_PLAN.md"
status: "completed"
reason: "v0.2 前端设计重构计划"
---

# Anchor 前端设计重构计划

> 目标：将当前简陋的 UI 重构成 Apple 产品级别的美观、简洁、直观、易用的界面。

---

## 一、当前设计审计

### 1.1 全局样式系统 — 完全缺失

| 问题 | 现状 | 影响 |
|------|------|------|
| 无 CSS 变量 | `index.css` 只有 `@tailwind` + `body { bg-gray-50 }` | 颜色/间距/圆角无法统一管理 |
| 无 Tailwind 主题扩展 | `tailwind.config.js` 的 `theme.extend` 为空对象 | 无法使用语义化 token |
| 无字体栈 | 完全依赖浏览器默认 | 不同系统显示效果差异大 |
| 无间距系统 | 各页面随意写 `p-4`, `gap-3` 等 | 视觉节奏混乱 |

### 1.2 导航栏 — 像 2010 年的后台系统

```tsx
// 当前代码
<nav className="bg-slate-800 text-white px-4 py-3 flex gap-6">
  <Link to="/" className="font-bold text-lg">Anchor</Link>
  <Link to="/" className="hover:underline">项目</Link>
  <Link to="/runs" className="hover:underline">运行</Link>
</nav>
```

**问题：**
- 深色 slate-800 背景像老式管理后台
- 纯文字导航，无图标，无品牌感
- 缺少导航高亮/激活状态
- 导航项缺失（缺少 Findings、Reports 入口）
- 没有 macOS 原生风格的窗口控制集成

### 1.3 配色体系 — 各页面各自为政

```tsx
// ProjectPage: 状态 badge 颜色
<span className="text-xs bg-red-100 text-red-700 px-2 py-0.5 rounded">已过期</span>
<span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded">未开始</span>
<span className="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded">进行中</span>

// FindingsPage: severity 颜色
const severityColors = {
  critical: "bg-red-600 text-white",
  high: "bg-orange-500 text-white",
  medium: "bg-yellow-400 text-black",
  low: "bg-blue-300 text-black",
  info: "bg-gray-200 text-gray-700",
};

// ReportsPage: 又一套 severity 颜色
const colors: Record<string, string> = {
  critical: "bg-red-700 text-white",
  high: "bg-orange-600 text-white",
  medium: "bg-yellow-500 text-black",
  low: "bg-blue-500 text-white",
  info: "bg-gray-400 text-white",
};
```

**问题：**
- 同一语义（severity）在不同页面颜色不同
- 没有统一的语义颜色 token
- 大量使用高饱和度的红黄绿，视觉上刺眼
- 没有暗色/浅色背景的自适应

### 1.4 组件复用度 — 接近零

| 组件 | 出现次数 | 状态 |
|------|---------|------|
| Button | 每页内联定义 | 无统一组件 |
| Badge/Tag | 每页内联定义 | 样式不统一 |
| Card | `bg-white p-4 rounded shadow` 重复 everywhere | 无封装 |
| Input | 每个 input 都写一遍 `w-full border rounded px-3 py-2` | 无封装 |
| Modal/Dialog | 无 | 完全缺失 |
| Toast/Notification | 无 | 完全缺失，用 `alert()` |
| EmptyState | 每页各自写 `暂无...` | 无统一组件 |
| Skeleton | 无 | 完全缺失 |

### 1.5 状态设计 — 严重缺失

| 状态 | 当前处理 | 问题 |
|------|---------|------|
| Loading | 部分页面有 `setLoading(true)`，但很多是简单文字 | 无统一 Loading 组件，无骨架屏 |
| Empty | `暂无项目，请创建第一个项目` 纯文字 | 无图标、无引导操作 |
| Error | `alert(String(err))` | 原生弹窗阻断操作，体验极差 |
| Success | `alert("资产发现工作流已启动")` | 同上 |

### 1.6 交互体验 — 粗糙

- **原生 alert/confirm**：ProjectPage 创建失败用 `alert()`，AssetPage 启动工作流用 `alert()`
- **无 hover/active 反馈**：按钮只有简单的 `hover:bg-slate-600`
- **无表单验证**：input 的 `required` 是唯一验证，没有错误提示样式
- **无加载遮罩**：长时间操作时界面仍可交互
- **无过渡动画**：页面切换、元素出现都是硬切

### 1.7 信息层级 — 混乱

- 页面标题和内容之间没有明确的层级分隔
- 卡片之间没有视觉呼吸感
- 表格没有行 hover 效果、没有斑马纹
- 关键信息（如 severity）不够突出
- 次要信息（创建时间等）喧宾夺主

---

## 二、Apple 设计语言规范

### 2.1 核心原则

1. **Clarity（清晰）**：内容为王，UI 不抢戏
2. **Deference（尊重）**：UI 帮助用户理解内容，而不是与之竞争
3. **Depth（深度）**：通过层级和动效建立空间感

### 2.2 字体系统

```
字体栈：-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", Roboto, sans-serif

标题层级：
  H1: 28px / 34px line-height / font-weight 700 / tracking -0.021em
  H2: 22px / 28px line-height / font-weight 600 / tracking -0.019em
  H3: 17px / 22px line-height / font-weight 600 / tracking -0.021em
  Body: 13px / 20px line-height / font-weight 400 / tracking -0.006em
  Small: 11px / 16px line-height / font-weight 400 / tracking -0.006em
  Caption: 10px / 13px line-height / font-weight 400 / tracking 0.006em
```

### 2.3 颜色体系

```
// 中性色（灰度）— Apple 风格偏冷
gray-50:  #F5F5F7   (背景)
gray-100: #E8E8ED   (边框、分隔线)
gray-200: #D2D2D7   (禁用态边框)
gray-400: #86868B   (次要文字)
gray-500: #6E6E73   (辅助文字)
gray-700: #1D1D1F   (主要文字)

// 强调色 — Apple 蓝
primary:     #007AFF   (主按钮、链接、激活态)
primary-hover: #0051D5
primary-light: #E8F1FF  (badge 背景)

// 功能色 — 降低饱和度，更柔和
success:     #34C759   (绿色)
warning:     #FF9F0A   (橙色)
error:       #FF3B30   (红色)
info:        #5AC8FA   (蓝色)

// Severity 映射（用于安全场景）
critical:    #FF3B30   (红)
high:        #FF9500   (橙)
medium:      #FFCC00   (黄)
low:         #5AC8FA   (蓝)
info:        #8E8E93   (灰)
```

### 2.4 间距系统

```
Apple 使用 8px 基网格：
  xs:  4px
  sm:  8px
  md:  12px
  lg:  16px
  xl:  24px
  2xl: 32px
  3xl: 48px

卡片内边距：16px-20px
卡片间距：16px-24px
页面边距：24px-32px
```

### 2.5 圆角系统

```
Apple 风格圆角偏保守：
  sm:  6px   (小按钮、badge)
  md:  8px   (卡片、输入框)
  lg:  10px  (大卡片、modal)
  xl:  12px  (特殊容器)
  full: 9999px (avatar、status dot)
```

### 2.6 阴影系统

```
Apple 使用极淡的阴影，几乎不可察觉：
  sm:  0 1px 2px rgba(0,0,0,0.04)
  md:  0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)
  lg:  0 4px 12px rgba(0,0,0,0.08)

原则：阴影用于建立层级，不是为了装饰。
```

### 2.7 玻璃拟态（Glassmorphism）

```
用于导航栏、浮动面板：
  background: rgba(255, 255, 255, 0.72)
  backdrop-filter: blur(20px) saturate(180%)
  border-bottom: 1px solid rgba(0, 0, 0, 0.08)
```

---

## 三、全局样式系统

### 3.1 tailwind.config.js 扩展

```js
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', '"SF Pro Text"', '"Segoe UI"', 'Roboto', 'sans-serif'],
      },
      colors: {
        // Apple 风格中性灰
        apple: {
          bg: '#F5F5F7',
          card: '#FFFFFF',
          border: '#E8E8ED',
          'border-strong': '#D2D2D7',
          'text-primary': '#1D1D1F',
          'text-secondary': '#86868B',
          'text-tertiary': '#6E6E73',
        },
        // 语义色
        accent: {
          DEFAULT: '#007AFF',
          hover: '#0051D5',
          light: '#E8F1FF',
        },
        semantic: {
          success: '#34C759',
          warning: '#FF9F0A',
          error: '#FF3B30',
          info: '#5AC8FA',
        },
        // Severity
        severity: {
          critical: '#FF3B30',
          high: '#FF9500',
          medium: '#FFCC00',
          low: '#5AC8FA',
          info: '#8E8E93',
        },
      },
      borderRadius: {
        'apple-sm': '6px',
        'apple-md': '8px',
        'apple-lg': '10px',
        'apple-xl': '12px',
      },
      boxShadow: {
        'apple-sm': '0 1px 2px rgba(0,0,0,0.04)',
        'apple-md': '0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)',
        'apple-lg': '0 4px 12px rgba(0,0,0,0.08)',
      },
      fontSize: {
        'apple-h1': ['28px', { lineHeight: '34px', letterSpacing: '-0.021em', fontWeight: '700' }],
        'apple-h2': ['22px', { lineHeight: '28px', letterSpacing: '-0.019em', fontWeight: '600' }],
        'apple-h3': ['17px', { lineHeight: '22px', letterSpacing: '-0.021em', fontWeight: '600' }],
        'apple-body': ['13px', { lineHeight: '20px', letterSpacing: '-0.006em', fontWeight: '400' }],
        'apple-small': ['11px', { lineHeight: '16px', letterSpacing: '-0.006em', fontWeight: '400' }],
        'apple-caption': ['10px', { lineHeight: '13px', letterSpacing: '0.006em', fontWeight: '400' }],
      },
    },
  },
  plugins: [],
};
```

### 3.2 index.css 全局样式

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  body {
    @apply bg-apple-bg text-apple-text-primary antialiased;
    font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", Roboto, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }

  /* 自定义滚动条 — macOS 风格 */
  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }
  ::-webkit-scrollbar-track {
    background: transparent;
  }
  ::-webkit-scrollbar-thumb {
    background: rgba(0, 0, 0, 0.15);
    border-radius: 4px;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: rgba(0, 0, 0, 0.25);
  }

  /* 选中文字颜色 */
  ::selection {
    background: rgba(0, 122, 255, 0.2);
  }
}
```

---

## 四、通用组件设计

### 4.1 组件目录结构

```
frontend/src/components/
├── ui/                    # 基础 UI 组件
│   ├── Button.tsx
│   ├── Card.tsx
│   ├── Input.tsx
│   ├── Badge.tsx
│   ├── Modal.tsx
│   ├── Toast.tsx
│   ├── EmptyState.tsx
│   ├── Skeleton.tsx
│   ├── Table.tsx
│   └── Tabs.tsx
├── layout/                # 布局组件
│   ├── Navbar.tsx
│   ├── PageHeader.tsx
│   └── Container.tsx
└── icons/                 # 图标组件（使用 Lucide）
    └── index.tsx
```

### 4.2 组件详细设计

#### Button

```tsx
// 变体：primary / secondary / ghost / danger
// 尺寸：sm / md / lg
// 状态：disabled / loading

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
}

// 样式映射：
// primary: bg-accent text-white hover:bg-accent-hover rounded-apple-md
// secondary: bg-white border border-apple-border hover:bg-apple-bg rounded-apple-md
// ghost: bg-transparent hover:bg-apple-bg rounded-apple-md
// danger: bg-semantic-error text-white hover:opacity-90 rounded-apple-md
```

#### Card

```tsx
// 默认：bg-white rounded-apple-lg shadow-apple-sm border border-apple-border
// hover 态：shadow-apple-md transition-shadow duration-200
// 无 padding 变体（内容自己控制）
```

#### Badge

```tsx
// 变体：default / success / warning / error / info
// 尺寸：sm / md
// 可带圆点指示器

// 样式：
// default: bg-apple-bg text-apple-text-secondary
// success: bg-green-50 text-semantic-success
// warning: bg-orange-50 text-semantic-warning
// error: bg-red-50 text-semantic-error
// info: bg-blue-50 text-semantic-info
```

#### Input

```tsx
// 基础：bg-white border border-apple-border rounded-apple-md px-3 py-2
// focus: ring-2 ring-accent/20 border-accent
// error: border-semantic-error ring-semantic-error/20
// disabled: bg-apple-bg opacity-60
```

#### Modal/Dialog

```tsx
// 遮罩：bg-black/30 backdrop-blur-sm
// 容器：bg-white rounded-apple-xl shadow-apple-lg max-w-lg
// 动画：fade-in + scale-in
// 可拖拽关闭（点击遮罩）
```

#### Toast

```tsx
// 位置：右上角
// 变体：success / error / warning / info
// 自动消失：3-5s
// 动画：slide-in-right + fade-out
// 样式：bg-white/90 backdrop-blur rounded-apple-lg shadow-apple-lg
```

#### EmptyState

```tsx
// 结构：图标 + 标题 + 描述 + 操作按钮
// 图标：Lucide icon，gray-300 颜色
// 标题：text-apple-h3 text-apple-text-primary
// 描述：text-apple-text-secondary
```

#### Skeleton

```tsx
// 基础：bg-apple-border rounded animate-pulse
// 变体：text / avatar / card / table-row
```

#### Table

```tsx
// 表头：bg-transparent border-b border-apple-border text-apple-text-secondary text-apple-small font-medium
// 行：hover:bg-apple-bg/50 transition-colors
// 无垂直边框，仅靠留白和对齐区分列
// 圆角：rounded-apple-lg 外包一层 bg-white
```

---

## 五、页面级重构要点

### 5.1 App.tsx — 导航栏重构

**目标**：macOS 风格顶部导航栏

```
┌─────────────────────────────────────────────┐
│ 🔵 🟡 🟢  Anchor     项目  资产  Findings    │  ← 玻璃拟态导航
├─────────────────────────────────────────────┤
│                                             │
│  页面内容区域                                   │
│                                             │
└─────────────────────────────────────────────┘
```

设计要点：
- 玻璃拟态背景：`rgba(255,255,255,0.72) + backdrop-blur(20px)`
- 底部 1px 分隔线
- Logo 带锚点图标
- 导航项带图标（Lucide）
- 当前页面高亮：accent 颜色底部指示条
- 补全导航项：项目、资产、Findings、报告、运行

### 5.2 ProjectPage — 项目管理

**当前问题：**
- 新建表单直接堆在列表上方，喧宾夺主
- 项目卡片信息密度低，视觉层次差
- 空状态简陋

**重构方案：**
- 页面布局：左侧项目列表（sidebar 风格）+ 右侧详情/操作区
- 或：顶部操作栏 + 卡片网格布局
- 新建项目用 Modal 替代 inline form
- 项目卡片：
  - 左侧色条表示状态（绿=进行中，黄=未开始，灰=无时限）
  - 项目名称 + 组织信息 + 时间窗口
  - hover 显示操作按钮（进入/编辑/删除）
- 空状态：大型图标 + "创建第一个安全测试项目" CTA

### 5.3 TargetPage — 目标与 Scope

**当前问题：**
- 信息堆砌，没有清晰的任务流
- 文件导入区域像草稿纸
- Scope 规则列表简陋

**重构方案：**
- 采用 Tab 布局：目标列表 / Scope 规则 / 导入
- 目标列表：Table 组件替代卡片
- 导入区域：拖拽区域美化（虚线边框 + 图标 + 动画）
- Scope 规则：带彩色标签的列表，可拖拽排序
- 执行计划预览：Progress + 统计卡片

### 5.4 AssetPage — 资产管理

**当前问题：**
- Tab 切换简陋
- 资产类型没有视觉区分
- 端口列表像纯文本

**重构方案：**
- 顶部统计栏：域名数 / IP数 / URL数 / Web端点数
- Tab 组件美化：胶囊式切换，带计数徽章
- 资产卡片：图标 + 类型 badge + 标签云
- WebEndpoint 卡片：预览缩略图占位 + 状态码 + 技术栈标签
- 端口列表：端口范围可视化，状态颜色指示

### 5.5 FindingsPage — Finding 管理

**当前问题：**
- 表格样式简陋
- Severity 颜色过于鲜艳刺眼
- 筛选按钮像 HTML 默认样式
- Detail 弹窗缺失

**重构方案：**
- 表格：Apple 风格 clean table，无垂直边框
- Severity badge：柔和背景色 + 深色文字（不用白字）
- 筛选栏：分段控制器（Segmented Control）替代按钮组
- Detail 弹窗：侧边滑出面板（Slide-over Panel）
- Evidence 展示：折叠面板，HTTP 请求/响应用代码块高亮

### 5.6 ReportsPage — 报告导出

**当前问题：**
- Markdown 预览区域简陋
- 导出按钮样式不一致

**重构方案：**
- 分栏布局：左侧 Finding 列表摘要 / 右侧 Markdown 预览
- Markdown 渲染：白色背景、优雅排版、代码块高亮
- 导出按钮组：主操作突出，次要操作弱化
- 预览模式切换：原始 Markdown / 渲染预览

### 5.7 RunsPage — 运行状态

**当前问题：**
- 工具健康卡片简陋
- 任务列表占位符

**重构方案：**
- 工具健康：网格卡片，带状态指示灯（绿点/红点脉冲动画）
- 任务列表：时间线样式，显示运行状态、耗时、结果数
- 实时日志：可折叠的终端风格输出区域

---

## 六、实施优先级

### 🔴 P0 — 必须先完成（基础框架）

1. **全局样式系统**
   - [ ] 重写 `tailwind.config.js`（Apple 主题 token）
   - [ ] 重写 `index.css`（全局样式、滚动条、选中色）
   - [ ] 安装 Lucide React 图标库

2. **通用组件**
   - [ ] Button 组件
   - [ ] Card 组件
   - [ ] Badge 组件
   - [ ] Input 组件
   - [ ] Toast 通知系统（替换所有 alert）

3. **导航栏**
   - [ ] 玻璃拟态 Navbar
   - [ ] 导航高亮状态
   - [ ] 补全导航项

### 🟡 P1 — 核心页面（高价值）

4. **ProjectPage 重构**
   - [ ] 卡片网格布局
   - [ ] Modal 新建项目
   - [ ] EmptyState 组件
   - [ ] Skeleton 加载态

5. **FindingsPage 重构**
   - [ ] Clean Table 组件
   - [ ] Severity badge 统一
   - [ ] Segmented Control 筛选
   - [ ] Slide-over Detail 面板

6. **AssetPage 重构**
   - [ ] Tabs 组件美化
   - [ ] 统计卡片
   - [ ] 资产卡片视觉优化

### 🟢 P2 — 其他页面和优化

7. **TargetPage 重构**
   - [ ] Tab 布局
   - [ ] 拖拽导入区域美化
   - [ ] Table 替换列表

8. **ReportsPage 重构**
   - [ ] 分栏布局
   - [ ] Markdown 预览美化

9. **RunsPage 重构**
   - [ ] 健康状态卡片
   - [ ] 时间线任务列表

10. **全局优化**
    - [ ] 过渡动画（页面切换、元素出现）
    - [ ] 响应式适配
    - [ ] 暗色模式支持（CSS 变量方案）

---

## 七、依赖安装

```bash
cd frontend
npm install lucide-react
```

---

## 八、验收标准

- [ ] 所有页面无 `alert()`，使用 Toast
- [ ] 所有页面有 Skeleton 加载态
- [ ] 所有页面有 EmptyState 空状态
- [ ] 配色统一使用 design token
- [ ] 圆角统一使用 apple 命名体系
- [ ] 阴影统一使用 apple 命名体系
- [ ] 字体使用系统字体栈
- [ ] 导航栏有玻璃拟态效果
- [ ] 按钮、卡片、Badge 使用统一组件
- [ ] 表格使用统一 Table 组件
- [ ] TypeScript 无编译错误
- [ ] 所有交互有 hover/active 反馈
- [ ] 页面切换有过渡动画

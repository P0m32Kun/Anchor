# CyberOS UI 改造设计文档

## 概述

将前端 UI 从 Vision Glass 风格全面改造为 CyberOS 红队工作台风格，增强科技感和攻击性视觉效果。

## 设计目标

- **视觉风格**：深色科技感，类似黑客/安全工具界面
- **颜色方案**：经典 CyberOS（深蓝背景 + 蓝色发光 + 红色警告）
- **动画效果**：发光脉冲、星空背景、悬停发光
- **字体**：等宽字体增强科技感
- **图标**：线性 SVG 图标

## 颜色系统

### 背景色
- 主背景：`#0b1120`
- 面板背景：`rgba(30, 41, 59, 0.4)`
- 深层背景：`rgba(11, 17, 32, 0.5)`

### 品牌色
- 主色-蓝：`#38bdf8`
- 警告-红：`#f87171`
- 成功-绿：`#4ade80`
- 警告-黄：`#facc15`
- 橙色：`#fb923c`

### 文本色
- 主文本：`#F0F6FC`
- 次文本：`#e2e8f0`
- 三级文本：`#94a3b8`
- 四级文本：`#64748b`

## 发光效果

### glow-blue
```css
border-color: rgba(56, 189, 248, 0.6);
box-shadow: 0 0 20px rgba(56, 189, 248, 0.15), inset 0 0 10px rgba(56, 189, 248, 0.05);
```

### glow-red
```css
border-color: rgba(248, 113, 113, 0.6);
box-shadow: 0 0 20px rgba(248, 113, 113, 0.15), inset 0 0 10px rgba(248, 113, 113, 0.05);
```

### text-glow-blue
```css
text-shadow: 0 0 10px rgba(56, 189, 248, 0.5);
```

### text-glow-red
```css
text-shadow: 0 0 10px rgba(248, 113, 113, 0.5);
```

## 背景效果

### 星空背景
```css
body {
  background-color: #0b1120;
  background-image:
    radial-gradient(circle at 50% 50%, rgba(15, 23, 42, 0) 0%, rgba(11, 17, 32, 1) 100%),
    url('https://www.transparenttextures.com/patterns/stardust.png');
}
```

## 动画效果

### 发光脉冲
```css
@keyframes glow-pulse {
  0%, 100% { box-shadow: 0 0 20px rgba(56, 189, 248, 0.15); }
  50% { box-shadow: 0 0 30px rgba(56, 189, 248, 0.3); }
}
```

### 悬停发光增强
```css
.glow-blue:hover {
  box-shadow: 0 0 30px rgba(56, 189, 248, 0.25), inset 0 0 15px rgba(56, 189, 248, 0.1);
}
```

## 组件设计

### Card 组件
- 基础样式：玻璃面板 + 圆角 12px
- 活跃状态：glow-blue 边框
- 危险状态：glow-red 边框
- 悬停效果：发光增强 + 轻微上移

### Button 组件
- 主按钮：蓝色背景 + 蓝色发光边框
- 次按钮：玻璃面板 + 灰色边框
- 危险按钮：红色渐变 + 红色发光
- 幽灵按钮：透明背景 + 文本色

### Badge 组件
- 半透明背景 + 彩色边框
- 严重程度标签：
  - Critical：红色
  - High：橙色
  - Medium：黄色
  - Low：绿色
  - Info：蓝色

### 状态指示器
- 圆点 + 发光效果
- 运行中：绿色发光
- 等待中：黄色发光
- 失败：红色发光
- 已完成：灰色无发光

### Table 组件
- 玻璃面板容器
- 表头：小写 uppercase + 三级文本
- 行：悬停高亮 + 左侧蓝色边框（可点击时）
- 边框：半透明灰色

### Navbar 组件
- 深色背景 + 毛玻璃效果
- Logo：蓝色发光
- 导航项：active 蓝色高亮 + 下划线
- 搜索框：玻璃面板 + 圆角

### 活动时间线
- 垂直时间线 + 渐变边框
- 节点：蓝色发光圆点
- 卡片：玻璃面板 + 深色背景

## 页面布局

### Dashboard 页面
- **顶部**：4 个统计卡片网格
  - Active Operations（glow-blue）
  - Total Assets（普通）
  - Recent Findings（glow-blue）
  - Key Vulnerabilities（glow-red）
- **中部**：操作按钮组
- **底部**：双栏布局
  - 左侧：活动时间线
  - 右侧：Findings 表格

### 其他页面
- 统一使用玻璃卡片布局
- 表格使用 CyberOS 风格
- 状态标签使用发光效果

## 字体

```css
font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
```

## 图标

使用线性 SVG 图标，风格：
- 细线条（stroke-width: 1.5-2）
- 圆角（stroke-linecap: round, stroke-linejoin: round）
- 蓝色主色，红色警告

## 实现策略

采用一次性全面改造：
1. 更新 CSS 基础样式（颜色、发光、动画）
2. 更新核心组件（Card, Button, Badge, Table）
3. 更新布局组件（Navbar, 时间线）
4. 更新页面（Dashboard, 其他页面）

## 保留功能

- 组件接口不变
- 路由布局不变
- 数据流不变
- 响应式设计不变

## 参考模板

- 文件：`example.htlm`
- 图片：`Gemini_Generated_Image_i8vp5zi8vp5zi8vp.png`
- 风格：CyberOS Red Team Workbench

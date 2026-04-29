# DashboardPage E2E Test Plan

## Test Purpose
验证 DashboardPage 的跨项目统计展示、最近活动列表、待处理 Findings 列表和快速操作按钮功能正常，空状态正确渲染。

## Prerequisites
1. 前端开发服务器已启动 (`npm run dev`)
2. 后端 API 服务已启动
3. 应用可访问于 `http://localhost:1420`

## Test Cases

### TC-1: 首次访问 Dashboard（空状态）
**Setup:** 确保后端没有任何项目数据

| Step | Command | Expected Result |
|------|---------|-----------------|
| 1 | `navigate http://localhost:1420` | 页面加载，显示 Dashboard |
| 2 | `expect text "总项目数"` | 统计卡片区域可见 |
| 3 | `expect text "0"` | 总项目数显示 0 |
| 4 | `expect text "欢迎使用 Dashboard"` | 空状态提示可见 |
| 5 | `click "创建项目"` | 导航到 `/projects` |
| 6 | `expect url "/projects"` | URL 正确变化 |

### TC-2: 有数据时的统计卡片
**Setup:** 后端至少存在 1 个项目、1 个 running run、1 个 pending_review finding、1 个 online worker

| Step | Command | Expected Result |
|------|---------|-----------------|
| 1 | `navigate http://localhost:1420` | Dashboard 加载完成 |
| 2 | `expect text "总项目数"` | 统计卡片可见 |
| 3 | `expect text "活跃扫描"` | 统计卡片可见 |
| 4 | `expect text "待处理 Findings"` | 统计卡片可见 |
| 5 | `expect text "在线 Worker"` | 统计卡片可见 |
| 6 | `expect text "running" or "运行中"` | 活跃扫描数 > 0 时高亮显示 |

### TC-3: 最近活动区域
**Setup:** 后端存在至少 1 个 run

| Step | Command | Expected Result |
|------|---------|-----------------|
| 1 | `navigate http://localhost:1420` | Dashboard 加载完成 |
| 2 | `expect text "最近活动"` | 区域标题可见 |
| 3 | `expect text "查看全部 →"` | 链接可见 |
| 4 | `click first run row` | 导航到 `/runs` |
| 5 | `expect url "/runs"` | URL 正确变化 |

### TC-4: 待处理 Findings 区域
**Setup:** 后端存在至少 1 个 pending_review finding

| Step | Command | Expected Result |
|------|---------|-----------------|
| 1 | `navigate http://localhost:1420` | Dashboard 加载完成 |
| 2 | `expect text "待处理 Findings"` | 区域标题可见 |
| 3 | `expect text "查看全部 →"` | 链接可见 |
| 4 | `expect element .SeverityBadge` | 严重级别标签可见 |
| 5 | `click first finding row` | 导航到 `/findings` |
| 6 | `expect url "/findings"` | URL 正确变化 |

### TC-5: 快速操作按钮
**Setup:** 无特殊要求

| Step | Command | Expected Result |
|------|---------|-----------------|
| 1 | `navigate http://localhost:1420` | Dashboard 加载完成 |
| 2 | `click "+ 创建项目"` | 导航到 `/projects` |
| 3 | `navigate http://localhost:1420` | 返回 Dashboard |
| 4 | `click "导入目标"` | 导航到 `/targets`（无当前项目时）或 `/projects/:id/targets`（有当前项目时） |

### TC-6: 空状态 — 有项目但无 runs
**Setup:** 后端存在项目但无任何 runs

| Step | Command | Expected Result |
|------|---------|-----------------|
| 1 | `navigate http://localhost:1420` | Dashboard 加载完成 |
| 2 | `expect text "暂无扫描活动"` | 最近活动区域显示 EmptyState |
| 3 | `expect text "暂无待处理 Findings"` | 待处理 Findings 区域显示 EmptyState（如无 findings） |

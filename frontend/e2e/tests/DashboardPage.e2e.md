# DashboardPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API on default port (for worker count display)

## Test 1: Page renders with all sections

### Steps

```bash
agent-browser navigate http://localhost:1420
agent-browser screenshot
```

### Expected Results
- [ ] "项目状态" status bar visible
- [ ] 4 stat cards: 当前项目, 在线 Worker, 运行中任务, Scope 状态
- [ ] 3-column grid: 高优先级资产, 待验证 Finding, 最近活动
- [ ] 2-column grid: 失败/部分成功任务, 复测队列

## Test 2: Quick-action buttons are functional

### Steps

```bash
agent-browser navigate http://localhost:1420

# Click "前往创建 →"
agent-browser click "前往创建"
agent-browser screenshot
# Expect: navigated to /projects

agent-browser navigate http://localhost:1420

# Click "导入目标 →"
agent-browser click "导入目标"
agent-browser screenshot
# Expect: navigated to /targets

agent-browser navigate http://localhost:1420

# Click "查看全部 →"
agent-browser click "查看全部"
agent-browser screenshot
# Expect: navigated to /findings
```

### Expected Results
- [ ] All 3 buttons navigate to valid routes
- [ ] No console errors after navigation

## Test 3: Worker count polling

### Steps

```bash
agent-browser navigate http://localhost:1420
# Wait 6+ seconds for polling interval
sleep 6
agent-browser screenshot
```

### Expected Results
- [ ] Worker count displays (number or 0)
- [ ] No errors from failed fetch

# E2E Test Plan — ProjectLayout + Nested Routes

## Test Purpose
验证 ProjectLayout 组件能正确加载项目、提供 projectId context，以及嵌套路由 `/projects/:projectId/*` 能正常访问各子页面。

## Prerequisite Steps
1. 启动后端服务（确保项目 API 可用）
2. 启动前端 dev server：`npm run dev`（端口 1420）
3. 确保数据库中至少有一个项目存在（如 ID 为 `proj_01`）

## Navigation Commands
```
navigate http://localhost:1420/projects
```

## Interaction Commands & Expected Results

### Test 1: 嵌套路由 — Targets 页面
```
click text="proj_01"  # 或通过 URL 直接访问
navigate http://localhost:1420/projects/proj_01/targets
expect text="目标" or text="Targets"
expect element .bg-accent-yellow/10  # 如项目已过期显示警告条
take screenshot
```

### Test 2: 嵌套路由 — Assets 页面
```
navigate http://localhost:1420/projects/proj_01/assets
expect text="Assets" or text="资产"
take screenshot
```

### Test 3: 嵌套路由 — Runs 页面
```
navigate http://localhost:1420/projects/proj_01/runs
expect text="扫描执行" or text="Runs"
take screenshot
```

### Test 4: 嵌套路由 — Findings 页面
```
navigate http://localhost:1420/projects/proj_01/findings
expect text="Findings" or text="安全发现"
take screenshot
```

### Test 5: 嵌套路由 — Reports 页面
```
navigate http://localhost:1420/projects/proj_01/reports
expect text="Reports" or text="报告"
take screenshot
```

### Test 6: 项目不存在时显示错误并跳转
```
navigate http://localhost:1420/projects/nonexistent/targets
expect text="项目不存在"
wait 2500
expect url "/projects"
take screenshot
```

### Test 7: Legacy 路由仍然可用
```
navigate http://localhost:1420/projects/proj_01/assets  # 注意 :id 形式
expect text="Assets" or text="资产"
take screenshot
```

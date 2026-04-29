# ReportsPage E2E Test Plan

## Test Purpose
Verify the Markdown preview, findings summary, outline navigation, and preview/raw toggle on the ReportsPage.

## Prerequisites
1. Backend API is running and accessible at `http://localhost:8080`
2. At least one project exists with findings in `confirmed` or `accepted_risk` status
3. The `/projects/:projectId/reports/export.md` endpoint returns a valid Markdown report

## Test Steps

### 1. Navigation
```
navigate http://localhost:1420/projects/:projectId/reports
```

### 2. Findings Summary Display
```
expect text "安全评估报告"
expect text "严重"
expect text "高危"
expect text "中危"
expect text "低危"
expect text "信息"
```

### 3. Report Outline
```
expect text "报告大纲"
expect text "Findings 列表"
click text "Findings 列表" button[title^="查看"]:first
```

### 4. Markdown Preview - Rendered Mode
```
click button "Markdown 预览"
wait 1000
expect text "Markdown 报告预览"
expect element div.prose
```

### 5. Preview/Raw Toggle
```
click button "原始"
expect element pre
click button "预览"
expect element div.prose
```

### 6. Close Preview
```
click button[aria-label="关闭预览"]
expect no text "Markdown 报告预览"
```

### 7. Export Buttons
```
expect button "导出 Markdown"
expect button "导出 JSON"
```

## Expected Results
- Findings summary cards show correct counts per severity level
- Report outline lists all findings with severity badges
- Clicking an outline item smooth-scrolls to the corresponding finding card
- Markdown preview renders HTML with proper typography and no raw script injection
- Raw mode shows the original Markdown source in a `<pre>` block
- Toggle between rendered and raw modes works instantly
- Close button dismisses the preview panel

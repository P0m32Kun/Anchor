# ReportsPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API available

## Test 1: Page loads with reports

### Steps

```bash
agent-browser navigate http://localhost:1420/reports
agent-browser screenshot
```

### Expected Results
- [ ] Page renders without errors
- [ ] Reports list or empty state visible
- [ ] "安全评估报告" title visible

## Test 2: Export buttons disabled when no findings

### Steps

```bash
agent-browser navigate http://localhost:1420/reports
agent-browser snapshot

# When no confirmed/accepted findings exist
agent-browser click @e<N> "导出 Markdown"
```

### Expected Results
- [ ] Toast warning "无 findings 可导出" appears
- [ ] No network request to export endpoint fired

## Test 3: Markdown export success (browser fallback)

### Steps

```bash
agent-browser navigate http://localhost:1420/reports
agent-browser snapshot

# When findings exist
agent-browser click @e<N> "导出 Markdown"
agent-browser screenshot
```

### Expected Results
- [ ] Button shows "导出中..." during request
- [ ] Toast success "下载已启动" appears
- [ ] Blob download triggered (check Network tab for /reports/export.md)

## Test 4: JSON export success (browser fallback)

### Steps

```bash
agent-browser navigate http://localhost:1420/reports
agent-browser snapshot

# When findings exist
agent-browser click @e<N> "导出 JSON"
agent-browser screenshot
```

### Expected Results
- [ ] Button shows "导出中..." during request
- [ ] Toast success "下载已启动" appears
- [ ] Blob download triggered (check Network tab for /reports/export.json)

## Test 5: Markdown preview

### Steps

```bash
agent-browser navigate http://localhost:1420/reports
agent-browser snapshot

agent-browser click @e<N> "Markdown 预览"
agent-browser screenshot
```

### Expected Results
- [ ] Preview panel appears with "Markdown 报告预览" header
- [ ] Close button (✕) visible
- [ ] Raw markdown content rendered in <pre> block

## Test 6: Error handling on export failure

### Steps

```bash
# Stop backend API or block /reports/export.md
agent-browser navigate http://localhost:1420/reports
agent-browser snapshot

agent-browser click @e<N> "导出 Markdown"
agent-browser screenshot
```

### Expected Results
- [ ] Toast error "导出失败：" appears
- [ ] Error banner visible on page
- [ ] Button returns to idle state (not stuck in "导出中...")

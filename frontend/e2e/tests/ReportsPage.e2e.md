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

## Test 2: Report generation

### Steps

```bash
agent-browser navigate http://localhost:1420/reports
agent-browser snapshot

# Verify generate/export buttons
```

### Expected Results
- [ ] Generate button is functional
- [ ] Export/download triggers correctly

# RunsPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API available

## Test 1: Page loads with run history

### Steps

```bash
agent-browser navigate http://localhost:1420/runs
agent-browser screenshot
```

### Expected Results
- [ ] Page renders without errors
- [ ] Run list or empty state visible

## Test 2: Run controls

### Steps

```bash
agent-browser navigate http://localhost:1420/runs
agent-browser snapshot

# Verify start/stop/retry buttons
```

### Expected Results
- [ ] All control buttons are functional
- [ ] Status badges display correctly

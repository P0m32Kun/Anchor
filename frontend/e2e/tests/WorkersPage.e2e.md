# WorkersPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API available

## Test 1: Page loads with worker list

### Steps

```bash
agent-browser navigate http://localhost:1420/workers
agent-browser screenshot
```

### Expected Results
- [ ] Page renders without errors
- [ ] Worker list with status visible

## Test 2: Worker actions

### Steps

```bash
agent-browser navigate http://localhost:1420/workers
agent-browser snapshot

# Verify add/remove/toggle buttons
```

### Expected Results
- [ ] All worker management buttons are functional
- [ ] Status updates reflect backend state

# FindingsPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API available

## Test 1: Page loads with findings list

### Steps

```bash
agent-browser navigate http://localhost:1420/findings
agent-browser screenshot
```

### Expected Results
- [ ] Page renders without errors
- [ ] Findings list or empty state visible

## Test 2: Finding actions

### Steps

```bash
agent-browser navigate http://localhost:1420/findings
agent-browser snapshot

# Test verify, dismiss, export actions
```

### Expected Results
- [ ] All action buttons have valid handlers
- [ ] Navigation from findings to detail works (if applicable)

# TargetPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API available

## Test 1: Page loads and shows targets

### Steps

```bash
agent-browser navigate http://localhost:1420/targets
agent-browser screenshot
```

### Expected Results
- [ ] Page renders without errors
- [ ] Target list or empty state visible

## Test 2: Import targets

### Steps

```bash
agent-browser navigate http://localhost:1420/targets

# Verify import button exists and is clickable
agent-browser snapshot
# agent-browser click "导入"
```

### Expected Results
- [ ] Import button is functional (not disabled without reason)
- [ ] Import flow opens or navigates correctly

## Known Issues
- ⬜ Verify all target action buttons have valid handlers

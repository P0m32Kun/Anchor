# AssetPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API available

## Test 1: Page loads with asset list

### Steps

```bash
agent-browser navigate http://localhost:1420/assets
agent-browser screenshot
```

### Expected Results
- [ ] Page renders without errors
- [ ] Asset list table/grid visible

## Test 2: Asset actions

### Steps

```bash
agent-browser navigate http://localhost:1420/assets
agent-browser snapshot

# Test any action buttons (view details, delete, etc.)
```

### Expected Results
- [ ] All action buttons have valid onClick handlers
- [ ] No dead clicks

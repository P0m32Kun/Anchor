# App E2E Tests — Routing & Navigation

Tests that all navbar links route to valid pages and no 404s occur.

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)

## Test 1: Navbar links are all valid

### Steps

```bash
agent-browser navigate http://localhost:1420

# Verify dashboard loads
agent-browser screenshot

# Click each navbar link and verify the page loads
agent-browser click "Projects"
agent-browser screenshot

agent-browser click "Targets"
agent-browser screenshot

agent-browser click "Assets"
agent-browser screenshot

agent-browser click "Runs"
agent-browser screenshot

agent-browser click "Findings"
agent-browser screenshot

agent-browser click "Reports"
agent-browser screenshot

agent-browser click "Workers"
agent-browser screenshot

agent-browser click "Settings"
agent-browser screenshot

# Return to Dashboard
agent-browser click "Dashboard"
agent-browser screenshot
```

### Expected Results
- [ ] All 9 navbar items navigate successfully
- [ ] No blank pages or 404 errors
- [ ] Each page shows distinct content (not identical)
- [ ] Active nav indicator follows current page

## Test 2: Dashboard quick-action buttons navigate correctly

### Steps

```bash
agent-browser navigate http://localhost:1420

# "前往创建 →" should go to /projects
agent-browser click "前往创建"
agent-browser screenshot
# Verify URL contains /projects

agent-browser navigate http://localhost:1420

# "导入目标 →" should go to /targets
agent-browser click "导入目标"
agent-browser screenshot
# Verify URL contains /targets

agent-browser navigate http://localhost:1420

# "查看全部 →" should go to /findings
agent-browser click "查看全部"
agent-browser screenshot
# Verify URL contains /findings
```

### Expected Results
- [ ] All 3 dashboard quick-action buttons navigate to correct routes
- [ ] Target routes exist in App.tsx `<Routes>`

## Test 3: Legacy routes handling

### Steps

```bash
# Legacy project detail route
agent-browser navigate http://localhost:1420/projects/123
agent-browser screenshot

# Legacy nested routes
agent-browser navigate http://localhost:1420/projects/123/assets
agent-browser screenshot

agent-browser navigate http://localhost:1420/projects/123/findings
agent-browser screenshot

agent-browser navigate http://localhost:1420/projects/123/reports
agent-browser screenshot
```

### Expected Results
- [ ] Legacy routes render without crashing
- [ ] Pages handle missing params gracefully

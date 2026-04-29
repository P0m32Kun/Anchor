# E2E Test Template

Copy this file when creating tests for a new page.

## Test: <Feature Name>

### Prerequisites
- Dev server running on http://localhost:1420
- Backend API available (if needed)

### Steps

```bash
# 1. Navigate to the page
agent-browser navigate http://localhost:1420/<path>

# 2. Verify page loads (screenshot for visual validation)
agent-browser screenshot

# 3. Identify interactive elements
agent-browser snapshot

# 4. Interact with key elements
# agent-browser click @e<N>
# agent-browser type @e<N> "test input"

# 5. Verify result
# agent-browser screenshot
# agent-browser expect @e<N> "expected text"
```

### Expected Results
- [ ] Page renders without errors
- [ ] All buttons/links are clickable (not dead)
- [ ] Navigation to other pages works
- [ ] No console errors

### Known Issues / TODO
- List any disabled buttons or incomplete features here

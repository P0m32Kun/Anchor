# ProjectPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)
- Backend API available

## Test 1: Project list loads

### Steps

```bash
agent-browser navigate http://localhost:1420/projects
agent-browser screenshot
```

### Expected Results
- [ ] Page title "项目管理" visible
- [ ] New project form visible with input fields
- [ ] Project list renders (empty or with items)

## Test 2: Create project form

### Steps

```bash
agent-browser navigate http://localhost:1420/projects

# Fill required field
agent-browser type "项目名称" "E2E Test Project"

# Optional fields
agent-browser type "组织/客户" "Test Org"
agent-browser type "目的/描述" "E2E test"

# Submit
agent-browser click "创建"
agent-browser screenshot
```

### Expected Results
- [ ] Form submits without error
- [ ] New project appears in list
- [ ] Input fields clear after creation

## Test 3: Project actions

### Steps

```bash
agent-browser navigate http://localhost:1420/projects

# If projects exist, test delete flow
# Click delete button on a project
# agent-browser click "删除"
# Confirm delete
# agent-browser screenshot
```

### Expected Results
- [ ] Delete confirmation appears
- [ ] Project removed from list after confirm

## Known Issues
- ⬜ Delete flow requires confirmation modal verification

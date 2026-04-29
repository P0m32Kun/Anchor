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
- [ ] Worker list or empty state visible
- [ ] Title "Workers" and subtitle visible

## Test 2: Empty state guidance

### Steps

```bash
agent-browser navigate http://localhost:1420/workers
agent-browser snapshot

# When no workers registered
agent-browser expect @e<N> "暂无 Worker"
```

### Expected Results
- [ ] EmptyState component renders with title "暂无 Worker"
- [ ] Description text explains Docker deployment

## Test 3: Polling without request storm

### Steps

```bash
agent-browser navigate http://localhost:1420/workers
agent-browser snapshot

# Wait for multiple poll cycles (15s)
# Open DevTools Network tab and verify no overlapping /workers requests
```

### Expected Results
- [ ] /workers requests occur every ~5 seconds
- [ ] No overlapping pending requests (previous aborted before new one)

## Test 4: Offline worker warning

### Steps

```bash
agent-browser navigate http://localhost:1420/workers
agent-browser snapshot

# When backend has offline workers
agent-browser expect @e<N> "检测到"
agent-browser expect @e<N> "个 Worker 离线"
```

### Expected Results
- [ ] Amber warning banner visible when offline workers exist
- [ ] Online/busy workers still listed below warning

## Test 5: Error state

### Steps

```bash
# Stop backend API
agent-browser navigate http://localhost:1420/workers
agent-browser snapshot
```

### Expected Results
- [ ] Error message "连接失败，请检查服务是否运行" displayed
- [ ] No uncaught exceptions in console

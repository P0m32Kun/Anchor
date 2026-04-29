# RunsPage E2E Test Plan — SSE Real-time Updates

## Test Purpose
Verify that RunsPage correctly uses SSE for real-time task/run updates and gracefully falls back to polling when SSE is unavailable.

## Prerequisite Steps
1. Backend is running with SSE endpoint `GET /events` available
2. At least one project exists with at least one run
3. Frontend dev server is running at `http://localhost:1420`

## Navigation Commands
```
navigate http://localhost:1420/runs
```

## Interaction Commands & Expected Results

### Test 1: SSE Connection Indicator
```
click "扫描执行"
expect selector ".text-brand-success" to contain "实时连接"
```
**Expected**: Green "实时连接" indicator with pulsing dot is visible when SSE is connected.

### Test 2: Run Status Updates via SSE
```
navigate http://localhost:1420/runs
# Trigger a backend SSE event (e.g., start a scan or manually emit task_update)
# Wait for UI to update
wait 3000
screenshot runs-after-sse-update
```
**Expected**: Run status badges update automatically without page refresh.

### Test 3: Task Details Refresh on SSE Event
```
navigate http://localhost:1420/runs
click element matching selector "[data-testid='run-row']" occurrence 1
wait 1000
# Trigger SSE task_update for selected run
wait 3000
screenshot tasks-after-sse
```
**Expected**: Task list in the detail panel refreshes automatically when a relevant SSE event arrives.

### Test 4: Polling Fallback when SSE Unavailable
```
navigate http://localhost:1420/runs
# Block SSE endpoint (e.g., via devtools network blocking or backend stop)
# Verify fallback to polling
wait 6000
expect selector ".text-accent-yellow" to contain "轮询中"
```
**Expected**: Yellow "轮询中" indicator appears after SSE fails; runs list continues to update every 5s.

### Test 5: Page Visibility Pause/Resume
```
navigate http://localhost:1420/runs
# Simulate tab hidden (this may require evaluating JS in browser)
execute "document.dispatchEvent(new Event('visibilitychange'))"
wait 1000
# Simulate tab visible
execute "document.dispatchEvent(new Event('visibilitychange'))"
wait 1000
expect selector ".text-brand-success" to contain "实时连接"
```
**Expected**: Connection reconnects when tab becomes visible again.

## Notes
- SSE requires backend `GET /events` endpoint to be active.
- The `project_id` query param is appended automatically when a project is selected.

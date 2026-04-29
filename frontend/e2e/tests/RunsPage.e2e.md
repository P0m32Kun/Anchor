# RunsPage E2E Test Plan

## Test Purpose
Verify RunsPage reliability features: duplicate submission prevention, cancel confirmation, SSE real-time updates with polling fallback, error handling with Toast, and empty state guidance.

## Prerequisites
1. Backend server running on `http://localhost:8080`
2. Frontend dev server running on `http://localhost:1420`
3. At least one project exists with at least one tool template

---

## Test 1: Empty State Guided Workflow

**Navigation:**
```
navigate http://localhost:1420/projects/:projectId/runs
```

**Steps:**
1. Ensure the selected project has no scan runs
2. Verify empty state displays with 3-step guide: 创建项目 → 导入目标 → 启动扫描

**Interactions:**
```
expect text "暂无扫描任务"
expect text "按照以下步骤开始你的第一次扫描"
expect text "创建项目"
expect text "导入目标"
expect text "启动扫描"
```

**Expected Result:**
- Empty state shows with lightning bolt icon
- Three step buttons are visible and clickable
- "启动扫描" button opens the create run modal

---

## Test 2: Create Run with Duplicate Submission Prevention

**Navigation:**
```
navigate http://localhost:1420/projects/:projectId/runs
```

**Steps:**
1. Click "新建扫描" button
2. Select a tool template
3. Enter scan name
4. Click "开始扫描" button
5. Attempt to click "开始扫描" again while "创建中..." is shown

**Interactions:**
```
click "新建扫描"
click text matching first template name
type "测试扫描" into input[placeholder="例如：外网初筛"]
click "开始扫描"
# Button should show "创建中..." and be disabled
expect button "创建中..." disabled
# Attempt second click - should be ignored
try click "创建中..."
```

**Expected Result:**
- Modal closes after successful creation
- Toast message "扫描任务已启动" appears
- Run list refreshes automatically with new run in "pending" or "running" status
- Button is disabled during creation, preventing duplicate submission

---

## Test 3: SSE Connection Status Indicator

**Navigation:**
```
navigate http://localhost:1420/projects/:projectId/runs
```

**Steps:**
1. Load the page with backend running
2. Observe connection status indicator
3. Stop backend server
4. Wait for SSE to disconnect and polling to take over

**Interactions:**
```
expect text "实时连接"
expect element .animate-ping
# After stopping backend:
# Wait for status change
sleep 5
expect text "轮询中"
```

**Expected Result:**
- Initial state shows "实时连接" with green animated dot
- After SSE failure, indicator changes to "轮询中" with yellow dot
- Run list continues to update via polling fallback

---

## Test 4: Cancel Run with Confirm Dialog

**Navigation:**
```
navigate http://localhost:1420/projects/:projectId/runs
```

**Prerequisites:**
- A run in "pending" or "running" status exists

**Steps:**
1. Click the cancel (×) icon next to a running/pending run
2. Verify ConfirmDialog appears
3. Click "返回" to cancel the dialog
4. Click the cancel icon again
5. Click "确认取消" to confirm

**Interactions:**
```
click svg near text "运行中" or "待启动"
expect text "确认取消扫描？"
expect text "即将取消扫描"
click "返回"
expect text "确认取消扫描？" missing
click svg near text "运行中" or "待启动"
click "确认取消"
```

**Expected Result:**
- ConfirmDialog opens with run name in description
- Cancel button returns to list without changes
- Confirm cancellation shows "扫描任务已取消" toast
- Run status changes to "已取消"
- Task details update if the run was selected

---

## Test 5: Error Handling on Create/Cancel Failure

**Navigation:**
```
navigate http://localhost:1420/projects/:projectId/runs
```

**Steps:**
1. Create a run without selecting a template (button should be disabled)
2. Verify button is disabled and shows why
3. Simulate network error during cancel (e.g., stop backend mid-request)

**Interactions:**
```
click "新建扫描"
# Try to submit without selecting template
expect button "开始扫描" disabled
# Select template and create
type "错误测试" into input[placeholder="例如：外网初筛"]
click first template
# Stop backend before clicking create
# ... or verify error toast appears on network failure
```

**Expected Result:**
- "开始扫描" button is disabled when no template selected
- Error toast appears with meaningful message on failure
- Loading states are properly reset after error

---

## Test 6: Run List Auto-Refresh

**Navigation:**
```
navigate http://localhost:1420/projects/:projectId/runs
```

**Steps:**
1. Create a new run
2. Observe that the run appears in the list without manual refresh
3. Wait for SSE or polling to update run status

**Interactions:**
```
click "新建扫描"
click first template
click "开始扫描"
# Run should appear in list automatically
expect text "未命名扫描" or typed name
```

**Expected Result:**
- New run appears in list automatically after creation
- Status updates propagate via SSE when connected
- Status updates propagate via polling when SSE is down

---

## Test 7: Task Details Expansion

**Navigation:**
```
navigate http://localhost:1420/projects/:projectId/runs
```

**Prerequisites:**
- At least one run exists

**Steps:**
1. Click on a run name to expand task details
2. Verify task list loads
3. Click on another run

**Interactions:**
```
click text of first run name
expect text "任务详情"
expect text "加载中..." or task tool names
```

**Expected Result:**
- Run row becomes highlighted when selected
- Task details section appears below run list
- Tasks show tool name, ID suffix, and status badge
- Loading state shown while fetching tasks

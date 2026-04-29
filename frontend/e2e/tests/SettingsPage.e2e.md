# SettingsPage E2E Tests

## Prerequisites
- Dev server: `npm run dev` (http://localhost:1420)

## Test 1: Page renders with all settings sections

### Steps

```bash
agent-browser navigate http://localhost:1420/settings
agent-browser screenshot
```

### Expected Results
- [ ] Page title "Settings" visible
- [ ] "Server 地址" section with input + buttons
- [ ] "扫描配置" section with port range dropdown
- [ ] "本地 Worker 自动启动" toggle (Tauri mode only)
- [ ] "数据目录" info (Tauri mode only)
- [ ] "版本" info visible

## Test 2: Server URL save/reset

### Steps

```bash
agent-browser navigate http://localhost:1420/settings

# Type new URL
agent-browser type "Server 地址" "http://localhost:9999"

# Click save
agent-browser click "保存并刷新"
agent-browser screenshot

# Expect "已保存 ✓" text appears
# Page should reload

# Click reset
agent-browser click "重置"
agent-browser screenshot
```

### Expected Results
- [ ] Save button works and shows "已保存 ✓"
- [ ] Page reloads after save
- [ ] Reset restores default URL

## Test 3: Port range configuration

### Steps

```bash
agent-browser navigate http://localhost:1420/settings

# Change dropdown selection
agent-browser click "端口范围"
agent-browser click "Top 1000 常用端口"
agent-browser screenshot

# Select custom
agent-browser click "端口范围"
agent-browser click "自定义"
agent-browser screenshot

# Custom input should appear
agent-browser type "自定义端口" "80,443,8080"
agent-browser screenshot
```

### Expected Results
- [ ] Dropdown changes selection
- [ ] Custom input appears when "自定义" selected
- [ ] Input accepts text

## Test 4: DISABLED buttons must be documented

### Steps

```bash
agent-browser navigate http://localhost:1420/settings
agent-browser snapshot
```

### Expected Results
- [ ] "保存扫描配置" button is DISABLED
  - Reason: Requires current project ID to call API
  - TODO: Wire to actual API when project context is available
- [ ] "本地 Worker 自动启动" toggle is static UI only
  - Reason: No state binding implemented
  - TODO: Connect to Tauri store or config

### CRITICAL: Any `disabled` button without a clear explanation is a BUG

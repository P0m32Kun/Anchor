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
- [ ] "API Token" section with input + show/hide toggle
- [ ] "本地 Worker 自动启动" toggle (Tauri mode only)
- [ ] "数据目录" info (Tauri mode only)
- [ ] "版本" info visible

> 端口扫描配置已下沉到 ScanModal 内部(-tp 预设 / -p 自定义),不再出现在 Settings 页;
> ScanModal 端口 UI 的 E2E 覆盖见 `qa-regression.spec.ts`、`full-flow.spec.ts`、
> `high-risk-pipeline.spec.ts`、`internal-scan-live.spec.ts`。

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

## Test 3: DISABLED buttons must be documented

### Steps

```bash
agent-browser navigate http://localhost:1420/settings
agent-browser snapshot
```

### Expected Results
- [ ] "本地 Worker 自动启动" toggle is static UI only
  - Reason: No state binding implemented
  - TODO: Connect to Tauri store or config

### CRITICAL: Any `disabled` button without a clear explanation is a BUG

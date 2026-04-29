# Error Paths — 异常路径端到端测试

覆盖系统在异常和边缘场景下的行为表现，验证用户体验的健壮性。

## 测试目的
验证以下 5 个异常路径的正确处理：
1. 后端未运行时页面的引导与提示
2. 访问不存在的项目时的 404 处理与跳转
3. 新创建项目的空数据页面引导
4. 网络断开与恢复时的 Toast 提示和自动恢复
5. 权限错误（403）的反馈处理

## 前置条件
- 前端 dev server 运行在 `http://localhost:1420`
- 后端 API 服务默认运行在 `http://localhost:8080`
- 浏览器会话干净（无残留状态干扰）
- 至少已存在一个正常项目（用于部分路径前置准备）

---

## 路径 1：后端未运行

### 操作

```bash
# 1.1 确保后端停止（在另一个终端执行）
# pkill -f "backend" 或停止对应服务

# 1.2 刷新 Dashboard 页面
agent-browser navigate http://localhost:1420/
agent-browser wait --load networkidle

# 1.3 截图验证页面状态
agent-browser screenshot error1-backend-down.png

# 1.4 验证错误横幅可见
agent-browser expect "加载数据失败"
agent-browser expect "重试"

# 1.5 验证 Toast 通知
agent-browser expect "网络错误"
agent-browser expect "网络连接失败，请检查后端服务是否运行"
```

### 预期结果
- [ ] Dashboard 页面加载完成，不白屏
- [ ] 显示红色错误横幅：`加载数据失败：网络连接失败，请检查后端服务是否运行`
- [ ] 错误横幅中包含"重试"按钮
- [ ] 顶部弹出 Toast 通知：`网络错误：网络连接失败，请检查后端服务是否运行`
- [ ] 统计卡片区域显示骨架屏或占位状态，不崩溃
- [ ] 控制台无未捕获的 JavaScript 错误

---

## 路径 2：404 项目

### 操作

```bash
# 2.1 确保后端已启动

# 2.2 直接访问一个不存在的项目路径
agent-browser navigate http://localhost:1420/projects/nonexistent-uuid-1234/targets
agent-browser wait --load networkidle

# 2.3 截图验证
agent-browser screenshot error2-project-not-found.png

# 2.4 验证 EmptyState 内容
agent-browser expect "项目不存在"
agent-browser expect "返回项目列表"

# 2.5 验证 Toast 提示
agent-browser expect "项目不存在或已被删除"

# 2.6 等待自动跳转（2 秒后）
agent-browser wait --load networkidle
agent-browser screenshot error2-redirected.png

# 2.7 验证已跳转到项目列表
agent-browser expect "项目管理"
```

### 预期结果
- [ ] 页面不白屏，正常渲染
- [ ] 显示 EmptyState 组件，标题为 `项目不存在`
- [ ] EmptyState 中包含 `返回项目列表` 按钮
- [ ] 顶部弹出 Toast：`项目不存在或已被删除`
- [ ] 约 2 秒后自动跳转至 `/projects` 项目列表页
- [ ] 跳转后页面标题 `项目管理` 可见
- [ ] 控制台无未捕获的 JavaScript 错误

---

## 路径 3：空数据页面

### 操作

```bash
# 3.1 确保后端已启动

# 3.2 导航到项目列表并创建一个新项目
agent-browser navigate http://localhost:1420/projects
agent-browser wait --text "项目管理"

# 3.3 填写并提交新项目
agent-browser fill "项目名称 *" "Empty Data Test Project"
agent-browser click "创建项目"
agent-browser wait --load networkidle

# 3.4 获取刚创建的项目 ID（从 URL 或页面元素中提取）
# 此处假设通过页面点击或 URL 获取到 projectId
# 实际执行时需替换 __PROJECT_ID__

# 3.5 访问 Targets 页面（空目标）
agent-browser navigate http://localhost:1420/projects/__PROJECT_ID__/targets
agent-browser wait --load networkidle
agent-browser screenshot error3-empty-targets.png
agent-browser expect "暂无目标"

# 3.6 访问 Findings 页面（空 findings）
agent-browser navigate http://localhost:1420/projects/__PROJECT_ID__/findings
agent-browser wait --load networkidle
agent-browser screenshot error3-empty-findings.png
agent-browser expect "暂无 Finding"
agent-browser expect "当前项目还没有任何安全发现，请先运行扫描任务"

# 3.7 访问 Runs 页面（空扫描任务）
agent-browser navigate http://localhost:1420/projects/__PROJECT_ID__/runs
agent-browser wait --load networkidle
agent-browser screenshot error3-empty-runs.png
agent-browser expect "暂无扫描任务"
agent-browser expect "按照以下步骤开始你的第一次扫描"

# 3.8 验证 Runs 页面的引导按钮
agent-browser expect "创建项目"
agent-browser expect "导入目标"
agent-browser expect "启动扫描"
```

### 预期结果
- [ ] Targets 页面：Table 区域显示空状态图标和文字 `暂无目标`
- [ ] Findings 页面：显示 EmptyState，标题 `暂无 Finding`，描述包含 `当前项目还没有任何安全发现，请先运行扫描任务`
- [ ] Runs 页面：显示 EmptyState，标题 `暂无扫描任务`，描述包含 `按照以下步骤开始你的第一次扫描`
- [ ] Runs 页面底部显示 3 步引导按钮：`创建项目` → `导入目标` → `启动扫描`
- [ ] 所有空数据页面均无加载骨架屏残留
- [ ] 控制台无未捕获的 JavaScript 错误

---

## 路径 4：网络断开恢复

### 操作

```bash
# 4.1 确保后端已启动，先加载 Dashboard
agent-browser navigate http://localhost:1420/
agent-browser wait --text "总项目数"
agent-browser screenshot error4-before-disconnect.png

# 4.2 停止后端服务（在另一个终端执行）
# pkill -f "backend" 或停止对应服务
# 等待服务完全停止
sleep 3

# 4.3 在前端触发一个网络请求（例如导航到 Projects）
agent-browser navigate http://localhost:1420/projects
agent-browser wait --load networkidle
agent-browser screenshot error4-after-disconnect.png

# 4.4 验证 Toast 显示网络错误
agent-browser expect "网络错误"
agent-browser expect "网络连接失败，请检查后端服务是否运行"

# 4.5 重新启动后端服务（在另一个终端执行）
# ./backend 或对应启动命令
# 等待服务完全就绪
sleep 5

# 4.6 触发页面重新请求（点击重试或刷新）
agent-browser click "重试"
agent-browser wait --load networkidle
agent-browser screenshot error4-after-recovery.png

# 4.7 验证页面恢复正常
agent-browser expect "项目管理"
```

### 预期结果
- [ ] 后端停止后，页面不白屏、不崩溃
- [ ] 顶部弹出 Toast：`网络错误：网络连接失败，请检查后端服务是否运行`
- [ ] 页面显示错误状态或重试按钮
- [ ] 后端重启后，点击"重试"或自动轮询后页面恢复正常数据展示
- [ ] 恢复后无残留的错误提示遮挡正常内容
- [ ] 控制台无未捕获的 JavaScript 错误

---

## 路径 5：权限错误（403）

### 操作

```bash
# 5.1 确保后端已启动且前端会话已建立
agent-browser navigate http://localhost:1420/
agent-browser wait --load networkidle

# 5.2 尝试访问一个需要特殊权限的资源
# 注：如果后端有模拟 403 的接口或特定资源，直接访问该端点
# 以下以前端页面触发 403 的场景为例（如尝试操作无权限的项目）
agent-browser navigate http://localhost:1420/projects
agent-browser wait --text "项目管理"

# 5.3 如果后端支持，通过 API 直接触发一个会返回 403 的操作
# 例如访问受限接口：
# agent-browser navigate http://localhost:8080/api/v1/admin/restricted
# 但前端页面应通过正常交互触发 403

# 若应用有 Admin 专用路由或受限资源，直接访问：
# agent-browser navigate http://localhost:1420/admin/restricted
# agent-browser wait --load networkidle

# 5.4 截图验证 Toast 提示
agent-browser screenshot error5-forbidden.png

# 5.5 验证 Toast 内容（403 被归类为 HTTP_4xx）
agent-browser expect "请求错误"
```

### 预期结果
- [ ] 当后端返回 403 时，页面不白屏
- [ ] 顶部弹出 Toast，标题为 `请求错误`
- [ ] Toast 内容包含后端返回的具体错误信息（如 `403: Forbidden` 或自定义消息）
- [ ] 若当前页面有数据，保留已有数据不丢失
- [ ] 控制台无未捕获的 JavaScript 错误

### 备注
- 本路径依赖后端实际返回 403 的场景。如果当前版本后端暂无权限控制，可在后端增加一个测试用的 `GET /api/v1/test-forbidden` 接口返回 403，或手动拦截请求模拟。
- 若应用后续增加专门的 403 页面或权限提示组件，需同步更新本测试用例。

---

## 通用验收标准

- [ ] 全部 5 个异常路径均有独立的操作步骤和截图验证
- [ ] 每个路径的"预期结果"复选框均可明确判定通过/失败
- [ ] 所有路径下控制台均无任何未捕获的 JavaScript 异常
- [ ] 错误提示文案使用中文，语义清晰
- [ ] 网络异常恢复后，页面数据能正确刷新，无旧错误状态残留

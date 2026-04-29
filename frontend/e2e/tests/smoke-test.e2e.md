# Smoke Test — 主流程端到端测试

覆盖从项目创建到报告导出的完整主流程，验证核心用户路径可用性。

## 测试目的
验证用户可以顺利完成以下 5 个主流程步骤：
1. 创建项目
2. 导入目标（含 Scope 确认）
3. 启动扫描
4. 查看 Findings
5. 导出报告

## 前置条件
- 前端 dev server 运行在 `http://localhost:1420`
- 后端 API 服务可用（默认 `http://localhost:8080`）
- 数据库已初始化，至少有一个工具模板已注册（供步骤 3 选择）
- 浏览器会话干净（无残留状态干扰）

---

## 步骤 1：创建项目

### 操作

```bash
# 1.1 导航到项目列表页
agent-browser navigate http://localhost:1420/projects
agent-browser wait --text "项目管理"

# 1.2 填写项目表单
agent-browser fill "项目名称 *" "E2E Smoke Test Project"
agent-browser fill "组织/客户" "Smoke Test Org"
agent-browser fill "目的/描述" "End-to-end smoke test"

# 1.3 提交表单
agent-browser click "创建项目"
agent-browser wait --load networkidle

# 1.4 截图验证
agent-browser screenshot step1-project-created.png
```

### 预期结果
- [ ] 页面标题 "项目管理" 可见
- [ ] 表单提交成功，无错误提示
- [ ] 新项目 "E2E Smoke Test Project" 出现在项目列表中
- [ ] 表单输入框已清空（创建后重置状态）

---

## 步骤 2：导入目标

### 操作

```bash
# 2.1 从项目列表点击进入目标管理（点击刚创建的项目）
agent-browser click "E2E Smoke Test Project"
agent-browser wait --url "**/projects/**"

# 2.2 导航到目标管理页（如未自动跳转）
agent-browser navigate http://localhost:1420/projects
agent-browser snapshot -i
# 找到刚创建的项目卡片，点击进入
agent-browser click "E2E Smoke Test Project"
agent-browser wait --load networkidle

# 切换到 Targets 标签或导航到 targets 页
agent-browser navigate http://localhost:1420/projects/__LAST_PROJECT_ID__/targets
# 注：实际执行时替换 __LAST_PROJECT_ID__ 为真实 projectId
# 或使用 find + click 方式从 Dashboard 进入

agent-browser wait --text "目标管理"

# 2.3 选择目标类型为域名
agent-browser select "自动检测" "domain"

# 2.4 填写域名目标
agent-browser fill "example.com 或 192.168.1.1 或 10.0.0.0/24 或 192.168.0.1-10" "smoke-test.example.com"

# 2.5 点击添加
agent-browser click "添加"
agent-browser wait --load networkidle

# 2.6 处理 Scope 确认弹窗（如果出现）
agent-browser snapshot -i
# 如果 Scope 确认对话框出现，点击确认
agent-browser click "添加规则并继续"
agent-browser wait --load networkidle

# 2.7 截图验证
agent-browser screenshot step2-target-added.png
```

### 预期结果
- [ ] 页面标题 "目标管理" 可见
- [ ] 目标类型下拉框可选择 "domain"
- [ ] 填写域名后点击"添加"成功
- [ ] 如触发 Scope 确认对话框，点击"添加规则并继续"后目标成功添加
- [ ] 新目标 "smoke-test.example.com" 出现在目标列表中
- [ ] 目标列表表格可见（类型、目标值、创建时间列）

---

## 步骤 3：启动扫描

### 操作

```bash
# 3.1 导航到扫描执行页
agent-browser navigate http://localhost:1420/projects/__PROJECT_ID__/runs
agent-browser wait --text "扫描执行"

# 3.2 点击新建扫描按钮
agent-browser click "新建扫描"
agent-browser wait --text "新建扫描"

# 3.3 填写扫描名称
agent-browser fill "扫描名称" "E2E Smoke Scan"

# 3.4 选择第一个可用模板
agent-browser snapshot -i
# 点击第一个模板卡片
agent-browser click @e4

# 3.5 点击开始扫描
agent-browser click "开始扫描"
agent-browser wait --load networkidle

# 3.6 截图验证
agent-browser screenshot step3-scan-started.png
```

### 预期结果
- [ ] 页面标题 "扫描执行" 可见
- [ ] "新建扫描"按钮可点击并打开模态框
- [ ] 模态框标题 "新建扫描" 可见
- [ ] 扫描名称输入框可填写
- [ ] 工具模板列表至少有一个可选模板
- [ ] 选择模板后"开始扫描"按钮变为可点击状态
- [ ] 点击"开始扫描"后模态框关闭，新 run 出现在执行历史列表中
- [ ] 新 run 显示状态标签（如"待启动"或"运行中"）

---

## 步骤 4：查看 Findings

### 操作

```bash
# 4.1 导航到 Findings 页
agent-browser navigate http://localhost:1420/projects/__PROJECT_ID__/findings
agent-browser wait --text "漏洞发现"

# 4.2 等待列表加载
agent-browser wait --load networkidle
agent-browser snapshot -i

# 4.3 截图验证
agent-browser screenshot step4-findings-loaded.png
```

### 预期结果
- [ ] 页面标题 "漏洞发现" 可见
- [ ] Findings 列表区域加载完成（显示列表或空状态）
- [ ] 过滤器栏可见（状态、严重级别、关键词搜索）
- [ ] 页面无加载错误提示
- [ ] 如果有 findings，每条显示：严重级别徽章、标题、状态、来源工具、可信度

---

## 步骤 5：导出报告

### 操作

```bash
# 5.1 导航到报告页
agent-browser navigate http://localhost:1420/projects/__PROJECT_ID__/reports
agent-browser wait --text "报告"

# 5.2 等待报告数据加载
agent-browser wait --load networkidle
agent-browser snapshot -i

# 5.3 点击导出 Markdown
agent-browser click "导出 Markdown"
agent-browser wait 2000

# 5.4 截图验证
agent-browser screenshot step5-report-exported.png
```

### 预期结果
- [ ] 页面标题包含 "报告" 字样
- [ ] 报告大纲区域可见（概览、范围、方法、风险统计、漏洞详情等）
- [ ] "导出 Markdown" 按钮可点击（如有 findings 则非禁用状态）
- [ ] 点击"导出 Markdown"后：
  - 浏览器环境：触发文件下载
  - Tauri 环境：弹出系统保存对话框
- [ ] 按钮文字在导出过程中变为 "导出中..."，导出完成后恢复
- [ ] 无错误提示弹窗

---

## 自动化变量说明

| 变量 | 获取方式 | 说明 |
|------|---------|------|
| `__PROJECT_ID__` | 步骤 1 创建项目后从 URL 或 API 响应提取 | 新建项目的 UUID，后续步骤复用 |

获取 projectId 的辅助命令：
```bash
# 创建项目后从当前 URL 提取
agent-browser get url
# 期望输出: http://localhost:1420/projects/<projectId>/...

# 或从页面中的项目卡片链接提取
agent-browser get attr "[href*='\/projects\/']" href
```

---

## 环境特定处理

### Tauri 桌面应用模式
如果应用以 Tauri 模式运行：
- 使用 `agent-browser --auto-connect` 连接已启动的应用窗口
- 导出报告时会触发系统文件保存对话框，需要额外处理：
```bash
agent-browser click "导出 Markdown"
agent-browser wait --text "保存"
agent-browser type ".file-name" "smoke-test-report"
agent-browser click "保存"
```

### 浏览器模式
```bash
agent-browser navigate http://localhost:1420
# 正常执行上述所有步骤
# 导出报告时通过下载事件验证
```

---

## 测试后清理（可选）

```bash
# 导航回项目列表
agent-browser navigate http://localhost:1420/projects
agent-browser snapshot -i

# 找到测试项目并删除
agent-browser click "删除项目" --nth 1
agent-browser wait --text "确认删除"
agent-browser click "确认"
agent-browser wait --load networkidle
```

---

## 失败排查指南

| 失败点 | 可能原因 | 排查建议 |
|--------|---------|---------|
| 步骤 1 表单提交失败 | 后端 API 不可用 | 检查后端是否运行在 :8080 |
| 步骤 2 Scope 弹窗未出现 | 目标值已匹配现有 Scope 规则 | 更换为新的域名测试 |
| 步骤 3 无可用模板 | 数据库未初始化模板数据 | 检查 tool_templates 表 |
| 步骤 4 findings 为空 | 扫描未完成或未产生发现 | 等待扫描完成或检查扫描配置 |
| 步骤 5 导出按钮禁用 | 无 confirmed/accepted_risk 状态的 findings | 先完成扫描并将 findings 状态改为 confirmed |

---

## 验收标准
- [ ] 5 个主流程步骤全部覆盖
- [ ] 每个步骤有明确的前置条件、操作命令、预期结果
- [ ] 使用 agent-browser CLI 语法编写
- [ ] 包含环境特定处理（浏览器 / Tauri）
- [ ] 包含失败排查指南

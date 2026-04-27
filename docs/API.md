# Anchor API 参考

> 基础路径：`http://localhost:8080`
> 最后更新：2026-04-27（M4）

---

## 通用约定

### 成功响应

直接返回数据对象或数组（**注意**：wiki 约定包裹 `{"data": ...}`，但当前实现未包裹，前后端已匹配）。

```json
{ "id": "...", "name": "..." }
```

### 错误响应

```json
{
  "error": {
    "code": "TOOL_NOT_FOUND",
    "message": "tool not found: subfinder",
    "detail": "..."
  }
}
```

### 错误码

| 错误码 | HTTP 状态 | 含义 |
|--------|-----------|------|
| `BAD_REQUEST` | 400 | 请求参数错误 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `SCOPE_DENIED` | 403 | 目标未通过 Scope Check |
| `TOOL_NOT_FOUND` | 404 | 工具未安装 |
| `TOOL_TIMEOUT` | 408 | 工具执行超时 |
| `TOOL_EXECUTION` | 500 | 工具执行失败（非零退出） |
| `PARSE_ERROR` | 422 | 输出解析失败 |
| `TRUNCATION_WARNING` | 200 | 输出超过大小上限 |
| `WORKDIR_ERROR` | 500 | workdir 不可写 |
| `INTERNAL_ERROR` | 500 | 内部错误 |

---

## 项目

### 创建项目
```http
POST /projects
Content-Type: application/json

{
  "name": "项目名称",
  "organization": "组织/客户",
  "purpose": "测试目的",
  "start_time": "2026-04-01T00:00:00Z",
  "end_time": "2026-04-30T23:59:59Z",
  "rate_limit": 150,
  "default_profile": "standard"
}
```

### 列出项目
```http
GET /projects
```

### 获取项目
```http
GET /projects/{id}
```

---

## 目标

### 添加目标
```http
POST /projects/{id}/targets
Content-Type: application/json

{
  "type": "domain",
  "value": "example.com"
}
```

### 批量导入
```http
POST /projects/{id}/targets/import
Content-Type: multipart/form-data

file: <TXT 或 CSV 文件>
```

响应：`ImportResult`
```json
{
  "imported": 5,
  "duplicates": 2,
  "denied_targets": [
    { "value": "evil.com", "reason": "命中排除规则" }
  ],
  "errors": []
}
```

### 列出目标
```http
GET /projects/{id}/targets
```

---

## Scope 规则

### 创建规则
```http
POST /scope-rules
Content-Type: application/json

{
  "project_id": "...",
  "action": "include",
  "type": "domain",
  "value": "example.com",
  "reason": "授权域名"
}
```

### 列出规则
```http
GET /scope-rules?project_id={project_id}
```

---

## 扫描计划

### 创建计划
```http
POST /scan-plans
Content-Type: application/json

{
  "project_id": "...",
  "workflow_type": "asset_discovery",
  "profile": "standard"
}
```

### 批准计划
```http
POST /scan-plans/{id}/approve
```

### 干运行
```http
POST /scan-plans/dry-run?project_id={project_id}
```

---

## 扫描任务

### 运行任务
```http
POST /tasks/run
Content-Type: application/json

{
  "project_id": "...",
  "tool": "subfinder",
  "target_id": "...",
  "command": "subfinder -d example.com -oJ"
}
```

### 获取任务
```http
GET /scan-tasks/{id}
```

### 取消任务
```http
POST /scan-tasks/{id}/cancel
```

### 列出 Artifact
```http
GET /tasks/{id}/artifacts
```

---

## 资产（M2 新增）

### 启动资产发现工作流
```http
POST /projects/{id}/workflows/asset-discovery
```

### 列出资产
```http
GET /projects/{id}/assets
```

### 列出 Web 端点
```http
GET /projects/{id}/web-endpoints
```

### 列出端口
```http
GET /assets/{id}/ports
```

### 列出服务
```http
GET /assets/{id}/services
```

---

## Finding（M3 新增）

### 启动 Web 初筛工作流
```http
POST /projects/{id}/workflows/web-screening
```

### 列出 Finding
```http
GET /projects/{id}/findings
```

### 获取 Finding
```http
GET /findings/{id}
```

### 更新 Finding 状态
```http
PATCH /findings/{id}/status
Content-Type: application/json

{
  "status": "confirmed"
}
```

状态枚举：`new` | `pending_review` | `confirmed` | `false_positive` | `accepted_risk` | `ignored` | `reported`

### 添加 Evidence
```http
POST /findings/{id}/evidence
Content-Type: application/json

{
  "type": "note",
  "excerpt": "验证说明..."
}
```

Evidence 类型：`request` | `response` | `screenshot` | `raw_output` | `note` | `file`

---

## 报告（M4 新增）

### 导出 Markdown 报告
```http
GET /projects/{id}/reports/export.md
```

响应：`text/markdown`

### 导出 JSON 数据包
```http
GET /projects/{id}/reports/export.json
```

响应：`application/json`

---

## 健康检查

### 列出工具健康状态
```http
GET /health/tools
```

### 运行健康检查
```http
POST /health/check
```

---

## SSE 事件流

```http
GET /events
```

推送事件：
```json
{"event": "task_update", "task_id": "...", "status": "running"}
```

---

## 数据模型参考

完整模型定义见 `internal/models/models.go`。核心类型：

| 模型 | 说明 |
|------|------|
| `Project` | 项目（含时间窗口、速率限制） |
| `Target` | 目标（domain/url/ip/cidr） |
| `ScopeRule` | Scope 规则（include/exclude） |
| `ScanPlan` / `ScanTask` | 扫描计划与任务 |
| `ToolInvocation` | 工具调用记录 |
| `Asset` / `Port` / `Service` / `WebEndpoint` | 资产发现结果 |
| `Finding` / `Evidence` | 漏洞发现与证据 |
| `RawArtifact` | 原始工具输出 |
| `ToolHealth` | 工具健康状态 |

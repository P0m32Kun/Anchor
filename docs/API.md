# SecBench API 参考

> 基础路径：`http://localhost:8080`

## 项目

### 创建项目
```http
POST /projects
Content-Type: application/json

{
  "name": "项目名称",
  "organization": "组织/客户",
  "purpose": "测试目的"
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

## 目标

### 创建目标
```http
POST /projects/{id}/targets
Content-Type: application/json

{
  "type": "domain",  // domain | url | ip | cidr
  "value": "example.com"
}
```

### 列出目标
```http
GET /projects/{id}/targets
```

## Scope 规则

### 创建规则
```http
POST /scope-rules
Content-Type: application/json

{
  "project_id": "...",
  "action": "include",  // include | exclude
  "type": "domain",     // domain | url | ip | cidr
  "value": "example.com",
  "reason": "授权域名"
}
```

### 列出规则
```http
GET /scope-rules?project_id={project_id}
```

## 扫描计划

### 创建计划
```http
POST /scan-plans
Content-Type: application/json

{
  "project_id": "...",
  "workflow_type": "asset_discovery",
  "profile": "standard"  // light | standard | deep
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

响应：
```json
{
  "mode": "dry-run",
  "project_id": "...",
  "results": [
    {
      "target": "example.com",
      "type": "domain",
      "decision": "allow",
      "reason": "命中包含规则: example.com"
    }
  ]
}
```

## 任务

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

## 健康检查

### 列出工具健康状态
```http
GET /health/tools
```

### 运行健康检查
```http
POST /health/check
```

## SSE 事件流

```http
GET /events
```

推送事件：
```json
{"event": "task_update", "task_id": "..."}
```

## 错误响应

所有错误统一格式：
```json
{
  "error": {
    "code": "TOOL_NOT_FOUND",
    "message": "tool not found: subfinder",
    "detail": "..."
  }
}
```

## 错误码

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

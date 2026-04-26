# 前后端 API 契约

> SecBench HTTP API 契约规范。前后端开发人员必须遵守。

---

## 1. 基础信息

- **基础路径**: `http://localhost:8080`
- **协议**: HTTP/1.1（MVP），后续可能 HTTPS
- **数据格式**: JSON
- **编码**: UTF-8

## 2. 通用响应格式

### 2.1 成功响应
```json
{
  "data": { ... }
}
```

### 2.2 错误响应
```json
{
  "error": {
    "code": "SCOPE_DENIED",
    "message": "目标不在授权范围内",
    "details": { ... }
  }
}
```

### 2.3 错误码表

| Code | HTTP Status | 说明 |
|------|-------------|------|
| `SCOPE_DENIED` | 403 | 目标不在授权范围内 |
| `TOOL_NOT_FOUND` | 404 | 工具未安装或路径错误 |
| `TOOL_TIMEOUT` | 504 | 工具执行超时 |
| `TOOL_EXECUTION_ERROR` | 502 | 工具执行失败（非零退出码） |
| `PARSE_ERROR` | 422 | 工具输出解析失败 |
| `TRUNCATION_WARNING` | 200 | 输出被截断（在 metadata 中标记） |
| `WORKDIR_ERROR` | 500 | workdir 创建/写入失败 |
| `INVALID_INPUT` | 400 | 请求参数无效 |
| `INTERNAL` | 500 | 内部错误 |

## 3. SSE 事件格式

### 3.1 连接
```
GET /events
Accept: text/event-stream
```

### 3.2 事件格式
```
event: task_update
data: {"task_id":"...","status":"running","progress":50,"message":"扫描中..."}

event: task_completed
data: {"task_id":"...","status":"completed","artifact_count":5}

event: task_failed
data: {"task_id":"...","status":"failed","error_code":"TOOL_TIMEOUT","error_message":"..."}
```

### 3.3 事件类型
| Event | 说明 |
|-------|------|
| `task_update` | 任务状态更新 |
| `task_completed` | 任务完成 |
| `task_failed` | 任务失败 |
| `health_update` | 工具健康状态更新 |

## 4. 核心 API 端点

### 4.1 项目
```
POST   /projects          创建项目
GET    /projects          列出项目
GET    /projects/:id      获取项目详情
PUT    /projects/:id      更新项目
DELETE /projects/:id      删除项目
```

### 4.2 目标
```
POST   /projects/:id/targets    添加目标
GET    /projects/:id/targets    列出目标
DELETE /targets/:id             删除目标
```

### 4.3 Scope 规则
```
POST   /scope-rules       创建规则
GET    /scope-rules       列出规则
DELETE /scope-rules/:id   删除规则
```

### 4.4 扫描任务
```
POST   /tasks/run         启动扫描任务
GET    /tasks/:id         获取任务详情
GET    /tasks/:id/artifacts   获取任务产出
POST   /tasks/:id/cancel    取消任务
```

### 4.5 工具健康
```
GET    /health/tools      获取所有工具健康状态
POST   /health/check      手动触发健康检查
```

## 5. 数据模型

### 5.1 Project
```json
{
  "id": "uuid",
  "name": "项目名称",
  "organization": "组织/客户",
  "purpose": "测试目的",
  "created_at": "2026-04-26T10:00:00Z",
  "updated_at": "2026-04-26T10:00:00Z"
}
```

### 5.2 Target
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "type": "domain",
  "value": "example.com",
  "created_at": "2026-04-26T10:00:00Z"
}
```

### 5.3 ScopeRule
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "action": "include",
  "type": "domain",
  "value": "*.example.com",
  "reason": "授权域名"
}
```

### 5.4 ScanTask
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "target_id": "uuid",
  "tool": "subfinder",
  "status": "completed",
  "started_at": "2026-04-26T10:00:00Z",
  "completed_at": "2026-04-26T10:05:00Z",
  "error": null
}
```

### 5.5 ToolInvocation
```json
{
  "id": "uuid",
  "task_id": "uuid",
  "tool": "subfinder",
  "args": ["-d", "example.com"],
  "exit_code": 0,
  "stdout": "...",
  "stderr": "",
  "duration_ms": 5000,
  "truncated": false
}
```

## 6. 变更规则

- **新增端点**: 需更新 `docs/API.md` 和此文件
- **变更响应格式**: 需同步更新前端类型定义
- **废弃端点**: 标记为 deprecated，保留至少一个版本

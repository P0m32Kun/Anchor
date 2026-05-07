# 前后端 API 契约

> Anchor HTTP API 契约规范。前后端开发人员必须遵守。
> 最后更新：2026-04-27（M4）

---

## 1. 基础信息

- **基础路径**: `http://localhost:8080`
- **协议**: HTTP/1.1（MVP），后续可能 HTTPS
- **数据格式**: JSON
- **编码**: UTF-8

## 2. 通用响应格式

### 2.1 成功响应

当前实现直接返回数据对象（未包裹 `{"data": ...}`），前后端已匹配。后续如需统一包裹，需同步更新前后端。

```json
{ "id": "uuid", "name": "..." }
```

### 2.2 错误响应

```json
{
  "error": {
    "code": "SCOPE_DENIED",
    "message": "目标不在授权范围内",
    "detail": "..."
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
POST   /projects                 创建项目
GET    /projects                 列出项目
GET    /projects/:id             获取项目详情
```

### 4.2 目标
```
POST   /projects/:id/targets          添加目标
POST   /projects/:id/targets/import   批量导入（multipart/file）
GET    /projects/:id/targets          列出目标
```

### 4.3 Scope 规则
```
POST   /scope-rules             创建规则
GET    /scope-rules             列出规则
DELETE /scope-rules/:id         删除规则
```

### 4.4 扫描计划
```
POST   /scan-plans              创建计划
POST   /scan-plans/:id/approve  批准计划
POST   /scan-plans/dry-run      干运行
```

### 4.5 扫描任务
```
POST   /tasks/run               启动扫描任务
GET    /scan-tasks/:id          获取任务详情
POST   /scan-tasks/:id/cancel   取消任务
GET    /tasks/:id/artifacts     获取任务产出
```

### 4.6 资产（M2）
```
POST   /projects/:id/workflows/asset-discovery  启动资产发现
GET    /projects/:id/assets                     列出资产
GET    /projects/:id/web-endpoints              列出 Web 端点
GET    /assets/:id/ports                        列出端口
GET    /assets/:id/services                     列出服务
```

### 4.7 Finding（M3）
```
POST   /projects/:id/workflows/web-screening   启动 Web 初筛
GET    /projects/:id/findings                  列出 Finding
GET    /findings/:id                           获取 Finding
PATCH  /findings/:id/status                    更新状态
POST   /findings/:id/evidence                  添加 Evidence
```

### 4.8 报告（M4）
```
GET    /projects/:id/reports/export.md    导出 Markdown
GET    /projects/:id/reports/export.json  导出 JSON
```

### 4.9 健康检查
```
GET    /health/tools            获取所有工具健康状态
POST   /health/check            手动触发健康检查
```

## 5. 数据模型

### 5.1 Project
```json
{
  "id": "uuid",
  "name": "项目名称",
  "organization": "组织/客户",
  "purpose": "测试目的",
  "start_time": "2026-04-01T00:00:00Z",
  "end_time": "2026-04-30T23:59:59Z",
  "rate_limit": 150,
  "default_profile": "standard",
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
  "source": "manual",
  "status": "active",
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
  "finished_at": "2026-04-26T10:05:00Z"
}
```

### 5.5 Asset（M2）
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "type": "domain",
  "value": "sub.example.com",
  "normalized_value": "sub.example.com",
  "source_tools": ["subfinder"],
  "first_seen": "2026-04-26T10:00:00Z",
  "last_seen": "2026-04-26T10:00:00Z"
}
```

### 5.6 WebEndpoint（M2）
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "asset_id": "uuid",
  "url": "https://sub.example.com/",
  "scheme": "https",
  "host": "sub.example.com",
  "status_code": 200,
  "title": "Example Site",
  "technologies": ["nginx", "WordPress"]
}
```

### 5.7 Finding（M3）
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "asset_id": "uuid",
  "source_tool": "nuclei",
  "dedup_key": "sha256hash",
  "title": "WordPress Login Exposed",
  "severity": "medium",
  "confidence": 75,
  "priority": 60,
  "status": "pending_review",
  "summary": "...",
  "remediation": "..."
}
```

### 5.8 Evidence（M3）
```json
{
  "id": "uuid",
  "finding_id": "uuid",
  "type": "request",
  "excerpt": "GET /wp-login.php HTTP/1.1...",
  "created_by": "user",
  "created_at": "2026-04-26T10:00:00Z"
}
```

## 6. 变更规则

- **新增端点**: 需更新 `docs/current/architecture.md` 中的接口叙事或相关当前文档，并同步更新此文件
- **变更响应格式**: 需同步更新前端类型定义
- **废弃端点**: 标记为 deprecated，保留至少一个版本

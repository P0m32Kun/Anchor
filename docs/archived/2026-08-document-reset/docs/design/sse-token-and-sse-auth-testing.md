---
status: stale
stale_date: "2026-06-17"
---

# SSE Token & SSE Auth — Proposal / Spec / Tasks

> 目的：用可观测的行为约束「SSE 实时更新链路」是否真的跑通，避免仅写了代码但 SSE 认证或前端订阅链路不可用。

## Proposal（动机与范围）

### 动机
- SSE 连接需要项目级的短生命周期认证（JWT），否则可能出现跨项目泄露。
- 前端 `useSSE` 需要在 `projectId` 存在时拉取 SSE token，并把 token 作为查询参数拼进 EventSource URL。
- 历史问题：测试容易“只断言 API 返回值”，但 SSE 认证或订阅失败时用户实际看不到数据。

### 范围
包含：
- `POST /projects/{id}/sse-token` 签发 SSE JWT
- `GET /projects/{id}/events` 使用 `SSEAuthMiddleware` 校验：
  - Bearer token 可用
  - 或 query `token` 的 JWT claims.project_id 必须与路径 `{id}` 一致
- `frontend/src/hooks/useSSE.ts`：当 `projectId` 存在时获取 token 并建立 EventSource

不包含（后续）：
- 端到端浏览器层面验证 SSE 消息流（本次用集成/单元测试先把认证链路稳住）

## Spec（验收标准：Given-When-Then）

### REQ-SSE-01：签发 token
- Given 已登录且 API Token 可用
- When 调用 `POST /projects/{projectId}/sse-token`
- Then 返回 JSON `{ token, expires_in, project_id }`
- And token 可被服务端 `ValidateSSEToken` 解析
- And claims：
  - `claims.project_id == projectId`
  - `claims.type == "sse"`
  - `claims.exp - now` 在 1h 内

### REQ-SSE-02：SSEAuthMiddleware 允许同项目 token
- Given token 来自 projectA 的 `GenerateSSEToken(projectA)`
- When 请求 `GET /projects/{projectA}/events?token={token}`
- Then 中间件通过，返回 200（到达下游 handler）

### REQ-SSE-03：SSEAuthMiddleware 拒绝跨项目 token
- Given token 来自 projectA 的 `GenerateSSEToken(projectA)`
- When 请求 `GET /projects/{projectB}/events?token={token}`（projectB != projectA）
- Then 中间件返回 401 Unauthorized

### REQ-UI-SSE-01：useSSE 需要 projectId 才拉 token
- Given 调用 `useSSE(url, { projectId })`
- When hook 发起连接
- Then 首先通过 `api.fetchSSEToken(projectId)` 获取 token
- And EventSource URL 包含 `token=` 查询参数

## Tasks（实现清单）

- [x] Backend：实现 `GenerateSSEToken` / `ValidateSSEToken`（`internal/api/sse_token.go`）
- [x] Backend：实现 `SSEAuthMiddleware`（`internal/api/middleware.go`）
- [x] Frontend：实现 `useSSE` 拉取 token 并拼进 EventSource URL（`frontend/src/hooks/useSSE.ts`）
- [x] TDD/单测：新增 SSE token/鉴权单测
  - [x] `internal/api/sse_token_test.go`
- [x] E2E：`frontend/e2e/tests/sse-realtime.spec.ts`（SSE 连接徽章 + 扫描完成后 UI 状态更新）
- [x] E2E：`frontend/e2e/tests/trace-audit.spec.ts`（工具调用日志 + Finding 调用溯源 UI）


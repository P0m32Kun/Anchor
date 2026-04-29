---
archived: true
archived_at: "2026-04-29"
archived_by: doc-archivist
version: "v0.1"
original_path: "wiki/decisions/002-sse-over-websocket.md"
status: "completed"
reason: "v0.1 架构决策记录，v0.2 阶段结束"
---

# ADR-002: SSE 替代 WebSocket

## 状态
✅ 已决策（2026-04-26）

## 背景
需要为扫描任务提供实时状态推送（queued → running → completed/failed）。

## 决策
**使用 SSE (Server-Sent Events) 替代 WebSocket。**

## 理由
| 因素 | SSE | WebSocket |
|------|-----|-----------|
| 方向 | ✅ 单向（服务端→客户端） | 双向 |
| 复杂度 | ✅ 基于 HTTP，极简单 | 需要握手、帧协议 |
| 重连 | ✅ 浏览器自动重连 | 需自行实现 |
| 防火墙 | ✅ 走标准 HTTP | 可能被封 |
| 双向通信 | ❌ 不支持 | ✅ 支持 |
| 二进制数据 | ❌ 不支持 | ✅ 支持 |

MVP 只需要服务端向客户端推送任务状态变更，SSE 完全够用且更简单。

## 当前实现
- Go 端：`http.ServeContent` 或手动实现 SSE writer
- 前端：`EventSource('/events')` 监听

## 风险
- 如果未来需要客户端向服务端实时推送（如实时日志输入），SSE 不够，需要 WebSocket 或轮询

## 迁移路径
如果未来需要双向通信，可以：
1. 保持 SSE 用于状态推送
2. 新增 WebSocket 用于双向场景
3. 或保持 SSE，用 HTTP POST 处理客户端→服务端

## 方案：Sprint 2a.5 — 后端 SSE 改造

### 当前现状

| 项目 | 现状 |
|------|------|
| 存储结构 | `sseClients map[string]chan []byte` — 全局客户端列表，无项目隔离 |
| 连接端点 | `GET /events` — 无 `project_id` 过滤 |
| 广播方式 | `broadcastSSE(data)` — 推送给**所有**客户端 |
| 心跳 | ❌ 无 |
| 断开处理 | ✅ 正常（无 error 日志） |

`broadcastSSE` 当前被 3 处调用，均具备 `project_id` 信息：
1. `handleRunTask` → `task.ProjectID`
2. `handleStartAssetDiscovery` → 参数 `projectID`
3. `handleStartWebScreening` → 参数 `projectID`

### 改造方案

**方案 A（推荐）：双层 map 按项目隔离**

将 `sseClients` 改为 `map[string]map[string]chan []byte`（`project_id` → `client_id` → `ch`），广播时只遍历目标项目的客户端，O(订阅该项目客户端数)。

1. **数据结构改造**
   ```go
   sseClients map[string]map[string]chan []byte  // projectID -> clientID -> ch
   ```

2. **新 endpoint**
   - `GET /projects/{id}/events` 替代 `GET /events`
   - 从 URL path 读取 `project_id`，存入对应项目分桶

3. **广播改造**
   ```go
   func (s *Server) broadcastProjectSSE(projectID string, data map[string]interface{}) {
       b, _ := json.Marshal(data)
       s.mu.Lock()
       clients, ok := s.sseClients[projectID]
       s.mu.Unlock()
       if !ok { return }
       for _, ch := range clients {
           select {
           case ch <- b:
           default:
           }
       }
   }
   ```

4. **心跳机制**
   - `handleSSE` 内启动 `time.NewTicker(30 * time.Second)`
   - 每 30s 发送 `data: {"event":"ping"}\n\n`
   - 客户端断开时（`<-r.Context().Done()`）停止 ticker，静默返回

5. **替换调用点**
   - 3 处 `broadcastSSE(...)` 全部替换为 `broadcastProjectSSE(projectID, ...)`

6. **清理旧路由**
   - 删除 `mux.HandleFunc("GET /events", s.handleSSE)`

### 执行步骤

1. 修改 `Server` 结构体：`sseClients` 类型 + 同步互斥（已有 `mu`）
2. 实现 `handleProjectSSE(w, r)` 并注册路由 `GET /projects/{id}/events`
3. 实现 `broadcastProjectSSE(projectID, data)`
4. 在 `handleProjectSSE` 内添加 30s 心跳 ticker
5. 替换 3 处 `broadcastSSE` 调用为 `broadcastProjectSSE`
6. 删除旧 `GET /events` 路由和 `handleSSE`
7. 运行 `go build ./...` 验证
8. 更新 `sprint2a-5-sse-backend.md`

### 风险

- 前端若仍连接旧 `/events`，会收到 404。需与前端改造同步。
- `handleRunTask` 中 goroutine 内的广播需要正确传递 `task.ProjectID`（当前已有）。

---

**是否同意此方案？如同意，我将开始实施。**
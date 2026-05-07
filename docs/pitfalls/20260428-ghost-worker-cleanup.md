# 幽灵 Worker 问题

## 现象

每次 Worker 容器重启后，Workers 页面和 API 都会多出一个新的 Worker 记录，旧的记录仍然显示为 online。数据库里积累了十几个"幽灵"Worker，只有一个是真正在运行的。

```json
// GET /workers
[
  {"id": "local", "status": "stopped"},
  {"id": "id-xxx-1", "status": "online", "endpoint": "http://worker:41561"},
  {"id": "id-xxx-2", "status": "online", "endpoint": "http://worker:33761"},  // 旧的，已死
  {"id": "id-xxx-3", "status": "online", "endpoint": "http://worker:37477"},  // 旧的，已死
  // ... 累计 13 个
]
```

## 根因

1. **注册无去重**：`handleRegisterWorker` 每次生成新 `util.GenerateID()` 直接插入，不管同一 endpoint 是否已有记录
2. **无心跳超时**：`handleWorkerHeartbeat` 只更新 `last_seen`，没有任何后台任务检查"多久没心跳"
3. **列表不过滤**：`handleListWorkers` 返回 DB 中所有记录，包括已经死掉的

## 修复

### 后端

1. **心跳清理 goroutine**（`internal/api/handlers.go`）
   ```go
   func (s *Server) cleanupStaleWorkers() {
       ticker := time.NewTicker(60 * time.Second)
       for range ticker.C {
           workers, _ := s.queries.ListWorkerNodes()
           for _, w := range workers {
               if now.Sub(*w.LastSeen) > 120*time.Second {
                   s.queries.UpdateWorkerNodeStatus(w.ID, models.WorkerStatusOffline, now)
                   delete(s.taskQueue, w.ID)
                   delete(s.taskResults, w.ID)
               }
           }
       }
   }
   ```

2. **列表不过滤（API 返回全部）**：前端按需过滤展示，保留历史数据

### 前端

WorkersPage 按 `w.status === "online"` 过滤显示，offline 的不展示。

## 教训

- **注册必须配过期**：任何"注册"机制必须有对应的"注销/过期"机制，否则就是内存/数据库泄漏
- **心跳 != 存活检测**：心跳只是更新状态，必须有人读这个状态做超时判断
- **endpoint 去重**：生产环境应考虑用 endpoint + name 做唯一键，避免重复注册

## 相关文件

- `internal/api/handlers.go` — `cleanupStaleWorkers()`
- `internal/api/worker_handlers.go` — `handleRegisterWorker`
- `frontend/src/pages/WorkersPage.tsx` — 状态过滤

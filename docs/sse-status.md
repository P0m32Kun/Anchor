# SSE 通信现状与架构决策

> **文档版本**：v1.0  
> **生成日期**：2026-04-29  
> **对应 Sprint**：0.7  
> **状态**：架构决策已确认，Sprint 2 实现

---

## 一、现状确认

### 1.1 后端 SSE 能力清单

| 项目 | 状态 | 详情 |
|------|:----:|------|
| SSE endpoint | ✅ | `GET /events` — `internal/api/handlers.go:905` |
| 客户端管理 | ✅ | `map[string]chan []byte` — channel-based |
| 初始事件 | ✅ | `{"event":"connected"}` |
| 断线处理 | ✅ | `r.Context().Done()` 自动清理 |
| 心跳机制 | ❌ | 无定期 ping，连接可能被中间层超时关闭 |
| 项目级过滤 | ❌ | `broadcastSSE` 全局广播，所有客户端收到所有事件 |

### 1.2 广播事件类型

| 事件类型 | 触发场景 | 包含 project_id |
|----------|---------|:---------------:|
| `connected` | 客户端连接时 | ❌ |
| `task_update` | 扫描任务状态变更 | ❌ |
| `asset_discovery_complete` | 资产发现 workflow 完成 | ✅ |
| `web_screening_complete` | Web 筛查 workflow 完成 | ✅ |

### 1.3 前端 SSE 缺失清单

| 缺失项 | 影响 |
|--------|------|
| 零 `EventSource` 代码 | 无法接收任何 SSE 事件 |
| 无 `useSSE` hook | 无 SSE 连接管理 |
| DashboardPage 5s 轮询 `/workers` | 低效，无实时通知 |
| WorkersPage 无实时更新 | 只能手动刷新 |
| RunsPage 一次性拉取 | 无任务进度实时更新 |

---

## 二、架构决策

### 2.1 技术路径选择

| 方案 | 说明 | 决策 |
|------|------|:----:|
| **A. 浏览器 EventSource 直连** | 前端直接用 `new EventSource('/events')` | ✅ **采用** |
| B. Tauri IPC 桥接 | Rust 层作为 bridge，前端通过 `invoke` 接收 | ❌ 不采用 |

**选择方案 A 的理由**：
1. SSE 是标准 HTTP，Tauri WebView 完全支持 `EventSource`
2. 无需额外 Rust 代码，减少复杂度
3. 浏览器 dev 模式和 Tauri 桌面模式行为一致
4. 无需处理 Tauri IPC 的额外开销

**已知限制**：
- `EventSource` 不支持自定义 headers（如 Auth Token）— 当前无 Auth，不影响
- Tauri WebView 的 CORS 行为与浏览器略有差异 — 已配置 CSP 和 proxy，已解决

### 2.2 后端改造建议

**`broadcastSSE` → `broadcastProjectSSE`**

```go
// 当前：全局广播
type sseClient struct {
    id string
    ch chan []byte
}

// 目标：按项目过滤
type sseClient struct {
    id        string
    projectID string  // 可为空（全局订阅）
    ch        chan []byte
}

func (s *Server) broadcastProjectSSE(projectID string, data map[string]interface{}) {
    b, _ := json.Marshal(data)
    for _, client := range s.sseClients {
        // 全局事件（connected）推给所有客户端
        // 项目事件只推给订阅了该项目的客户端
        if projectID != "" && client.projectID != "" && client.projectID != projectID {
            continue
        }
        select {
        case client.ch <- b:
        default:
            log.Printf("[SSE] client %s channel full, dropping message", client.id)
        }
    }
}
```

**心跳机制**：
```go
// 每 30s 发送一次 ping
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()
for {
    select {
    case <-ticker.C:
        broadcastSSE(map[string]interface{}{"event": "ping"})
    case <-ctx.Done():
        return
    }
}
```

**SSE 客户端注册时携带 project_id**：
```
GET /events?project_id=xxx
```

---

## 三、`useSSE` Hook 设计草案

### 3.1 签名

```typescript
interface UseSSEOptions {
  projectId?: string;           // 过滤：只接收该 project 的事件
  onEvent: (event: SSEEvent) => void;  // 事件回调
  autoReconnect?: boolean;      // 默认 true
  maxReconnectDelay?: number;   // 默认 30000ms
}

interface UseSSEReturn {
  status: 'connecting' | 'open' | 'closed' | 'error';
  reconnect: () => void;
  close: () => void;
}

function useSSE(options: UseSSEOptions): UseSSEReturn;
```

### 3.2 自动重连策略

```typescript
// 指数退避
let delay = 1000;
const maxDelay = options.maxReconnectDelay || 30000;

function reconnect() {
  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(() => {
    connect();
    delay = Math.min(delay * 2, maxDelay);
  }, delay);
}

// 连接成功时重置延迟
function onOpen() {
  delay = 1000;
  status = 'open';
}
```

### 3.3 页面可见性管理

```typescript
useEffect(() => {
  const handleVisibility = () => {
    if (document.hidden) {
      eventSource?.close();
      status = 'closed';
    } else {
      reconnect();
    }
  };
  document.addEventListener('visibilitychange', handleVisibility);
  return () => document.removeEventListener('visibilitychange', handleVisibility);
}, []);
```

### 3.4 断线降级到 Polling

```typescript
// useRealtimeData.ts — SSE 优先 + Polling 降级
function useRealtimeData<T>(options: {
  sseUrl?: string;
  pollFn?: () => Promise<T>;
  interval?: number;
}): { data: T | null; isLive: boolean } {
  const [data, setData] = useState<T | null>(null);
  const [isLive, setIsLive] = useState(false);

  useSSE({
    onEvent: (e) => {
      setData(e.data);
      setIsLive(true);
    },
  });

  // SSE 断线时，usePolling 自动接管
  usePolling(() => {
    if (!isLive) {
      options.pollFn?.().then(setData);
    }
  }, options.interval || 5000);

  return { data, isLive };
}
```

---

## 四、与 `usePolling` 的协作模式

### 4.1 职责划分

| 页面 | 首选模式 | 降级模式 | 说明 |
|------|---------|---------|------|
| **RunsPage** | SSE | Polling 5s | 任务状态实时更新最关键 |
| **DashboardPage** | SSE | Polling 5s | Worker 状态 + 任务完成通知 |
| **WorkersPage** | SSE | Polling 5s | Worker 在线状态 |
| **FindingsPage** | 手动刷新 | — | 数据变更不频繁，手动刷新足够 |
| **AssetPage** | 手动刷新 | — | 同上 |

### 4.2 互斥逻辑

```typescript
// 每个页面只使用一种数据源
function usePageData<T>(options: {
  sseEnabled: boolean;
  sseUrl?: string;
  pollFn: () => Promise<T>;
  interval: number;
}) {
  if (options.sseEnabled) {
    // SSE 模式：接收实时事件，不轮询
    return useSSE({ ... });
  } else {
    // Polling 模式
    return usePolling(options.pollFn, options.interval);
  }
}
```

---

## 五、实施计划

| 步骤 | 内容 | 负责人 | Sprint |
|------|------|--------|--------|
| 1 | 后端：`broadcastProjectSSE` + 心跳 + 客户端 project_id 过滤 | 后端 | Sprint 2a.5 |
| 2 | 前端：`useSSE` hook 实现（自动重连、可见性管理） | 前端 | Sprint 2a.4 |
| 3 | 前端：`usePolling` hook 实现（指数退避、可见性暂停） | 前端 | Sprint 2a.6 |
| 4 | 前端：`useRealtimeData`（SSE + Polling 协作） | 前端 | Sprint 2a.7 |
| 5 | RunsPage 接入 SSE | 前端 | Sprint 2b.3 |
| 6 | DashboardPage/WorkersPage 接入 SSE | 前端 | Sprint 2b.3 |
| 7 | E2E 测试验证 SSE 实时更新 | QA | Sprint 4 |

---

## 六、风险与缓解

| 风险 | 缓解 |
|------|------|
| SSE 在 Tauri WebView 中不稳定 | 保留 Polling 降级机制；Sprint 4 集中验证 |
| 高频事件导致前端性能问题 | 后端消息合并（如 1s 内多个 task_update 合并为最新状态） |
| 多窗口/多标签 SSE 连接过多 | 限制每个客户端最多 1 个 SSE 连接；心跳超时后清理 |
| 后端 `broadcastSSE` 全局广播导致信息泄漏 | 按 project_id 过滤，确保 Project A 事件不推给 Project B 用户 |

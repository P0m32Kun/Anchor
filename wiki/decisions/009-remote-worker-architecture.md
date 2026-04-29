# ADR-009: 远程 Worker 架构

## 状态
已确认 ✅

## 上下文

v0.1 的 Worker 是同进程内的 goroutine，只适合单机运行。v0.2 需要支持：
1. 云主机上部署 Worker，桌面端在本地
2. 家庭 WiFi 笔记本作为 Worker，Server 在公网 VPS
3. 内网渗透场景：带着笔记本进客户内网，Worker 在内网执行扫描

核心约束：**Worker 不应该需要公网 IP**。在家庭 WiFi、企业内网、NAT 后面的 Worker 无法被 Server 主动连接。

## 决策

Worker 通过**出站 HTTP 长轮询**连接 Server，所有控制流都是 Worker 主动发起的：

```
Worker ──POST /workers/register────→ Server
Worker ──POST /workers/{id}/heartbeat──→ Server
Worker ──GET  /workers/{id}/tasks/poll──→ Server（长轮询，阻塞等待任务）
Worker ──POST /tasks/{id}/result────→ Server
```

- Worker 启动时注册自己，获取 worker_id 和 token
- 每 30 秒发送心跳
- 持续长轮询拉取任务，有任务立即执行，执行完上报结果
- Server 通过心跳超时检测 Worker 存活（120s 无心跳标记 offline）

## 理由

1. **NAT 友好**：Worker 只需要能出站访问 Server 的 IP/域名即可，无需公网 IP、无需端口映射
2. **安全**：Worker 不暴露任何入站端口，攻击面最小
3. **行业标准**：C2 框架（Cobalt Strike、Sliver）、Rapid7 InsightVM、HostedScan 等都采用此模式
4. **简单**：基于 HTTP REST，无需引入 WebSocket、gRPC 或自定义协议

## 后果

- **Server 不回连 Worker**：所有 Server → Worker 的通信必须通过 poll 响应"包装"
- **延迟**：任务下发有最长 5s 的 poll 间隔延迟（可接受）
- **Worker endpoint 仅作记录**：Server 存储 Worker 上报的 endpoint，但当前不回连。如需回连（如拉取截图），未来需通过反向隧道或 Worker 主动上传

## 相关文件

- `main.go` — 双模式入口
- `internal/worker/remote_client.go` — Worker 端远程客户端
- `internal/api/worker_handlers.go` — Server 端注册/心跳/轮询/撤销

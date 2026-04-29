---
archived: true
archived_at: "2026-04-29"
archived_by: doc-archivist
version: "v0.1"
original_path: "wiki/decisions/005-worker-in-process.md"
status: "completed"
reason: "v0.1 架构决策记录，v0.2 阶段结束"
---

# ADR-005: Worker 同进程模型 (MVP)

## 状态
✅ 已决策（2026-04-26）

## 背景
Worker 负责执行外部安全工具（Subfinder 等）作为子进程。需要确定 Worker 的运行模式。

## 决策
**MVP 中 Worker 作为 Control Plane 内的 goroutine 运行（同进程）。v0.2 再评估是否拆分为独立进程。**

## 理由
| 因素 | 同进程 goroutine | 独立进程 |
|------|-----------------|----------|
| 复杂度 | ✅ 最低 | 需要 IPC、进程管理 |
| 资源隔离 | ❌ 共享内存 | ✅ 独立内存空间 |
| 崩溃隔离 | ❌ Worker panic 影响主进程 | ✅ 互不影响 |
| 调试 | ✅ 简单 | 需要多进程调试 |
| 扩展性 | ❌ 无法水平扩展 | ✅ 可启动多个 Worker |

MVP 追求最小可行产品，同进程 goroutine 足够。

## 当前实现
- `internal/worker/worker.go` — goroutine 内实现
- 每个扫描任务启动一个 goroutine 执行 Worker.Run()
- 使用 `exec.CommandContext()` 启动外部工具子进程

## 风险
- Worker 子进程失控（如无限循环）可能影响主进程
- 内存泄漏无法隔离

## 缓解措施
- 超时控制（per-task）
- 输出大小截断（100MB）
- v0.2 考虑拆分为独立进程

## 迁移路径
v0.2 时评估拆分为独立 Worker 进程，通过 HTTP 或 gRPC 与 Control Plane 通信。

# ADR-004: SQLite WAL 模式

## 状态
✅ 已决策（2026-04-26）

## 背景
桌面应用需要本地数据库，选择 SQLite。需要确定 journaling 模式。

## 决策
**使用 SQLite WAL (Write-Ahead Logging) 模式。**

## 理由
| 因素 | WAL 模式 | DELETE/Rollback 模式 |
|------|----------|---------------------|
| 读写并发 | ✅ 读写可并发 | ❌ 写时阻塞读 |
| 性能 | ✅ 写性能更好 | 写性能较差 |
| 崩溃恢复 | ✅ 更可靠 | 一般 |
| 文件数量 | ⚠️ 额外 .wal/.shm 文件 | 单一文件 |
| 兼容性 | ⚠️ 需支持 WAL 的 SQLite 版本 | 最通用 |

桌面应用需要同时处理 UI 查询和后台写入，WAL 模式是最佳选择。

## 当前实现
- data_dir = `~/.secbench`
- 数据库文件：`~/.secbench/secbench.db`
- WAL 文件：`~/.secbench/secbench.db-wal`、`-shm`
- 初始化时检查目录可写权限

## 风险
- WAL 文件可能与主数据库文件不同步（如进程崩溃后）
- 某些文件系统（如 NFS）对 WAL 支持不佳

## 缓解措施
- 启动时检查 WAL 文件完整性
- 定期执行 `PRAGMA wal_checkpoint`

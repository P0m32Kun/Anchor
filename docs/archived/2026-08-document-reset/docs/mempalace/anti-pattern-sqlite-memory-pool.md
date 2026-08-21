# Anti-Pattern: Go + SQLite :memory: 连接池陷阱

## 类型
anti-pattern（跨项目可复用）

## 现象

`sql.Open("sqlite3", ":memory:")` 返回 `*sql.DB` 连接池。每个物理连接获得**独立的空数据库**。

典型表现：
- 迁移（`Migrate(db)`）成功，表存在
- 后续查询报 `no such table: xxx`
- 调试时 `SELECT name FROM sqlite_master` 能看到表，但 handler 里看不到

## 根因

Go `database/sql` 管理连接池。`:memory:` SQLite 每个连接 = 独立数据库实例。迁移在连接 A 执行，查询从池中取到连接 B → 连接 B 是空库。

## 修复

```go
rawDB, _ := sql.Open("sqlite3", ":memory:")
rawDB.SetMaxOpenConns(1) // 强制所有操作共享同一连接
```

## 何时触发

- Go 单元测试使用 `:memory:` SQLite
- `setupTestServer` / `setupTestDB` 等测试 helper
- 任何 `sql.Open("sqlite3", ":memory:")` 的场景

## 日期
2026-06-16

## 来源
Anchor 项目护网优化验收，`internal/api/handlers_test.go` 修复

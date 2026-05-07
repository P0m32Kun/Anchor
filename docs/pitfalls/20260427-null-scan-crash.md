# NULL `created_by` 导致 sql.Scan 崩溃

## 现象
报告生成时 `ListEvidenceByFinding` 查询崩溃：`sql: Scan error on column index 5, name "created_by": converting NULL to string is unsupported`。

## 原因
数据库中 `created_by` 列可为 NULL，但 Scan 到 `models.Evidence.CreatedBy`（`string` 类型）时不支持 NULL 转换。

## 解决
`ListEvidenceByFinding` 使用 `sql.NullString` 接收 `created_by`，通过 `.Valid` 判断后赋值。

## 预防
- 所有可为 NULL 的数据库列 Scan 时必须使用 `sql.NullString`/`sql.NullInt64` 等
- CI 中增加 `go test` 覆盖含 NULL 数据的场景

## 相关文件
- `internal/db/queries.go` (ListEvidenceByFinding)
- `internal/models/models.go` (Evidence)

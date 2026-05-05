# Schema Migrations

## Strategy

Anchor uses SQLite's built-in `PRAGMA user_version` to track the database schema version. This is a lightweight, dependency-free approach suitable for a desktop application with a single local SQLite database.

### How it works

1. On startup, `migrate()` reads `PRAGMA user_version`.
2. It compares the current version against the target version.
3. For each missing version, it runs the corresponding migration step (typically `ALTER TABLE` or data seeding).
4. After each successful step, it updates `PRAGMA user_version`.
5. If a step fails, the version is **not** bumped. On the next startup, the migration resumes from the last successful version.

All migration steps are designed to be **idempotent** (e.g., using `CREATE TABLE IF NOT EXISTS` or checking `pragma_table_info` before `ALTER TABLE`), so resuming from a partial migration is safe.

### Adding a new migration

When you need to change the schema:

1. Increment the target version number.
2. Add a new `if version < N` block in `internal/db/db.go` → `migrate()`.
3. Implement the migration logic in a dedicated function (e.g., `migrateV4`).
4. Update the version history table below.
5. Run `go test ./...` and `go build ./...`.

Example:

```go
if version < 4 {
    if err := migrateV4(db); err != nil {
        return fmt.Errorf("migrate v4: %w", err)
    }
    if _, err := db.Exec("PRAGMA user_version = 4"); err != nil {
        return fmt.Errorf("set user_version 4: %w", err)
    }
    version = 4
}
```

## Version History

| Version | Description | Date |
|---------|-------------|------|
| 1 | Initial schema: projects, targets, scope_rules, scan_plans, scan_tasks, tool_invocations, scope_decisions, raw_artifacts, audit_logs, tool_health, assets, ports, services, web_endpoints, findings, evidence, tool_templates, scan_steps, worker_nodes, worker_health_checks, runs, ip_discovery_results, screenshots | 2026-04 |
| 2 | Add `rate_limit` column to `projects` table | 2026-04 |
| 3 | v0.2 schema updates: add `steps_json`, `tool_template_id` to `scan_tasks`; add `raw_request`, `raw_response`, `matched_template` to `findings`; add `run_id` to `scan_tasks`; insert default `tool_templates` | 2026-04 |
| 4 | Recreate `targets` table to add `company` type to CHECK constraint | 2026-04 |
| 5 | Recreate `projects` table to remove `start_time`/`end_time` columns | 2026-04 |
| 6 | Create `pipeline_runs` table for unified scan pipeline | 2026-04 |
| 7 | Create `pipeline_run_stages` table for per-stage tracking | 2026-04 |
| 8 | Add `mode` column to `pipeline_runs` | 2026-05 |
| 9 | Create `engine_credentials` table; migrate FOFA credentials from `projects` | 2026-05 |

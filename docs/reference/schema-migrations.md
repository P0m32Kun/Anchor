# SQLite migration policy

- Status: reference
- Updated: 2026-08-20

`internal/db/` is the source of truth for schema versions, migration order, constraints, and compatibility behavior. Do not copy migration counts or a full schema into prose.

## Rules

- Add a forward migration; do not silently rewrite a migration that may already exist in user databases.
- Make migrations transactional where SQLite permits it and safe to retry after interruption.
- Keep model enums, SQL constraints, inserts, scans, and query filters consistent in the same change.
- Test upgrades from the nearest relevant prior schema and test a fresh database.
- Use temporary database files when WAL, locking, or connection behavior matters; in-memory databases may use separate schemas per connection.
- Define defaults and backfill behavior for existing rows explicitly.
- Preserve user data on failure and return contextual errors.

Replacing SQLite is a product and architecture decision, not an ordinary migration. It requires an ADR that updates the current baseline.

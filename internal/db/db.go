package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func Open(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "secbench.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS projects (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	organization TEXT,
	purpose TEXT,
	start_time DATETIME,
	end_time DATETIME,
	default_profile TEXT DEFAULT 'standard',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS targets (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	type TEXT NOT NULL CHECK(type IN ('domain','url','ip','cidr')),
	value TEXT NOT NULL,
	source TEXT DEFAULT 'manual',
	status TEXT DEFAULT 'active',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scope_rules (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	action TEXT NOT NULL CHECK(action IN ('include','exclude')),
	type TEXT NOT NULL CHECK(type IN ('domain','url','ip','cidr','regex')),
	value TEXT NOT NULL,
	reason TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scan_plans (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	workflow_type TEXT,
	profile TEXT DEFAULT 'standard',
	status TEXT DEFAULT 'draft',
	created_by TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	approved_at DATETIME
);

CREATE TABLE IF NOT EXISTS scan_tasks (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	plan_id TEXT REFERENCES scan_plans(id) ON DELETE CASCADE,
	depends_on_task_id TEXT REFERENCES scan_tasks(id) ON DELETE SET NULL,
	target_id TEXT REFERENCES targets(id) ON DELETE SET NULL,
	tool TEXT NOT NULL,
	command_template TEXT,
	arguments_redacted TEXT,
	status TEXT DEFAULT 'created' CHECK(status IN ('created','queued','running','completed','partial_success','failed','cancelled','scope_denied')),
	started_at DATETIME,
	finished_at DATETIME,
	exit_code INTEGER,
	worker_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tool_invocations (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	task_id TEXT REFERENCES scan_tasks(id) ON DELETE CASCADE,
	tool TEXT NOT NULL,
	binary_path TEXT,
	version TEXT,
	command_redacted TEXT,
	workdir TEXT,
	started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	exit_code INTEGER
);

CREATE TABLE IF NOT EXISTS scope_decisions (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	target_value TEXT NOT NULL,
	task_id TEXT REFERENCES scan_tasks(id) ON DELETE SET NULL,
	decision TEXT NOT NULL CHECK(decision IN ('allow','deny')),
	matched_rule_id TEXT,
	reason TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS raw_artifacts (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	task_id TEXT REFERENCES scan_tasks(id) ON DELETE CASCADE,
	type TEXT NOT NULL CHECK(type IN ('stdout','stderr','jsonl','screenshot','request','response','file')),
	path TEXT NOT NULL,
	sha256 TEXT,
	size INTEGER DEFAULT 0,
	redaction_status TEXT DEFAULT 'unchecked',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_logs (
	id TEXT PRIMARY KEY,
	project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
	actor TEXT,
	action TEXT NOT NULL,
	resource_type TEXT,
	resource_id TEXT,
	summary TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tool_health (
	id TEXT PRIMARY KEY,
	tool TEXT NOT NULL UNIQUE,
	binary_path TEXT,
	version TEXT,
	template_path TEXT,
	workdir_writable INTEGER DEFAULT 0,
	network_available INTEGER DEFAULT 0,
	dns_available INTEGER DEFAULT 0,
	proxy_reachable INTEGER,
	last_check_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_targets_project ON targets(project_id);
CREATE INDEX IF NOT EXISTS idx_scope_rules_project ON scope_rules(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_plan ON scan_tasks(plan_id);
CREATE INDEX IF NOT EXISTS idx_tasks_project ON scan_tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_scope_decisions_project ON scope_decisions(project_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_task ON raw_artifacts(task_id);
CREATE INDEX IF NOT EXISTS idx_audit_project ON audit_logs(project_id);
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	// M1 migration: add rate_limit column to projects if it doesn't exist.
	if err := migrateAddRateLimit(db); err != nil {
		return fmt.Errorf("migrate rate_limit: %w", err)
	}

	return nil
}

// migrateAddRateLimit adds the rate_limit column to projects table.
// SQLite does not support IF NOT EXISTS on ALTER TABLE ADD COLUMN,
// so we check pragma_table_info first.
func migrateAddRateLimit(db *sql.DB) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info('projects') WHERE name = 'rate_limit'`)
	if err != nil {
		return fmt.Errorf("check rate_limit column: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return nil // column already exists
	}
	if _, err := db.Exec(`ALTER TABLE projects ADD COLUMN rate_limit INTEGER DEFAULT 0`); err != nil {
		return fmt.Errorf("add rate_limit column: %w", err)
	}
	return nil
}

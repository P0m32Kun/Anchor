package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func Open(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "anchor.db")
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

func Migrate(db *sql.DB) error { return migrate(db) }

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	if version < 1 {
		if err := migrateV1(db); err != nil {
			return fmt.Errorf("migrate v1: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 1"); err != nil {
			return fmt.Errorf("set user_version 1: %w", err)
		}
		version = 1
	}

	if version < 2 {
		if err := migrateAddRateLimit(db); err != nil {
			return fmt.Errorf("migrate v2 (rate_limit): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 2"); err != nil {
			return fmt.Errorf("set user_version 2: %w", err)
		}
		version = 2
	}

	if version < 3 {
		if err := migrateV02(db); err != nil {
			return fmt.Errorf("migrate v3 (v0.2): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 3"); err != nil {
			return fmt.Errorf("set user_version 3: %w", err)
		}
		version = 3
	}

	if version < 4 {
		if err := migrateV04(db); err != nil {
			return fmt.Errorf("migrate v4 (v0.4 pipeline): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 4"); err != nil {
			return fmt.Errorf("set user_version 4: %w", err)
		}
		version = 4
	}

	if version < 5 {
		if err := migrateV05(db); err != nil {
			return fmt.Errorf("migrate v5 (drop start_time/end_time): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 5"); err != nil {
			return fmt.Errorf("set user_version 5: %w", err)
		}
		version = 5
	}

	if version < 6 {
		if err := migrateV06(db); err != nil {
			return fmt.Errorf("migrate v6 (pipeline_runs): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 6"); err != nil {
			return fmt.Errorf("set user_version 6: %w", err)
		}
		version = 6
	}

	if version < 7 {
		if err := migrateV07(db); err != nil {
			return fmt.Errorf("migrate v7 (pipeline_run_stages): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 7"); err != nil {
			return fmt.Errorf("set user_version 7: %w", err)
		}
		version = 7
	}

	if version < 8 {
		if err := migrateV08(db); err != nil {
			return fmt.Errorf("migrate v8 (pipeline_runs mode): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 8"); err != nil {
			return fmt.Errorf("set user_version 8: %w", err)
		}
		version = 8
	}

	if version < 9 {
		if err := migrateV09(db); err != nil {
			return fmt.Errorf("migrate v9 (engine_credentials): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 9"); err != nil {
			return fmt.Errorf("set user_version 9: %w", err)
		}
		version = 9
	}

	if version < 10 {
		if err := migrateV10(db); err != nil {
			return fmt.Errorf("migrate v10 (nuclei custom): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 10"); err != nil {
			return fmt.Errorf("set user_version 10: %w", err)
		}
		version = 10
	}

	if version < 11 {
		if err := migrateV11(db); err != nil {
			return fmt.Errorf("migrate v11 (drop fofa email): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 11"); err != nil {
			return fmt.Errorf("set user_version 11: %w", err)
		}
		version = 11
	}

	if version < 12 {
		if err := migrateV12(db); err != nil {
			return fmt.Errorf("migrate v12 (fix scan_tasks.run_id FK): %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version = 12"); err != nil {
			return fmt.Errorf("set user_version 12: %w", err)
		}
		version = 12
	}

	if err := ensureProjectsColumns(db); err != nil {
		return fmt.Errorf("ensure projects columns: %w", err)
	}

	return nil
}

func migrateV1(db *sql.DB) error {
	schema := `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS projects (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	organization TEXT,
	purpose TEXT,
	default_profile TEXT DEFAULT 'standard',
	port_range TEXT,
	fofa_email TEXT,
	fofa_api_key TEXT,
	pipeline_config TEXT,
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

CREATE TABLE IF NOT EXISTS assets (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('domain', 'ip', 'url')),
    value TEXT NOT NULL,
    normalized_value TEXT NOT NULL,
    source_tools TEXT,
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    tags TEXT,
    UNIQUE(project_id, normalized_value)
);

CREATE TABLE IF NOT EXISTS ports (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    port INTEGER NOT NULL,
    protocol TEXT DEFAULT 'tcp',
    state TEXT DEFAULT 'open' CHECK(state IN ('open', 'closed', 'filtered')),
    source_tool TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(asset_id, port)
);

CREATE TABLE IF NOT EXISTS services (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    port_id TEXT REFERENCES ports(id) ON DELETE SET NULL,
    name TEXT,
    product TEXT,
    version TEXT,
    banner TEXT,
    confidence INTEGER DEFAULT 0,
    source_tool TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS web_endpoints (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    scheme TEXT,
    host TEXT,
    port INTEGER,
    path TEXT,
    status_code INTEGER,
    title TEXT,
    technologies TEXT,
    screenshot_artifact_id TEXT REFERENCES raw_artifacts(id) ON DELETE SET NULL,
    source_tool TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, url)
);

CREATE INDEX IF NOT EXISTS idx_assets_project ON assets(project_id);
CREATE INDEX IF NOT EXISTS idx_assets_normalized ON assets(project_id, normalized_value);
CREATE INDEX IF NOT EXISTS idx_ports_asset ON ports(asset_id);
CREATE INDEX IF NOT EXISTS idx_services_asset ON services(asset_id);
CREATE INDEX IF NOT EXISTS idx_web_endpoints_asset ON web_endpoints(asset_id);
CREATE INDEX IF NOT EXISTS idx_web_endpoints_url ON web_endpoints(project_id, url);

CREATE TABLE IF NOT EXISTS findings (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    asset_id TEXT REFERENCES assets(id) ON DELETE SET NULL,
    service_id TEXT REFERENCES services(id) ON DELETE SET NULL,
    web_endpoint_id TEXT REFERENCES web_endpoints(id) ON DELETE SET NULL,
    source_tool TEXT NOT NULL,
    source_rule_id TEXT,
    dedup_key TEXT NOT NULL,
    title TEXT NOT NULL,
    severity TEXT NOT NULL CHECK(severity IN ('info','low','medium','high','critical')),
    confidence INTEGER DEFAULT 0,
    priority INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending_review' CHECK(status IN ('new','pending_review','confirmed','false_positive','accepted_risk','ignored','reported')),
    summary TEXT,
    remediation TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, dedup_key)
);

CREATE TABLE IF NOT EXISTS evidence (
    id TEXT PRIMARY KEY,
    finding_id TEXT NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('request','response','screenshot','raw_output','note','file')),
    artifact_id TEXT REFERENCES raw_artifacts(id) ON DELETE SET NULL,
    excerpt TEXT,
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_findings_project ON findings(project_id);
CREATE INDEX IF NOT EXISTS idx_findings_dedup ON findings(project_id, dedup_key);
CREATE INDEX IF NOT EXISTS idx_findings_status ON findings(status);
CREATE INDEX IF NOT EXISTS idx_evidence_finding ON evidence(finding_id);

-- v0.2 M0: tool_templates
CREATE TABLE IF NOT EXISTS tool_templates (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	profile_type TEXT NOT NULL CHECK(profile_type IN ('external','internal','restricted','lab')),
	tools_json TEXT NOT NULL,           -- [{"tool":"subfinder","enabled":true,"rate":100}]
	default_max_concurrency INTEGER DEFAULT 10,
	screenshot_enabled BOOLEAN DEFAULT FALSE,
	directory_bruteforce_enabled BOOLEAN DEFAULT FALSE,
	nuclei_severity_filter TEXT DEFAULT 'critical,high',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- v0.2 M0: scan_steps
CREATE TABLE IF NOT EXISTS scan_steps (
	id TEXT PRIMARY KEY,
	task_id TEXT NOT NULL REFERENCES scan_tasks(id) ON DELETE CASCADE,
	name TEXT NOT NULL CHECK(name IN ('scope_check','prepare_input','run_tool','collect_artifacts','parse_output','normalize_result','score_result','cleanup')),
	status TEXT DEFAULT 'pending' CHECK(status IN ('pending','running','completed','failed','skipped')),
	started_at DATETIME,
	finished_at DATETIME,
	error_code TEXT,
	error_summary TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scan_steps_task ON scan_steps(task_id);
CREATE INDEX IF NOT EXISTS idx_tool_templates_profile ON tool_templates(profile_type);

-- v0.2 M2: worker_nodes
CREATE TABLE IF NOT EXISTS worker_nodes (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	endpoint TEXT,
	mode TEXT NOT NULL DEFAULT 'remote',
	status TEXT DEFAULT 'offline' CHECK(status IN ('online','offline','busy','error')),
	trust_level TEXT DEFAULT 'standard' CHECK(trust_level IN ('low','standard','high')),
	network_profile TEXT DEFAULT 'external',
	capabilities TEXT,
	tool_versions TEXT,
	template_versions TEXT,
	max_concurrency INTEGER DEFAULT 10,
	last_seen DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	revoked_at DATETIME
);

-- v0.2 M2: worker_health_checks
CREATE TABLE IF NOT EXISTS worker_health_checks (
	id TEXT PRIMARY KEY,
	worker_id TEXT NOT NULL REFERENCES worker_nodes(id) ON DELETE CASCADE,
	tool TEXT NOT NULL,
	status TEXT NOT NULL CHECK(status IN ('ready','missing','version_mismatch','config_error','permission_error')),
	version TEXT,
	details TEXT,
	checked_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_worker_health_worker ON worker_health_checks(worker_id);

-- v0.2 M3: runs
CREATE TABLE IF NOT EXISTS runs (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	tool_template_id TEXT REFERENCES tool_templates(id) ON DELETE SET NULL,
	name TEXT NOT NULL,
	status TEXT DEFAULT 'pending' CHECK(status IN ('pending','running','completed','failed','cancelled')),
	started_at DATETIME,
	finished_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- v0.2 M4: ip_discovery_results
CREATE TABLE IF NOT EXISTS ip_discovery_results (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	target_id TEXT REFERENCES targets(id) ON DELETE SET NULL,
	ip TEXT NOT NULL,
	hostname TEXT,
	source TEXT DEFAULT 'naabu',
	alive BOOLEAN DEFAULT TRUE,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ip_discovery_project ON ip_discovery_results(project_id);
CREATE INDEX IF NOT EXISTS idx_ip_discovery_target ON ip_discovery_results(target_id);

-- v0.2 M3: screenshots
CREATE TABLE IF NOT EXISTS screenshots (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	asset_id TEXT REFERENCES assets(id) ON DELETE SET NULL,
	task_id TEXT REFERENCES scan_tasks(id) ON DELETE SET NULL,
	url TEXT NOT NULL,
	original_path TEXT NOT NULL,
	thumbnail_path TEXT NOT NULL,
	width INTEGER,
	height INTEGER,
	taken_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runs_project ON runs(project_id);
CREATE INDEX IF NOT EXISTS idx_screenshots_project ON screenshots(project_id);
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema v1: %w", err)
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

// migrateV02 handles v0.2 schema migrations.
func migrateV02(db *sql.DB) error {
	// Add steps_json and tool_template_id to scan_tasks
	cols := []struct {
		table string
		name  string
		def   string
	}{
		{"projects", "port_range", "TEXT"},
		{"scan_tasks", "steps_json", "TEXT"},
		{"scan_tasks", "tool_template_id", "TEXT"},
		{"findings", "raw_request", "TEXT"},
		{"findings", "raw_response", "TEXT"},
		{"findings", "matched_template", "TEXT"},
	}
	for _, col := range cols {
		var exists bool
		err := db.QueryRow(
			`SELECT 1 FROM pragma_table_info(?) WHERE name = ?`,
			col.table, col.name,
		).Scan(&exists)
		if err == nil && exists {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, col.table, col.name, col.def)); err != nil {
			return fmt.Errorf("add column %s to %s: %w", col.name, col.table, err)
		}
	}

	// Insert default tool templates if none exist
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tool_templates`).Scan(&count); err != nil {
		return fmt.Errorf("count tool_templates: %w", err)
	}
	if count == 0 {
		templates := []struct {
			id, name, desc, profile, tools, severity string
			concurrency                              int
			screenshot                               bool
		}{
			{
				id: "external-quick", name: "外网快速摸底", desc: "轻量端口扫描 + 存活探测 + 高危漏洞",
				profile: "external", tools: `[{"tool":"naabu","enabled":true,"rate":1000},{"tool":"httpx","enabled":true,"rate":50},{"tool":"nuclei","enabled":true,"rate":50}]`,
				severity: "critical,high", concurrency: 10, screenshot: false,
			},
			{
				id: "external-standard", name: "外网标准初筛", desc: "完整资产发现 + 标准漏洞扫描",
				profile: "external", tools: `[{"tool":"subfinder","enabled":true,"rate":100},{"tool":"naabu","enabled":true,"rate":1000},{"tool":"httpx","enabled":true,"rate":50},{"tool":"nuclei","enabled":true,"rate":50}]`,
				severity: "critical,high,medium", concurrency: 20, screenshot: true,
			},
			{
				id: "internal-slow", name: "内网慢扫", desc: "低并发内网巡检，禁用高影响模板",
				profile: "internal", tools: `[{"tool":"naabu","enabled":true,"rate":100},{"tool":"httpx","enabled":true,"rate":10},{"tool":"nuclei","enabled":true,"rate":10}]`,
				severity: "critical,high", concurrency: 5, screenshot: false,
			},
			{
				id: "retest", name: "复测模式", desc: "只扫描已确认 Finding 对应目标",
				profile: "external", tools: `[{"tool":"nuclei","enabled":true,"rate":10}]`,
				severity: "critical,high,medium", concurrency: 5, screenshot: false,
			},
		}
		for _, t := range templates {
			_, err := db.Exec(`
				INSERT INTO tool_templates (id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, nuclei_severity_filter)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, t.id, t.name, t.desc, t.profile, t.tools, t.concurrency, t.screenshot, t.severity)
			if err != nil {
				return fmt.Errorf("insert template %s: %w", t.id, err)
			}
		}
	}
	if err := migrateV02RunID(db); err != nil {
		return fmt.Errorf("migrate v0.2 run_id: %w", err)
	}
	return nil
}

func migrateV02RunID(db *sql.DB) error {
	// Check if column exists
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('scan_tasks') WHERE name = 'run_id'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = db.Exec(`ALTER TABLE scan_tasks ADD COLUMN run_id TEXT REFERENCES runs(id) ON DELETE CASCADE`)
		return err
	}
	return nil
}

func migrateV04(db *sql.DB) error {
	// 1. Update targets table CHECK constraint to support 'company' type
	// SQLite doesn't support ALTER TABLE DROP CONSTRAINT, so we need to recreate
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS targets_new (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			type TEXT NOT NULL CHECK(type IN ('domain','url','ip','cidr','company')),
			value TEXT NOT NULL,
			source TEXT DEFAULT 'manual',
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create targets_new: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO targets_new (id, project_id, type, value, source, status, created_at)
		SELECT id, project_id, type, value, source, status, created_at FROM targets;
	`)
	if err != nil {
		return fmt.Errorf("copy targets: %w", err)
	}

	_, err = db.Exec(`DROP TABLE targets;`)
	if err != nil {
		return fmt.Errorf("drop old targets: %w", err)
	}

	_, err = db.Exec(`ALTER TABLE targets_new RENAME TO targets;`)
	if err != nil {
		return fmt.Errorf("rename targets_new: %w", err)
	}

	// Recreate index
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_targets_project ON targets(project_id);`)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	// 2. Add FOFA config to projects
	var colCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'fofa_email'`).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("check fofa_email column: %w", err)
	}
	if colCount == 0 {
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN fofa_email TEXT;`)
		if err != nil {
			return fmt.Errorf("add fofa_email: %w", err)
		}
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN fofa_api_key TEXT;`)
		if err != nil {
			return fmt.Errorf("add fofa_api_key: %w", err)
		}
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN pipeline_config TEXT;`)
		if err != nil {
			return fmt.Errorf("add pipeline_config: %w", err)
		}
	}

	// 3. Create DNS records table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS dns_records (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			domain TEXT NOT NULL,
			ips TEXT NOT NULL,
			cnames TEXT,
			ttl INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, domain)
		);
	`)
	if err != nil {
		return fmt.Errorf("create dns_records: %w", err)
	}

	// 4. Create CDN results table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cdn_results (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			ip TEXT NOT NULL,
			is_cdn BOOLEAN DEFAULT FALSE,
			cdn_provider TEXT,
			cdn_type TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, ip)
		);
	`)
	if err != nil {
		return fmt.Errorf("create cdn_results: %w", err)
	}

	// 5. Create service fingerprints table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS service_fingerprints (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT DEFAULT 'tcp',
			is_web BOOLEAN DEFAULT FALSE,
			service TEXT NOT NULL,
			metadata TEXT,
			source TEXT DEFAULT 'nerva',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, ip, port)
		);
	`)
	if err != nil {
		return fmt.Errorf("create service_fingerprints: %w", err)
	}

	return nil
}

func migrateV05(db *sql.DB) error {
	// Check if start_time column exists in projects
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'start_time'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		// Column already removed, nothing to do
		return nil
	}

	// SQLite doesn't support DROP COLUMN, so we need to recreate the table
	_, err = db.Exec(`
		CREATE TABLE projects_new (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			organization TEXT,
			purpose TEXT,
			default_profile TEXT DEFAULT 'standard',
			port_range TEXT,
			fofa_email TEXT,
			fofa_api_key TEXT,
			pipeline_config TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create projects_new: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO projects_new (id, name, organization, purpose, default_profile, port_range, fofa_email, fofa_api_key, pipeline_config, created_at, updated_at)
		SELECT id, name, organization, purpose, default_profile, port_range, fofa_email, fofa_api_key, pipeline_config, created_at, updated_at
		FROM projects;
	`)
	if err != nil {
		return fmt.Errorf("copy projects: %w", err)
	}

	_, err = db.Exec(`DROP TABLE projects;`)
	if err != nil {
		return fmt.Errorf("drop old projects: %w", err)
	}

	_, err = db.Exec(`ALTER TABLE projects_new RENAME TO projects;`)
	if err != nil {
		return fmt.Errorf("rename projects_new: %w", err)
	}

	return nil
}

func migrateV06(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_runs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			status TEXT NOT NULL DEFAULT 'running',
			stage TEXT,
			error TEXT,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_pipeline_runs_project ON pipeline_runs(project_id);
	`)
	if err != nil {
		return fmt.Errorf("create pipeline_runs: %w", err)
	}
	return nil
}

// ensureProjectsColumns verifies that all expected columns exist on the
// projects table and adds any that are missing. This acts as a final
// safety net for databases that may have skipped an intermediate
// migration or had their schema altered outside the normal flow.
func ensureProjectsColumns(db *sql.DB) error {
	columns := []struct {
		name string
		def  string
	}{
		{"rate_limit", "INTEGER DEFAULT 0"},
	}

	for _, col := range columns {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = ?",
			col.name,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("check column %s: %w", col.name, err)
		}
		if count == 0 {
			_, err := db.Exec(fmt.Sprintf("ALTER TABLE projects ADD COLUMN %s %s", col.name, col.def))
			if err != nil {
				return fmt.Errorf("add column %s: %w", col.name, err)
			}
		}
	}
	return nil
}

func migrateV08(db *sql.DB) error {
	// Add mode column to pipeline_runs
	_, err := db.Exec(`ALTER TABLE pipeline_runs ADD COLUMN mode TEXT NOT NULL DEFAULT 'standard'`)
	if err != nil {
		// Column may already exist
		log.Printf("migration v8 (mode column): %v", err)
	}
	return nil
}

func migrateV09(db *sql.DB) error {
	// 1. Create engine_credentials table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS engine_credentials (
			id TEXT PRIMARY KEY,
			engine TEXT NOT NULL UNIQUE,
			api_key TEXT NOT NULL,
			email TEXT,
			extra TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create engine_credentials: %w", err)
	}

	// 2. Migrate existing FOFA credentials from projects table
	var fofaEmail, fofaAPIKey string
	err = db.QueryRow(`SELECT fofa_email, fofa_api_key FROM projects WHERE fofa_email IS NOT NULL AND fofa_email != '' AND fofa_api_key IS NOT NULL AND fofa_api_key != '' LIMIT 1`).Scan(&fofaEmail, &fofaAPIKey)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("migrate fofa credentials: %w", err)
	}
	if err == nil && fofaEmail != "" && fofaAPIKey != "" {
		now := time.Now().UTC()
		_, err = db.Exec(`
			INSERT INTO engine_credentials (id, engine, api_key, email, created_at, updated_at)
			VALUES (?, 'fofa', ?, ?, ?, ?)
			ON CONFLICT(engine) DO UPDATE SET
				api_key = excluded.api_key,
				email = excluded.email,
				updated_at = excluded.updated_at;
		`, fmt.Sprintf("cred-%d", now.UnixNano()), fofaAPIKey, fofaEmail, now, now)
		if err != nil {
			return fmt.Errorf("insert fofa credential: %w", err)
		}
		log.Printf("migration v9: migrated FOFA credentials from projects to engine_credentials")
	}

	return nil
}

func migrateV07(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_run_stages (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			stage TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','running','completed','failed','skipped')),
			error TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_pipeline_run_stages_run ON pipeline_run_stages(run_id);
		CREATE INDEX IF NOT EXISTS idx_pipeline_run_stages_stage ON pipeline_run_stages(run_id, stage);
	`)
	if err != nil {
		return fmt.Errorf("create pipeline_run_stages: %w", err)
	}
	return nil
}

// migrateV10 introduces server-side custom Nuclei template management:
//   - nuclei_custom_sources: user-defined source registry
//   - nuclei_custom_bundles: immutable published bundles (Phase 2 fills this)
//   - scan_tasks.nuclei_custom_bundle_version: per-task bundle attribution
//     (Phase 4 populates; Phase 1 ships the column to avoid a follow-up migration)
func migrateV10(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS nuclei_custom_sources (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('git','upload','file')),
			uri TEXT,
			branch TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			routing_policy TEXT NOT NULL DEFAULT 'manual',
			status TEXT NOT NULL DEFAULT 'draft',
			last_sync_at DATETIME,
			last_validate_at DATETIME,
			last_error TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_nuclei_custom_sources_status ON nuclei_custom_sources(status);
	`)
	if err != nil {
		return fmt.Errorf("create nuclei_custom_sources: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS nuclei_custom_bundles (
			version TEXT PRIMARY KEY,
			manifest_json TEXT NOT NULL,
			archive_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL,
			activated_at DATETIME
		);
	`)
	if err != nil {
		return fmt.Errorf("create nuclei_custom_bundles: %w", err)
	}

	var colCount int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('scan_tasks') WHERE name = 'nuclei_custom_bundle_version'`,
	).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("check nuclei_custom_bundle_version column: %w", err)
	}
	if colCount == 0 {
		if _, err := db.Exec(`ALTER TABLE scan_tasks ADD COLUMN nuclei_custom_bundle_version TEXT`); err != nil {
			return fmt.Errorf("add nuclei_custom_bundle_version column: %w", err)
		}
	}

	return nil
}

// migrateV11 drops legacy FOFA email columns now that the FOFA API no longer
// requires an email parameter. It rebuilds projects without fofa_email/
// fofa_api_key and engine_credentials without email.
func migrateV11(db *sql.DB) error {
	// 1. Rebuild projects table without fofa_email and fofa_api_key
	var fofaEmailCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name = 'fofa_email'`).Scan(&fofaEmailCount); err != nil {
		return fmt.Errorf("check fofa_email column: %w", err)
	}
	if fofaEmailCount > 0 {
		_, err := db.Exec(`
			CREATE TABLE projects_new (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				organization TEXT,
				purpose TEXT,
				default_profile TEXT DEFAULT 'standard',
				port_range TEXT,
				rate_limit INTEGER DEFAULT 0,
				pipeline_config TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("create projects_new: %w", err)
		}

		_, err = db.Exec(`
			INSERT INTO projects_new (id, name, organization, purpose, default_profile, port_range, rate_limit, pipeline_config, created_at, updated_at)
			SELECT id, name, organization, purpose, default_profile, port_range, COALESCE(rate_limit, 0), pipeline_config, created_at, updated_at
			FROM projects;
		`)
		if err != nil {
			return fmt.Errorf("copy projects: %w", err)
		}

		if _, err := db.Exec(`DROP TABLE projects;`); err != nil {
			return fmt.Errorf("drop old projects: %w", err)
		}

		if _, err := db.Exec(`ALTER TABLE projects_new RENAME TO projects;`); err != nil {
			return fmt.Errorf("rename projects_new: %w", err)
		}
	}

	// 2. Rebuild engine_credentials table without email
	var emailCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('engine_credentials') WHERE name = 'email'`).Scan(&emailCount); err != nil {
		return fmt.Errorf("check engine_credentials.email column: %w", err)
	}
	if emailCount > 0 {
		_, err := db.Exec(`
			CREATE TABLE engine_credentials_new (
				id TEXT PRIMARY KEY,
				engine TEXT NOT NULL UNIQUE,
				api_key TEXT NOT NULL,
				extra TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("create engine_credentials_new: %w", err)
		}

		_, err = db.Exec(`
			INSERT INTO engine_credentials_new (id, engine, api_key, extra, created_at, updated_at)
			SELECT id, engine, api_key, extra, created_at, updated_at
			FROM engine_credentials;
		`)
		if err != nil {
			return fmt.Errorf("copy engine_credentials: %w", err)
		}

		if _, err := db.Exec(`DROP TABLE engine_credentials;`); err != nil {
			return fmt.Errorf("drop old engine_credentials: %w", err)
		}

		if _, err := db.Exec(`ALTER TABLE engine_credentials_new RENAME TO engine_credentials;`); err != nil {
			return fmt.Errorf("rename engine_credentials_new: %w", err)
		}
	}

	return nil
}

// migrateV12 fixes a long-standing bug where scan_tasks.run_id was declared as
// REFERENCES runs(id) by an early v0.2 migration. The application now records
// pipeline run IDs from pipeline_runs (added in v6), so every scan_tasks INSERT
// fails with FOREIGN KEY constraint failed and the entire scan silently does
// nothing. This rebuilds scan_tasks with run_id REFERENCES pipeline_runs(id).
func migrateV12(db *sql.DB) error {
	// Detect whether the bad FK is present. If the column is missing or already
	// points at pipeline_runs there is nothing to do.
	rows, err := db.Query(`SELECT "table" FROM pragma_foreign_key_list('scan_tasks') WHERE "from" = 'run_id'`)
	if err != nil {
		return fmt.Errorf("query scan_tasks foreign keys: %w", err)
	}
	var refTable string
	if rows.Next() {
		_ = rows.Scan(&refTable)
	}
	rows.Close()
	if refTable == "" || refTable == "pipeline_runs" {
		return nil
	}

	// Rebuild scan_tasks with the correct FK. SQLite requires foreign_keys=OFF
	// during the table rebuild because other tables reference scan_tasks.
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys: %w", err)
	}
	defer func() { _, _ = db.Exec(`PRAGMA foreign_keys = ON`) }()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Exec(`
		CREATE TABLE scan_tasks_new (
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			steps_json TEXT,
			tool_template_id TEXT,
			run_id TEXT REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			nuclei_custom_bundle_version TEXT
		);
	`); err != nil {
		return fmt.Errorf("create scan_tasks_new: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO scan_tasks_new (id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, worker_id, created_at, steps_json, tool_template_id, run_id, nuclei_custom_bundle_version)
		SELECT id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, worker_id, created_at, steps_json, tool_template_id,
			CASE WHEN run_id IN (SELECT id FROM pipeline_runs) THEN run_id ELSE NULL END,
			nuclei_custom_bundle_version
		FROM scan_tasks;
	`); err != nil {
		return fmt.Errorf("copy scan_tasks: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE scan_tasks`); err != nil {
		return fmt.Errorf("drop scan_tasks: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE scan_tasks_new RENAME TO scan_tasks`); err != nil {
		return fmt.Errorf("rename scan_tasks_new: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_plan ON scan_tasks(plan_id)`); err != nil {
		return fmt.Errorf("recreate idx_tasks_plan: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_project ON scan_tasks(project_id)`); err != nil {
		return fmt.Errorf("recreate idx_tasks_project: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	committed = true

	return nil
}

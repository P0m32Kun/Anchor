package db

import (
	"database/sql"
	"fmt"
)

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

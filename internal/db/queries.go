package db

import (
	"database/sql"
	"fmt"
	"time"

	"secbench/internal/models"
)

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type Queries struct{ db DBTX }

func New(db DBTX) *Queries { return &Queries{db: db} }

// --- Projects ---

func (q *Queries) CreateProject(p *models.Project) error {
	_, err := q.db.Exec(`
		INSERT INTO projects (id, name, organization, purpose, start_time, end_time, default_profile, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Organization, p.Purpose, p.StartTime, p.EndTime, p.DefaultProfile, p.CreatedAt, p.UpdatedAt)
	return err
}

func (q *Queries) GetProject(id string) (*models.Project, error) {
	row := q.db.QueryRow(`SELECT id, name, organization, purpose, start_time, end_time, default_profile, created_at, updated_at FROM projects WHERE id = ?`, id)
	p := &models.Project{}
	err := row.Scan(&p.ID, &p.Name, &p.Organization, &p.Purpose, &p.StartTime, &p.EndTime, &p.DefaultProfile, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (q *Queries) ListProjects() ([]*models.Project, error) {
	rows, err := q.db.Query(`SELECT id, name, organization, purpose, start_time, end_time, default_profile, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Project
	for rows.Next() {
		p := &models.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Organization, &p.Purpose, &p.StartTime, &p.EndTime, &p.DefaultProfile, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// --- Targets ---

func (q *Queries) CreateTarget(t *models.Target) error {
	_, err := q.db.Exec(`INSERT INTO targets (id, project_id, type, value, source, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.Type, t.Value, t.Source, t.Status, t.CreatedAt)
	return err
}

func (q *Queries) GetTarget(id string) (*models.Target, error) {
	row := q.db.QueryRow(`SELECT id, project_id, type, value, source, status, created_at FROM targets WHERE id = ?`, id)
	t := &models.Target{}
	err := row.Scan(&t.ID, &t.ProjectID, &t.Type, &t.Value, &t.Source, &t.Status, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (q *Queries) ListTargetsByProject(projectID string) ([]*models.Target, error) {
	rows, err := q.db.Query(`SELECT id, project_id, type, value, source, status, created_at FROM targets WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Target
	for rows.Next() {
		t := &models.Target{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Type, &t.Value, &t.Source, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// --- Scope Rules ---

func (q *Queries) CreateScopeRule(r *models.ScopeRule) error {
	_, err := q.db.Exec(`INSERT INTO scope_rules (id, project_id, action, type, value, reason, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ProjectID, r.Action, r.Type, r.Value, r.Reason, r.CreatedAt, r.UpdatedAt)
	return err
}

func (q *Queries) ListScopeRulesByProject(projectID string) ([]*models.ScopeRule, error) {
	rows, err := q.db.Query(`SELECT id, project_id, action, type, value, reason, created_at, updated_at FROM scope_rules WHERE project_id = ? ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.ScopeRule
	for rows.Next() {
		r := &models.ScopeRule{}
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Action, &r.Type, &r.Value, &r.Reason, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) GetMaxScopeRuleUpdatedAt(projectID string) (time.Time, error) {
	var s sql.NullString
	err := q.db.QueryRow(`SELECT MAX(updated_at) FROM scope_rules WHERE project_id = ?`, projectID).Scan(&s)
	if err != nil || !s.Valid || s.String == "" {
		return time.Time{}, err
	}
	// SQLite stores datetime in various formats; try common ones.
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999+00:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s.String); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s.String)
}

// --- Scan Plans ---

func (q *Queries) CreateScanPlan(p *models.ScanPlan) error {
	_, err := q.db.Exec(`INSERT INTO scan_plans (id, project_id, workflow_type, profile, status, created_by, created_at, approved_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.ProjectID, p.WorkflowType, p.Profile, p.Status, p.CreatedBy, p.CreatedAt, p.ApprovedAt)
	return err
}

// --- Scan Tasks ---

func (q *Queries) CreateScanTask(t *models.ScanTask) error {
	_, err := q.db.Exec(`INSERT INTO scan_tasks (id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.PlanID, t.DependsOnTaskID, t.TargetID, t.Tool, t.CommandTemplate, t.ArgumentsRedacted, t.Status, t.CreatedAt)
	return err
}

func (q *Queries) GetScanTask(id string) (*models.ScanTask, error) {
	row := q.db.QueryRow(`SELECT id, project_id, plan_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, worker_id, created_at FROM scan_tasks WHERE id = ?`, id)
	t := &models.ScanTask{}
	err := row.Scan(&t.ID, &t.ProjectID, &t.PlanID, &t.DependsOnTaskID, &t.TargetID, &t.Tool, &t.CommandTemplate, &t.ArgumentsRedacted, &t.Status, &t.StartedAt, &t.FinishedAt, &t.ExitCode, &t.WorkerID, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (q *Queries) UpdateScanTaskStatus(id string, status models.TaskStatus, exitCode *int, finishedAt *time.Time) error {
	_, err := q.db.Exec(`UPDATE scan_tasks SET status = ?, exit_code = ?, finished_at = ? WHERE id = ?`, status, exitCode, finishedAt, id)
	return err
}

func (q *Queries) SetScanTaskRunning(id string, startedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE scan_tasks SET status = ?, started_at = ? WHERE id = ?`, models.TaskRunning, startedAt, id)
	return err
}

func (q *Queries) ListScanTasksByPlan(planID string) ([]*models.ScanTask, error) {
	rows, err := q.db.Query(`SELECT id, project_id, plan_id, depends_on_task_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, worker_id, created_at FROM scan_tasks WHERE plan_id = ? ORDER BY created_at`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.ScanTask
	for rows.Next() {
		t := &models.ScanTask{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.PlanID, &t.DependsOnTaskID, &t.Tool, &t.CommandTemplate, &t.ArgumentsRedacted, &t.Status, &t.StartedAt, &t.FinishedAt, &t.ExitCode, &t.WorkerID, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// --- Scope Decisions ---

func (q *Queries) CreateScopeDecision(d *models.ScopeDecision) error {
	_, err := q.db.Exec(`INSERT INTO scope_decisions (id, project_id, target_value, task_id, decision, matched_rule_id, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.ProjectID, d.TargetValue, d.TaskID, d.Decision, d.MatchedRuleID, d.Reason, d.CreatedAt)
	return err
}

func (q *Queries) GetLatestScopeDecision(projectID, targetValue string) (*models.ScopeDecision, error) {
	row := q.db.QueryRow(`SELECT id, project_id, target_value, task_id, decision, matched_rule_id, reason, created_at FROM scope_decisions WHERE project_id = ? AND target_value = ? ORDER BY created_at DESC LIMIT 1`, projectID, targetValue)
	d := &models.ScopeDecision{}
	err := row.Scan(&d.ID, &d.ProjectID, &d.TargetValue, &d.TaskID, &d.Decision, &d.MatchedRuleID, &d.Reason, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (q *Queries) ListScopeDecisionsByPlan(projectID string) ([]*models.ScopeDecision, error) {
	rows, err := q.db.Query(`SELECT id, project_id, target_value, task_id, decision, matched_rule_id, reason, created_at FROM scope_decisions WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.ScopeDecision
	for rows.Next() {
		d := &models.ScopeDecision{}
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.TargetValue, &d.TaskID, &d.Decision, &d.MatchedRuleID, &d.Reason, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// --- Raw Artifacts ---

func (q *Queries) CreateRawArtifact(a *models.RawArtifact) error {
	_, err := q.db.Exec(`INSERT INTO raw_artifacts (id, project_id, task_id, type, path, sha256, size, redaction_status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ProjectID, a.TaskID, a.Type, a.Path, a.SHA256, a.Size, a.RedactionStatus, a.CreatedAt)
	return err
}

func (q *Queries) ListRawArtifactsByTask(taskID string) ([]*models.RawArtifact, error) {
	rows, err := q.db.Query(`SELECT id, project_id, task_id, type, path, sha256, size, redaction_status, created_at FROM raw_artifacts WHERE task_id = ? ORDER BY created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.RawArtifact
	for rows.Next() {
		a := &models.RawArtifact{}
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.TaskID, &a.Type, &a.Path, &a.SHA256, &a.Size, &a.RedactionStatus, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// --- Audit Logs ---

func (q *Queries) CreateAuditLog(a *models.AuditLog) error {
	_, err := q.db.Exec(`INSERT INTO audit_logs (id, project_id, actor, action, resource_type, resource_id, summary, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ProjectID, a.Actor, a.Action, a.ResourceType, a.ResourceID, a.Summary, a.CreatedAt)
	return err
}

// --- Tool Health ---

func (q *Queries) UpsertToolHealth(h *models.ToolHealth) error {
	_, err := q.db.Exec(`
		INSERT INTO tool_health (id, tool, binary_path, version, template_path, workdir_writable, network_available, dns_available, proxy_reachable, last_check_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tool) DO UPDATE SET
			binary_path = excluded.binary_path,
			version = excluded.version,
			template_path = excluded.template_path,
			workdir_writable = excluded.workdir_writable,
			network_available = excluded.network_available,
			dns_available = excluded.dns_available,
			proxy_reachable = excluded.proxy_reachable,
			last_check_at = excluded.last_check_at`,
		h.ID, h.Tool, h.BinaryPath, h.Version, h.TemplatePath, h.WorkdirWritable, h.NetworkAvailable, h.DNSAvailable, h.ProxyReachable, h.LastCheckAt)
	return err
}

func (q *Queries) ListToolHealth() ([]*models.ToolHealth, error) {
	rows, err := q.db.Query(`SELECT id, tool, binary_path, version, template_path, workdir_writable, network_available, dns_available, proxy_reachable, last_check_at FROM tool_health ORDER BY tool`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.ToolHealth
	for rows.Next() {
		h := &models.ToolHealth{}
		var proxyReachable sql.NullBool
		var templatePath sql.NullString
		if err := rows.Scan(&h.ID, &h.Tool, &h.BinaryPath, &h.Version, &templatePath, &h.WorkdirWritable, &h.NetworkAvailable, &h.DNSAvailable, &proxyReachable, &h.LastCheckAt); err != nil {
			return nil, err
		}
		h.TemplatePath = nullableString(templatePath)
		h.ProxyReachable = nullableBool(proxyReachable)
		list = append(list, h)
	}
	return list, rows.Err()
}

func (q *Queries) CreateToolInvocation(inv *models.ToolInvocation) error {
	_, err := q.db.Exec(`INSERT INTO tool_invocations (id, project_id, task_id, tool, binary_path, version, command_redacted, workdir, started_at, finished_at, exit_code) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.ProjectID, inv.TaskID, inv.Tool, inv.BinaryPath, inv.Version, inv.CommandRedacted, inv.Workdir, inv.StartedAt, inv.FinishedAt, inv.ExitCode)
	return err
}

func (q *Queries) UpdateToolInvocation(taskID string, finishedAt time.Time, exitCode int) error {
	_, err := q.db.Exec(`UPDATE tool_invocations SET finished_at = ?, exit_code = ? WHERE task_id = ?`,
		finishedAt, exitCode, taskID)
	return err
}

func nullableString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}
func nullableBool(nb sql.NullBool) *bool {
	if nb.Valid {
		v := nb.Bool
		return &v
	}
	return nil
}

// WithTx runs fn inside a transaction. rawDB must be *sql.DB.
func WithTx(rawDB *sql.DB, fn func(*Queries) error) error {
	tx, err := rawDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

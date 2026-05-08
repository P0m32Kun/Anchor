package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type Queries struct{ db DBTX }

func New(db DBTX) *Queries { return &Queries{db: db} }

// --- Scan Plans ---

func (q *Queries) CreateScanPlan(p *models.ScanPlan) error {
	_, err := q.db.Exec(`INSERT INTO scan_plans (id, project_id, workflow_type, profile, status, created_by, created_at, approved_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.ProjectID, p.WorkflowType, p.Profile, p.Status, p.CreatedBy, p.CreatedAt, p.ApprovedAt)
	return err
}

// --- Scan Tasks ---

func (q *Queries) CreateScanTask(t *models.ScanTask) error {
	// Convert empty strings to NULL for foreign key fields
	planID := sqlNullStringValue(t.PlanID)
	runID := sqlNullString(t.RunID)
	dependsOn := sqlNullString(t.DependsOnTaskID)
	targetID := sqlNullString(t.TargetID)
	customVersion := sqlNullString(t.NucleiCustomBundleVersion)
	_, err := q.db.Exec(`INSERT INTO scan_tasks (id, project_id, plan_id, run_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, nuclei_custom_bundle_version, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, planID, runID, dependsOn, targetID, t.Tool, t.CommandTemplate, t.ArgumentsRedacted, t.Status, customVersion, t.CreatedAt)
	return err
}

func (q *Queries) GetScanTask(id string) (*models.ScanTask, error) {
	row := q.db.QueryRow(`SELECT id, project_id, plan_id, run_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, error_message, worker_id, nuclei_custom_bundle_version, created_at FROM scan_tasks WHERE id = ?`, id)
	t := &models.ScanTask{}
	var planID, runID, customVersion sql.NullString
	var startedAt, finishedAt sql.NullTime
	var exitCode sql.NullInt64
	var errorMsg sql.NullString
	err := row.Scan(&t.ID, &t.ProjectID, &planID, &runID, &t.DependsOnTaskID, &t.TargetID, &t.Tool, &t.CommandTemplate, &t.ArgumentsRedacted, &t.Status, &startedAt, &finishedAt, &exitCode, &errorMsg, &t.WorkerID, &customVersion, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if planID.Valid {
		t.PlanID = planID.String
	}
	if runID.Valid {
		t.RunID = &runID.String
	}
	if customVersion.Valid {
		t.NucleiCustomBundleVersion = &customVersion.String
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		t.FinishedAt = &finishedAt.Time
	}
	if exitCode.Valid {
		ec := int(exitCode.Int64)
		t.ExitCode = &ec
	}
	if errorMsg.Valid {
		t.ErrorMessage = errorMsg.String
	}
	return t, nil
}

func (q *Queries) UpdateScanTaskStatus(id string, status models.TaskStatus, exitCode *int, finishedAt *time.Time) error {
	_, err := q.db.Exec(`UPDATE scan_tasks SET status = ?, exit_code = ?, finished_at = ? WHERE id = ?`, status, exitCode, finishedAt, id)
	return err
}

func (q *Queries) UpdateScanTaskErrorMessage(id string, errorMsg string) error {
	_, err := q.db.Exec(`UPDATE scan_tasks SET error_message = ? WHERE id = ?`, errorMsg, id)
	return err
}

func (q *Queries) SetScanTaskRunning(id string, startedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE scan_tasks SET status = ?, started_at = ? WHERE id = ?`, models.TaskRunning, startedAt, id)
	return err
}

func (q *Queries) ListScanTasksByPlan(planID string) ([]*models.ScanTask, error) {
	rows, err := q.db.Query(`SELECT id, project_id, plan_id, depends_on_task_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, error_message, worker_id, nuclei_custom_bundle_version, created_at FROM scan_tasks WHERE plan_id = ? ORDER BY created_at`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ScanTask, 0)
	for rows.Next() {
		t := &models.ScanTask{}
		var customVersion sql.NullString
		var errorMsg sql.NullString
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.PlanID, &t.DependsOnTaskID, &t.Tool, &t.CommandTemplate, &t.ArgumentsRedacted, &t.Status, &t.StartedAt, &t.FinishedAt, &t.ExitCode, &errorMsg, &t.WorkerID, &customVersion, &t.CreatedAt); err != nil {
			return nil, err
		}
		if customVersion.Valid {
			t.NucleiCustomBundleVersion = &customVersion.String
		}
		if errorMsg.Valid {
			t.ErrorMessage = errorMsg.String
		}
		list = append(list, t)
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
	list := make([]*models.RawArtifact, 0)
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
	list := make([]*models.ToolHealth, 0)
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

// ListToolInvocationsByProject returns all tool invocations for a project.
func (q *Queries) ListToolInvocationsByProject(projectID string) ([]*models.ToolInvocation, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, task_id, tool, binary_path, version, command_redacted, workdir, started_at, finished_at, exit_code
		FROM tool_invocations WHERE project_id = ? ORDER BY started_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ToolInvocation, 0)
	for rows.Next() {
		inv := &models.ToolInvocation{}
		if err := rows.Scan(&inv.ID, &inv.ProjectID, &inv.TaskID, &inv.Tool, &inv.BinaryPath, &inv.Version, &inv.CommandRedacted, &inv.Workdir, &inv.StartedAt, &inv.FinishedAt, &inv.ExitCode); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, rows.Err()
}

func nullableString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func sqlNullString(s *string) sql.NullString {
	if s != nil && *s != "" {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{}
}

func sqlNullStringValue(s string) sql.NullString {
	if s != "" {
		return sql.NullString{String: s, Valid: true}
	}
	return sql.NullString{}
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

// --- ToolTemplate ---

func (q *Queries) ListToolTemplates() ([]*models.ToolTemplate, error) {
	rows, err := q.db.Query(`
		SELECT id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, directory_bruteforce_enabled, nuclei_severity_filter, created_at, updated_at
		FROM tool_templates ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ToolTemplate, 0)
	for rows.Next() {
		t := &models.ToolTemplate{}
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Description, &t.ProfileType, &t.ToolsJSON,
			&t.DefaultMaxConcurrency, &t.ScreenshotEnabled, &t.DirectoryBruteforceEnabled,
			&t.NucleiSeverityFilter, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (q *Queries) GetToolTemplate(id string) (*models.ToolTemplate, error) {
	row := q.db.QueryRow(`
		SELECT id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, directory_bruteforce_enabled, nuclei_severity_filter, created_at, updated_at
		FROM tool_templates WHERE id = ?`, id)
	t := &models.ToolTemplate{}
	if err := row.Scan(
		&t.ID, &t.Name, &t.Description, &t.ProfileType, &t.ToolsJSON,
		&t.DefaultMaxConcurrency, &t.ScreenshotEnabled, &t.DirectoryBruteforceEnabled,
		&t.NucleiSeverityFilter, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// --- ScanStep ---

func (q *Queries) CreateScanStep(s *models.ScanStep) error {
	_, err := q.db.Exec(`
		INSERT INTO scan_steps (id, task_id, name, status, started_at, finished_at, error_code, error_summary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TaskID, s.Name, s.Status, s.StartedAt, s.FinishedAt, s.ErrorCode, s.ErrorSummary, s.CreatedAt)
	return err
}

func (q *Queries) UpdateScanStepStatus(id string, status models.StepStatus, finishedAt *time.Time, errorCode, errorSummary string) error {
	_, err := q.db.Exec(`
		UPDATE scan_steps SET status = ?, finished_at = ?, error_code = ?, error_summary = ? WHERE id = ?`,
		status, finishedAt, errorCode, errorSummary, id)
	return err
}

func (q *Queries) ListScanStepsByTask(taskID string) ([]*models.ScanStep, error) {
	rows, err := q.db.Query(`
		SELECT id, task_id, name, status, started_at, finished_at, error_code, error_summary, created_at
		FROM scan_steps WHERE task_id = ? ORDER BY created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ScanStep, 0)
	for rows.Next() {
		s := &models.ScanStep{}
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Name, &s.Status, &s.StartedAt, &s.FinishedAt, &s.ErrorCode, &s.ErrorSummary, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- WorkerNode ---

func (q *Queries) CreateWorkerNode(w *models.WorkerNode) error {
	_, err := q.db.Exec(`
		INSERT INTO worker_nodes (id, name, endpoint, mode, status, trust_level, network_profile, capabilities, tool_versions, template_versions, max_concurrency, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.Name, w.Endpoint, w.Mode, w.Status, w.TrustLevel, w.NetworkProfile, w.Capabilities, w.ToolVersions, w.TemplateVersions, w.MaxConcurrency, w.LastSeen, w.CreatedAt)
	return err
}

func (q *Queries) GetWorkerNode(id string) (*models.WorkerNode, error) {
	row := q.db.QueryRow(`
		SELECT id, name, endpoint, mode, status, trust_level, network_profile, capabilities, tool_versions, template_versions, max_concurrency, last_seen, created_at, revoked_at
		FROM worker_nodes WHERE id = ?`, id)
	w := &models.WorkerNode{}
	var lastSeen, revokedAt sql.NullTime
	if err := row.Scan(&w.ID, &w.Name, &w.Endpoint, &w.Mode, &w.Status, &w.TrustLevel, &w.NetworkProfile, &w.Capabilities, &w.ToolVersions, &w.TemplateVersions, &w.MaxConcurrency, &lastSeen, &w.CreatedAt, &revokedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastSeen.Valid {
		w.LastSeen = &lastSeen.Time
	}
	if revokedAt.Valid {
		w.RevokedAt = &revokedAt.Time
	}
	return w, nil
}

func (q *Queries) ListWorkerNodes() ([]*models.WorkerNode, error) {
	rows, err := q.db.Query(`
		SELECT id, name, endpoint, mode, status, trust_level, network_profile, capabilities, tool_versions, template_versions, max_concurrency, last_seen, created_at, revoked_at
		FROM worker_nodes ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.WorkerNode, 0)
	for rows.Next() {
		w := &models.WorkerNode{}
		var lastSeen, revokedAt sql.NullTime
		if err := rows.Scan(&w.ID, &w.Name, &w.Endpoint, &w.Mode, &w.Status, &w.TrustLevel, &w.NetworkProfile, &w.Capabilities, &w.ToolVersions, &w.TemplateVersions, &w.MaxConcurrency, &lastSeen, &w.CreatedAt, &revokedAt); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			w.LastSeen = &lastSeen.Time
		}
		if revokedAt.Valid {
			w.RevokedAt = &revokedAt.Time
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateWorkerNodeStatus(id string, status models.WorkerStatus, lastSeen time.Time) error {
	_, err := q.db.Exec(`UPDATE worker_nodes SET status = ?, last_seen = ? WHERE id = ?`, status, lastSeen, id)
	return err
}

// UpdateWorkerNodeTemplateVersions persists the worker's template version
// report (JSON blob) alongside the heartbeat status update.
func (q *Queries) UpdateWorkerNodeTemplateVersions(id string, status models.WorkerStatus, lastSeen time.Time, templateVersions string) error {
	_, err := q.db.Exec(`UPDATE worker_nodes SET status = ?, last_seen = ?, template_versions = ? WHERE id = ?`,
		status, lastSeen, templateVersions, id)
	return err
}

func (q *Queries) RevokeWorkerNode(id string, revokedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE worker_nodes SET status = ?, revoked_at = ? WHERE id = ?`, models.WorkerStatusOffline, revokedAt, id)
	return err
}

func (q *Queries) DeleteWorkerNode(id string) error {
	_, err := q.db.Exec(`DELETE FROM worker_nodes WHERE id = ?`, id)
	return err
}

// --- WorkerHealthCheck ---

func (q *Queries) CreateWorkerHealthCheck(h *models.WorkerHealthCheck) error {
	_, err := q.db.Exec(`
		INSERT INTO worker_health_checks (id, worker_id, tool, status, version, details, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		h.ID, h.WorkerID, h.Tool, h.Status, h.Version, h.Details, h.CheckedAt)
	return err
}

func (q *Queries) ListWorkerHealthChecks(workerID string) ([]*models.WorkerHealthCheck, error) {
	rows, err := q.db.Query(`
		SELECT id, worker_id, tool, status, version, details, checked_at
		FROM worker_health_checks WHERE worker_id = ? ORDER BY checked_at DESC`, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.WorkerHealthCheck, 0)
	for rows.Next() {
		h := &models.WorkerHealthCheck{}
		if err := rows.Scan(&h.ID, &h.WorkerID, &h.Tool, &h.Status, &h.Version, &h.Details, &h.CheckedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

// --- Run ---

func (q *Queries) CreateRun(r *models.Run) error {
	_, err := q.db.Exec(`
		INSERT INTO runs (id, project_id, tool_template_id, name, status, started_at, finished_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ProjectID, r.ToolTemplateID, r.Name, r.Status, r.StartedAt, r.FinishedAt, r.CreatedAt)
	return err
}

func (q *Queries) GetRun(id string) (*models.Run, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, tool_template_id, name, status, started_at, finished_at, created_at
		FROM runs WHERE id = ?`, id)
	r := &models.Run{}
	var templateID sql.NullString
	var startedAt, finishedAt sql.NullTime
	if err := row.Scan(&r.ID, &r.ProjectID, &templateID, &r.Name, &r.Status, &startedAt, &finishedAt, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if templateID.Valid {
		r.ToolTemplateID = &templateID.String
	}
	if startedAt.Valid {
		r.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		r.FinishedAt = &finishedAt.Time
	}
	return r, nil
}

func (q *Queries) ListRunsByProject(projectID string) ([]*models.Run, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, tool_template_id, name, status, started_at, finished_at, created_at
		FROM runs WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Run, 0)
	for rows.Next() {
		r := &models.Run{}
		var templateID sql.NullString
		var startedAt, finishedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &templateID, &r.Name, &r.Status, &startedAt, &finishedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if templateID.Valid {
			r.ToolTemplateID = &templateID.String
		}
		if startedAt.Valid {
			r.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			r.FinishedAt = &finishedAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) CountRunsByProject(projectID string) (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM runs WHERE project_id = ?`, projectID)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListRunsByProjectPaginated(projectID string, limit, offset int) ([]*models.Run, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, tool_template_id, name, status, started_at, finished_at, created_at
		FROM runs WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Run, 0)
	for rows.Next() {
		r := &models.Run{}
		var templateID sql.NullString
		var startedAt, finishedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &templateID, &r.Name, &r.Status, &startedAt, &finishedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if templateID.Valid {
			r.ToolTemplateID = &templateID.String
		}
		if startedAt.Valid {
			r.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			r.FinishedAt = &finishedAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// --- Screenshot ---

func (q *Queries) CreateScreenshot(s *models.Screenshot) error {
	_, err := q.db.Exec(`
		INSERT INTO screenshots (id, project_id, asset_id, task_id, url, original_path, thumbnail_path, width, height, taken_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ProjectID, s.AssetID, s.TaskID, s.URL, s.OriginalPath, s.ThumbnailPath, s.Width, s.Height, s.TakenAt)
	return err
}

func (q *Queries) ListScreenshotsByProject(projectID string) ([]*models.Screenshot, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, task_id, url, original_path, thumbnail_path, width, height, taken_at
		FROM screenshots WHERE project_id = ? ORDER BY taken_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Screenshot, 0)
	for rows.Next() {
		s := &models.Screenshot{}
		var assetID, taskID sql.NullString
		if err := rows.Scan(&s.ID, &s.ProjectID, &assetID, &taskID, &s.URL, &s.OriginalPath, &s.ThumbnailPath, &s.Width, &s.Height, &s.TakenAt); err != nil {
			return nil, err
		}
		if assetID.Valid {
			s.AssetID = &assetID.String
		}
		if taskID.Valid {
			s.TaskID = &taskID.String
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- ScanTask scheduling (v0.2) ---

func (q *Queries) ListScanTasksByRun(runID string) ([]*models.ScanTask, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, plan_id, run_id, depends_on_task_id, target_id, tool, command_template, arguments_redacted, status, started_at, finished_at, exit_code, error_message, worker_id, nuclei_custom_bundle_version, created_at
		FROM scan_tasks WHERE run_id = ? ORDER BY created_at`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ScanTask, 0)
	for rows.Next() {
		t := &models.ScanTask{}
		var rid, pid, dep, tid, wid, cv sql.NullString
		var sa, fa sql.NullTime
		var ec sql.NullInt64
		var errorMsg sql.NullString
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &pid, &rid, &dep, &tid, &t.Tool, &t.CommandTemplate, &t.ArgumentsRedacted, &t.Status, &sa, &fa, &ec, &errorMsg, &wid, &cv, &t.CreatedAt); err != nil {
			return nil, err
		}
		if pid.Valid {
			t.PlanID = pid.String
		}
		if rid.Valid {
			t.RunID = &rid.String
		}
		if dep.Valid {
			t.DependsOnTaskID = &dep.String
		}
		if tid.Valid {
			t.TargetID = &tid.String
		}
		if errorMsg.Valid {
			t.ErrorMessage = errorMsg.String
		}
		if cv.Valid {
			t.NucleiCustomBundleVersion = &cv.String
		}
		if sa.Valid {
			t.StartedAt = &sa.Time
		}
		if fa.Valid {
			t.FinishedAt = &fa.Time
		}
		if ec.Valid {
			v := int(ec.Int64)
			t.ExitCode = &v
		}
		if wid.Valid {
			t.WorkerID = &wid.String
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateRunStatus(id string, status models.RunStatus, startedAt, finishedAt *time.Time) error {
	_, err := q.db.Exec(`UPDATE runs SET status = ?, started_at = ?, finished_at = ? WHERE id = ?`, status, startedAt, finishedAt, id)
	return err
}

// --- v0.4 Pipeline tables ---

func (q *Queries) SaveDNSRecord(r *models.DNSRecord) error {
	_, err := q.db.Exec(`
		INSERT INTO dns_records (id, project_id, domain, ips, cnames, ttl, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, domain) DO UPDATE SET
			ips = excluded.ips,
			cnames = excluded.cnames,
			ttl = excluded.ttl,
			created_at = excluded.created_at
	`, r.ID, r.ProjectID, r.Domain, strings.Join(r.IPs, ","), strings.Join(r.CNAMEs, ","), r.TTL, r.CreatedAt)
	return err
}

func (q *Queries) ListDNSRecordsByProject(projectID string) ([]*models.DNSRecord, error) {
	rows, err := q.db.Query(`SELECT id, project_id, domain, ips, cnames, ttl, created_at FROM dns_records WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.DNSRecord
	for rows.Next() {
		r := &models.DNSRecord{}
		var ips, cnames string
		var ttl sql.NullInt64
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Domain, &ips, &cnames, &ttl, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.IPs = strings.Split(ips, ",")
		r.CNAMEs = strings.Split(cnames, ",")
		if ttl.Valid {
			r.TTL = uint32(ttl.Int64)
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) SaveCDNResult(r *models.CDNResult) error {
	_, err := q.db.Exec(`
		INSERT INTO cdn_results (id, project_id, ip, is_cdn, cdn_provider, cdn_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, ip) DO UPDATE SET
			is_cdn = excluded.is_cdn,
			cdn_provider = excluded.cdn_provider,
			cdn_type = excluded.cdn_type,
			created_at = excluded.created_at
	`, r.ID, r.ProjectID, r.IP, r.IsCDN, r.Provider, r.Type, r.CreatedAt)
	return err
}

func (q *Queries) ListCDNResultsByProject(projectID string) ([]*models.CDNResult, error) {
	rows, err := q.db.Query(`SELECT id, project_id, ip, is_cdn, cdn_provider, cdn_type, created_at FROM cdn_results WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.CDNResult
	for rows.Next() {
		r := &models.CDNResult{}
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.IP, &r.IsCDN, &r.Provider, &r.Type, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) SaveServiceFingerprint(r *models.ServiceFingerprint) error {
	metaJSON, _ := json.Marshal(r.Metadata)
	_, err := q.db.Exec(`
		INSERT INTO service_fingerprints (id, project_id, ip, port, protocol, is_web, service, metadata, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, ip, port) DO UPDATE SET
			protocol = excluded.protocol,
			is_web = excluded.is_web,
			service = excluded.service,
			metadata = excluded.metadata,
			source = excluded.source,
			created_at = excluded.created_at
	`, r.ID, r.ProjectID, r.IP, r.Port, r.Protocol, r.IsWeb, r.Service, string(metaJSON), r.Source, r.CreatedAt)
	return err
}

func (q *Queries) ListServiceFingerprintsByProject(projectID string) ([]*models.ServiceFingerprint, error) {
	rows, err := q.db.Query(`SELECT id, project_id, ip, port, protocol, is_web, service, metadata, source, created_at FROM service_fingerprints WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.ServiceFingerprint
	for rows.Next() {
		r := &models.ServiceFingerprint{}
		var metaJSON string
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.IP, &r.Port, &r.Protocol, &r.IsWeb, &r.Service, &metaJSON, &r.Source, &r.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metaJSON), &r.Metadata); err != nil {
			// silently ignore unmarshal errors for backward compatibility
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateProjectPipelineConfig(projectID string, cfgJSON string) error {
	_, err := q.db.Exec(`UPDATE projects SET pipeline_config = ? WHERE id = ?`, cfgJSON, projectID)
	return err
}

func (q *Queries) CreatePipelineRun(r *models.PipelineRun) error {
	_, err := q.db.Exec(`INSERT INTO pipeline_runs (id, project_id, mode, status, stage, error, started_at, completed_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ProjectID, r.Mode, r.Status, r.Stage, r.Error, r.StartedAt, r.CompletedAt, r.CreatedAt)
	return err
}

func (q *Queries) UpdatePipelineRunStatus(id, status string) error {
	_, err := q.db.Exec(`UPDATE pipeline_runs SET status = ? WHERE id = ?`, status, id)
	return err
}

func (q *Queries) UpdatePipelineRunStage(id, stage string) error {
	_, err := q.db.Exec(`UPDATE pipeline_runs SET stage = ? WHERE id = ?`, stage, id)
	return err
}

func (q *Queries) UpdatePipelineRunError(id, errMsg string) error {
	_, err := q.db.Exec(`UPDATE pipeline_runs SET error = ? WHERE id = ?`, errMsg, id)
	return err
}

func (q *Queries) UpdatePipelineRunCompleted(id string, completedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE pipeline_runs SET status = 'completed', completed_at = ? WHERE id = ?`, completedAt, id)
	return err
}

func (q *Queries) GetPipelineRun(id string) (*models.PipelineRun, error) {
	row := q.db.QueryRow(`SELECT id, project_id, mode, status, stage, error, started_at, completed_at, created_at FROM pipeline_runs WHERE id = ?`, id)
	r := &models.PipelineRun{}
	var completedAt sql.NullTime
	err := row.Scan(&r.ID, &r.ProjectID, &r.Mode, &r.Status, &r.Stage, &r.Error, &r.StartedAt, &completedAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	return r, err
}

func (q *Queries) ListPipelineRunsByProject(projectID string) ([]*models.PipelineRun, error) {
	rows, err := q.db.Query(`SELECT id, project_id, mode, status, stage, error, started_at, completed_at, created_at FROM pipeline_runs WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.PipelineRun
	for rows.Next() {
		r := &models.PipelineRun{}
		var completedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Mode, &r.Status, &r.Stage, &r.Error, &r.StartedAt, &completedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) CountPipelineRunsByProject(projectID string) (int, error) {
	var count int
	err := q.db.QueryRow(`SELECT COUNT(*) FROM pipeline_runs WHERE project_id = ?`, projectID).Scan(&count)
	return count, err
}

func (q *Queries) ListPipelineRunsByProjectPaginated(projectID string, limit, offset int) ([]*models.PipelineRun, error) {
	rows, err := q.db.Query(
		`SELECT id, project_id, mode, status, stage, error, started_at, completed_at, created_at
		 FROM pipeline_runs WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		projectID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.PipelineRun
	for rows.Next() {
		r := &models.PipelineRun{}
		var completedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Mode, &r.Status, &r.Stage, &r.Error, &r.StartedAt, &completedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// --- Pipeline Run Stages ---

func (q *Queries) CreatePipelineRunStage(s *models.PipelineRunStage) error {
	_, err := q.db.Exec(`
		INSERT INTO pipeline_run_stages (id, run_id, stage, status, error, started_at, completed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.RunID, s.Stage, s.Status, s.Error, s.StartedAt, s.CompletedAt, s.CreatedAt)
	return err
}

func (q *Queries) UpdatePipelineRunStageRecord(id string, status models.PipelineRunStageStatus, errMsg string, completedAt *time.Time) error {
	_, err := q.db.Exec(`
		UPDATE pipeline_run_stages SET status = ?, error = ?, completed_at = ? WHERE id = ?`,
		status, errMsg, completedAt, id)
	return err
}

func (q *Queries) GetPipelineRunStage(runID, stage string) (*models.PipelineRunStage, error) {
	row := q.db.QueryRow(`
		SELECT id, run_id, stage, status, error, started_at, completed_at, created_at
		FROM pipeline_run_stages WHERE run_id = ? AND stage = ?`, runID, stage)
	s := &models.PipelineRunStage{}
	var startedAt, completedAt sql.NullTime
	err := row.Scan(&s.ID, &s.RunID, &s.Stage, &s.Status, &s.Error, &startedAt, &completedAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if startedAt.Valid {
		s.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		s.CompletedAt = &completedAt.Time
	}
	return s, err
}

func (q *Queries) ListPipelineRunStages(runID string) ([]*models.PipelineRunStage, error) {
	rows, err := q.db.Query(`
		SELECT id, run_id, stage, status, error, started_at, completed_at, created_at
		FROM pipeline_run_stages WHERE run_id = ? ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.PipelineRunStage
	for rows.Next() {
		s := &models.PipelineRunStage{}
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.RunID, &s.Stage, &s.Status, &s.Error, &startedAt, &completedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			s.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			s.CompletedAt = &completedAt.Time
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- Dashboard ---

func (q *Queries) CountActiveRuns() (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM runs WHERE status = 'running'`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) CountPendingFindings() (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM findings WHERE status = 'pending_review'`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) CountOnlineWorkers() (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM worker_nodes WHERE status IN ('online', 'busy') AND (revoked_at IS NULL OR revoked_at = '')`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListRecentRuns(limit int) ([]*models.DashboardRunItem, error) {
	rows, err := q.db.Query(`
		SELECT r.id, r.project_id, p.name, r.name, r.status, r.started_at, r.created_at
		FROM runs r
		JOIN projects p ON r.project_id = p.id
		ORDER BY r.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.DashboardRunItem, 0)
	for rows.Next() {
		item := &models.DashboardRunItem{}
		var startedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ProjectName, &item.Name, &item.Status, &startedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			item.StartedAt = &startedAt.Time
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (q *Queries) ListRecentFindingsByStatus(status models.FindingStatus, limit int) ([]*models.DashboardFindingItem, error) {
	rows, err := q.db.Query(`
		SELECT f.id, f.project_id, p.name, f.title, f.severity, f.created_at
		FROM findings f
		JOIN projects p ON f.project_id = p.id
		WHERE f.status = ?
		ORDER BY f.priority DESC, f.created_at DESC LIMIT ?`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.DashboardFindingItem, 0)
	for rows.Next() {
		item := &models.DashboardFindingItem{}
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ProjectName, &item.Title, &item.Severity, &item.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// --- Engine Credentials ---

func (q *Queries) GetEngineCredential(engine string) (*models.EngineCredential, error) {
	row := q.db.QueryRow(`SELECT id, engine, api_key, extra, created_at, updated_at FROM engine_credentials WHERE engine = ?`, engine)
	c := &models.EngineCredential{}
	var extra sql.NullString
	err := row.Scan(&c.ID, &c.Engine, &c.APIKey, &extra, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if extra.Valid {
		c.Extra = &extra.String
	}
	return c, err
}

func (q *Queries) ListEngineCredentials() ([]*models.EngineCredential, error) {
	rows, err := q.db.Query(`SELECT id, engine, api_key, extra, created_at, updated_at FROM engine_credentials ORDER BY engine`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.EngineCredential, 0)
	for rows.Next() {
		c := &models.EngineCredential{}
		var extra sql.NullString
		if err := rows.Scan(&c.ID, &c.Engine, &c.APIKey, &extra, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if extra.Valid {
			c.Extra = &extra.String
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (q *Queries) SaveEngineCredential(c *models.EngineCredential) error {
	_, err := q.db.Exec(`
		INSERT INTO engine_credentials (id, engine, api_key, extra, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(engine) DO UPDATE SET
			api_key = excluded.api_key,
			extra = excluded.extra,
			updated_at = excluded.updated_at;
	`, c.ID, c.Engine, c.APIKey, c.Extra, c.CreatedAt, c.UpdatedAt)
	return err
}

func (q *Queries) DeleteEngineCredential(engine string) error {
	_, err := q.db.Exec(`DELETE FROM engine_credentials WHERE engine = ?`, engine)
	return err
}

// --- Nuclei Custom Sources ---

func (q *Queries) CreateNucleiCustomSource(s *models.NucleiCustomSource) error {
	_, err := q.db.Exec(`
		INSERT INTO nuclei_custom_sources (
			id, name, type, uri, branch, enabled, routing_policy, status,
			last_sync_at, last_validate_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, string(s.Type), s.URI, s.Branch,
		boolToInt(s.Enabled), s.RoutingPolicy, string(s.Status),
		s.LastSyncAt, s.LastValidateAt, s.LastError, s.CreatedAt, s.UpdatedAt)
	return err
}

func (q *Queries) GetNucleiCustomSource(id string) (*models.NucleiCustomSource, error) {
	row := q.db.QueryRow(`
		SELECT id, name, type, uri, branch, enabled, routing_policy, status,
		       last_sync_at, last_validate_at, last_error, created_at, updated_at
		FROM nuclei_custom_sources WHERE id = ?`, id)
	return scanNucleiCustomSource(row)
}

func (q *Queries) ListNucleiCustomSources() ([]*models.NucleiCustomSource, error) {
	rows, err := q.db.Query(`
		SELECT id, name, type, uri, branch, enabled, routing_policy, status,
		       last_sync_at, last_validate_at, last_error, created_at, updated_at
		FROM nuclei_custom_sources ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.NucleiCustomSource, 0)
	for rows.Next() {
		s, err := scanNucleiCustomSource(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateNucleiCustomSource(s *models.NucleiCustomSource) error {
	_, err := q.db.Exec(`
		UPDATE nuclei_custom_sources SET
			name = ?, type = ?, uri = ?, branch = ?, enabled = ?, routing_policy = ?,
			status = ?, last_sync_at = ?, last_validate_at = ?, last_error = ?, updated_at = ?
		WHERE id = ?`,
		s.Name, string(s.Type), s.URI, s.Branch, boolToInt(s.Enabled), s.RoutingPolicy,
		string(s.Status), s.LastSyncAt, s.LastValidateAt, s.LastError, s.UpdatedAt, s.ID)
	return err
}

func (q *Queries) DeleteNucleiCustomSource(id string) error {
	_, err := q.db.Exec(`DELETE FROM nuclei_custom_sources WHERE id = ?`, id)
	return err
}

// scanRow is the minimal interface satisfied by both *sql.Row and *sql.Rows
// (single-row scan calls), which lets the helper handle GET/LIST equally.
type scanRow interface {
	Scan(dest ...any) error
}

func scanNucleiCustomSource(row scanRow) (*models.NucleiCustomSource, error) {
	s := &models.NucleiCustomSource{}
	var typeStr, statusStr string
	var enabledInt int
	var uri, branch, lastError sql.NullString
	var lastSyncAt, lastValidateAt sql.NullTime
	err := row.Scan(
		&s.ID, &s.Name, &typeStr, &uri, &branch, &enabledInt, &s.RoutingPolicy,
		&statusStr, &lastSyncAt, &lastValidateAt, &lastError, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Type = models.NucleiCustomSourceType(typeStr)
	s.Status = models.NucleiCustomSourceStatus(statusStr)
	s.Enabled = enabledInt != 0
	if uri.Valid {
		s.URI = &uri.String
	}
	if branch.Valid {
		s.Branch = &branch.String
	}
	if lastError.Valid {
		s.LastError = &lastError.String
	}
	if lastSyncAt.Valid {
		t := lastSyncAt.Time
		s.LastSyncAt = &t
	}
	if lastValidateAt.Valid {
		t := lastValidateAt.Time
		s.LastValidateAt = &t
	}
	return s, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Nuclei Custom Bundles (Phase 2 wires these; Phase 1 ships placeholders) ---

func (q *Queries) CreateNucleiCustomBundle(b *models.NucleiCustomBundle) error {
	_, err := q.db.Exec(`
		INSERT INTO nuclei_custom_bundles (version, manifest_json, archive_path, status, created_at, activated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		b.Version, b.ManifestJSON, b.ArchivePath, b.Status, b.CreatedAt, b.ActivatedAt)
	return err
}

func (q *Queries) GetNucleiCustomBundle(version string) (*models.NucleiCustomBundle, error) {
	row := q.db.QueryRow(`
		SELECT version, manifest_json, archive_path, status, created_at, activated_at
		FROM nuclei_custom_bundles WHERE version = ?`, version)
	b := &models.NucleiCustomBundle{}
	var activatedAt sql.NullTime
	err := row.Scan(&b.Version, &b.ManifestJSON, &b.ArchivePath, &b.Status, &b.CreatedAt, &activatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if activatedAt.Valid {
		t := activatedAt.Time
		b.ActivatedAt = &t
	}
	return b, nil
}

func (q *Queries) ListNucleiCustomBundles() ([]*models.NucleiCustomBundle, error) {
	rows, err := q.db.Query(`
		SELECT version, manifest_json, archive_path, status, created_at, activated_at
		FROM nuclei_custom_bundles ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.NucleiCustomBundle, 0)
	for rows.Next() {
		b := &models.NucleiCustomBundle{}
		var activatedAt sql.NullTime
		if err := rows.Scan(&b.Version, &b.ManifestJSON, &b.ArchivePath, &b.Status, &b.CreatedAt, &activatedAt); err != nil {
			return nil, err
		}
		if activatedAt.Valid {
			t := activatedAt.Time
			b.ActivatedAt = &t
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (q *Queries) SetNucleiCustomBundleStatus(version, status string, activatedAt *time.Time) error {
	_, err := q.db.Exec(`UPDATE nuclei_custom_bundles SET status = ?, activated_at = ? WHERE version = ?`,
		status, activatedAt, version)
	return err
}

// GetActiveNucleiCustomBundleVersion returns the version string of the
// currently active bundle, or "" if none is active.
func (q *Queries) GetActiveNucleiCustomBundleVersion() (string, error) {
	var version string
	err := q.db.QueryRow(`SELECT version FROM nuclei_custom_bundles WHERE status = 'active' LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return version, nil
}

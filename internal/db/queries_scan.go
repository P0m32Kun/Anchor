package db

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

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

func (q *Queries) SetScanTaskWorker(id, workerID string) error {
	_, err := q.db.Exec(`UPDATE scan_tasks SET worker_id = ? WHERE id = ?`, workerID, id)
	return err
}

// ResetScanTaskForRetry clears terminal state on a task so it can be re-dispatched.
// Used by the dispatcher to retry tasks that were idle-killed by the worker watchdog.
func (q *Queries) ResetScanTaskForRetry(id string) error {
	_, err := q.db.Exec(`UPDATE scan_tasks SET status = ?, exit_code = NULL, finished_at = NULL, error_message = '' WHERE id = ?`, models.TaskCreated, id)
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

func (q *Queries) UpdateRunStatus(id string, status models.RunStatus, startedAt, finishedAt *time.Time) error {
	_, err := q.db.Exec(`UPDATE runs SET status = ?, started_at = ?, finished_at = ? WHERE id = ?`, status, startedAt, finishedAt, id)
	return err
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

func (q *Queries) GetRawArtifact(id string) (*models.RawArtifact, error) {
	row := q.db.QueryRow(`SELECT id, project_id, task_id, type, path, sha256, size, redaction_status, created_at FROM raw_artifacts WHERE id = ?`, id)
	a := &models.RawArtifact{}
	err := row.Scan(&a.ID, &a.ProjectID, &a.TaskID, &a.Type, &a.Path, &a.SHA256, &a.Size, &a.RedactionStatus, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
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
		INSERT INTO service_fingerprints (id, project_id, ip, port, protocol, is_web, service, product, version, metadata, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, ip, port) DO UPDATE SET
			protocol = excluded.protocol,
			is_web = excluded.is_web,
			service = excluded.service,
			product = excluded.product,
			version = excluded.version,
			metadata = excluded.metadata,
			source = excluded.source,
			created_at = excluded.created_at
	`, r.ID, r.ProjectID, r.IP, r.Port, r.Protocol, r.IsWeb, r.Service, r.Product, r.Version, string(metaJSON), r.Source, r.CreatedAt)
	return err
}

func (q *Queries) ListServiceFingerprintsByProject(projectID string) ([]*models.ServiceFingerprint, error) {
	rows, err := q.db.Query(`SELECT id, project_id, ip, port, protocol, is_web, service, product, version, metadata, source, created_at FROM service_fingerprints WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.ServiceFingerprint
	for rows.Next() {
		r := &models.ServiceFingerprint{}
		var metaJSON string
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.IP, &r.Port, &r.Protocol, &r.IsWeb, &r.Service, &r.Product, &r.Version, &metaJSON, &r.Source, &r.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metaJSON), &r.Metadata); err != nil {
			// silently ignore unmarshal errors for backward compatibility
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// --- Pipeline Runs ---

func (q *Queries) CreatePipelineRun(r *models.PipelineRun) error {
	_, err := q.db.Exec(`INSERT INTO pipeline_runs (id, project_id, mode, status, stage, error, engine_state, last_new_asset_at, started_at, completed_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ProjectID, r.Mode, r.Status, r.Stage, r.Error, r.EngineState, r.LastNewAssetAt, r.StartedAt, r.CompletedAt, r.CreatedAt)
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
	row := q.db.QueryRow(`SELECT id, project_id, mode, status, stage, error, engine_state, last_new_asset_at, started_at, completed_at, created_at FROM pipeline_runs WHERE id = ?`, id)
	r := &models.PipelineRun{}
	var completedAt, lastNewAssetAt sql.NullTime
	err := row.Scan(&r.ID, &r.ProjectID, &r.Mode, &r.Status, &r.Stage, &r.Error, &r.EngineState, &lastNewAssetAt, &r.StartedAt, &completedAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	if lastNewAssetAt.Valid {
		r.LastNewAssetAt = &lastNewAssetAt.Time
	}
	return r, err
}

func (q *Queries) ListPipelineRunsByProject(projectID string) ([]*models.PipelineRun, error) {
	rows, err := q.db.Query(`SELECT id, project_id, mode, status, stage, error, engine_state, last_new_asset_at, started_at, completed_at, created_at FROM pipeline_runs WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.PipelineRun
	for rows.Next() {
		r := &models.PipelineRun{}
		var completedAt, lastNewAssetAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Mode, &r.Status, &r.Stage, &r.Error, &r.EngineState, &lastNewAssetAt, &r.StartedAt, &completedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		if lastNewAssetAt.Valid {
			r.LastNewAssetAt = &lastNewAssetAt.Time
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
		`SELECT id, project_id, mode, status, stage, error, engine_state, last_new_asset_at, started_at, completed_at, created_at
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
		var completedAt, lastNewAssetAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Mode, &r.Status, &r.Stage, &r.Error, &r.EngineState, &lastNewAssetAt, &r.StartedAt, &completedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		if lastNewAssetAt.Valid {
			r.LastNewAssetAt = &lastNewAssetAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// --- Pipeline Run Stages ---

func (q *Queries) CreatePipelineRunStage(s *models.PipelineRunStage) error {
	_, err := q.db.Exec(`
		INSERT INTO pipeline_run_stages (id, run_id, stage, status, error, work_total, work_done, work_running, round, started_at, completed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.RunID, s.Stage, s.Status, s.Error, s.WorkTotal, s.WorkDone, s.WorkRunning, s.Round, s.StartedAt, s.CompletedAt, s.CreatedAt)
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
		SELECT id, run_id, stage, status, error, work_total, work_done, work_running, round, started_at, completed_at, created_at
		FROM pipeline_run_stages WHERE run_id = ? AND stage = ?`, runID, stage)
	s := &models.PipelineRunStage{}
	var startedAt, completedAt sql.NullTime
	err := row.Scan(&s.ID, &s.RunID, &s.Stage, &s.Status, &s.Error, &s.WorkTotal, &s.WorkDone, &s.WorkRunning, &s.Round, &startedAt, &completedAt, &s.CreatedAt)
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
		SELECT id, run_id, stage, status, error, work_total, work_done, work_running, round, started_at, completed_at, created_at
		FROM pipeline_run_stages WHERE run_id = ? ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.PipelineRunStage
	for rows.Next() {
		s := &models.PipelineRunStage{}
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.RunID, &s.Stage, &s.Status, &s.Error, &s.WorkTotal, &s.WorkDone, &s.WorkRunning, &s.Round, &startedAt, &completedAt, &s.CreatedAt); err != nil {
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
	row := q.db.QueryRow(`SELECT COUNT(*) FROM runs r JOIN projects p ON r.project_id = p.id WHERE r.status = 'running'`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) CountPendingFindings() (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM findings f JOIN projects p ON f.project_id = p.id WHERE f.status = 'pending_review'`)
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

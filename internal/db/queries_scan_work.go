package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

const scanWorkItemColumns = `id, run_id, project_id, asset_id, action, task_id, status, skip_reason, stage, error,
	input_file, member_asset_ids, bucket_key, generation, batch_mode, started_at, completed_at, created_at`

// --- ScanWorkItem ---

func (q *Queries) CreateScanWorkItem(w *models.ScanWorkItem) error {
	batchMode := 0
	if w.BatchMode {
		batchMode = 1
	}
	_, err := q.db.Exec(`
		INSERT INTO scan_work_items (id, run_id, project_id, asset_id, action, task_id, status, skip_reason, stage, error,
			input_file, member_asset_ids, bucket_key, generation, batch_mode, started_at, completed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.RunID, w.ProjectID, w.AssetID, w.Action, w.TaskID, w.Status,
		sqlNullStringValue(w.SkipReason), sqlNullStringValue(w.Stage), sqlNullStringValue(w.Error),
		sqlNullStringValue(w.InputFile), sqlNullStringValue(w.MemberAssetIDs), sqlNullStringValue(w.BucketKey),
		sqlNullIntValue(w.Generation), batchMode,
		w.StartedAt, w.CompletedAt, w.CreatedAt)
	return err
}

func (q *Queries) GetScanWorkItem(id string) (*models.ScanWorkItem, error) {
	row := q.db.QueryRow(`SELECT `+scanWorkItemColumns+` FROM scan_work_items WHERE id = ?`, id)
	w, err := scanWorkItemRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

func (q *Queries) GetScanWorkItemByRunAssetAction(runID, assetID, action string) (*models.ScanWorkItem, error) {
	row := q.db.QueryRow(`SELECT `+scanWorkItemColumns+` FROM scan_work_items WHERE run_id = ? AND asset_id = ? AND action = ?`, runID, assetID, action)
	w, err := scanWorkItemRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// ClaimScanWorkItem atomically transitions pending → running.
func (q *Queries) ClaimScanWorkItem(id string) (*models.ScanWorkItem, error) {
	now := time.Now().UTC()
	res, err := q.db.Exec(`
		UPDATE scan_work_items SET status = ?, started_at = ?
		WHERE id = ? AND status = ?`,
		models.WorkStatusRunning, now, id, models.WorkStatusPending)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	return q.GetScanWorkItem(id)
}

func (q *Queries) UpdateScanWorkItemTaskID(id, taskID string) error {
	_, err := q.db.Exec(`UPDATE scan_work_items SET task_id = ? WHERE id = ?`, taskID, id)
	return err
}

func (q *Queries) UpdateScanWorkItemStatus(id string, status models.WorkStatus, startedAt, completedAt *time.Time) error {
	_, err := q.db.Exec(`
		UPDATE scan_work_items SET status = ?, started_at = ?, completed_at = ? WHERE id = ?`,
		status, startedAt, completedAt, id)
	return err
}

func (q *Queries) UpdateScanWorkItemSkip(id string, status models.WorkStatus, skipReason string, completedAt *time.Time) error {
	_, err := q.db.Exec(`
		UPDATE scan_work_items SET status = ?, skip_reason = ?, completed_at = ? WHERE id = ?`,
		status, skipReason, completedAt, id)
	return err
}

func (q *Queries) UpdateScanWorkItemError(id string, status models.WorkStatus, errMsg string, completedAt *time.Time) error {
	_, err := q.db.Exec(`
		UPDATE scan_work_items SET status = ?, error = ?, completed_at = ? WHERE id = ?`,
		status, errMsg, completedAt, id)
	return err
}

func (q *Queries) ListScanWorkItemsByRun(runID string) ([]*models.ScanWorkItem, error) {
	return q.ListScanWorkItemsByRunPaginated(runID, 0, 0)
}

func (q *Queries) CountScanWorkItemsByRun(runID string) (int, error) {
	var count int
	err := q.db.QueryRow(`SELECT COUNT(*) FROM scan_work_items WHERE run_id = ?`, runID).Scan(&count)
	return count, err
}

func (q *Queries) ListScanWorkItemsByRunPaginated(runID string, limit, offset int) ([]*models.ScanWorkItem, error) {
	query := `SELECT ` + scanWorkItemColumns + ` FROM scan_work_items WHERE run_id = ? ORDER BY created_at`
	var rows *sql.Rows
	var err error
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		rows, err = q.db.Query(query, runID, limit, offset)
	} else {
		rows, err = q.db.Query(query, runID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkItemRows(rows)
}

func (q *Queries) ListScanWorkItemsByRunAndStatus(runID string, status models.WorkStatus) ([]*models.ScanWorkItem, error) {
	rows, err := q.db.Query(`SELECT `+scanWorkItemColumns+` FROM scan_work_items WHERE run_id = ? AND status = ? ORDER BY created_at`, runID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkItemRows(rows)
}

func (q *Queries) ListScanWorkItemsByAsset(runID, assetID string) ([]*models.ScanWorkItem, error) {
	rows, err := q.db.Query(`SELECT `+scanWorkItemColumns+` FROM scan_work_items WHERE run_id = ? AND asset_id = ? ORDER BY started_at`, runID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkItemRows(rows)
}

func (q *Queries) CountScanWorkItemsByStatus(runID string) (pending, running, done, skipped, failed int, err error) {
	rows, err2 := q.db.Query(`
		SELECT status, COUNT(*) FROM scan_work_items WHERE run_id = ? GROUP BY status`, runID)
	if err2 != nil {
		return 0, 0, 0, 0, 0, err2
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return 0, 0, 0, 0, 0, err
		}
		switch models.WorkStatus(status) {
		case models.WorkStatusPending:
			pending = count
		case models.WorkStatusRunning:
			running = count
		case models.WorkStatusDone:
			done = count
		case models.WorkStatusSkipped:
			skipped = count
		case models.WorkStatusFailed:
			failed = count
		}
	}
	return pending, running, done, skipped, failed, rows.Err()
}

func (q *Queries) AllWorkItemsTerminal(runID string) (bool, error) {
	var count int
	err := q.db.QueryRow(`
		SELECT COUNT(*) FROM scan_work_items
		WHERE run_id = ? AND status NOT IN ('done', 'skipped', 'failed')`, runID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// --- ScanRunMetrics ---

func (q *Queries) GetScanRunMetrics(runID string) (*models.ScanRunMetrics, error) {
	run, err := q.GetPipelineRun(runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}

	pending, running, done, skipped, failed, err := q.CountScanWorkItemsByStatus(runID)
	if err != nil {
		return nil, err
	}

	// Count assets discovered for this run
	var assetsDiscovered int
	if err := q.db.QueryRow(`
		SELECT COUNT(DISTINCT asset_id) FROM scan_work_items WHERE run_id = ?`, runID).Scan(&assetsDiscovered); err != nil {
		return nil, err
	}

	m := &models.ScanRunMetrics{
		EngineState:      run.EngineState,
		AssetsDiscovered: assetsDiscovered,
		WorksPending:     pending,
		WorksDone:        done,
		WorksSkipped:     skipped,
		WorksRunning:     running,
		WorksFailed:      failed,
		LastNewAssetAt:   run.LastNewAssetAt,
	}
	return m, nil
}

// --- PipelineRun extensions ---

func (q *Queries) UpdatePipelineRunEngineState(id, engineState string) error {
	_, err := q.db.Exec(`UPDATE pipeline_runs SET engine_state = ? WHERE id = ?`, engineState, id)
	return err
}

func (q *Queries) UpdatePipelineRunLastNewAssetAt(id string, t time.Time) error {
	_, err := q.db.Exec(`UPDATE pipeline_runs SET last_new_asset_at = ? WHERE id = ?`, t, id)
	return err
}

// --- PipelineRunStage work count extensions ---

func (q *Queries) UpdatePipelineRunStageWorkCounts(runID, stage string, workTotal, workDone, workRunning, round int) error {
	_, err := q.db.Exec(`
		UPDATE pipeline_run_stages SET work_total = ?, work_done = ?, work_running = ?, round = ?
		WHERE run_id = ? AND stage = ?`,
		workTotal, workDone, workRunning, round, runID, stage)
	return err
}

// --- helper ---

func scanWorkItemRows(rows *sql.Rows) ([]*models.ScanWorkItem, error) {
	var list []*models.ScanWorkItem
	for rows.Next() {
		w, err := scanWorkItemRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func scanWorkItemRow(scanner interface{ Scan(dest ...any) error }) (*models.ScanWorkItem, error) {
	w := &models.ScanWorkItem{}
	var skipReason, stage, errMsg sql.NullString
	var taskID, inputFile, memberAssetIDs, bucketKey sql.NullString
	var generation sql.NullInt64
	var batchMode int
	var startedAt, completedAt sql.NullTime
	if err := scanner.Scan(&w.ID, &w.RunID, &w.ProjectID, &w.AssetID, &w.Action, &taskID, &w.Status,
		&skipReason, &stage, &errMsg, &inputFile, &memberAssetIDs, &bucketKey, &generation, &batchMode,
		&startedAt, &completedAt, &w.CreatedAt); err != nil {
		return nil, err
	}
	if inputFile.Valid {
		w.InputFile = inputFile.String
	}
	if memberAssetIDs.Valid {
		w.MemberAssetIDs = memberAssetIDs.String
	}
	if bucketKey.Valid {
		w.BucketKey = bucketKey.String
	}
	if generation.Valid {
		w.Generation = int(generation.Int64)
	}
	w.BatchMode = batchMode != 0
	applyScanWorkItemNullables(w, skipReason, stage, errMsg, taskID, startedAt, completedAt)
	return w, nil
}

func applyScanWorkItemNullables(w *models.ScanWorkItem, skipReason, stage, errMsg, taskID sql.NullString, startedAt, completedAt sql.NullTime) {
	if skipReason.Valid {
		w.SkipReason = skipReason.String
	}
	if stage.Valid {
		w.Stage = stage.String
	}
	if errMsg.Valid {
		w.Error = errMsg.String
	}
	if taskID.Valid {
		w.TaskID = &taskID.String
	}
	if startedAt.Valid {
		w.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		w.CompletedAt = &completedAt.Time
	}
}

// ListAssetsByRun returns assets discovered during a specific pipeline run,
// joined through scan_work_items.
func (q *Queries) ListAssetsByRun(projectID, runID string) ([]*models.Asset, error) {
	rows, err := q.db.Query(`
		SELECT DISTINCT a.id, a.project_id, a.type, a.value, a.normalized_value,
			a.source_tools, a.first_seen, a.last_seen, a.tags
		FROM assets a
		INNER JOIN scan_work_items swi ON swi.asset_id = a.id
		WHERE swi.run_id = ? AND a.project_id = ?
		ORDER BY a.last_seen DESC`, runID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Asset, 0)
	for rows.Next() {
		a, err := scanAssetRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// scanAssetRow scans an Asset from a row interface.
func scanAssetRow(scanner interface{ Scan(dest ...any) error }) (*models.Asset, error) {
	a := &models.Asset{}
	var sourceToolsJSON, tagsJSON string
	if err := scanner.Scan(&a.ID, &a.ProjectID, &a.Type, &a.Value, &a.NormalizedValue,
		&sourceToolsJSON, &a.FirstSeen, &a.LastSeen, &tagsJSON); err != nil {
		return nil, err
	}
	if sourceToolsJSON != "" {
		if err := json.Unmarshal([]byte(sourceToolsJSON), &a.SourceTools); err != nil {
			return nil, fmt.Errorf("unmarshal source_tools: %w", err)
		}
	}
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &a.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}
	}
	return a, nil
}

// ToolStats holds aggregated statistics for a single tool.
type ToolStats struct {
	Tool         string
	TotalCalls   int
	SuccessCount int
	FailedCount  int
	SkippedCount int
	AvgDuration  float64 // seconds
}

// GetToolStatsByRun returns aggregated tool statistics for a run.
func (q *Queries) GetToolStatsByRun(runID string) ([]*ToolStats, error) {
	rows, err := q.db.Query(`
		SELECT
			action as tool,
			COUNT(*) as total_calls,
			SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END) as skipped_count,
			AVG(CASE
				WHEN started_at IS NOT NULL AND completed_at IS NOT NULL
				THEN (julianday(completed_at) - julianday(started_at)) * 86400
				ELSE NULL
			END) as avg_duration
		FROM scan_work_items
		WHERE run_id = ?
		GROUP BY action`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*ToolStats
	for rows.Next() {
		s := &ToolStats{}
		var avgDuration sql.NullFloat64
		if err := rows.Scan(&s.Tool, &s.TotalCalls, &s.SuccessCount, &s.FailedCount, &s.SkippedCount, &avgDuration); err != nil {
			return nil, err
		}
		if avgDuration.Valid {
			s.AvgDuration = avgDuration.Float64
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// ToolErrorStats holds error distribution for a tool.
type ToolErrorStats struct {
	Tool  string
	Error string
	Count int
}

// GetToolErrorStatsByRun returns error distribution for tools in a run.
func (q *Queries) GetToolErrorStatsByRun(runID string) ([]*ToolErrorStats, error) {
	rows, err := q.db.Query(`
		SELECT action, error, COUNT(*) as cnt
		FROM scan_work_items
		WHERE run_id = ? AND status = 'failed' AND error != ''
		GROUP BY action, error
		ORDER BY cnt DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*ToolErrorStats
	for rows.Next() {
		s := &ToolErrorStats{}
		if err := rows.Scan(&s.Tool, &s.Error, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

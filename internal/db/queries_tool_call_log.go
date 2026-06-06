package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// toolCallLogColumns is the canonical column list for the tool_call_logs table.
const toolCallLogColumns = "id, run_id, work_item_id, task_id, tool, action, asset_id, params_json, started_at, finished_at, duration_ms, exit_code, status, output_summary, error_message, created_at"

// CreateToolCallLog inserts a new tool call log entry.
func (q *Queries) CreateToolCallLog(l *models.ToolCallLog) error {
	_, err := q.db.Exec(`
		INSERT INTO tool_call_logs (`+toolCallLogColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.RunID, l.WorkItemID, l.TaskID, l.Tool, l.Action, l.AssetID,
		l.ParamsJSON, l.StartedAt, l.FinishedAt, l.DurationMs, l.ExitCode,
		l.Status, l.OutputSummary, l.ErrorMessage, l.CreatedAt)
	return err
}

// UpdateToolCallLogOnComplete updates a tool call log when the tool finishes.
func (q *Queries) UpdateToolCallLogOnComplete(id string, finishedAt time.Time, exitCode *int, status models.ToolCallStatus, durationMs int64, outputSummary, errorMsg string) error {
	_, err := q.db.Exec(`
		UPDATE tool_call_logs
		SET finished_at = ?, exit_code = ?, status = ?, duration_ms = ?, output_summary = ?, error_message = ?
		WHERE id = ?`,
		finishedAt, exitCode, status, durationMs, sqlNullStringValue(outputSummary), sqlNullStringValue(errorMsg), id)
	return err
}

// UpdateToolCallLogTaskID links a tool call log to its scan task.
func (q *Queries) UpdateToolCallLogTaskID(id, taskID string) error {
	_, err := q.db.Exec(`UPDATE tool_call_logs SET task_id = ? WHERE id = ?`, taskID, id)
	return err
}

// GetToolCallLog returns a single tool call log by ID.
func (q *Queries) GetToolCallLog(id string) (*models.ToolCallLog, error) {
	row := q.db.QueryRow(`
		SELECT `+toolCallLogColumns+`
		FROM tool_call_logs WHERE id = ?`, id)
	return scanToolCallLog(row)
}

// ListToolCallLogsByRun returns all tool call logs for a pipeline run, ordered by start time.
func (q *Queries) ListToolCallLogsByRun(runID string) ([]*models.ToolCallLog, error) {
	rows, err := q.db.Query(`
		SELECT `+toolCallLogColumns+`
		FROM tool_call_logs WHERE run_id = ? ORDER BY started_at`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanToolCallLogRows(rows)
}

// ListToolCallLogsByTaskID returns all tool call logs linked to a scan task.
func (q *Queries) ListToolCallLogsByTaskID(taskID string) ([]*models.ToolCallLog, error) {
	rows, err := q.db.Query(`
		SELECT `+toolCallLogColumns+`
		FROM tool_call_logs WHERE task_id = ? ORDER BY started_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanToolCallLogRows(rows)
}

// ListToolCallLogsByWorkItem returns all tool call logs for a work item.
func (q *Queries) ListToolCallLogsByWorkItem(workItemID string) ([]*models.ToolCallLog, error) {
	rows, err := q.db.Query(`
		SELECT `+toolCallLogColumns+`
		FROM tool_call_logs WHERE work_item_id = ? ORDER BY started_at`, workItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanToolCallLogRows(rows)
}

// GetToolCallTraceByFinding returns the full trace chain for a finding.
// Chain: finding → scan_task (via source_task_id) → scan_work_item (via task_id) → pipeline_run
func (q *Queries) GetToolCallTraceByFinding(findingID string) (*models.ToolCallTrace, error) {
	trace := &models.ToolCallTrace{}

	// 1. Get finding
	finding, err := q.GetFinding(findingID)
	if err != nil {
		return nil, err
	}
	if finding == nil {
		return nil, nil
	}
	trace.Finding = finding

	// 2. Get scan task via source_task_id
	if finding.SourceTaskID != nil && *finding.SourceTaskID != "" {
		task, err := q.GetScanTask(*finding.SourceTaskID)
		if err == nil && task != nil {
			trace.Task = task
		}
	}

	// 3. Get scan work item that links to this task
	if trace.Task != nil {
		// Find work item by task_id
		workItem, err := q.getScanWorkItemByTaskID(trace.Task.ID)
		if err == nil && workItem != nil {
			trace.WorkItem = workItem
		}
	}

	// 4. Get pipeline run
	if finding.RunID != nil && *finding.RunID != "" {
		run, err := q.GetPipelineRun(*finding.RunID)
		if err == nil && run != nil {
			trace.Run = run
		}
	}

	// 5. Get tool call log (prefer work_item_id, fallback to task_id)
	if trace.WorkItem != nil {
		logs, err := q.ListToolCallLogsByWorkItem(trace.WorkItem.ID)
		if err == nil && len(logs) > 0 {
			trace.ToolCallLog = logs[0]
		}
	}
	if trace.ToolCallLog == nil && trace.Task != nil {
		logs, err := q.ListToolCallLogsByTaskID(trace.Task.ID)
		if err == nil && len(logs) > 0 {
			trace.ToolCallLog = logs[0]
		}
	}

	return trace, nil
}

// getScanWorkItemByTaskID finds a work item by its task_id.
func (q *Queries) getScanWorkItemByTaskID(taskID string) (*models.ScanWorkItem, error) {
	row := q.db.QueryRow(`
		SELECT id, run_id, project_id, asset_id, action, task_id, status, skip_reason, stage, error, started_at, completed_at, created_at
		FROM scan_work_items WHERE task_id = ?`, taskID)
	w := &models.ScanWorkItem{}
	var skipReason, stage, errMsg sql.NullString
	var taskIDNull sql.NullString
	var startedAt, completedAt sql.NullTime
	err := row.Scan(&w.ID, &w.RunID, &w.ProjectID, &w.AssetID, &w.Action, &taskIDNull, &w.Status,
		&skipReason, &stage, &errMsg, &startedAt, &completedAt, &w.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if skipReason.Valid {
		w.SkipReason = skipReason.String
	}
	if stage.Valid {
		w.Stage = stage.String
	}
	if errMsg.Valid {
		w.Error = errMsg.String
	}
	if taskIDNull.Valid {
		w.TaskID = &taskIDNull.String
	}
	if startedAt.Valid {
		w.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		w.CompletedAt = &completedAt.Time
	}
	return w, nil
}

// --- helpers ---

func scanToolCallLog(row interface {
	Scan(dest ...any) error
}) (*models.ToolCallLog, error) {
	l := &models.ToolCallLog{}
	var workItemID, taskID, assetID sql.NullString
	var finishedAt sql.NullTime
	var durationMs sql.NullInt64
	var exitCode sql.NullInt32
	var outputSummary, errorMessage sql.NullString

	err := row.Scan(
		&l.ID, &l.RunID, &workItemID, &taskID, &l.Tool, &l.Action, &assetID,
		&l.ParamsJSON, &l.StartedAt, &finishedAt, &durationMs, &exitCode,
		&l.Status, &outputSummary, &errorMessage, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	l.WorkItemID = nullableString(workItemID)
	l.TaskID = nullableString(taskID)
	l.AssetID = nullableString(assetID)
	l.OutputSummary = nullableString(outputSummary)
	l.ErrorMessage = nullableString(errorMessage)

	if finishedAt.Valid {
		l.FinishedAt = &finishedAt.Time
	}
	if durationMs.Valid {
		v := durationMs.Int64
		l.DurationMs = &v
	}
	if exitCode.Valid {
		v := int(exitCode.Int32)
		l.ExitCode = &v
	}

	return l, nil
}

func scanToolCallLogRows(rows *sql.Rows) ([]*models.ToolCallLog, error) {
	var list []*models.ToolCallLog
	for rows.Next() {
		l, err := scanToolCallLog(rows)
		if err != nil {
			return nil, err
		}
		if l != nil {
			list = append(list, l)
		}
	}
	return list, rows.Err()
}

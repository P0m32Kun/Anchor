package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (q *Queries) CreateSlowScanTask(t *models.SlowScanTask) error {
	_, err := q.db.Exec(`
		INSERT INTO slow_scan_tasks (id, project_id, target_id, run_id, tool, status, config_json, rate_limit, timeout, error_message, started_at, finished_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.TargetID, t.RunID, t.Tool, t.Status, t.ConfigJSON, t.RateLimit, t.Timeout, t.ErrorMessage, t.StartedAt, t.FinishedAt, t.CreatedAt, t.UpdatedAt)
	return err
}

func (q *Queries) GetSlowScanTask(id string) (*models.SlowScanTask, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, target_id, run_id, tool, status, config_json, rate_limit, timeout, error_message, started_at, finished_at, created_at, updated_at
		FROM slow_scan_tasks WHERE id = ?`, id)
	t := &models.SlowScanTask{}
	err := row.Scan(&t.ID, &t.ProjectID, &t.TargetID, &t.RunID, &t.Tool, &t.Status, &t.ConfigJSON, &t.RateLimit, &t.Timeout, &t.ErrorMessage, &t.StartedAt, &t.FinishedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (q *Queries) ListSlowScanTasksByProject(projectID string) ([]*models.SlowScanTask, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, target_id, run_id, tool, status, config_json, rate_limit, timeout, error_message, started_at, finished_at, created_at, updated_at
		FROM slow_scan_tasks WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.SlowScanTask, 0)
	for rows.Next() {
		t := &models.SlowScanTask{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.TargetID, &t.RunID, &t.Tool, &t.Status, &t.ConfigJSON, &t.RateLimit, &t.Timeout, &t.ErrorMessage, &t.StartedAt, &t.FinishedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (q *Queries) ListPendingSlowScanTasks(limit int) ([]*models.SlowScanTask, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := q.db.Query(`
		SELECT id, project_id, target_id, run_id, tool, status, config_json, rate_limit, timeout, error_message, started_at, finished_at, created_at, updated_at
		FROM slow_scan_tasks WHERE status = ? ORDER BY created_at ASC LIMIT ?`, models.SlowScanPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.SlowScanTask, 0)
	for rows.Next() {
		t := &models.SlowScanTask{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.TargetID, &t.RunID, &t.Tool, &t.Status, &t.ConfigJSON, &t.RateLimit, &t.Timeout, &t.ErrorMessage, &t.StartedAt, &t.FinishedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateSlowScanStatus(id string, status models.SlowScanStatus, errorMsg string, finishedAt *time.Time) error {
	_, err := q.db.Exec(`
		UPDATE slow_scan_tasks
		SET status = ?, error_message = ?, finished_at = ?, updated_at = ?
		WHERE id = ?`,
		status, errorMsg, finishedAt, time.Now().UTC(), id)
	return err
}

func (q *Queries) SetSlowScanRunning(id string, startedAt time.Time) error {
	_, err := q.db.Exec(`
		UPDATE slow_scan_tasks
		SET status = ?, started_at = ?, updated_at = ?
		WHERE id = ?`,
		models.SlowScanRunning, startedAt, time.Now().UTC(), id)
	return err
}

func (q *Queries) DeleteSlowScanTask(id string) error {
	_, err := q.db.Exec(`DELETE FROM slow_scan_tasks WHERE id = ?`, id)
	return err
}

func (q *Queries) CancelSlowScanTasksByRun(runID string) error {
	_, err := q.db.Exec(`
		UPDATE slow_scan_tasks
		SET status = ?, error_message = ?, updated_at = ?
		WHERE run_id = ? AND status IN (?, ?)`,
		models.SlowScanCancelled, "cancelled by run", time.Now().UTC(), runID, models.SlowScanPending, models.SlowScanRunning)
	return err
}

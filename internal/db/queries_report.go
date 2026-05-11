package db

import (
	"database/sql"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Audit Logs ---

func (q *Queries) CreateAuditLog(a *models.AuditLog) error {
	_, err := q.db.Exec(`INSERT INTO audit_logs (id, project_id, actor, action, resource_type, resource_id, summary, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ProjectID, a.Actor, a.Action, a.ResourceType, a.ResourceID, a.Summary, a.CreatedAt)
	return err
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

// --- Reports ---

func (q *Queries) CreateReport(r *models.Report) error {
	_, err := q.db.Exec(`INSERT INTO reports (id, run_id, status, title, finding_count, evidence_count, file_path, file_size_bytes, error_message, created_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.RunID, string(r.Status), r.Title, r.FindingCount, r.EvidenceCount, r.FilePath, r.FileSizeBytes, r.ErrorMessage, r.CreatedAt, r.CompletedAt)
	return err
}

func (q *Queries) GetReport(id string) (*models.Report, error) {
	row := q.db.QueryRow(`SELECT id, run_id, status, title, finding_count, evidence_count, file_path, file_size_bytes, error_message, created_at, completed_at FROM reports WHERE id = ?`, id)
	r := &models.Report{}
	var title, filePath, errorMsg sql.NullString
	var completedAt sql.NullTime
	err := row.Scan(&r.ID, &r.RunID, &r.Status, &title, &r.FindingCount, &r.EvidenceCount, &filePath, &r.FileSizeBytes, &errorMsg, &r.CreatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if title.Valid {
		r.Title = title.String
	}
	if filePath.Valid {
		r.FilePath = filePath.String
	}
	if errorMsg.Valid {
		r.ErrorMessage = errorMsg.String
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	return r, err
}

func (q *Queries) GetReportByRunID(runID string) (*models.Report, error) {
	row := q.db.QueryRow(`SELECT id, run_id, status, title, finding_count, evidence_count, file_path, file_size_bytes, error_message, created_at, completed_at FROM reports WHERE run_id = ?`, runID)
	r := &models.Report{}
	var title, filePath, errorMsg sql.NullString
	var completedAt sql.NullTime
	err := row.Scan(&r.ID, &r.RunID, &r.Status, &title, &r.FindingCount, &r.EvidenceCount, &filePath, &r.FileSizeBytes, &errorMsg, &r.CreatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if title.Valid {
		r.Title = title.String
	}
	if filePath.Valid {
		r.FilePath = filePath.String
	}
	if errorMsg.Valid {
		r.ErrorMessage = errorMsg.String
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	return r, err
}

func (q *Queries) UpdateReport(r *models.Report) error {
	_, err := q.db.Exec(`UPDATE reports SET status = ?, title = ?, finding_count = ?, evidence_count = ?, file_path = ?, file_size_bytes = ?, error_message = ?, completed_at = ? WHERE id = ?`,
		string(r.Status), r.Title, r.FindingCount, r.EvidenceCount, r.FilePath, r.FileSizeBytes, r.ErrorMessage, r.CompletedAt, r.ID)
	return err
}

func (q *Queries) DeleteReport(id string) error {
	_, err := q.db.Exec(`DELETE FROM reports WHERE id = ?`, id)
	return err
}

func (q *Queries) ListReports(cursor string, limit int) ([]*models.Report, bool, error) {
	query := `SELECT id, run_id, status, title, finding_count, evidence_count, file_path, file_size_bytes, error_message, created_at, completed_at FROM reports`
	args := []interface{}{}
	if cursor != "" {
		query += ` WHERE created_at < (SELECT created_at FROM reports WHERE id = ?)`
		args = append(args, cursor)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var list []*models.Report
	for rows.Next() {
		r := &models.Report{}
		var title, filePath, errorMsg sql.NullString
		var completedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.RunID, &r.Status, &title, &r.FindingCount, &r.EvidenceCount, &filePath, &r.FileSizeBytes, &errorMsg, &r.CreatedAt, &completedAt); err != nil {
			return nil, false, err
		}
		if title.Valid {
			r.Title = title.String
		}
		if filePath.Valid {
			r.FilePath = filePath.String
		}
		if errorMsg.Valid {
			r.ErrorMessage = errorMsg.String
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		list = append(list, r)
	}

	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return list, hasMore, rows.Err()
}

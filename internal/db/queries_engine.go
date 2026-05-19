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

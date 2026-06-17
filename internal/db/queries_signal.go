package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// CreateSignal inserts a new signal.
func (q *Queries) CreateSignal(s *models.Signal) error {
	if s.ID == "" {
		s.ID = util.GenerateID()
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}
	if s.FirstSeen.IsZero() {
		s.FirstSeen = now
	}
	if s.LastSeen.IsZero() {
		s.LastSeen = now
	}
	_, err := q.db.Exec(`
		INSERT INTO signals (id, project_id, source_kind, source_id, title, severity, score, scope_status, status, metadata, first_seen, last_seen, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ProjectID, s.SourceKind, s.SourceID, s.Title, s.Severity, s.Score,
		s.ScopeStatus, s.Status, s.Metadata, s.FirstSeen, s.LastSeen, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

// ListSignalsByProject returns all signals for a project, newest first.
func (q *Queries) ListSignalsByProject(projectID string) ([]*models.Signal, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, source_kind, source_id, title, severity, score, scope_status, status, metadata, first_seen, last_seen, created_at, updated_at
		FROM signals
		WHERE project_id = ?
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Signal, 0)
	for rows.Next() {
		s := &models.Signal{}
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.SourceKind, &s.SourceID, &s.Title, &s.Severity, &s.Score,
			&s.ScopeStatus, &s.Status, &s.Metadata, &s.FirstSeen, &s.LastSeen, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// UpdateSignalStatus updates the status of a signal.
func (q *Queries) UpdateSignalStatus(signalID string, status string) error {
	_, err := q.db.Exec(`UPDATE signals SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC(), signalID)
	return err
}

// CountSignalsByProject returns the count of signals for a project, optionally filtered by status.
func (q *Queries) CountSignalsByProject(projectID string, status *string) (int, error) {
	var count int
	var row *sql.Row
	if status != nil && *status != "" {
		row = q.db.QueryRow(`SELECT COUNT(*) FROM signals WHERE project_id = ? AND status = ?`, projectID, *status)
	} else {
		row = q.db.QueryRow(`SELECT COUNT(*) FROM signals WHERE project_id = ?`, projectID)
	}
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetSignalsBySource returns existing signals for a source to enable dedup/upsert.
func (q *Queries) GetSignalsBySource(sourceKind, sourceID string) ([]*models.Signal, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, source_kind, source_id, title, severity, score, scope_status, status, metadata, first_seen, last_seen, created_at, updated_at
		FROM signals
		WHERE source_kind = ? AND source_id = ?
		ORDER BY created_at DESC`, sourceKind, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Signal, 0)
	for rows.Next() {
		s := &models.Signal{}
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.SourceKind, &s.SourceID, &s.Title, &s.Severity, &s.Score,
			&s.ScopeStatus, &s.Status, &s.Metadata, &s.FirstSeen, &s.LastSeen, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// UpdateSignalLastSeen bumps the last_seen timestamp for an existing signal.
func (q *Queries) UpdateSignalLastSeen(signalID string, score int) error {
	_, err := q.db.Exec(`UPDATE signals SET last_seen = ?, score = ?, updated_at = ? WHERE id = ?`,
		time.Now().UTC(), score, time.Now().UTC(), signalID)
	return err
}

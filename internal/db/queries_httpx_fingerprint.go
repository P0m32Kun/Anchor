package db

import (
	"database/sql"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (q *Queries) CreateHttpxFingerprint(f *models.HttpxFingerprint) error {
	_, err := q.db.Exec(`
		INSERT INTO httpx_fingerprints (id, name, description, type, file_path, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.Name, f.Description, f.Type, f.FilePath, f.Enabled, f.CreatedAt, f.UpdatedAt)
	return err
}

func (q *Queries) GetHttpxFingerprint(id string) (*models.HttpxFingerprint, error) {
	row := q.db.QueryRow(`
		SELECT id, name, description, type, file_path, enabled, created_at, updated_at
		FROM httpx_fingerprints WHERE id = ?`, id)
	f := &models.HttpxFingerprint{}
	err := row.Scan(&f.ID, &f.Name, &f.Description, &f.Type, &f.FilePath, &f.Enabled, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return f, err
}

func (q *Queries) ListHttpxFingerprints(fpType string) ([]*models.HttpxFingerprint, error) {
	var rows *sql.Rows
	var err error
	if fpType != "" {
		rows, err = q.db.Query(`
			SELECT id, name, description, type, file_path, enabled, created_at, updated_at
			FROM httpx_fingerprints WHERE type = ? ORDER BY created_at DESC`, fpType)
	} else {
		rows, err = q.db.Query(`
			SELECT id, name, description, type, file_path, enabled, created_at, updated_at
			FROM httpx_fingerprints ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.HttpxFingerprint, 0)
	for rows.Next() {
		f := &models.HttpxFingerprint{}
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Type, &f.FilePath, &f.Enabled, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (q *Queries) ListEnabledHttpxFingerprints(fpType string) ([]*models.HttpxFingerprint, error) {
	var rows *sql.Rows
	var err error
	if fpType != "" {
		rows, err = q.db.Query(`
			SELECT id, name, description, type, file_path, enabled, created_at, updated_at
			FROM httpx_fingerprints WHERE type = ? AND enabled = 1 ORDER BY created_at DESC`, fpType)
	} else {
		rows, err = q.db.Query(`
			SELECT id, name, description, type, file_path, enabled, created_at, updated_at
			FROM httpx_fingerprints WHERE enabled = 1 ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.HttpxFingerprint, 0)
	for rows.Next() {
		f := &models.HttpxFingerprint{}
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Type, &f.FilePath, &f.Enabled, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateHttpxFingerprint(f *models.HttpxFingerprint) error {
	_, err := q.db.Exec(`
		UPDATE httpx_fingerprints
		SET name = ?, description = ?, type = ?, file_path = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		f.Name, f.Description, f.Type, f.FilePath, f.Enabled, f.UpdatedAt, f.ID)
	return err
}

func (q *Queries) DeleteHttpxFingerprint(id string) error {
	_, err := q.db.Exec(`DELETE FROM httpx_fingerprints WHERE id = ?`, id)
	return err
}

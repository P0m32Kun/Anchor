package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (q *Queries) CreateHttpxFingerprint(f *models.HttpxFingerprint) error {
	_, err := q.db.Exec(`
		INSERT INTO httpx_fingerprints (id, name, description, type, file_path, enabled, builtin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.Name, f.Description, f.Type, f.FilePath,
		boolToInt(f.Enabled), boolToInt(f.Builtin), f.CreatedAt, f.UpdatedAt)
	return err
}

func (q *Queries) GetHttpxFingerprint(id string) (*models.HttpxFingerprint, error) {
	row := q.db.QueryRow(`
		SELECT id, name, description, type, file_path, enabled, builtin, created_at, updated_at
		FROM httpx_fingerprints WHERE id = ?`, id)
	return scanHttpxFingerprint(row)
}

func (q *Queries) ListHttpxFingerprints(fpType string) ([]*models.HttpxFingerprint, error) {
	var rows *sql.Rows
	var err error
	if fpType != "" {
		rows, err = q.db.Query(`
			SELECT id, name, description, type, file_path, enabled, builtin, created_at, updated_at
			FROM httpx_fingerprints WHERE type = ? ORDER BY created_at DESC`, fpType)
	} else {
		rows, err = q.db.Query(`
			SELECT id, name, description, type, file_path, enabled, builtin, created_at, updated_at
			FROM httpx_fingerprints ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.HttpxFingerprint, 0)
	for rows.Next() {
		f, err := scanHttpxFingerprint(rows)
		if err != nil {
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
			SELECT id, name, description, type, file_path, enabled, builtin, created_at, updated_at
			FROM httpx_fingerprints WHERE type = ? AND enabled = 1 ORDER BY created_at DESC`, fpType)
	} else {
		rows, err = q.db.Query(`
			SELECT id, name, description, type, file_path, enabled, builtin, created_at, updated_at
			FROM httpx_fingerprints WHERE enabled = 1 ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.HttpxFingerprint, 0)
	for rows.Next() {
		f, err := scanHttpxFingerprint(rows)
		if err != nil {
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
		f.Name, f.Description, f.Type, f.FilePath, boolToInt(f.Enabled), f.UpdatedAt, f.ID)
	return err
}

func (q *Queries) UpdateHttpxFingerprintEnabled(id string, enabled bool, updatedAt time.Time) error {
	_, err := q.db.Exec(`
		UPDATE httpx_fingerprints SET enabled = ?, updated_at = ?
		WHERE id = ? AND builtin = 1`,
		boolToInt(enabled), updatedAt, id)
	return err
}

func (q *Queries) DeleteHttpxFingerprint(id string) error {
	_, err := q.db.Exec(`DELETE FROM httpx_fingerprints WHERE id = ?`, id)
	return err
}

func scanHttpxFingerprint(row scanRow) (*models.HttpxFingerprint, error) {
	f := &models.HttpxFingerprint{}
	var enabledInt, builtinInt int
	err := row.Scan(
		&f.ID, &f.Name, &f.Description, &f.Type, &f.FilePath,
		&enabledInt, &builtinInt, &f.CreatedAt, &f.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.Enabled = enabledInt != 0
	f.Builtin = builtinInt != 0
	return f, nil
}

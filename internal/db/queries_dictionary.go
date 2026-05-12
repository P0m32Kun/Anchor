package db

import (
	"database/sql"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (q *Queries) CreateDictionary(d *models.Dictionary) error {
	_, err := q.db.Exec(`
		INSERT INTO dictionaries (id, name, description, category, file_path, line_count, size_bytes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.Name, d.Description, d.Category, d.FilePath, d.LineCount, d.SizeBytes, d.CreatedAt, d.UpdatedAt)
	return err
}

func (q *Queries) GetDictionary(id string) (*models.Dictionary, error) {
	row := q.db.QueryRow(`
		SELECT id, name, description, category, file_path, line_count, size_bytes, created_at, updated_at
		FROM dictionaries WHERE id = ?`, id)
	d := &models.Dictionary{}
	err := row.Scan(&d.ID, &d.Name, &d.Description, &d.Category, &d.FilePath, &d.LineCount, &d.SizeBytes, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (q *Queries) ListDictionaries(category string) ([]*models.Dictionary, error) {
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = q.db.Query(`
			SELECT id, name, description, category, file_path, line_count, size_bytes, created_at, updated_at
			FROM dictionaries WHERE category = ? ORDER BY created_at DESC`, category)
	} else {
		rows, err = q.db.Query(`
			SELECT id, name, description, category, file_path, line_count, size_bytes, created_at, updated_at
			FROM dictionaries ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.Dictionary, 0)
	for rows.Next() {
		d := &models.Dictionary{}
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.Category, &d.FilePath, &d.LineCount, &d.SizeBytes, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateDictionary(d *models.Dictionary) error {
	_, err := q.db.Exec(`
		UPDATE dictionaries
		SET name = ?, description = ?, category = ?, file_path = ?, line_count = ?, size_bytes = ?, updated_at = ?
		WHERE id = ?`,
		d.Name, d.Description, d.Category, d.FilePath, d.LineCount, d.SizeBytes, d.UpdatedAt, d.ID)
	return err
}

func (q *Queries) DeleteDictionary(id string) error {
	_, err := q.db.Exec(`DELETE FROM dictionaries WHERE id = ?`, id)
	return err
}

package db

import (
	"database/sql"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (q *Queries) CreateDictionary(d *models.Dictionary) error {
	_, err := q.db.Exec(`
		INSERT INTO dictionaries (id, name, description, category, file_path, line_count, size_bytes, builtin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.Name, d.Description, d.Category, d.FilePath, d.LineCount, d.SizeBytes, boolToInt(d.Builtin), d.CreatedAt, d.UpdatedAt)
	return err
}

func (q *Queries) GetDictionary(id string) (*models.Dictionary, error) {
	row := q.db.QueryRow(`
		SELECT id, name, description, category, file_path, line_count, size_bytes, builtin, created_at, updated_at
		FROM dictionaries WHERE id = ?`, id)
	return scanDictionary(row)
}

func (q *Queries) ListDictionaries(category string) ([]*models.Dictionary, error) {
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = q.db.Query(`
			SELECT id, name, description, category, file_path, line_count, size_bytes, builtin, created_at, updated_at
			FROM dictionaries WHERE category = ? ORDER BY builtin DESC, name ASC`, category)
	} else {
		rows, err = q.db.Query(`
			SELECT id, name, description, category, file_path, line_count, size_bytes, builtin, created_at, updated_at
			FROM dictionaries ORDER BY builtin DESC, category ASC, name ASC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.Dictionary, 0)
	for rows.Next() {
		d, err := scanDictionary(rows)
		if err != nil {
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

// ListBuiltinDictionaries returns every builtin dictionary regardless of category.
// Used by the seed routine to reconcile DB state with on-disk /opt/dict files.
func (q *Queries) ListBuiltinDictionaries() ([]*models.Dictionary, error) {
	rows, err := q.db.Query(`
		SELECT id, name, description, category, file_path, line_count, size_bytes, builtin, created_at, updated_at
		FROM dictionaries WHERE builtin = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.Dictionary, 0)
	for rows.Next() {
		d, err := scanDictionary(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func scanDictionary(row scanRow) (*models.Dictionary, error) {
	d := &models.Dictionary{}
	var builtinInt int
	err := row.Scan(&d.ID, &d.Name, &d.Description, &d.Category, &d.FilePath, &d.LineCount, &d.SizeBytes, &builtinInt, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.Builtin = builtinInt != 0
	return d, nil
}

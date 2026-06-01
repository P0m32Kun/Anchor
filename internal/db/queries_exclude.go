package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Excluded Domains ---

func (q *Queries) CreateExcludedDomain(d *models.ExcludedDomain) error {
	_, err := q.db.Exec(`INSERT INTO excluded_domains (id, domain, reason, builtin, created_at) VALUES (?, ?, ?, ?, ?)`,
		d.ID, d.Domain, d.Reason, d.Builtin, d.CreatedAt)
	return err
}

func (q *Queries) GetExcludedDomain(domain string) (*models.ExcludedDomain, error) {
	row := q.db.QueryRow(`SELECT id, domain, reason, builtin, created_at FROM excluded_domains WHERE domain = ?`, domain)
	d := &models.ExcludedDomain{}
	err := row.Scan(&d.ID, &d.Domain, &d.Reason, &d.Builtin, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (q *Queries) ListExcludedDomains() ([]*models.ExcludedDomain, error) {
	rows, err := q.db.Query(`SELECT id, domain, reason, builtin, created_at FROM excluded_domains ORDER BY builtin DESC, domain ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ExcludedDomain, 0)
	for rows.Next() {
		d := &models.ExcludedDomain{}
		if err := rows.Scan(&d.ID, &d.Domain, &d.Reason, &d.Builtin, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (q *Queries) ListCustomExcludedDomains() ([]*models.ExcludedDomain, error) {
	rows, err := q.db.Query(`SELECT id, domain, reason, builtin, created_at FROM excluded_domains WHERE builtin = 0 ORDER BY domain ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ExcludedDomain, 0)
	for rows.Next() {
		d := &models.ExcludedDomain{}
		if err := rows.Scan(&d.ID, &d.Domain, &d.Reason, &d.Builtin, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (q *Queries) DeleteExcludedDomain(domain string) error {
	_, err := q.db.Exec(`DELETE FROM excluded_domains WHERE domain = ? AND builtin = 0`, domain)
	return err
}

func (q *Queries) DeleteAllCustomExcludedDomains() error {
	_, err := q.db.Exec(`DELETE FROM excluded_domains WHERE builtin = 0`)
	return err
}

func (q *Queries) ExcludedDomainExists(domain string) (bool, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(1) FROM excluded_domains WHERE domain = ?`, domain)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// BulkCreateExcludedDomains inserts multiple excluded domains.
func (q *Queries) BulkCreateExcludedDomains(domains []*models.ExcludedDomain) error {
	now := time.Now().UTC()
	for _, d := range domains {
		if d.CreatedAt.IsZero() {
			d.CreatedAt = now
		}
		if _, err := q.db.Exec(`INSERT OR IGNORE INTO excluded_domains (id, domain, reason, builtin, created_at) VALUES (?, ?, ?, ?, ?)`,
			d.ID, d.Domain, d.Reason, d.Builtin, d.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

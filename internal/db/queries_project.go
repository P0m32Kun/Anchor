package db

import (
	"database/sql"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Projects ---

func (q *Queries) CreateProject(p *models.Project) error {
	_, err := q.db.Exec(`
		INSERT INTO projects (id, name, organization, purpose, rate_limit, port_range, default_profile, pipeline_config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Organization, p.Purpose, p.RateLimit, p.PortRange, p.DefaultProfile, p.PipelineConfig, p.CreatedAt, p.UpdatedAt)
	return err
}

func (q *Queries) GetProject(id string) (*models.Project, error) {
	row := q.db.QueryRow(`SELECT id, name, organization, purpose, rate_limit, port_range, default_profile, pipeline_config, created_at, updated_at FROM projects WHERE id = ?`, id)
	p := &models.Project{}
	err := row.Scan(&p.ID, &p.Name, &p.Organization, &p.Purpose, &p.RateLimit, &p.PortRange, &p.DefaultProfile, &p.PipelineConfig, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (q *Queries) ListProjects() ([]*models.Project, error) {
	rows, err := q.db.Query(`SELECT id, name, organization, purpose, rate_limit, port_range, default_profile, pipeline_config, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Project, 0)
	for rows.Next() {
		p := &models.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Organization, &p.Purpose, &p.RateLimit, &p.PortRange, &p.DefaultProfile, &p.PipelineConfig, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (q *Queries) CountProjects() (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM projects`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListProjectsPaginated(limit, offset int) ([]*models.Project, error) {
	rows, err := q.db.Query(`SELECT id, name, organization, purpose, rate_limit, port_range, default_profile, pipeline_config, created_at, updated_at FROM projects ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Project, 0)
	for rows.Next() {
		p := &models.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Organization, &p.Purpose, &p.RateLimit, &p.PortRange, &p.DefaultProfile, &p.PipelineConfig, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (q *Queries) DeleteProject(id string) error {
	_, err := q.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (q *Queries) UpdateProjectPipelineConfig(projectID string, cfgJSON string) error {
	_, err := q.db.Exec(`UPDATE projects SET pipeline_config = ? WHERE id = ?`, cfgJSON, projectID)
	return err
}

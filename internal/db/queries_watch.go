package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// ListWatchEnabledProjects returns projects with watch_enabled=1.
func (q *Queries) ListWatchEnabledProjects() ([]*models.WatchProject, error) {
	rows, err := q.db.Query(`
		SELECT id, name, watch_enabled, watch_interval_hours, watch_passive_only, watch_last_tick_at
		FROM projects
		WHERE watch_enabled = 1
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.WatchProject, 0)
	for rows.Next() {
		p := &models.WatchProject{}
		var enabled int
		var passiveOnly int
		if err := rows.Scan(&p.ID, &p.Name, &enabled, &p.WatchIntervalHours, &passiveOnly, &p.WatchLastTickAt); err != nil {
			return nil, err
		}
		p.WatchEnabled = enabled != 0
		p.WatchPassiveOnly = passiveOnly != 0
		list = append(list, p)
	}
	return list, rows.Err()
}

// UpdateProjectWatchTick updates the watch_last_tick_at timestamp for a project.
func (q *Queries) UpdateProjectWatchTick(projectID string, tickAt time.Time) error {
	_, err := q.db.Exec(`UPDATE projects SET watch_last_tick_at = ? WHERE id = ?`, tickAt, projectID)
	return err
}

// UpdateProjectWatchConfig updates watch settings on a project.
func (q *Queries) UpdateProjectWatchConfig(projectID string, enabled bool, intervalHours int, passiveOnly bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	passiveInt := 0
	if passiveOnly {
		passiveInt = 1
	}
	_, err := q.db.Exec(
		`UPDATE projects SET watch_enabled = ?, watch_interval_hours = ?, watch_passive_only = ? WHERE id = ?`,
		enabledInt, intervalHours, passiveInt, projectID,
	)
	return err
}

// GetWatchProject returns watch config for a single project regardless of watch_enabled.
func (q *Queries) GetWatchProject(projectID string) (*models.WatchProject, error) {
	row := q.db.QueryRow(`
		SELECT id, name, watch_enabled, watch_interval_hours, watch_passive_only, watch_last_tick_at
		FROM projects
		WHERE id = ?`, projectID)
	p := &models.WatchProject{}
	var enabled int
	var passiveOnly int
	if err := row.Scan(&p.ID, &p.Name, &enabled, &p.WatchIntervalHours, &passiveOnly, &p.WatchLastTickAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.WatchEnabled = enabled != 0
	p.WatchPassiveOnly = passiveOnly != 0
	return p, nil
}

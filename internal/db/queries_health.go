package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Tool Health ---

func (q *Queries) UpsertToolHealth(h *models.ToolHealth) error {
	_, err := q.db.Exec(`
		INSERT INTO tool_health (id, tool, binary_path, version, template_path, workdir_writable, network_available, dns_available, proxy_reachable, last_check_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tool) DO UPDATE SET
			binary_path = excluded.binary_path,
			version = excluded.version,
			template_path = excluded.template_path,
			workdir_writable = excluded.workdir_writable,
			network_available = excluded.network_available,
			dns_available = excluded.dns_available,
			proxy_reachable = excluded.proxy_reachable,
			last_check_at = excluded.last_check_at`,
		h.ID, h.Tool, h.BinaryPath, h.Version, h.TemplatePath, h.WorkdirWritable, h.NetworkAvailable, h.DNSAvailable, h.ProxyReachable, h.LastCheckAt)
	return err
}

func (q *Queries) ListToolHealth() ([]*models.ToolHealth, error) {
	rows, err := q.db.Query(`SELECT id, tool, binary_path, version, template_path, workdir_writable, network_available, dns_available, proxy_reachable, last_check_at FROM tool_health ORDER BY tool`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ToolHealth, 0)
	for rows.Next() {
		h := &models.ToolHealth{}
		var proxyReachable sql.NullBool
		var templatePath sql.NullString
		if err := rows.Scan(&h.ID, &h.Tool, &h.BinaryPath, &h.Version, &templatePath, &h.WorkdirWritable, &h.NetworkAvailable, &h.DNSAvailable, &proxyReachable, &h.LastCheckAt); err != nil {
			return nil, err
		}
		h.TemplatePath = nullableString(templatePath)
		h.ProxyReachable = nullableBool(proxyReachable)
		list = append(list, h)
	}
	return list, rows.Err()
}

// --- ToolTemplate ---

func (q *Queries) ListToolTemplates() ([]*models.ToolTemplate, error) {
	rows, err := q.db.Query(`
		SELECT id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, directory_bruteforce_enabled, nuclei_severity_filter, created_at, updated_at
		FROM tool_templates ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ToolTemplate, 0)
	for rows.Next() {
		t := &models.ToolTemplate{}
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Description, &t.ProfileType, &t.ToolsJSON,
			&t.DefaultMaxConcurrency, &t.ScreenshotEnabled, &t.DirectoryBruteforceEnabled,
			&t.NucleiSeverityFilter, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (q *Queries) GetToolTemplate(id string) (*models.ToolTemplate, error) {
	row := q.db.QueryRow(`
		SELECT id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, directory_bruteforce_enabled, nuclei_severity_filter, created_at, updated_at
		FROM tool_templates WHERE id = ?`, id)
	t := &models.ToolTemplate{}
	if err := row.Scan(
		&t.ID, &t.Name, &t.Description, &t.ProfileType, &t.ToolsJSON,
		&t.DefaultMaxConcurrency, &t.ScreenshotEnabled, &t.DirectoryBruteforceEnabled,
		&t.NucleiSeverityFilter, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// --- ToolInvocation ---

func (q *Queries) CreateToolInvocation(inv *models.ToolInvocation) error {
	_, err := q.db.Exec(`INSERT INTO tool_invocations (id, project_id, task_id, tool, binary_path, version, command_redacted, workdir, started_at, finished_at, exit_code) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.ProjectID, inv.TaskID, inv.Tool, inv.BinaryPath, inv.Version, inv.CommandRedacted, inv.Workdir, inv.StartedAt, inv.FinishedAt, inv.ExitCode)
	return err
}

func (q *Queries) UpdateToolInvocation(taskID string, finishedAt time.Time, exitCode int) error {
	_, err := q.db.Exec(`UPDATE tool_invocations SET finished_at = ?, exit_code = ? WHERE task_id = ?`,
		finishedAt, exitCode, taskID)
	return err
}

// ListToolInvocationsByProject returns all tool invocations for a project.
func (q *Queries) ListToolInvocationsByProject(projectID string) ([]*models.ToolInvocation, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, task_id, tool, binary_path, version, command_redacted, workdir, started_at, finished_at, exit_code
		FROM tool_invocations WHERE project_id = ? ORDER BY started_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ToolInvocation, 0)
	for rows.Next() {
		inv := &models.ToolInvocation{}
		if err := rows.Scan(&inv.ID, &inv.ProjectID, &inv.TaskID, &inv.Tool, &inv.BinaryPath, &inv.Version, &inv.CommandRedacted, &inv.Workdir, &inv.StartedAt, &inv.FinishedAt, &inv.ExitCode); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, rows.Err()
}

package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- WorkerNode ---

func (q *Queries) CreateWorkerNode(w *models.WorkerNode) error {
	_, err := q.db.Exec(`
		INSERT INTO worker_nodes (id, name, endpoint, mode, status, trust_level, network_profile, capabilities, tool_versions, template_versions, max_concurrency, last_seen, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.Name, w.Endpoint, w.Mode, w.Status, w.TrustLevel, w.NetworkProfile, w.Capabilities, w.ToolVersions, w.TemplateVersions, w.MaxConcurrency, w.LastSeen, w.CreatedAt)
	return err
}

func (q *Queries) GetWorkerNode(id string) (*models.WorkerNode, error) {
	row := q.db.QueryRow(`
		SELECT id, name, endpoint, mode, status, trust_level, network_profile, capabilities, tool_versions, template_versions, max_concurrency, last_seen, created_at, revoked_at
		FROM worker_nodes WHERE id = ?`, id)
	w := &models.WorkerNode{}
	var lastSeen, revokedAt sql.NullTime
	if err := row.Scan(&w.ID, &w.Name, &w.Endpoint, &w.Mode, &w.Status, &w.TrustLevel, &w.NetworkProfile, &w.Capabilities, &w.ToolVersions, &w.TemplateVersions, &w.MaxConcurrency, &lastSeen, &w.CreatedAt, &revokedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastSeen.Valid {
		w.LastSeen = &lastSeen.Time
	}
	if revokedAt.Valid {
		w.RevokedAt = &revokedAt.Time
	}
	return w, nil
}

func (q *Queries) ListWorkerNodes() ([]*models.WorkerNode, error) {
	rows, err := q.db.Query(`
		SELECT id, name, endpoint, mode, status, trust_level, network_profile, capabilities, tool_versions, template_versions, max_concurrency, last_seen, cpu_percent, mem_percent, disk_percent, metrics_updated_at, created_at, revoked_at
		FROM worker_nodes ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.WorkerNode, 0)
	for rows.Next() {
		w := &models.WorkerNode{}
		var lastSeen, revokedAt, metricsAt sql.NullTime
		var cpu, mem, disk sql.NullFloat64
		if err := rows.Scan(&w.ID, &w.Name, &w.Endpoint, &w.Mode, &w.Status, &w.TrustLevel, &w.NetworkProfile, &w.Capabilities, &w.ToolVersions, &w.TemplateVersions, &w.MaxConcurrency, &lastSeen, &cpu, &mem, &disk, &metricsAt, &w.CreatedAt, &revokedAt); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			w.LastSeen = &lastSeen.Time
		}
		if revokedAt.Valid {
			w.RevokedAt = &revokedAt.Time
		}
		if cpu.Valid {
			w.CPUPercent = &cpu.Float64
		}
		if mem.Valid {
			w.MemPercent = &mem.Float64
		}
		if disk.Valid {
			w.DiskPercent = &disk.Float64
		}
		if metricsAt.Valid {
			w.MetricsUpdatedAt = &metricsAt.Time
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateWorkerNodeStatus(id string, status models.WorkerStatus, lastSeen time.Time) error {
	_, err := q.db.Exec(`UPDATE worker_nodes SET status = ?, last_seen = ? WHERE id = ?`, status, lastSeen, id)
	return err
}

// UpdateWorkerNodeTemplateVersions persists the worker's template version
// report (JSON blob) alongside the heartbeat status update.
func (q *Queries) UpdateWorkerNodeTemplateVersions(id string, status models.WorkerStatus, lastSeen time.Time, templateVersions string) error {
	_, err := q.db.Exec(`UPDATE worker_nodes SET status = ?, last_seen = ?, template_versions = ? WHERE id = ?`,
		status, lastSeen, templateVersions, id)
	return err
}

// UpdateWorkerNodeMetrics persists the worker's resource metrics.
func (q *Queries) UpdateWorkerNodeMetrics(id string, cpu, mem, disk *float64, updatedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE worker_nodes SET cpu_percent = ?, mem_percent = ?, disk_percent = ?, metrics_updated_at = ? WHERE id = ?`,
		cpu, mem, disk, updatedAt, id)
	return err
}

func (q *Queries) RevokeWorkerNode(id string, revokedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE worker_nodes SET status = ?, revoked_at = ? WHERE id = ?`, models.WorkerStatusOffline, revokedAt, id)
	return err
}

func (q *Queries) DeleteWorkerNode(id string) error {
	_, err := q.db.Exec(`DELETE FROM worker_nodes WHERE id = ?`, id)
	return err
}

// --- WorkerHealthCheck ---

func (q *Queries) CreateWorkerHealthCheck(h *models.WorkerHealthCheck) error {
	_, err := q.db.Exec(`
		INSERT INTO worker_health_checks (id, worker_id, tool, status, version, details, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		h.ID, h.WorkerID, h.Tool, h.Status, h.Version, h.Details, h.CheckedAt)
	return err
}

func (q *Queries) ListWorkerHealthChecks(workerID string) ([]*models.WorkerHealthCheck, error) {
	rows, err := q.db.Query(`
		SELECT id, worker_id, tool, status, version, details, checked_at
		FROM worker_health_checks WHERE worker_id = ? ORDER BY checked_at DESC`, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.WorkerHealthCheck, 0)
	for rows.Next() {
		h := &models.WorkerHealthCheck{}
		if err := rows.Scan(&h.ID, &h.WorkerID, &h.Tool, &h.Status, &h.Version, &h.Details, &h.CheckedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

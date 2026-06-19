package db

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// CreateAssetChange inserts a new asset change record.
func (q *Queries) CreateAssetChange(c *models.AssetChange) error {
	if c.ID == "" {
		c.ID = util.GenerateID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	_, err := q.db.Exec(`
		INSERT INTO asset_changes (id, project_id, run_id, asset_id, asset_value, asset_type, change_type, change_summary, detail_json, severity, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.ProjectID, c.RunID, c.AssetID, c.AssetValue, c.AssetType,
		c.ChangeType, c.ChangeSummary, c.DetailJSON, c.Severity, c.CreatedAt,
	)
	return err
}

// ListAssetChangesByProject returns recent asset changes for a project.
func (q *Queries) ListAssetChangesByProject(projectID string, limit, offset int) ([]*models.AssetChange, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := q.db.Query(`
		SELECT id, project_id, run_id, asset_id, asset_value, asset_type, change_type, change_summary, detail_json, severity, created_at
		FROM asset_changes
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.AssetChange, 0)
	for rows.Next() {
		c := &models.AssetChange{}
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.RunID, &c.AssetID, &c.AssetValue, &c.AssetType,
			&c.ChangeType, &c.ChangeSummary, &c.DetailJSON, &c.Severity, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// ListAssetChangesByAsset returns the change timeline for a specific asset.
func (q *Queries) ListAssetChangesByAsset(assetID string) ([]*models.AssetChange, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, run_id, asset_id, asset_value, asset_type, change_type, change_summary, detail_json, severity, created_at
		FROM asset_changes
		WHERE asset_id = ?
		ORDER BY created_at DESC
		LIMIT 200`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.AssetChange, 0)
	for rows.Next() {
		c := &models.AssetChange{}
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.RunID, &c.AssetID, &c.AssetValue, &c.AssetType,
			&c.ChangeType, &c.ChangeSummary, &c.DetailJSON, &c.Severity, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// GetAlertWebhook returns the alert webhook config for a project.
func (q *Queries) GetAlertWebhook(projectID string) (*models.AlertWebhook, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, enabled, url, secret, min_severity, on_new_asset, on_asset_gone,
		    on_port_change, on_service_change, on_cert_expiry, created_at, updated_at
		FROM alert_webhooks
		WHERE project_id = ?`, projectID)
	w := &models.AlertWebhook{}
	err := row.Scan(&w.ID, &w.ProjectID, &w.Enabled, &w.URL, &w.Secret, &w.MinSeverity,
		&w.OnNewAsset, &w.OnAssetGone, &w.OnPortChange, &w.OnServiceChange, &w.OnCertExpiry,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// UpsertAlertWebhook creates or updates the alert webhook config for a project.
func (q *Queries) UpsertAlertWebhook(w *models.AlertWebhook) error {
	if w.ID == "" {
		w.ID = util.GenerateID()
	}
	now := time.Now().UTC()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	w.UpdatedAt = now
	_, err := q.db.Exec(`
		INSERT INTO alert_webhooks (id, project_id, enabled, url, secret, min_severity,
		    on_new_asset, on_asset_gone, on_port_change, on_service_change, on_cert_expiry, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			enabled = excluded.enabled,
			url = excluded.url,
			secret = excluded.secret,
			min_severity = excluded.min_severity,
			on_new_asset = excluded.on_new_asset,
			on_asset_gone = excluded.on_asset_gone,
			on_port_change = excluded.on_port_change,
			on_service_change = excluded.on_service_change,
			on_cert_expiry = excluded.on_cert_expiry,
			updated_at = excluded.updated_at`,
		w.ID, w.ProjectID, boolToInt(w.Enabled), w.URL, w.Secret, w.MinSeverity,
		boolToInt(w.OnNewAsset), boolToInt(w.OnAssetGone), boolToInt(w.OnPortChange),
		boolToInt(w.OnServiceChange), boolToInt(w.OnCertExpiry), w.CreatedAt, w.UpdatedAt,
	)
	return err
}

// DeleteAlertWebhook removes the alert webhook config for a project.
func (q *Queries) DeleteAlertWebhook(projectID string) error {
	_, err := q.db.Exec(`DELETE FROM alert_webhooks WHERE project_id = ?`, projectID)
	return err
}

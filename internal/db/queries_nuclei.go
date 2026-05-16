package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Nuclei Custom Sources ---

func (q *Queries) CreateNucleiCustomSource(s *models.NucleiCustomSource) error {
	_, err := q.db.Exec(`
		INSERT INTO nuclei_custom_sources (
			id, name, install_path, type, uri, branch, enabled, routing_policy, status,
			last_sync_at, last_validate_at, last_error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.InstallPath, string(s.Type), s.URI, s.Branch,
		boolToInt(s.Enabled), s.RoutingPolicy, string(s.Status),
		s.LastSyncAt, s.LastValidateAt, s.LastError, s.CreatedAt, s.UpdatedAt)
	return err
}

func (q *Queries) GetNucleiCustomSource(id string) (*models.NucleiCustomSource, error) {
	row := q.db.QueryRow(`
		SELECT id, name, install_path, type, uri, branch, enabled, routing_policy, status,
		       last_sync_at, last_validate_at, last_error, created_at, updated_at
		FROM nuclei_custom_sources WHERE id = ?`, id)
	return scanNucleiCustomSource(row)
}

// ListEnabledNucleiCustomSourceIDs returns the IDs of all enabled custom sources.
// Used by the pipeline to construct precise -w workflow paths.
func (q *Queries) ListEnabledNucleiCustomSourceIDs() ([]string, error) {
	rows, err := q.db.Query(`SELECT id FROM nuclei_custom_sources WHERE enabled = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (q *Queries) ListNucleiCustomSources() ([]*models.NucleiCustomSource, error) {
	rows, err := q.db.Query(`
		SELECT id, name, install_path, type, uri, branch, enabled, routing_policy, status,
		       last_sync_at, last_validate_at, last_error, created_at, updated_at
		FROM nuclei_custom_sources ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.NucleiCustomSource, 0)
	for rows.Next() {
		s, err := scanNucleiCustomSource(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (q *Queries) UpdateNucleiCustomSource(s *models.NucleiCustomSource) error {
	_, err := q.db.Exec(`
		UPDATE nuclei_custom_sources SET
			name = ?, install_path = ?, type = ?, uri = ?, branch = ?, enabled = ?, routing_policy = ?,
			status = ?, last_sync_at = ?, last_validate_at = ?, last_error = ?, updated_at = ?
		WHERE id = ?`,
		s.Name, s.InstallPath, string(s.Type), s.URI, s.Branch, boolToInt(s.Enabled), s.RoutingPolicy,
		string(s.Status), s.LastSyncAt, s.LastValidateAt, s.LastError, s.UpdatedAt, s.ID)
	return err
}

func (q *Queries) DeleteNucleiCustomSource(id string) error {
	_, err := q.db.Exec(`DELETE FROM nuclei_custom_sources WHERE id = ?`, id)
	return err
}

// scanRow is the minimal interface satisfied by both *sql.Row and *sql.Rows
// (single-row scan calls), which lets the helper handle GET/LIST equally.
type scanRow interface {
	Scan(dest ...any) error
}

func scanNucleiCustomSource(row scanRow) (*models.NucleiCustomSource, error) {
	s := &models.NucleiCustomSource{}
	var typeStr, statusStr string
	var enabledInt int
	var uri, branch, lastError sql.NullString
	var lastSyncAt, lastValidateAt sql.NullTime
	err := row.Scan(
		&s.ID, &s.Name, &s.InstallPath, &typeStr, &uri, &branch, &enabledInt, &s.RoutingPolicy,
		&statusStr, &lastSyncAt, &lastValidateAt, &lastError, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Type = models.NucleiCustomSourceType(typeStr)
	s.Status = models.NucleiCustomSourceStatus(statusStr)
	s.Enabled = enabledInt != 0
	if uri.Valid {
		s.URI = &uri.String
	}
	if branch.Valid {
		s.Branch = &branch.String
	}
	if lastError.Valid {
		s.LastError = &lastError.String
	}
	if lastSyncAt.Valid {
		t := lastSyncAt.Time
		s.LastSyncAt = &t
	}
	if lastValidateAt.Valid {
		t := lastValidateAt.Time
		s.LastValidateAt = &t
	}
	return s, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Nuclei Custom Bundles (Phase 2 wires these; Phase 1 ships placeholders) ---

func (q *Queries) CreateNucleiCustomBundle(b *models.NucleiCustomBundle) error {
	_, err := q.db.Exec(`
		INSERT INTO nuclei_custom_bundles (version, manifest_json, archive_path, status, created_at, activated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		b.Version, b.ManifestJSON, b.ArchivePath, b.Status, b.CreatedAt, b.ActivatedAt)
	return err
}

func (q *Queries) GetNucleiCustomBundle(version string) (*models.NucleiCustomBundle, error) {
	row := q.db.QueryRow(`
		SELECT version, manifest_json, archive_path, status, created_at, activated_at
		FROM nuclei_custom_bundles WHERE version = ?`, version)
	b := &models.NucleiCustomBundle{}
	var activatedAt sql.NullTime
	err := row.Scan(&b.Version, &b.ManifestJSON, &b.ArchivePath, &b.Status, &b.CreatedAt, &activatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if activatedAt.Valid {
		t := activatedAt.Time
		b.ActivatedAt = &t
	}
	return b, nil
}

func (q *Queries) ListNucleiCustomBundles() ([]*models.NucleiCustomBundle, error) {
	rows, err := q.db.Query(`
		SELECT version, manifest_json, archive_path, status, created_at, activated_at
		FROM nuclei_custom_bundles ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.NucleiCustomBundle, 0)
	for rows.Next() {
		b := &models.NucleiCustomBundle{}
		var activatedAt sql.NullTime
		if err := rows.Scan(&b.Version, &b.ManifestJSON, &b.ArchivePath, &b.Status, &b.CreatedAt, &activatedAt); err != nil {
			return nil, err
		}
		if activatedAt.Valid {
			t := activatedAt.Time
			b.ActivatedAt = &t
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (q *Queries) SetNucleiCustomBundleStatus(version, status string, activatedAt *time.Time) error {
	_, err := q.db.Exec(`UPDATE nuclei_custom_bundles SET status = ?, activated_at = ? WHERE version = ?`,
		status, activatedAt, version)
	return err
}

// GetActiveNucleiCustomBundleVersion returns the version string of the
// currently active bundle, or "" if none is active.
func (q *Queries) GetActiveNucleiCustomBundleVersion() (string, error) {
	var version string
	err := q.db.QueryRow(`SELECT version FROM nuclei_custom_bundles WHERE status = 'active' LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return version, nil
}

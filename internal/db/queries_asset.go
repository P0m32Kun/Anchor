package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Assets ---

func (q *Queries) CreateAsset(a *models.Asset) error {
	sourceToolsJSON, err := json.Marshal(a.SourceTools)
	if err != nil {
		return fmt.Errorf("marshal source_tools: %w", err)
	}
	tagsJSON, err := json.Marshal(a.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	_, err = q.db.Exec(`
		INSERT INTO assets (id, project_id, type, value, normalized_value, source_tools, first_seen, last_seen, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ProjectID, a.Type, a.Value, a.NormalizedValue, string(sourceToolsJSON), a.FirstSeen, a.LastSeen, string(tagsJSON))
	return err
}

func (q *Queries) GetAssetByNormalizedValue(projectID, normalizedValue string) (*models.Asset, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, type, value, normalized_value, source_tools, first_seen, last_seen, tags
		FROM assets WHERE project_id = ? AND normalized_value = ?`, projectID, normalizedValue)
	return scanAsset(row)
}

func (q *Queries) UpdateAssetLastSeen(id string, lastSeen time.Time, sourceTools []string) error {
	sourceToolsJSON, err := json.Marshal(sourceTools)
	if err != nil {
		return fmt.Errorf("marshal source_tools: %w", err)
	}
	_, err = q.db.Exec(`UPDATE assets SET last_seen = ?, source_tools = ? WHERE id = ?`, lastSeen, string(sourceToolsJSON), id)
	return err
}

func (q *Queries) ListAssetsByProject(projectID string) ([]*models.Asset, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, type, value, normalized_value, source_tools, first_seen, last_seen, tags
		FROM assets WHERE project_id = ? ORDER BY last_seen DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Asset, 0)
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (q *Queries) CountAssetsByProject(projectID string) (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM assets WHERE project_id = ?`, projectID)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListAssetsByProjectPaginated(projectID string, limit, offset int) ([]*models.Asset, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, type, value, normalized_value, source_tools, first_seen, last_seen, tags
		FROM assets WHERE project_id = ? ORDER BY last_seen DESC LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Asset, 0)
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func scanAsset(row interface {
	Scan(dest ...any) error
}) (*models.Asset, error) {
	a := &models.Asset{}
	var sourceToolsJSON, tagsJSON string
	err := row.Scan(&a.ID, &a.ProjectID, &a.Type, &a.Value, &a.NormalizedValue, &sourceToolsJSON, &a.FirstSeen, &a.LastSeen, &tagsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if sourceToolsJSON != "" && sourceToolsJSON != "null" {
		if err := json.Unmarshal([]byte(sourceToolsJSON), &a.SourceTools); err != nil {
			// silently ignore unmarshal errors for backward compatibility
		}
	}
	if tagsJSON != "" && tagsJSON != "null" {
		if err := json.Unmarshal([]byte(tagsJSON), &a.Tags); err != nil {
			// silently ignore unmarshal errors for backward compatibility
		}
	}
	return a, nil
}

// --- Ports ---

func (q *Queries) CreatePort(p *models.Port) error {
	_, err := q.db.Exec(`INSERT INTO ports (id, asset_id, port, protocol, state, source_tool, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.AssetID, p.Port, p.Protocol, p.State, p.SourceTool, p.CreatedAt)
	return err
}

func (q *Queries) ListPortsByAsset(assetID string) ([]*models.Port, error) {
	rows, err := q.db.Query(`SELECT id, asset_id, port, protocol, state, source_tool, created_at FROM ports WHERE asset_id = ? ORDER BY port`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPorts(rows)
}

func (q *Queries) ListPortsByProject(projectID string) ([]*models.Port, error) {
	rows, err := q.db.Query(`
		SELECT p.id, p.asset_id, p.port, p.protocol, p.state, p.source_tool, p.created_at
		FROM ports p
		JOIN assets a ON p.asset_id = a.id
		WHERE a.project_id = ?
		ORDER BY p.port`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPorts(rows)
}

func scanPorts(rows *sql.Rows) ([]*models.Port, error) {
	list := make([]*models.Port, 0)
	for rows.Next() {
		p := &models.Port{}
		if err := rows.Scan(&p.ID, &p.AssetID, &p.Port, &p.Protocol, &p.State, &p.SourceTool, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (q *Queries) PortExists(assetID string, port int) (bool, error) {
	row := q.db.QueryRow(`SELECT COUNT(1) FROM ports WHERE asset_id = ? AND port = ?`, assetID, port)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// --- Services ---

func (q *Queries) CreateService(s *models.Service) error {
	_, err := q.db.Exec(`INSERT INTO services (id, asset_id, port_id, name, product, version, banner, confidence, source_tool, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.AssetID, s.PortID, s.Name, s.Product, s.Version, s.Banner, s.Confidence, s.SourceTool, s.CreatedAt)
	return err
}

func (q *Queries) ListServicesByAsset(assetID string) ([]*models.Service, error) {
	rows, err := q.db.Query(`SELECT id, asset_id, port_id, name, product, version, banner, confidence, source_tool, created_at FROM services WHERE asset_id = ?`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Service, 0)
	for rows.Next() {
		s := &models.Service{}
		var portID sql.NullString
		if err := rows.Scan(&s.ID, &s.AssetID, &portID, &s.Name, &s.Product, &s.Version, &s.Banner, &s.Confidence, &s.SourceTool, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.PortID = nullableString(portID)
		list = append(list, s)
	}
	return list, rows.Err()
}

// --- Web Endpoints ---

func (q *Queries) CreateWebEndpoint(we *models.WebEndpoint) error {
	techJSON, err := json.Marshal(we.Technologies)
	if err != nil {
		return fmt.Errorf("marshal technologies: %w", err)
	}
	_, err = q.db.Exec(`
		INSERT INTO web_endpoints (id, project_id, asset_id, url, scheme, host, port, path, status_code, title, technologies, screenshot_artifact_id, source_tool, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		we.ID, we.ProjectID, we.AssetID, we.URL, we.Scheme, we.Host, we.Port, we.Path, we.StatusCode, we.Title, string(techJSON), we.ScreenshotArtifactID, we.SourceTool, we.CreatedAt)
	return err
}

func (q *Queries) ListWebEndpointsByAsset(assetID string) ([]*models.WebEndpoint, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, url, scheme, host, port, path, status_code, title, technologies, screenshot_artifact_id, source_tool, created_at
		FROM web_endpoints WHERE asset_id = ? ORDER BY url`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebEndpoints(rows)
}

func (q *Queries) ListWebEndpointsByProject(projectID string) ([]*models.WebEndpoint, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, url, scheme, host, port, path, status_code, title, technologies, screenshot_artifact_id, source_tool, created_at
		FROM web_endpoints WHERE project_id = ? ORDER BY url`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebEndpoints(rows)
}

func (q *Queries) CountWebEndpointsByProject(projectID string) (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM web_endpoints WHERE project_id = ?`, projectID)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListWebEndpointsByProjectPaginated(projectID string, limit, offset int) ([]*models.WebEndpoint, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, url, scheme, host, port, path, status_code, title, technologies, screenshot_artifact_id, source_tool, created_at
		FROM web_endpoints WHERE project_id = ? ORDER BY url LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebEndpoints(rows)
}

func (q *Queries) WebEndpointExists(projectID, url string) (bool, error) {
	row := q.db.QueryRow(`SELECT COUNT(1) FROM web_endpoints WHERE project_id = ? AND url = ?`, projectID, url)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func scanWebEndpoints(rows *sql.Rows) ([]*models.WebEndpoint, error) {
	list := make([]*models.WebEndpoint, 0)
	for rows.Next() {
		we := &models.WebEndpoint{}
		var port sql.NullInt64
		var statusCode sql.NullInt64
		var screenshotID sql.NullString
		var techJSON string
		err := rows.Scan(&we.ID, &we.ProjectID, &we.AssetID, &we.URL, &we.Scheme, &we.Host, &port, &we.Path, &statusCode, &we.Title, &techJSON, &screenshotID, &we.SourceTool, &we.CreatedAt)
		if err != nil {
			return nil, err
		}
		if port.Valid {
			p := int(port.Int64)
			we.Port = &p
		}
		if statusCode.Valid {
			sc := int(statusCode.Int64)
			we.StatusCode = &sc
		}
		we.ScreenshotArtifactID = nullableString(screenshotID)
		if techJSON != "" && techJSON != "null" {
			if err := json.Unmarshal([]byte(techJSON), &we.Technologies); err != nil {
				// silently ignore unmarshal errors for backward compatibility
			}
		}
		list = append(list, we)
	}
	return list, rows.Err()
}

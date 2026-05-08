package db

import (
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Findings ---

func (q *Queries) CreateFinding(f *models.Finding) error {
	_, err := q.db.Exec(`
		INSERT INTO findings (id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.ProjectID, f.AssetID, f.ServiceID, f.WebEndpointID, f.SourceTool, f.SourceRuleID, f.DedupKey, f.Title, f.Severity, f.Confidence, f.Priority, f.Status, f.Summary, f.Remediation, f.CreatedAt, f.UpdatedAt)
	return err
}

func (q *Queries) GetFindingByDedupKey(projectID, dedupKey string) (*models.Finding, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE project_id = ? AND dedup_key = ?`, projectID, dedupKey)
	return scanFinding(row)
}

func (q *Queries) GetFinding(id string) (*models.Finding, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE id = ?`, id)
	return scanFinding(row)
}

func (q *Queries) UpdateFindingStatus(id string, status models.FindingStatus, updatedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE findings SET status = ?, updated_at = ? WHERE id = ?`, status, updatedAt, id)
	return err
}

func (q *Queries) UpdateFindingEvidence(id string, severity models.FindingSeverity, confidence, priority int, summary, remediation string, updatedAt time.Time) error {
	_, err := q.db.Exec(`UPDATE findings SET severity = ?, confidence = ?, priority = ?, summary = ?, remediation = ?, updated_at = ? WHERE id = ?`,
		severity, confidence, priority, summary, remediation, updatedAt, id)
	return err
}

func (q *Queries) ListFindingsByProject(projectID string) ([]*models.Finding, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE project_id = ? ORDER BY priority DESC, created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Finding, 0)
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (q *Queries) ListFindingsByStatus(projectID string, status models.FindingStatus) ([]*models.Finding, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE project_id = ? AND status = ? ORDER BY priority DESC, created_at DESC`, projectID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Finding, 0)
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (q *Queries) CountFindingsByProject(projectID string, status models.FindingStatus) (int, error) {
	var count int
	var err error
	if status != "" {
		err = q.db.QueryRow(`SELECT COUNT(*) FROM findings WHERE project_id = ? AND status = ?`, projectID, status).Scan(&count)
	} else {
		err = q.db.QueryRow(`SELECT COUNT(*) FROM findings WHERE project_id = ?`, projectID).Scan(&count)
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListFindingsByProjectPaginated(projectID string, limit, offset int) ([]*models.Finding, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE project_id = ? ORDER BY priority DESC, created_at DESC LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Finding, 0)
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (q *Queries) ListFindingsByStatusPaginated(projectID string, status models.FindingStatus, limit, offset int) ([]*models.Finding, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE project_id = ? AND status = ? ORDER BY priority DESC, created_at DESC LIMIT ? OFFSET ?`, projectID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Finding, 0)
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func scanFinding(row interface {
	Scan(dest ...any) error
}) (*models.Finding, error) {
	f := &models.Finding{}
	var assetID, serviceID, webEndpointID sql.NullString
	err := row.Scan(&f.ID, &f.ProjectID, &assetID, &serviceID, &webEndpointID, &f.SourceTool, &f.SourceRuleID, &f.DedupKey, &f.Title, &f.Severity, &f.Confidence, &f.Priority, &f.Status, &f.Summary, &f.Remediation, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.AssetID = nullableString(assetID)
	f.ServiceID = nullableString(serviceID)
	f.WebEndpointID = nullableString(webEndpointID)
	return f, nil
}

// ListFindingsForReport returns findings with status IN ('confirmed', 'accepted_risk') for a project.
// Used by report aggregation to select only report-eligible findings.
func (q *Queries) ListFindingsForReport(projectID string) ([]*models.Finding, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, asset_id, service_id, web_endpoint_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at
		FROM findings WHERE project_id = ? AND status IN ('confirmed', 'accepted_risk') ORDER BY priority DESC, created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Finding, 0)
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

// --- Evidence ---

func (q *Queries) CreateEvidence(e *models.Evidence) error {
	_, err := q.db.Exec(`
		INSERT INTO evidence (id, finding_id, type, artifact_id, excerpt, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.FindingID, e.Type, e.ArtifactID, e.Excerpt, e.CreatedBy, e.CreatedAt)
	return err
}

func (q *Queries) ListEvidenceByFinding(findingID string) ([]*models.Evidence, error) {
	rows, err := q.db.Query(`
		SELECT id, finding_id, type, artifact_id, excerpt, created_by, created_at
		FROM evidence WHERE finding_id = ? ORDER BY created_at`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Evidence, 0)
	for rows.Next() {
		e := &models.Evidence{}
		var artifactID, createdBy sql.NullString
		if err := rows.Scan(&e.ID, &e.FindingID, &e.Type, &artifactID, &e.Excerpt, &createdBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.ArtifactID = nullableString(artifactID)
		if createdBy.Valid {
			e.CreatedBy = createdBy.String
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// --- RetestRun ---

func (q *Queries) CreateRetestRun(r *models.RetestRun) error {
	_, err := q.db.Exec(`
		INSERT INTO retest_runs (id, finding_id, task_id, result, evidence_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.FindingID, r.TaskID, r.Result, r.EvidenceID, r.CreatedAt)
	return err
}

func (q *Queries) ListRetestRunsByFinding(findingID string) ([]*models.RetestRun, error) {
	rows, err := q.db.Query(`
		SELECT id, finding_id, task_id, result, evidence_id, created_at
		FROM retest_runs WHERE finding_id = ? ORDER BY created_at DESC`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.RetestRun, 0)
	for rows.Next() {
		r := &models.RetestRun{}
		var evidenceID sql.NullString
		if err := rows.Scan(&r.ID, &r.FindingID, &r.TaskID, &r.Result, &evidenceID, &r.CreatedAt); err != nil {
			return nil, err
		}
		if evidenceID.Valid {
			r.EvidenceID = &evidenceID.String
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

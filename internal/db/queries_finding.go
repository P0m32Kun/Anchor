package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// findingColumns is the canonical column list for the findings table.
// Keep this in sync with the schema and scanFinding.
const findingColumns = "id, project_id, asset_id, service_id, web_endpoint_id, run_id, source_tool, source_rule_id, dedup_key, title, severity, confidence, priority, status, summary, remediation, created_at, updated_at"

// findingInsertArgs returns the VALUES arguments for a single finding.
func findingInsertArgs(f *models.Finding) []any {
	return []any{
		f.ID, f.ProjectID, f.AssetID, f.ServiceID, f.WebEndpointID, f.RunID,
		f.SourceTool, f.SourceRuleID, f.DedupKey, f.Title, f.Severity,
		f.Confidence, f.Priority, f.Status, f.Summary, f.Remediation,
		f.CreatedAt, f.UpdatedAt,
	}
}

// --- Findings ---

func (q *Queries) CreateFinding(f *models.Finding) error {
	_, err := q.db.Exec(
		"INSERT INTO findings ("+findingColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		findingInsertArgs(f)...)
	return err
}

func isRetryableDBError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "busy")
}

// BatchInsertFindings inserts multiple findings in a single transaction.
// Uses individual INSERT statements wrapped in a transaction to avoid
// SQLite's SQLITE_MAX_VARIABLE_NUMBER limit (default 999).
// Falls back to individual inserts if the underlying DBTX is not *sql.DB.
// Retries up to maxBatchRetries on transient lock errors with exponential backoff.
func (q *Queries) BatchInsertFindings(findings []*models.Finding) error {
	if len(findings) == 0 {
		return nil
	}

	const maxBatchRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxBatchRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 50 * time.Millisecond)
		}

		lastErr = q.batchInsertOnce(findings)
		if lastErr == nil {
			return nil
		}
		if !isRetryableDBError(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func (q *Queries) batchInsertOnce(findings []*models.Finding) error {
	db, ok := q.db.(*sql.DB)
	if !ok {
		for _, f := range findings {
			if err := q.CreateFinding(f); err != nil {
				return err
			}
		}
		return nil
	}
	return WithTx(db, func(tq *Queries) error {
		for _, f := range findings {
			if err := tq.CreateFinding(f); err != nil {
				return err
			}
		}
		return nil
	})
}

func (q *Queries) GetFindingByDedupKey(projectID, dedupKey string) (*models.Finding, error) {
	row := q.db.QueryRow(
		"SELECT "+findingColumns+" FROM findings WHERE project_id = ? AND dedup_key = ?",
		projectID, dedupKey)
	return scanFinding(row)
}

func (q *Queries) GetFinding(id string) (*models.Finding, error) {
	row := q.db.QueryRow(
		"SELECT "+findingColumns+" FROM findings WHERE id = ?", id)
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
	rows, err := q.db.Query(
		"SELECT "+findingColumns+" FROM findings WHERE project_id = ? ORDER BY priority DESC, created_at DESC", projectID)
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
	rows, err := q.db.Query(
		"SELECT "+findingColumns+" FROM findings WHERE project_id = ? AND status = ? ORDER BY priority DESC, created_at DESC", projectID, status)
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
	rows, err := q.db.Query(
		"SELECT "+findingColumns+" FROM findings WHERE project_id = ? ORDER BY priority DESC, created_at DESC LIMIT ? OFFSET ?", projectID, limit, offset)
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
	rows, err := q.db.Query(
		"SELECT "+findingColumns+" FROM findings WHERE project_id = ? AND status = ? ORDER BY priority DESC, created_at DESC LIMIT ? OFFSET ?", projectID, status, limit, offset)
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
	var assetID, serviceID, webEndpointID, runID sql.NullString
	err := row.Scan(&f.ID, &f.ProjectID, &assetID, &serviceID, &webEndpointID, &runID, &f.SourceTool, &f.SourceRuleID, &f.DedupKey, &f.Title, &f.Severity, &f.Confidence, &f.Priority, &f.Status, &f.Summary, &f.Remediation, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.AssetID = nullableString(assetID)
	f.ServiceID = nullableString(serviceID)
	f.WebEndpointID = nullableString(webEndpointID)
	f.RunID = nullableString(runID)
	return f, nil
}

// ListFindingsForReport returns all findings for a project, ordered for report rendering.
// Status filtering is deferred to the report templates so that pending_review and
// false_positive findings are visible to the auditor before they make a decision.
func (q *Queries) ListFindingsForReport(projectID string) ([]*models.Finding, error) {
	rows, err := q.db.Query(
		"SELECT "+findingColumns+" FROM findings WHERE project_id = ? ORDER BY priority DESC, created_at DESC", projectID)
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

// ListFindingsByRun returns findings scoped to a specific pipeline run.
func (q *Queries) ListFindingsByRun(projectID, runID string) ([]*models.Finding, error) {
	rows, err := q.db.Query(
		"SELECT "+findingColumns+" FROM findings WHERE project_id = ? AND run_id = ? ORDER BY priority DESC, created_at DESC", projectID, runID)
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

// ListEvidenceByFindingIDs returns evidence for multiple findings in one query (avoids N+1).
func (q *Queries) ListEvidenceByFindingIDs(findingIDs []string) (map[string][]*models.Evidence, error) {
	if len(findingIDs) == 0 {
		return nil, nil
	}
	// Build placeholders: ?,?,?
	placeholders := ""
	args := make([]interface{}, 0, len(findingIDs))
	for i, id := range findingIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}
	rows, err := q.db.Query(`
		SELECT id, finding_id, type, artifact_id, excerpt, created_by, created_at
		FROM evidence WHERE finding_id IN (`+placeholders+`) ORDER BY created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]*models.Evidence)
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
		result[e.FindingID] = append(result[e.FindingID], e)
	}
	return result, rows.Err()
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

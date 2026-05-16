package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- Targets ---

func (q *Queries) CreateTarget(t *models.Target) error {
	_, err := q.db.Exec(`INSERT INTO targets (id, project_id, type, value, source, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.Type, t.Value, t.Source, t.Status, t.CreatedAt)
	return err
}

// BulkCreateTargets inserts multiple targets within the current transaction.
// Callers should use WithTx to wrap the call.
func (q *Queries) BulkCreateTargets(targets []*models.Target) error {
	for _, t := range targets {
		_, err := q.db.Exec(`INSERT INTO targets (id, project_id, type, value, source, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			t.ID, t.ProjectID, t.Type, t.Value, t.Source, t.Status, t.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// TargetExistsByValue checks if a target with the given value already exists for the project.
func (q *Queries) TargetExistsByValue(projectID, value string) (bool, error) {
	row := q.db.QueryRow(`SELECT COUNT(1) FROM targets WHERE project_id = ? AND value = ?`, projectID, value)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (q *Queries) GetTarget(id string) (*models.Target, error) {
	row := q.db.QueryRow(`SELECT id, project_id, type, value, source, status, created_at FROM targets WHERE id = ?`, id)
	t := &models.Target{}
	err := row.Scan(&t.ID, &t.ProjectID, &t.Type, &t.Value, &t.Source, &t.Status, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (q *Queries) ListTargetsByProject(projectID string) ([]*models.Target, error) {
	rows, err := q.db.Query(`SELECT id, project_id, type, value, source, status, created_at FROM targets WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Target, 0)
	for rows.Next() {
		t := &models.Target{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Type, &t.Value, &t.Source, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (q *Queries) CountTargetsByProject(projectID string) (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM targets WHERE project_id = ?`, projectID)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListTargetsByProjectPaginated(projectID string, limit, offset int) ([]*models.Target, error) {
	rows, err := q.db.Query(`SELECT id, project_id, type, value, source, status, created_at FROM targets WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.Target, 0)
	for rows.Next() {
		t := &models.Target{}
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Type, &t.Value, &t.Source, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// DeleteTarget deletes a target by ID.
func (q *Queries) DeleteTarget(id string) error {
	_, err := q.db.Exec(`DELETE FROM targets WHERE id = ?`, id)
	return err
}

// --- IP Discovery Results ---

func (q *Queries) CreateIPDiscoveryResult(r *models.IPDiscoveryResult) error {
	_, err := q.db.Exec(`
		INSERT INTO ip_discovery_results (id, project_id, target_id, ip, hostname, source, alive, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ProjectID, r.TargetID, r.IP, r.Hostname, r.Source, r.Alive, r.CreatedAt)
	return err
}

func (q *Queries) ListIPDiscoveryResultsByProject(projectID string) ([]*models.IPDiscoveryResult, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, target_id, ip, hostname, source, alive, created_at
		FROM ip_discovery_results
		WHERE project_id = ?
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.IPDiscoveryResult, 0)
	for rows.Next() {
		r := &models.IPDiscoveryResult{}
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.TargetID, &r.IP, &r.Hostname, &r.Source, &r.Alive, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) ListIPDiscoveryResultsByTarget(targetID string) ([]*models.IPDiscoveryResult, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, target_id, ip, hostname, source, alive, created_at
		FROM ip_discovery_results
		WHERE target_id = ?
		ORDER BY created_at DESC`, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.IPDiscoveryResult, 0)
	for rows.Next() {
		r := &models.IPDiscoveryResult{}
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.TargetID, &r.IP, &r.Hostname, &r.Source, &r.Alive, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// --- Scope Rules ---

func (q *Queries) CreateScopeRule(r *models.ScopeRule) error {
	_, err := q.db.Exec(`INSERT INTO scope_rules (id, project_id, action, type, value, reason, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ProjectID, r.Action, r.Type, r.Value, r.Reason, r.CreatedAt, r.UpdatedAt)
	return err
}

func (q *Queries) ListScopeRulesByProject(projectID string) ([]*models.ScopeRule, error) {
	rows, err := q.db.Query(`SELECT id, project_id, action, type, value, reason, created_at, updated_at FROM scope_rules WHERE project_id = ? ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ScopeRule, 0)
	for rows.Next() {
		r := &models.ScopeRule{}
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Action, &r.Type, &r.Value, &r.Reason, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (q *Queries) CountScopeRulesByProject(projectID string) (int, error) {
	var count int
	row := q.db.QueryRow(`SELECT COUNT(*) FROM scope_rules WHERE project_id = ?`, projectID)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (q *Queries) ListScopeRulesByProjectPaginated(projectID string, limit, offset int) ([]*models.ScopeRule, error) {
	rows, err := q.db.Query(`SELECT id, project_id, action, type, value, reason, created_at, updated_at FROM scope_rules WHERE project_id = ? ORDER BY created_at LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ScopeRule, 0)
	for rows.Next() {
		r := &models.ScopeRule{}
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Action, &r.Type, &r.Value, &r.Reason, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// DeleteScopeRule deletes a scope rule by ID.
func (q *Queries) DeleteScopeRule(id string) error {
	_, err := q.db.Exec(`DELETE FROM scope_rules WHERE id = ?`, id)
	return err
}

func (q *Queries) GetMaxScopeRuleUpdatedAt(projectID string) (time.Time, error) {
	var s sql.NullString
	err := q.db.QueryRow(`SELECT MAX(updated_at) FROM scope_rules WHERE project_id = ?`, projectID).Scan(&s)
	if err != nil || !s.Valid || s.String == "" {
		return time.Time{}, err
	}
	// SQLite stores datetime in various formats; try common ones.
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999+00:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s.String); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s.String)
}

// --- Scope Decisions ---

func (q *Queries) CreateScopeDecision(d *models.ScopeDecision) error {
	_, err := q.db.Exec(`INSERT INTO scope_decisions (id, project_id, target_value, task_id, decision, matched_rule_id, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.ProjectID, d.TargetValue, d.TaskID, d.Decision, d.MatchedRuleID, d.Reason, d.CreatedAt)
	return err
}

func (q *Queries) GetLatestScopeDecision(projectID, targetValue string) (*models.ScopeDecision, error) {
	row := q.db.QueryRow(`SELECT id, project_id, target_value, task_id, decision, matched_rule_id, reason, created_at FROM scope_decisions WHERE project_id = ? AND target_value = ? ORDER BY created_at DESC LIMIT 1`, projectID, targetValue)
	d := &models.ScopeDecision{}
	err := row.Scan(&d.ID, &d.ProjectID, &d.TargetValue, &d.TaskID, &d.Decision, &d.MatchedRuleID, &d.Reason, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (q *Queries) ListScopeDecisionsByPlan(projectID string) ([]*models.ScopeDecision, error) {
	rows, err := q.db.Query(`SELECT id, project_id, target_value, task_id, decision, matched_rule_id, reason, created_at FROM scope_decisions WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*models.ScopeDecision, 0)
	for rows.Next() {
		d := &models.ScopeDecision{}
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.TargetValue, &d.TaskID, &d.Decision, &d.MatchedRuleID, &d.Reason, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

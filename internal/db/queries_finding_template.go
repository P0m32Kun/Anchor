package db

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// CreateFindingTemplate inserts a new template.
func (q *Queries) CreateFindingTemplate(t *models.FindingTemplate) error {
	matchKeysJSON, err := json.Marshal(t.MatchKeys)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(`
		INSERT INTO finding_templates (id, source_tool, match_key, match_keys, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.SourceTool, t.MatchKey, string(matchKeysJSON), t.Title, t.Severity, t.Summary, t.Remediation, boolToInt(t.Enabled), boolToInt(t.IsBuiltin), boolToInt(t.UserModified), t.BuiltinPayload, t.CreatedAt, t.UpdatedAt)
	return err
}

// GetFindingTemplate fetches a template by id.
func (q *Queries) GetFindingTemplate(id string) (*models.FindingTemplate, error) {
	row := q.db.QueryRow(`
		SELECT id, source_tool, match_key, match_keys, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at
		FROM finding_templates WHERE id = ?`, id)
	return scanFindingTemplate(row)
}

// ListFindingTemplates returns all templates, optionally filtered by source_tool.
func (q *Queries) ListFindingTemplates(sourceTool string) ([]*models.FindingTemplate, error) {
	var rows *sql.Rows
	var err error
	if sourceTool != "" {
		rows, err = q.db.Query(`
			SELECT id, source_tool, match_key, match_keys, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at
			FROM finding_templates WHERE source_tool = ? ORDER BY created_at DESC`, sourceTool)
	} else {
		rows, err = q.db.Query(`
			SELECT id, source_tool, match_key, match_keys, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at
			FROM finding_templates ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.FindingTemplate, 0)
	for rows.Next() {
		t, err := scanFindingTemplate(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// ListBuiltinFindingTemplates returns only templates seeded from the repo.
func (q *Queries) ListBuiltinFindingTemplates() ([]*models.FindingTemplate, error) {
	rows, err := q.db.Query(`
		SELECT id, source_tool, match_key, match_keys, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at
		FROM finding_templates WHERE is_builtin = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*models.FindingTemplate, 0)
	for rows.Next() {
		t, err := scanFindingTemplate(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// UpdateFindingTemplate updates all mutable fields of a template.
func (q *Queries) UpdateFindingTemplate(t *models.FindingTemplate) error {
	matchKeysJSON, err := json.Marshal(t.MatchKeys)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(`
		UPDATE finding_templates
		SET source_tool = ?, match_key = ?, match_keys = ?, title = ?, severity = ?, summary = ?, remediation = ?, enabled = ?,
		    is_builtin = ?, user_modified = ?, builtin_payload = ?, updated_at = ?
		WHERE id = ?`,
		t.SourceTool, t.MatchKey, string(matchKeysJSON), t.Title, t.Severity, t.Summary, t.Remediation, boolToInt(t.Enabled),
		boolToInt(t.IsBuiltin), boolToInt(t.UserModified), t.BuiltinPayload, t.UpdatedAt, t.ID)
	return err
}

// DeleteFindingTemplate removes a template by id.
func (q *Queries) DeleteFindingTemplate(id string) error {
	_, err := q.db.Exec(`DELETE FROM finding_templates WHERE id = ?`, id)
	return err
}

// GetFindingTemplateForFinding looks up an enabled template for a finding,
// trying the match keys in priority order: ruleID → matchedTemplate → title.
// Returns (nil, nil) when no matching enabled template exists.
func (q *Queries) GetFindingTemplateForFinding(sourceTool, ruleID, matchedTemplate, title string) (*models.FindingTemplate, error) {
	tool := strings.TrimSpace(sourceTool)
	if tool == "" {
		return nil, nil
	}
	for _, key := range []string{ruleID, matchedTemplate, title} {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		row := q.db.QueryRow(`
			SELECT id, source_tool, match_key, match_keys, title, severity, summary, remediation, enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at
			FROM finding_templates
			WHERE source_tool = ? AND match_key = ? AND enabled = 1
			LIMIT 1`, tool, key)
		t, err := scanFindingTemplate(row)
		if err != nil {
			return nil, err
		}
		if t != nil {
			return t, nil
		}
	}
	return nil, nil
}

// scanFindingTemplate accepts either *sql.Row or *sql.Rows and reads one row.
func scanFindingTemplate(scanner rowScanner) (*models.FindingTemplate, error) {
	t := &models.FindingTemplate{}
	var enabled, isBuiltin, userModified int
	var matchKeysJSON string
	err := scanner.Scan(
		&t.ID, &t.SourceTool, &t.MatchKey, &matchKeysJSON,
		&t.Title, &t.Severity, &t.Summary, &t.Remediation,
		&enabled, &isBuiltin, &userModified, &t.BuiltinPayload,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Enabled = enabled != 0
	t.IsBuiltin = isBuiltin != 0
	t.UserModified = userModified != 0

	// 反序列化 match_keys JSON；为空或失败时 fallback 到老 match_key
	if matchKeysJSON != "" {
		if err := json.Unmarshal([]byte(matchKeysJSON), &t.MatchKeys); err != nil {
			t.MatchKeys = nil
		}
	}
	if len(t.MatchKeys) == 0 && t.MatchKey != "" {
		t.MatchKeys = []string{t.MatchKey}
	}
	return t, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

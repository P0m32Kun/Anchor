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

// listEnabledTemplatesByTool 读取某工具的所有启用词条。
func (q *Queries) listEnabledTemplatesByTool(sourceTool string) ([]*models.FindingTemplate, error) {
	rows, err := q.db.Query(`
		SELECT id, source_tool, match_key, match_keys, title, severity, summary, remediation,
		       enabled, is_builtin, user_modified, builtin_payload, created_at, updated_at
		FROM finding_templates
		WHERE source_tool = ? AND enabled = 1`, sourceTool)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.FindingTemplate
	for rows.Next() {
		t, err := scanFindingTemplate(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// containsExact 判断 key 是否在 keys 列表中(精确匹配)。
func containsExact(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

// GetFindingTemplateForFinding 查找匹配 finding 的启用词条。
// 匹配优先级: source_rule_id → title(都为精确字符串匹配)。
// 返回 (nil, nil) 表示未找到。
func (q *Queries) GetFindingTemplateForFinding(sourceTool, sourceRuleID, title string) (*models.FindingTemplate, error) {
	tool := strings.TrimSpace(sourceTool)
	if tool == "" {
		return nil, nil
	}

	templates, err := q.listEnabledTemplatesByTool(tool)
	if err != nil {
		return nil, err
	}

	// Tier 1: 按 source_ruleID 匹配
	if k := strings.TrimSpace(sourceRuleID); k != "" {
		for _, t := range templates {
			if containsExact(t.MatchKeys, k) {
				return t, nil
			}
		}
	}

	// Tier 2: 按 title 匹配(兜底)
	if k := strings.TrimSpace(title); k != "" {
		for _, t := range templates {
			if containsExact(t.MatchKeys, k) {
				return t, nil
			}
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

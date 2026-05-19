package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// SeedFindingTemplate 是 docs/templates/vuln-templates.json 中单条记录的形态。
// MatchKey 保留以兼容老文件；MatchKeys 优先。
type SeedFindingTemplate struct {
	SourceTool  string   `json:"source_tool"`
	MatchKey    string   `json:"match_key,omitempty"`    // 兼容老文件
	MatchKeys   []string `json:"match_keys,omitempty"`   // 新形态
	Title       string   `json:"title,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"` // 默认 true
}

// SyncResult summarises a sync run so callers (and tests) can verify behaviour.
type SyncResult struct {
	Inserted  int // new builtin rows created
	Updated   int // existing builtin rows whose content was overwritten from JSON (user_modified=0)
	Preserved int // user_modified rows whose content was kept; builtin_payload refreshed
	Deleted   int // builtin rows removed because the JSON no longer contains them (user_modified=0)
	Skipped   int // builtin rows kept despite removal from JSON (user_modified=1)
}

// SyncFindingTemplatesFromFile loads the seed JSON at path and synchronises the
// finding_templates table. Missing file → no-op (returns nil). Behaviour:
//
//   - JSON entry not in DB                       → INSERT  (is_builtin=1, user_modified=0, builtin_payload=JSON)
//   - DB row exists, user_modified=0             → UPDATE fields + builtin_payload
//   - DB row exists, user_modified=1             → UPDATE only builtin_payload (UI will surface "upstream new version")
//   - DB row missing from JSON, user_modified=0  → DELETE
//   - DB row missing from JSON, user_modified=1  → KEEP (user owns it now)
//
// Non-builtin rows (is_builtin=0) are never touched by this routine.
func (q *Queries) SyncFindingTemplatesFromFile(path string) (*SyncResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return &SyncResult{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &SyncResult{}, nil
		}
		return nil, fmt.Errorf("read seed file %s: %w", path, err)
	}

	var seeds []SeedFindingTemplate
	if len(strings.TrimSpace(string(data))) == 0 {
		seeds = nil
	} else if err := json.Unmarshal(data, &seeds); err != nil {
		return nil, fmt.Errorf("parse seed JSON %s: %w", path, err)
	}

	return q.applyFindingTemplateSeeds(seeds)
}

// applyFindingTemplateSeeds runs the diff against the current DB state.
// Exposed for tests that prefer feeding entries directly without a file.
func (q *Queries) applyFindingTemplateSeeds(seeds []SeedFindingTemplate) (*SyncResult, error) {
	res := &SyncResult{}

	// Index existing builtin rows by (source_tool, match_key).
	existing, err := q.ListBuiltinFindingTemplates()
	if err != nil {
		return nil, fmt.Errorf("list builtin templates: %w", err)
	}
	type key struct{ tool, match string }
	byKey := make(map[key]*models.FindingTemplate, len(existing))
	for _, e := range existing {
		byKey[key{e.SourceTool, e.MatchKey}] = e
	}

	seenKeys := make(map[key]struct{}, len(seeds))

	for _, s := range seeds {
		tool := strings.TrimSpace(s.SourceTool)
		// 优先用 MatchKeys，回退到 MatchKey（兼容老文件）
		keys := make([]string, 0, len(s.MatchKeys))
		for _, k := range s.MatchKeys {
			if trimmed := strings.TrimSpace(k); trimmed != "" {
				keys = append(keys, trimmed)
			}
		}
		if len(keys) == 0 && s.MatchKey != "" {
			keys = []string{strings.TrimSpace(s.MatchKey)}
		}
		if tool == "" || len(keys) == 0 {
			continue // skip malformed entries quietly
		}
		match := keys[0]
		k := key{tool, match}
		seenKeys[k] = struct{}{}

		payload, err := json.Marshal(s)
		if err != nil {
			return nil, fmt.Errorf("marshal seed payload (%s/%s): %w", tool, match, err)
		}
		enabled := true
		if s.Enabled != nil {
			enabled = *s.Enabled
		}
		now := time.Now().UTC()

		row, found := byKey[k]
		if !found {
			t := &models.FindingTemplate{
				ID:             util.GenerateID(),
				SourceTool:     tool,
				MatchKey:       match,
				MatchKeys:      keys,
				Title:          strings.TrimSpace(s.Title),
				Severity:       strings.TrimSpace(s.Severity),
				Summary:        s.Summary,
				Remediation:    s.Remediation,
				Enabled:        enabled,
				IsBuiltin:      true,
				UserModified:   false,
				BuiltinPayload: string(payload),
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := q.CreateFindingTemplate(t); err != nil {
				return nil, fmt.Errorf("insert seed (%s/%s): %w", tool, match, err)
			}
			res.Inserted++
			continue
		}

		// Always refresh the builtin_payload so UI can show "upstream updated".
		row.BuiltinPayload = string(payload)

		if row.UserModified {
			// Preserve user edits; only payload pointer moves.
			row.UpdatedAt = now
			if err := q.UpdateFindingTemplate(row); err != nil {
				return nil, fmt.Errorf("refresh payload (%s/%s): %w", tool, match, err)
			}
			res.Preserved++
			continue
		}

		// Adopt upstream content.
		row.Title = strings.TrimSpace(s.Title)
		row.Severity = strings.TrimSpace(s.Severity)
		row.Summary = s.Summary
		row.Remediation = s.Remediation
		row.MatchKeys = keys
		row.Enabled = enabled
		row.IsBuiltin = true
		row.UpdatedAt = now
		if err := q.UpdateFindingTemplate(row); err != nil {
			return nil, fmt.Errorf("update seed (%s/%s): %w", tool, match, err)
		}
		res.Updated++
	}

	// Anything still in `byKey` that wasn't in seeds is candidate for removal.
	for k, row := range byKey {
		if _, kept := seenKeys[k]; kept {
			continue
		}
		if row.UserModified {
			res.Skipped++
			continue
		}
		if err := q.DeleteFindingTemplate(row.ID); err != nil {
			return nil, fmt.Errorf("delete obsolete (%s/%s): %w", k.tool, k.match, err)
		}
		res.Deleted++
	}

	return res, nil
}

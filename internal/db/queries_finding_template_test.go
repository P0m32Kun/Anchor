package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func makeTpl(id, tool, key string, enabled bool) *models.FindingTemplate {
	now := time.Now().UTC().Truncate(time.Second)
	return &models.FindingTemplate{
		ID:          id,
		SourceTool:  tool,
		MatchKey:    key,
		MatchKeys:   []string{key},
		Title:       "T-" + id,
		Severity:    "high",
		Summary:     "summary " + id,
		Remediation: "fix " + id,
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestFindingTemplateCRUD_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))

	tpl := makeTpl("ft-1", "nuclei", "exposed-git", true)
	if err := q.CreateFindingTemplate(tpl); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := q.GetFindingTemplate("ft-1")
	if err != nil || got == nil {
		t.Fatalf("get: %v / %v", got, err)
	}
	if got.SourceTool != "nuclei" || got.MatchKey != "exposed-git" || !got.Enabled {
		t.Fatalf("get returned unexpected: %+v", got)
	}

	tpl.Title = "Updated Title"
	tpl.Enabled = false
	tpl.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := q.UpdateFindingTemplate(tpl); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = q.GetFindingTemplate("ft-1")
	if got.Title != "Updated Title" || got.Enabled {
		t.Fatalf("update did not persist: %+v", got)
	}

	list, err := q.ListFindingTemplates("")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: len=%d err=%v", len(list), err)
	}

	if err := q.DeleteFindingTemplate("ft-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ = q.GetFindingTemplate("ft-1")
	if got != nil {
		t.Fatalf("expected nil after delete, got %+v", got)
	}
}

func TestFindingTemplate_DuplicateUniqueIndex(t *testing.T) {
	q := New(openTestDB(t))
	if err := q.CreateFindingTemplate(makeTpl("a", "nuclei", "exposed-git", true)); err != nil {
		t.Fatal(err)
	}
	err := q.CreateFindingTemplate(makeTpl("b", "nuclei", "exposed-git", true))
	if err == nil {
		t.Fatal("expected uniqueness violation on (source_tool, match_key)")
	}
}

func TestGetFindingTemplateForFinding_PriorityFallback(t *testing.T) {
	q := New(openTestDB(t))

	// Seed two enabled templates for the same tool, different match keys.
	if err := q.CreateFindingTemplate(makeTpl("a", "nuclei", "rule-id-x", true)); err != nil {
		t.Fatal(err)
	}
	if err := q.CreateFindingTemplate(makeTpl("c", "nuclei", "Title X", true)); err != nil {
		t.Fatal(err)
	}

	// Tier 1: ruleID wins over title.
	tpl, err := q.GetFindingTemplateForFinding("nuclei", "rule-id-x", "Title X")
	if err != nil || tpl == nil || tpl.ID != "a" {
		t.Fatalf("ruleID priority: got %+v err %v", tpl, err)
	}

	// Tier 2: Falls back to title when ruleID missing.
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "", "Title X")
	if err != nil || tpl == nil || tpl.ID != "c" {
		t.Fatalf("title fallback: got %+v err %v", tpl, err)
	}

	// No match returns (nil, nil).
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "nope", "Nope")
	if err != nil || tpl != nil {
		t.Fatalf("no match: got %+v err %v", tpl, err)
	}

	// Disabled templates are skipped.
	if err := q.CreateFindingTemplate(makeTpl("d", "hydra", "ssh-weakpass", false)); err != nil {
		t.Fatal(err)
	}
	tpl, err = q.GetFindingTemplateForFinding("hydra", "ssh-weakpass", "")
	if err != nil || tpl != nil {
		t.Fatalf("disabled should be skipped: got %+v err %v", tpl, err)
	}

	// Different source_tool does not bleed across.
	tpl, err = q.GetFindingTemplateForFinding("sqlmap", "rule-id-x", "")
	if err != nil || tpl != nil {
		t.Fatalf("source_tool isolation: got %+v err %v", tpl, err)
	}

	// Empty source_tool returns (nil, nil) without querying.
	tpl, err = q.GetFindingTemplateForFinding("", "rule-id-x", "Title X")
	if err != nil || tpl != nil {
		t.Fatalf("empty tool short-circuit: got %+v err %v", tpl, err)
	}

	// Multi-match keys: a template with multiple MatchKeys should match ANY of them.
	multiKeyTpl := &models.FindingTemplate{
		ID:          "multi",
		SourceTool:  "nuclei",
		MatchKey:    "first-key",
		MatchKeys:   []string{"matched-template-x", "title-key-z", "third-key"},
		Title:       "T-multi",
		Severity:    "high",
		Summary:     "summary multi",
		Remediation: "fix multi",
		Enabled:     true,
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
		UpdatedAt:   time.Now().UTC().Truncate(time.Second),
	}
	if err := q.CreateFindingTemplate(multiKeyTpl); err != nil {
		t.Fatal(err)
	}

	// Hit the first key via sourceRuleID (Tier 1).
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "matched-template-x", "")
	if err != nil || tpl == nil || tpl.ID != "multi" {
		t.Fatalf("multi-key tier1 first key: got %+v err %v", tpl, err)
	}

	// Hit the second key via title (Tier 2).
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "", "title-key-z")
	if err != nil || tpl == nil || tpl.ID != "multi" {
		t.Fatalf("multi-key tier2 second key: got %+v err %v", tpl, err)
	}

	// Hit the third key via sourceRuleID.
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "third-key", "")
	if err != nil || tpl == nil || tpl.ID != "multi" {
		t.Fatalf("multi-key tier1 third key: got %+v err %v", tpl, err)
	}
}

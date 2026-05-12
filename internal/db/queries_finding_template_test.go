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

	// Seed three enabled templates for the same tool, different match keys.
	if err := q.CreateFindingTemplate(makeTpl("a", "nuclei", "rule-id-x", true)); err != nil {
		t.Fatal(err)
	}
	if err := q.CreateFindingTemplate(makeTpl("b", "nuclei", "matched-template-x", true)); err != nil {
		t.Fatal(err)
	}
	if err := q.CreateFindingTemplate(makeTpl("c", "nuclei", "Title X", true)); err != nil {
		t.Fatal(err)
	}

	// ruleID wins over matchedTemplate and title.
	tpl, err := q.GetFindingTemplateForFinding("nuclei", "rule-id-x", "matched-template-x", "Title X")
	if err != nil || tpl == nil || tpl.ID != "a" {
		t.Fatalf("ruleID priority: got %+v err %v", tpl, err)
	}

	// Falls back to matchedTemplate when ruleID missing.
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "", "matched-template-x", "Title X")
	if err != nil || tpl == nil || tpl.ID != "b" {
		t.Fatalf("matchedTemplate priority: got %+v err %v", tpl, err)
	}

	// Falls back to title when both missing.
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "", "", "Title X")
	if err != nil || tpl == nil || tpl.ID != "c" {
		t.Fatalf("title fallback: got %+v err %v", tpl, err)
	}

	// No match returns (nil, nil).
	tpl, err = q.GetFindingTemplateForFinding("nuclei", "nope", "nope", "Nope")
	if err != nil || tpl != nil {
		t.Fatalf("no match: got %+v err %v", tpl, err)
	}

	// Disabled templates are skipped.
	if err := q.CreateFindingTemplate(makeTpl("d", "hydra", "ssh-weakpass", false)); err != nil {
		t.Fatal(err)
	}
	tpl, err = q.GetFindingTemplateForFinding("hydra", "ssh-weakpass", "", "")
	if err != nil || tpl != nil {
		t.Fatalf("disabled should be skipped: got %+v err %v", tpl, err)
	}

	// Different source_tool does not bleed across.
	tpl, err = q.GetFindingTemplateForFinding("sqlmap", "rule-id-x", "", "")
	if err != nil || tpl != nil {
		t.Fatalf("source_tool isolation: got %+v err %v", tpl, err)
	}

	// Empty source_tool returns (nil, nil) without querying.
	tpl, err = q.GetFindingTemplateForFinding("", "rule-id-x", "matched-template-x", "Title X")
	if err != nil || tpl != nil {
		t.Fatalf("empty tool short-circuit: got %+v err %v", tpl, err)
	}
}

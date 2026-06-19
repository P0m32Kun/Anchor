package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func intPtr(v int) *int { return &v }

func TestToolHealth_UpsertAndList(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	proxyReachable := true
	templatePath := "/opt/nuclei-templates"
	h := &models.ToolHealth{
		ID: util.GenerateID(), Tool: "nuclei", BinaryPath: "/usr/bin/nuclei",
		Version: "3.0.0", TemplatePath: &templatePath,
		WorkdirWritable: true, NetworkAvailable: true, DNSAvailable: true,
		ProxyReachable: &proxyReachable, LastCheckAt: now,
	}
	if err := q.UpsertToolHealth(h); err != nil {
		t.Fatalf("UpsertToolHealth: %v", err)
	}

	list, err := q.ListToolHealth()
	if err != nil {
		t.Fatalf("ListToolHealth: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	if list[0].Tool != "nuclei" {
		t.Errorf("tool = %q, want nuclei", list[0].Tool)
	}
	if list[0].ProxyReachable == nil || !*list[0].ProxyReachable {
		t.Error("expected proxy_reachable=true")
	}
	if list[0].TemplatePath == nil || *list[0].TemplatePath != templatePath {
		t.Errorf("template_path = %v, want %q", list[0].TemplatePath, templatePath)
	}

	// Upsert again (update)
	h.Version = "3.1.0"
	if err := q.UpsertToolHealth(h); err != nil {
		t.Fatalf("UpsertToolHealth update: %v", err)
	}
	list2, err := q.ListToolHealth()
	if err != nil {
		t.Fatalf("ListToolHealth after update: %v", err)
	}
	if len(list2) != 1 {
		t.Fatalf("list len = %d, want 1", len(list2))
	}
	if list2[0].Version != "3.1.0" {
		t.Errorf("version = %q, want 3.1.0", list2[0].Version)
	}
}

func TestToolTemplate_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	// Insert via raw SQL since there's no CreateToolTemplate
	_, err := q.db.Exec(`
		INSERT INTO tool_templates (id, name, description, profile_type, tools_json, default_max_concurrency, screenshot_enabled, directory_bruteforce_enabled, nuclei_severity_filter, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"tmpl-1", "standard", "Standard scan profile", "external",
		`[{"tool":"nuclei","enabled":true}]`, 5, true, false, "medium", now, now,
	)
	if err != nil {
		t.Fatalf("insert tool_template: %v", err)
	}

	got, err := q.GetToolTemplate("tmpl-1")
	if err != nil {
		t.Fatalf("GetToolTemplate: %v", err)
	}
	if got == nil {
		t.Fatal("template not found")
	}
	if got.Name != "standard" {
		t.Errorf("name = %q, want standard", got.Name)
	}
	if got.DefaultMaxConcurrency != 5 {
		t.Errorf("max_concurrency = %d, want 5", got.DefaultMaxConcurrency)
	}

	// List
	list, err := q.ListToolTemplates()
	if err != nil {
		t.Fatalf("ListToolTemplates: %v", err)
	}
	if len(list) < 5 {
		t.Errorf("list len = %d, want at least 5 (4 builtin + 1 inserted)", len(list))
	}

	// Get nonexistent
	got2, err := q.GetToolTemplate("nonexistent")
	if err != nil {
		t.Fatalf("GetToolTemplate nonexistent: %v", err)
	}
	if got2 != nil {
		t.Error("expected nil for nonexistent template")
	}
}

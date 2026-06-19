package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestProject_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	p := &models.Project{
		ID: util.GenerateID(), Name: "test-project", Organization: "TestOrg",
		Purpose: "testing", RateLimit: 50, PortRange: strPtr("top100"),
		DefaultProfile: string(models.ProfileStandard),
		PipelineConfig: strPtr(`{"enable_nuclei":true}`),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateProject(p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Get
	got, err := q.GetProject(p.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got == nil {
		t.Fatal("project not found")
	}
	if got.Name != "test-project" {
		t.Errorf("name = %q, want test-project", got.Name)
	}
	if got.RateLimit != 50 {
		t.Errorf("rate_limit = %d, want 50", got.RateLimit)
	}

	// Get nonexistent
	got2, err := q.GetProject("nonexistent")
	if err != nil {
		t.Fatalf("GetProject nonexistent: %v", err)
	}
	if got2 != nil {
		t.Error("expected nil for nonexistent project")
	}
}

func TestListProjects(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		p := &models.Project{
			ID: util.GenerateID(), Name: "proj" + string(rune('a'+i)),
			RateLimit: 10, DefaultProfile: string(models.ProfileStandard),
			CreatedAt: now.Add(time.Duration(i) * time.Minute), UpdatedAt: now,
		}
		if err := q.CreateProject(p); err != nil {
			t.Fatalf("CreateProject %d: %v", i, err)
		}
	}

	list, err := q.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("list len = %d, want 3", len(list))
	}
}

func TestCountProjects(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)

	count, err := q.CountProjects()
	if err != nil {
		t.Fatalf("CountProjects: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestListProjectsPaginated(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		p := &models.Project{
			ID: util.GenerateID(), Name: "pag" + string(rune('a'+i)),
			RateLimit: 10, DefaultProfile: string(models.ProfileStandard),
			CreatedAt: now.Add(time.Duration(i) * time.Minute), UpdatedAt: now,
		}
		if err := q.CreateProject(p); err != nil {
			t.Fatalf("CreateProject %d: %v", i, err)
		}
	}

	page, err := q.ListProjectsPaginated(2, 1)
	if err != nil {
		t.Fatalf("ListProjectsPaginated: %v", err)
	}
	if len(page) != 2 {
		t.Errorf("page len = %d, want 2", len(page))
	}
}

func TestDeleteProject(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)

	if err := q.DeleteProject("proj-1"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	got, err := q.GetProject("proj-1")
	if err != nil {
		t.Fatalf("GetProject after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestUpdateProjectPipelineConfig(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)

	if err := q.UpdateProjectPipelineConfig("proj-1", `{"enable_nuclei":false}`); err != nil {
		t.Fatalf("UpdateProjectPipelineConfig: %v", err)
	}
	got, err := q.GetProject("proj-1")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.PipelineConfig == nil || *got.PipelineConfig != `{"enable_nuclei":false}` {
		t.Errorf("pipeline_config = %v", *got.PipelineConfig)
	}
}

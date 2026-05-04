package service

import (
	"context"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/errors"
)

func TestProjectService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockProjectRepository()
		audit := NewMockAuditLogger()
		svc := NewProjectServiceWithDeps(repo, audit)

		project, err := svc.Create(context.Background(), CreateProjectRequest{
			Name:         "Test Project",
			Organization: "Test Org",
			Purpose:      "testing",
			RateLimit:    10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if project.Name != "Test Project" {
			t.Errorf("name = %q, want %q", project.Name, "Test Project")
		}
		if project.Organization != "Test Org" {
			t.Errorf("organization = %q, want %q", project.Organization, "Test Org")
		}
		if project.DefaultProfile != "standard" {
			t.Errorf("default_profile = %q, want %q", project.DefaultProfile, "standard")
		}

		// Verify audit log was created
		logs := audit.GetAll()
		if len(logs) != 1 {
			t.Errorf("audit logs = %d, want 1", len(logs))
		}
		if logs[0].Action != "project.create" {
			t.Errorf("audit action = %q, want %q", logs[0].Action, "project.create")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		repo := NewMockProjectRepository()
		audit := NewMockAuditLogger()
		svc := NewProjectServiceWithDeps(repo, audit)

		_, err := svc.Create(context.Background(), CreateProjectRequest{
			Name: "",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Fatalf("expected AppError, got %T", err)
		}
		if appErr.Code != errors.ErrBadRequest {
			t.Errorf("error code = %q, want %q", appErr.Code, errors.ErrBadRequest)
		}
	})

	t.Run("negative rate limit", func(t *testing.T) {
		repo := NewMockProjectRepository()
		audit := NewMockAuditLogger()
		svc := NewProjectServiceWithDeps(repo, audit)

		_, err := svc.Create(context.Background(), CreateProjectRequest{
			Name:      "Test",
			RateLimit: -1,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Fatalf("expected AppError, got %T", err)
		}
		if appErr.Code != errors.ErrBadRequest {
			t.Errorf("error code = %q, want %q", appErr.Code, errors.ErrBadRequest)
		}
	})
}

func TestProjectService_Get(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo := NewMockProjectRepository()
		audit := NewMockAuditLogger()
		svc := NewProjectServiceWithDeps(repo, audit)

		// Create a project first
		created, err := svc.Create(context.Background(), CreateProjectRequest{
			Name: "Test Project",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Get the project
		got, err := svc.Get(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("id = %q, want %q", got.ID, created.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := NewMockProjectRepository()
		audit := NewMockAuditLogger()
		svc := NewProjectServiceWithDeps(repo, audit)

		_, err := svc.Get(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Fatalf("expected AppError, got %T", err)
		}
		if appErr.Code != errors.ErrNotFound {
			t.Errorf("error code = %q, want %q", appErr.Code, errors.ErrNotFound)
		}
	})
}

func TestProjectService_List(t *testing.T) {
	repo := NewMockProjectRepository()
	audit := NewMockAuditLogger()
	svc := NewProjectServiceWithDeps(repo, audit)

	// Create some projects
	for i := 0; i < 3; i++ {
		_, err := svc.Create(context.Background(), CreateProjectRequest{
			Name: "Project " + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	projects, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("len(projects) = %d, want 3", len(projects))
	}
}

func TestProjectService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockProjectRepository()
		audit := NewMockAuditLogger()
		svc := NewProjectServiceWithDeps(repo, audit)

		// Create a project first
		created, err := svc.Create(context.Background(), CreateProjectRequest{
			Name: "To Delete",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Delete the project
		err = svc.Delete(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it's deleted
		_, err = svc.Get(context.Background(), created.ID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Fatalf("expected AppError, got %T", err)
		}
		if appErr.Code != errors.ErrNotFound {
			t.Errorf("error code = %q, want %q", appErr.Code, errors.ErrNotFound)
		}

		// Verify audit log was created
		logs := audit.GetAll()
		if len(logs) != 2 { // create + delete
			t.Errorf("audit logs = %d, want 2", len(logs))
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := NewMockProjectRepository()
		audit := NewMockAuditLogger()
		svc := NewProjectServiceWithDeps(repo, audit)

		err := svc.Delete(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*errors.AppError)
		if !ok {
			t.Fatalf("expected AppError, got %T", err)
		}
		if appErr.Code != errors.ErrNotFound {
			t.Errorf("error code = %q, want %q", appErr.Code, errors.ErrNotFound)
		}
	})
}

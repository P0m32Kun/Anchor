package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestEngineCredential_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	extra := `{"email":"test@example.com"}`
	cred := &models.EngineCredential{
		ID: util.GenerateID(), Engine: "fofa", APIKey: "key123",
		Extra: &extra, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.SaveEngineCredential(cred); err != nil {
		t.Fatalf("SaveEngineCredential: %v", err)
	}

	// Get
	got, err := q.GetEngineCredential("fofa")
	if err != nil {
		t.Fatalf("GetEngineCredential: %v", err)
	}
	if got == nil {
		t.Fatal("credential not found")
	}
	if got.APIKey != "key123" {
		t.Errorf("api_key = %q, want key123", got.APIKey)
	}
	if got.Extra == nil || *got.Extra != extra {
		t.Errorf("extra = %v, want %q", got.Extra, extra)
	}

	// List
	cred2 := &models.EngineCredential{
		ID: util.GenerateID(), Engine: "hunter", APIKey: "key456",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := q.SaveEngineCredential(cred2); err != nil {
		t.Fatalf("SaveEngineCredential hunter: %v", err)
	}

	list, err := q.ListEngineCredentials()
	if err != nil {
		t.Fatalf("ListEngineCredentials: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("list len = %d, want 2", len(list))
	}

	// Upsert
	cred.APIKey = "newkey"
	cred.UpdatedAt = time.Now().UTC()
	if err := q.SaveEngineCredential(cred); err != nil {
		t.Fatalf("SaveEngineCredential upsert: %v", err)
	}
	got2, err := q.GetEngineCredential("fofa")
	if err != nil {
		t.Fatalf("GetEngineCredential after upsert: %v", err)
	}
	if got2.APIKey != "newkey" {
		t.Errorf("api_key = %q, want newkey", got2.APIKey)
	}

	// Delete
	if err := q.DeleteEngineCredential("fofa"); err != nil {
		t.Fatalf("DeleteEngineCredential: %v", err)
	}
	got3, err := q.GetEngineCredential("fofa")
	if err != nil {
		t.Fatalf("GetEngineCredential after delete: %v", err)
	}
	if got3 != nil {
		t.Error("expected nil after delete")
	}
}

func TestGetEngineCredential_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetEngineCredential("nonexistent")
	if err != nil {
		t.Fatalf("GetEngineCredential: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent engine")
	}
}

func TestCreateAuditLog(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	a := &models.AuditLog{
		ID: util.GenerateID(), ProjectID: "proj-1", Actor: "user",
		Action: "create", ResourceType: "target", ResourceID: "tgt-1",
		Summary: "created target", CreatedAt: now,
	}
	if err := q.CreateAuditLog(a); err != nil {
		t.Fatalf("CreateAuditLog: %v", err)
	}
}

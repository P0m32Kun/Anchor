package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestExcludedDomain_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	d := &models.ExcludedDomain{
		ID: util.GenerateID(), Domain: "example.com", Reason: "test",
		Builtin: false, CreatedAt: now,
	}
	if err := q.CreateExcludedDomain(d); err != nil {
		t.Fatalf("CreateExcludedDomain: %v", err)
	}

	// Get
	got, err := q.GetExcludedDomain("example.com")
	if err != nil {
		t.Fatalf("GetExcludedDomain: %v", err)
	}
	if got == nil {
		t.Fatal("excluded domain not found")
	}
	if got.Domain != "example.com" {
		t.Errorf("domain = %q, want example.com", got.Domain)
	}

	// Exists
	exists, err := q.ExcludedDomainExists("example.com")
	if err != nil {
		t.Fatalf("ExcludedDomainExists: %v", err)
	}
	if !exists {
		t.Error("expected domain to exist")
	}

	exists, err = q.ExcludedDomainExists("nonexistent.com")
	if err != nil {
		t.Fatalf("ExcludedDomainExists nonexistent: %v", err)
	}
	if exists {
		t.Error("expected domain not to exist")
	}

	// Delete
	if err := q.DeleteExcludedDomain("example.com"); err != nil {
		t.Fatalf("DeleteExcludedDomain: %v", err)
	}
	got2, err := q.GetExcludedDomain("example.com")
	if err != nil {
		t.Fatalf("GetExcludedDomain after delete: %v", err)
	}
	if got2 != nil {
		t.Error("expected nil after delete")
	}
}

func TestListExcludedDomains(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	domains := []*models.ExcludedDomain{
		{ID: util.GenerateID(), Domain: "builtin.com", Reason: "built-in", Builtin: true, CreatedAt: now},
		{ID: util.GenerateID(), Domain: "custom.com", Reason: "custom", Builtin: false, CreatedAt: now},
	}
	for _, d := range domains {
		if err := q.CreateExcludedDomain(d); err != nil {
			t.Fatalf("CreateExcludedDomain %s: %v", d.Domain, err)
		}
	}

	all, err := q.ListExcludedDomains()
	if err != nil {
		t.Fatalf("ListExcludedDomains: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("all len = %d, want 2", len(all))
	}
	// Builtin first
	if !all[0].Builtin {
		t.Error("expected builtin domain first")
	}
}

func TestListCustomExcludedDomains(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	domains := []*models.ExcludedDomain{
		{ID: util.GenerateID(), Domain: "builtin2.com", Reason: "built-in", Builtin: true, CreatedAt: now},
		{ID: util.GenerateID(), Domain: "custom2.com", Reason: "custom", Builtin: false, CreatedAt: now},
	}
	for _, d := range domains {
		if err := q.CreateExcludedDomain(d); err != nil {
			t.Fatalf("CreateExcludedDomain %s: %v", d.Domain, err)
		}
	}

	custom, err := q.ListCustomExcludedDomains()
	if err != nil {
		t.Fatalf("ListCustomExcludedDomains: %v", err)
	}
	if len(custom) != 1 {
		t.Errorf("custom len = %d, want 1", len(custom))
	}
	if custom[0].Domain != "custom2.com" {
		t.Errorf("domain = %q, want custom2.com", custom[0].Domain)
	}
}

func TestDeleteAllCustomExcludedDomains(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	domains := []*models.ExcludedDomain{
		{ID: util.GenerateID(), Domain: "builtin3.com", Reason: "built-in", Builtin: true, CreatedAt: now},
		{ID: util.GenerateID(), Domain: "custom3.com", Reason: "custom", Builtin: false, CreatedAt: now},
	}
	for _, d := range domains {
		if err := q.CreateExcludedDomain(d); err != nil {
			t.Fatalf("CreateExcludedDomain %s: %v", d.Domain, err)
		}
	}

	if err := q.DeleteAllCustomExcludedDomains(); err != nil {
		t.Fatalf("DeleteAllCustomExcludedDomains: %v", err)
	}

	all, err := q.ListExcludedDomains()
	if err != nil {
		t.Fatalf("ListExcludedDomains: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("remaining len = %d, want 1 (builtin only)", len(all))
	}
	if !all[0].Builtin {
		t.Error("expected remaining domain to be builtin")
	}
}

func TestBulkCreateExcludedDomains(t *testing.T) {
	q := New(openTestDB(t))

	domains := []*models.ExcludedDomain{
		{ID: util.GenerateID(), Domain: "bulk1.com", Reason: "test", Builtin: false},
		{ID: util.GenerateID(), Domain: "bulk2.com", Reason: "test", Builtin: false},
		{ID: util.GenerateID(), Domain: "bulk3.com", Reason: "test", Builtin: false},
	}
	if err := q.BulkCreateExcludedDomains(domains); err != nil {
		t.Fatalf("BulkCreateExcludedDomains: %v", err)
	}

	all, err := q.ListExcludedDomains()
	if err != nil {
		t.Fatalf("ListExcludedDomains: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all len = %d, want 3", len(all))
	}
}

func TestBulkCreateExcludedDomains_Idempotent(t *testing.T) {
	q := New(openTestDB(t))

	domains := []*models.ExcludedDomain{
		{ID: util.GenerateID(), Domain: "idem.com", Reason: "test", Builtin: false},
	}
	if err := q.BulkCreateExcludedDomains(domains); err != nil {
		t.Fatalf("first bulk: %v", err)
	}
	// INSERT OR IGNORE should not error
	if err := q.BulkCreateExcludedDomains(domains); err != nil {
		t.Fatalf("second bulk: %v", err)
	}

	all, err := q.ListExcludedDomains()
	if err != nil {
		t.Fatalf("ListExcludedDomains: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("all len = %d, want 1 (idempotent)", len(all))
	}
}

package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestHttpxFingerprint_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	f := &models.HttpxFingerprint{
		ID: util.GenerateID(), Name: "nginx-favicon", Description: "nginx favicon hash",
		Type: models.HttpxFingerprintTypeFavicon, FilePath: "/opt/fp/nginx.ico",
		Enabled: true, Builtin: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateHttpxFingerprint(f); err != nil {
		t.Fatalf("CreateHttpxFingerprint: %v", err)
	}

	// Get
	got, err := q.GetHttpxFingerprint(f.ID)
	if err != nil {
		t.Fatalf("GetHttpxFingerprint: %v", err)
	}
	if got == nil {
		t.Fatal("fingerprint not found")
	}
	if got.Name != "nginx-favicon" {
		t.Errorf("name = %q, want nginx-favicon", got.Name)
	}
	if !got.Builtin {
		t.Error("expected builtin=true")
	}

	// Update
	f.Name = "nginx-favicon-v2"
	f.UpdatedAt = time.Now().UTC()
	if err := q.UpdateHttpxFingerprint(f); err != nil {
		t.Fatalf("UpdateHttpxFingerprint: %v", err)
	}
	got2, err := q.GetHttpxFingerprint(f.ID)
	if err != nil {
		t.Fatalf("GetHttpxFingerprint after update: %v", err)
	}
	if got2.Name != "nginx-favicon-v2" {
		t.Errorf("name = %q, want nginx-favicon-v2", got2.Name)
	}

	// Delete
	if err := q.DeleteHttpxFingerprint(f.ID); err != nil {
		t.Fatalf("DeleteHttpxFingerprint: %v", err)
	}
	got3, err := q.GetHttpxFingerprint(f.ID)
	if err != nil {
		t.Fatalf("GetHttpxFingerprint after delete: %v", err)
	}
	if got3 != nil {
		t.Error("expected nil after delete")
	}
}

func TestListHttpxFingerprints(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	fps := []*models.HttpxFingerprint{
		{ID: util.GenerateID(), Name: "fp1", Type: models.HttpxFingerprintTypeFavicon, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), Name: "fp2", Type: models.HttpxFingerprintTypeTechDetect, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), Name: "fp3", Type: models.HttpxFingerprintTypeFavicon, Enabled: false, CreatedAt: now, UpdatedAt: now},
	}
	for _, fp := range fps {
		if err := q.CreateHttpxFingerprint(fp); err != nil {
			t.Fatalf("CreateHttpxFingerprint %s: %v", fp.Name, err)
		}
	}

	// All
	all, err := q.ListHttpxFingerprints("")
	if err != nil {
		t.Fatalf("ListHttpxFingerprints all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all len = %d, want 3", len(all))
	}

	// By type
	favicons, err := q.ListHttpxFingerprints(string(models.HttpxFingerprintTypeFavicon))
	if err != nil {
		t.Fatalf("ListHttpxFingerprints favicon: %v", err)
	}
	if len(favicons) != 2 {
		t.Errorf("favicons len = %d, want 2", len(favicons))
	}
}

func TestListEnabledHttpxFingerprints(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	fps := []*models.HttpxFingerprint{
		{ID: util.GenerateID(), Name: "en1", Type: models.HttpxFingerprintTypeFavicon, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), Name: "dis1", Type: models.HttpxFingerprintTypeFavicon, Enabled: false, CreatedAt: now, UpdatedAt: now},
	}
	for _, fp := range fps {
		if err := q.CreateHttpxFingerprint(fp); err != nil {
			t.Fatalf("CreateHttpxFingerprint %s: %v", fp.Name, err)
		}
	}

	enabled, err := q.ListEnabledHttpxFingerprints("")
	if err != nil {
		t.Fatalf("ListEnabledHttpxFingerprints: %v", err)
	}
	if len(enabled) != 1 {
		t.Errorf("enabled len = %d, want 1", len(enabled))
	}
}

func TestUpdateHttpxFingerprintEnabled(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	f := &models.HttpxFingerprint{
		ID: util.GenerateID(), Name: "builtin-fp", Type: models.HttpxFingerprintTypeFavicon,
		Enabled: true, Builtin: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateHttpxFingerprint(f); err != nil {
		t.Fatalf("CreateHttpxFingerprint: %v", err)
	}

	if err := q.UpdateHttpxFingerprintEnabled(f.ID, false, time.Now().UTC()); err != nil {
		t.Fatalf("UpdateHttpxFingerprintEnabled: %v", err)
	}

	got, err := q.GetHttpxFingerprint(f.ID)
	if err != nil {
		t.Fatalf("GetHttpxFingerprint: %v", err)
	}
	if got.Enabled {
		t.Error("expected enabled=false after update")
	}
}

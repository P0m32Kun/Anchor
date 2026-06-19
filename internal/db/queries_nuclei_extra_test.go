package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- ListEnabledNucleiCustomSourceIDs ---

func TestListEnabledNucleiCustomSourceIDs(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	// Create enabled and disabled sources
	if err := q.CreateNucleiCustomSource(&models.NucleiCustomSource{
		ID: "src-en", Name: "enabled", InstallPath: "/opt/en",
		Type: models.NucleiCustomSourceTypeGit, Enabled: true,
		Status: models.NucleiCustomSourceStatusReady, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create enabled source: %v", err)
	}
	if err := q.CreateNucleiCustomSource(&models.NucleiCustomSource{
		ID: "src-dis", Name: "disabled", InstallPath: "/opt/dis",
		Type: models.NucleiCustomSourceTypeGit, Enabled: false,
		Status: models.NucleiCustomSourceStatusReady, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create disabled source: %v", err)
	}

	ids, err := q.ListEnabledNucleiCustomSourceIDs()
	if err != nil {
		t.Fatalf("ListEnabledNucleiCustomSourceIDs: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 enabled, got %d", len(ids))
	}
	if ids[0] != "src-en" {
		t.Errorf("id = %q, want src-en", ids[0])
	}
}

func TestListEnabledNucleiCustomSourceIDs_Empty(t *testing.T) {
	q := New(openTestDB(t))

	ids, err := q.ListEnabledNucleiCustomSourceIDs()
	if err != nil {
		t.Fatalf("ListEnabledNucleiCustomSourceIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0, got %d", len(ids))
	}
}

// --- intToBool ---

func TestIntToBool(t *testing.T) {
	if intToBool(0) {
		t.Error("intToBool(0) should be false")
	}
	if !intToBool(1) {
		t.Error("intToBool(1) should be true")
	}
	if !intToBool(42) {
		t.Error("intToBool(42) should be true")
	}
	if !intToBool(-1) {
		t.Error("intToBool(-1) should be true (non-zero)")
	}
}

// --- GetActiveNucleiCustomBundleVersion ---

func TestGetActiveNucleiCustomBundleVersion(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	// No active bundle
	ver, err := q.GetActiveNucleiCustomBundleVersion()
	if err != nil {
		t.Fatalf("GetActiveNucleiCustomBundleVersion empty: %v", err)
	}
	if ver != "" {
		t.Errorf("expected empty, got %q", ver)
	}

	// Create bundles
	if err := q.CreateNucleiCustomBundle(&models.NucleiCustomBundle{
		Version: "v1.0", ManifestJSON: "{}", ArchivePath: "/tmp/v1.zip",
		Status: "active", CreatedAt: now,
	}); err != nil {
		t.Fatalf("create active bundle: %v", err)
	}
	if err := q.CreateNucleiCustomBundle(&models.NucleiCustomBundle{
		Version: "v0.9", ManifestJSON: "{}", ArchivePath: "/tmp/v0.9.zip",
		Status: "superseded", CreatedAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create superseded bundle: %v", err)
	}

	ver, err = q.GetActiveNucleiCustomBundleVersion()
	if err != nil {
		t.Fatalf("GetActiveNucleiCustomBundleVersion: %v", err)
	}
	if ver != "v1.0" {
		t.Errorf("version = %q, want v1.0", ver)
	}
}

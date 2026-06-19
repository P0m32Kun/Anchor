package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- CreateAsset with empty tags/sourceTools ---

func TestCreateAsset_EmptyJSONFields(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	q.CreateProject(&models.Project{
		ID: "proj-ej", Name: "ej-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	})

	// Asset with nil SourceTools and nil Tags
	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: "proj-ej",
		Type: models.AssetTypeDomain, Value: "empty.example.com",
		NormalizedValue: "empty.example.com",
		FirstSeen: now, LastSeen: now,
	}
	if err := q.CreateAsset(asset); err != nil {
		t.Fatalf("CreateAsset: %v", err)
	}

	got, err := q.GetAssetByID(asset.ID)
	if err != nil {
		t.Fatalf("GetAssetByID: %v", err)
	}
	if got == nil {
		t.Fatal("asset not found")
	}
	// scanAsset returns nil for empty JSON — this is expected backward-compat behavior
	if got.SourceTools != nil && len(got.SourceTools) != 0 {
		t.Errorf("source_tools = %v, want nil or empty", got.SourceTools)
	}
	if got.Tags != nil && len(got.Tags) != 0 {
		t.Errorf("tags = %v, want nil or empty", got.Tags)
	}
}

// --- GetAssetByID not found ---

func TestGetAssetByID_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetAssetByID("nonexistent")
	if err != nil {
		t.Fatalf("GetAssetByID: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent asset")
	}
}

// --- GetAssetByNormalizedValue not found ---

func TestGetAssetByNormalizedValue_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetAssetByNormalizedValue("proj-none", "nope.example.com")
	if err != nil {
		t.Fatalf("GetAssetByNormalizedValue: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent asset")
	}
}

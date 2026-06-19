package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- GetAssetState edge cases ---

func TestGetAssetState_NotFound(t *testing.T) {
	q := New(openTestDB(t))

	state, err := q.GetAssetState("nonexistent")
	if err != nil {
		t.Fatalf("GetAssetState: %v", err)
	}
	if state.Fingerprinted {
		t.Error("expected zero value for nonexistent asset")
	}
	if state.Alive != nil {
		t.Error("expected nil Alive for nonexistent asset")
	}
}

func TestGetAssetState_WithTechnologies(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-as", Name: "as-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	assetID := util.GenerateID()
	if err := q.CreateAsset(&models.Asset{
		ID: assetID, ProjectID: "proj-as", Type: models.AssetTypeDomain,
		Value: "tech.example.com", NormalizedValue: "tech.example.com",
		FirstSeen: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	alive := true
	statusCode := 200
	attrs := core.AssetAttrs{
		Alive:         &alive,
		Fingerprinted: true,
		Technologies:  []string{"nginx", "php"},
		StatusCode:    &statusCode,
		Sensitivity:   "high",
		RequiresAuth:  true,
		Tags:          []string{"admin", "login"},
	}
	if err := q.UpdateAssetState(assetID, attrs); err != nil {
		t.Fatalf("UpdateAssetState: %v", err)
	}

	got, err := q.GetAssetState(assetID)
	if err != nil {
		t.Fatalf("GetAssetState: %v", err)
	}
	if !got.Fingerprinted {
		t.Error("expected fingerprinted=true")
	}
	if len(got.Technologies) != 2 {
		t.Errorf("technologies len = %d, want 2", len(got.Technologies))
	}
	if got.StatusCode == nil || *got.StatusCode != 200 {
		t.Errorf("status_code = %v, want 200", got.StatusCode)
	}
	if got.Sensitivity != "high" {
		t.Errorf("sensitivity = %q, want high", got.Sensitivity)
	}
	if !got.RequiresAuth {
		t.Error("expected requires_auth=true")
	}
	if len(got.Tags) != 2 {
		t.Errorf("tags len = %d, want 2", len(got.Tags))
	}
}

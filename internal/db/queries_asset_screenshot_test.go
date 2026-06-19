package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- UpdateWebEndpointScreenshotArtifactID ---

func TestUpdateWebEndpointScreenshotArtifactID(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-ws", Name: "ws-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	assetID := util.GenerateID()
	if err := q.CreateAsset(&models.Asset{
		ID: assetID, ProjectID: "proj-ws", Type: models.AssetTypeDomain,
		Value: "example.com", NormalizedValue: "example.com",
		FirstSeen: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	we := &models.WebEndpoint{
		ID: "we-1", ProjectID: "proj-ws", AssetID: assetID,
		URL: "https://example.com/", Scheme: "https", Host: "example.com",
		Path: "/", SourceTool: "httpx", CreatedAt: now,
	}
	if err := q.CreateWebEndpoint(we); err != nil {
		t.Fatalf("CreateWebEndpoint: %v", err)
	}

	// Create raw_artifact first (FK on screenshot_artifact_id)
	if err := q.CreateRawArtifact(&models.RawArtifact{
		ID: "art-ss-1", ProjectID: "proj-ws", Type: "screenshot",
		Path: "/tmp/ss.png", CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateRawArtifact: %v", err)
	}

	// Update screenshot artifact ID
	if err := q.UpdateWebEndpointScreenshotArtifactID("we-1", "art-ss-1"); err != nil {
		t.Fatalf("UpdateWebEndpointScreenshotArtifactID: %v", err)
	}

	got, err := q.GetWebEndpointByURL("proj-ws", "https://example.com/")
	if err != nil {
		t.Fatalf("GetWebEndpointByURL: %v", err)
	}
	if got == nil {
		t.Fatal("expected endpoint, got nil")
	}
	if got.ScreenshotArtifactID == nil || *got.ScreenshotArtifactID != "art-ss-1" {
		t.Errorf("screenshot_artifact_id = %v, want art-ss-1", got.ScreenshotArtifactID)
	}
}

package asset_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/asset"
	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestEncodeAssetState(t *testing.T) {
	attrs := core.AssetAttrs{
		Fingerprinted: true,
		Technologies:  []string{"nginx", "php"},
	}
	encoded, err := asset.EncodeAssetState(attrs)
	if err != nil {
		t.Fatalf("EncodeAssetState: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected non-empty encoded string")
	}
	var decoded core.AssetAttrs
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("encoded state is not valid JSON: %v", err)
	}
	if !decoded.Fingerprinted {
		t.Error("expected Fingerprinted=true in decoded")
	}
	if len(decoded.Technologies) != 2 {
		t.Errorf("expected 2 technologies, got %d", len(decoded.Technologies))
	}
}

func TestEncodeAssetState_Empty(t *testing.T) {
	encoded, err := asset.EncodeAssetState(core.AssetAttrs{})
	if err != nil {
		t.Fatalf("EncodeAssetState empty: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected non-empty for empty attrs")
	}
}

func TestMergeAndSaveState(t *testing.T) {
	raw, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = raw.Close() })

	q := db.New(raw)
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-merge", Name: "merge", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	assetID := util.GenerateID()
	if err := q.CreateAsset(&models.Asset{
		ID: assetID, ProjectID: "proj-merge",
		Type: models.AssetTypeURL, Value: "https://bar.example.com",
		NormalizedValue: "https://bar.example.com",
		FirstSeen: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	status := 200
	if err := asset.MergeAndSaveState(q, assetID, core.AssetAttrs{
		Fingerprinted: true,
		Technologies:  []string{"jenkins"},
		StatusCode:    &status,
	}); err != nil {
		t.Fatalf("merge save: %v", err)
	}

	attrs, err := asset.LoadAttrsForEngine(q, assetID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !attrs.Fingerprinted || len(attrs.Technologies) != 1 {
		t.Fatalf("attrs = %+v", attrs)
	}
}

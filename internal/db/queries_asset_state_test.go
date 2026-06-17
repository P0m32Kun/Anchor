package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestAssetState_ReadWriteMerge(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-state", Name: "state", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	assetID := util.GenerateID()
	if err := q.CreateAsset(&models.Asset{
		ID: assetID, ProjectID: "proj-state",
		Type: models.AssetTypeDomain, Value: "foo.example.com",
		NormalizedValue: "foo.example.com",
		FirstSeen: now, LastSeen: now,
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	alive := true
	if err := q.UpdateAssetState(assetID, core.AssetAttrs{Alive: &alive, Fingerprinted: false}); err != nil {
		t.Fatalf("update state: %v", err)
	}

	state, err := q.GetAssetState(assetID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state.Alive == nil || !*state.Alive {
		t.Fatalf("alive = %+v, want true", state.Alive)
	}

	fp := true
	status := 200
	aliveStill := true
	if err := q.UpdateAssetState(assetID, core.AssetAttrs{
		Alive:         &aliveStill,
		Fingerprinted: fp,
		Technologies:  []string{"nginx"},
		StatusCode:    &status,
	}); err != nil {
		t.Fatalf("update state 2: %v", err)
	}

	state, err = q.GetAssetState(assetID)
	if err != nil {
		t.Fatalf("get state 2: %v", err)
	}
	if !state.Fingerprinted {
		t.Error("fingerprinted should be true")
	}
	if len(state.Technologies) != 1 || state.Technologies[0] != "nginx" {
		t.Errorf("technologies = %v", state.Technologies)
	}
}

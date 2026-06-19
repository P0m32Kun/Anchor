package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestAssetSnapshot_CreateAndGetLatest(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)
	now := time.Now().UTC()

	snap1 := &AssetSnapshot{
		ID: util.GenerateID(), ProjectID: "proj-1", RunID: "run-1",
		AssetCount: 10, PortCount: 5, EndpointCount: 3, ServiceCount: 2,
		AssetChangesJSON: `{"new":2}`, CreatedAt: now.Add(-time.Hour),
	}
	if err := q.CreateAssetSnapshot(snap1); err != nil {
		t.Fatalf("CreateAssetSnapshot 1: %v", err)
	}

	snap2 := &AssetSnapshot{
		ID: util.GenerateID(), ProjectID: "proj-1", RunID: "run-2",
		AssetCount: 12, PortCount: 6, EndpointCount: 4, ServiceCount: 3,
		AssetChangesJSON: `{"new":3}`, CreatedAt: now,
	}
	if err := q.CreateAssetSnapshot(snap2); err != nil {
		t.Fatalf("CreateAssetSnapshot 2: %v", err)
	}

	got, err := q.GetLatestAssetSnapshot("proj-1", "run-3")
	if err != nil {
		t.Fatalf("GetLatestAssetSnapshot: %v", err)
	}
	if got == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if got.AssetCount != 12 {
		t.Errorf("asset_count = %d, want 12", got.AssetCount)
	}
	if got.PortCount != 6 {
		t.Errorf("port_count = %d, want 6", got.PortCount)
	}
}

func TestAssetSnapshot_AutoID(t *testing.T) {
	q := New(openTestDB(t))
	createTestProject(t, q)

	snap := &AssetSnapshot{
		ProjectID: "proj-1", RunID: "run-auto",
		AssetCount: 5, PortCount: 2, EndpointCount: 1, ServiceCount: 1,
		CreatedAt: time.Now().UTC(),
	}
	if err := q.CreateAssetSnapshot(snap); err != nil {
		t.Fatalf("CreateAssetSnapshot: %v", err)
	}
	if snap.ID == "" {
		t.Error("expected auto-generated ID")
	}
	if snap.AssetChangesJSON != "{}" {
		t.Errorf("asset_changes_json = %q, want {}", snap.AssetChangesJSON)
	}
}

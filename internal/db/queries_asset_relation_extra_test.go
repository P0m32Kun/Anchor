package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- UpsertAssetRelation edge cases ---

func TestUpsertAssetRelation_NilReturnsError(t *testing.T) {
	q := New(openTestDB(t))

	if err := q.UpsertAssetRelation(nil); err == nil {
		t.Error("expected error for nil relation")
	}
}

func TestUpsertAssetRelation_EmptyIDReturnsError(t *testing.T) {
	q := New(openTestDB(t))

	if err := q.UpsertAssetRelation(&models.AssetRelation{
		ProjectID: "proj-1", SourceType: "target", SourceID: "s",
		TargetType: "asset", TargetID: "t", RelationType: "expanded_by",
	}); err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestUpsertAssetRelation_NilRunID(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-nilrun", Name: "nilrun", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	rel := &models.AssetRelation{
		ID: util.GenerateID(), ProjectID: "proj-nilrun",
		SourceType: "target", SourceID: "tgt-1",
		TargetType: "asset", TargetID: "ast-1",
		RelationType: "expanded_by", SourceEngine: "fofa",
		// RunID is nil
	}
	if err := q.UpsertAssetRelation(rel); err != nil {
		t.Fatalf("UpsertAssetRelation with nil RunID: %v", err)
	}

	incoming, err := q.ListIncomingAssetRelations("proj-nilrun", "ast-1", nil)
	if err != nil {
		t.Fatalf("ListIncomingAssetRelations: %v", err)
	}
	if len(incoming) != 1 {
		t.Fatalf("expected 1, got %d", len(incoming))
	}
	if incoming[0].RunID != nil {
		t.Errorf("expected nil run_id, got %v", *incoming[0].RunID)
	}
}

func TestUpsertAssetRelation_ZeroCreatedAt(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-zero", Name: "zero", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	rel := &models.AssetRelation{
		ID: util.GenerateID(), ProjectID: "proj-zero",
		SourceType: "target", SourceID: "tgt-1",
		TargetType: "asset", TargetID: "ast-1",
		RelationType: "expanded_by",
		// CreatedAt is zero
	}
	if err := q.UpsertAssetRelation(rel); err != nil {
		t.Fatalf("UpsertAssetRelation zero CreatedAt: %v", err)
	}
	if rel.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be auto-set")
	}
}

func TestListIncomingAssetRelations_WithRunID(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-filter", Name: "filter", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	run1 := "run-1"
	run2 := "run-2"
	q.UpsertAssetRelation(&models.AssetRelation{
		ID: util.GenerateID(), ProjectID: "proj-filter", RunID: &run1,
		SourceType: "target", SourceID: "tgt-1",
		TargetType: "asset", TargetID: "ast-1",
		RelationType: "expanded_by", SourceEngine: "fofa", CreatedAt: now,
	})
	q.UpsertAssetRelation(&models.AssetRelation{
		ID: util.GenerateID(), ProjectID: "proj-filter", RunID: &run2,
		SourceType: "target", SourceID: "tgt-1",
		TargetType: "asset", TargetID: "ast-1",
		RelationType: "expanded_by", SourceEngine: "hunter", CreatedAt: now,
	})

	// Filter by run-1
	list, err := q.ListIncomingAssetRelations("proj-filter", "ast-1", &run1)
	if err != nil {
		t.Fatalf("ListIncomingAssetRelations with runID: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].SourceEngine != "fofa" {
		t.Errorf("engine = %q, want fofa", list[0].SourceEngine)
	}
}

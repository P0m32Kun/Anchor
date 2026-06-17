package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// FT-LINEAGE-01: asset_relations CRUD and lineage walk.
func TestAssetRelation_UpsertAndLineage(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	project := &models.Project{
		ID: "proj-lineage", Name: "lineage-test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	target := &models.Target{
		ID: util.GenerateID(), ProjectID: project.ID,
		Type: models.TargetTypeCompany, Value: "TestCorp",
		Source: "manual", Status: "active", CreatedAt: now,
	}
	if err := q.CreateTarget(target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	domainAsset := &models.Asset{
		ID: util.GenerateID(), ProjectID: project.ID,
		Type: models.AssetTypeDomain, Value: "foo.example.com",
		NormalizedValue: "foo.example.com",
		FirstSeen: now, LastSeen: now,
	}
	if err := q.CreateAsset(domainAsset); err != nil {
		t.Fatalf("create domain asset: %v", err)
	}

	childAsset := &models.Asset{
		ID: util.GenerateID(), ProjectID: project.ID,
		Type: models.AssetTypeURL, Value: "https://foo.example.com/admin",
		NormalizedValue: "https://foo.example.com/admin",
		FirstSeen: now, LastSeen: now,
	}
	if err := q.CreateAsset(childAsset); err != nil {
		t.Fatalf("create child asset: %v", err)
	}

	runID := "run-lineage-1"
	runIDPtr := &runID

	if err := q.UpsertAssetRelation(&models.AssetRelation{
		ID: util.GenerateID(), ProjectID: project.ID, RunID: runIDPtr,
		SourceType: models.RelationSourceTarget, SourceID: target.ID,
		TargetType: models.RelationTargetAsset, TargetID: domainAsset.ID,
		RelationType: models.RelationExpandedBy, SourceEngine: "fofa",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("upsert expanded_by: %v", err)
	}

	if err := q.UpsertAssetRelation(&models.AssetRelation{
		ID: util.GenerateID(), ProjectID: project.ID, RunID: runIDPtr,
		SourceType: models.RelationSourceAsset, SourceID: domainAsset.ID,
		TargetType: models.RelationTargetAsset, TargetID: childAsset.ID,
		RelationType: models.RelationDiscoveredFrom, SourceEngine: "httpx",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("upsert discovered_from: %v", err)
	}

	lineage, err := q.BuildAssetLineage(project.ID, childAsset.ID, runIDPtr)
	if err != nil {
		t.Fatalf("BuildAssetLineage: %v", err)
	}
	if len(lineage.Chain) != 3 {
		t.Fatalf("chain len = %d, want 3", len(lineage.Chain))
	}
	if lineage.Chain[0].NodeType != "target" || lineage.Chain[0].Value != "TestCorp" {
		t.Errorf("root = %+v, want target TestCorp", lineage.Chain[0])
	}
	if lineage.Chain[1].Relation != models.RelationExpandedBy || lineage.Chain[1].SourceEngine != "fofa" {
		t.Errorf("domain hop = %+v, want expanded_by/fofa", lineage.Chain[1])
	}
	if lineage.Chain[2].Relation != models.RelationDiscoveredFrom {
		t.Errorf("child hop relation = %q, want discovered_from", lineage.Chain[2].Relation)
	}
}

func TestAssetRelation_UpsertIdempotent(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	if err := q.CreateProject(&models.Project{
		ID: "proj-idem", Name: "idem", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}

	rel := &models.AssetRelation{
		ID: util.GenerateID(), ProjectID: "proj-idem",
		SourceType: models.RelationSourceTarget, SourceID: "tgt-1",
		TargetType: models.RelationTargetAsset, TargetID: "ast-1",
		RelationType: models.RelationExpandedBy, SourceEngine: "fofa",
		CreatedAt: now,
	}
	if err := q.UpsertAssetRelation(rel); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	rel2 := *rel
	rel2.ID = util.GenerateID()
	rel2.SourceEngine = "hunter"
	if err := q.UpsertAssetRelation(&rel2); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	incoming, err := q.ListIncomingAssetRelations("proj-idem", "ast-1", nil)
	if err != nil {
		t.Fatalf("list incoming: %v", err)
	}
	if len(incoming) != 1 {
		t.Fatalf("incoming count = %d, want 1", len(incoming))
	}
	if incoming[0].SourceEngine != "hunter" {
		t.Errorf("source_engine = %q, want hunter", incoming[0].SourceEngine)
	}
}

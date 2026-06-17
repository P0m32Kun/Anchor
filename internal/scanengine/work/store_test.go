package work

import (
	"database/sql"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/pool"
)

func newTestQueries(t *testing.T) *db.Queries {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	sqlDB.SetMaxOpenConns(1)
	if err := db.Migrate(sqlDB); err != nil {
		t.Fatal(err)
	}
	return db.New(sqlDB)
}

func TestStore_CreatePooledBatch(t *testing.T) {
	q := newTestQueries(t)
	store := NewStore(q)

	if err := q.CreateProject(&models.Project{ID: "proj1", Name: "test", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := q.CreatePipelineRun(&models.PipelineRun{ID: "run1", ProjectID: "proj1", Status: "running", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	w, err := store.CreatePooledBatch(PooledBatchInput{
		RunID:     "run1",
		ProjectID: "proj1",
		Action:    core.ActionDNSResolve,
		Stage:     "resolve",
		InputFile: "/tmp/batch.txt",
		Members: []pool.Member{
			{Value: "a.example.com", AssetID: "asset-a"},
			{Value: "b.example.com", AssetID: "asset-b"},
		},
		BucketKey:  "tier1:DNS_RESOLVE",
		Generation: 1,
	})
	if err != nil {
		t.Fatalf("CreatePooledBatch: %v", err)
	}
	if !w.BatchMode {
		t.Fatal("expected batch_mode")
	}
	got, err := q.GetScanWorkItem(w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.BatchMode || got.MemberAssetIDs == "" {
		t.Fatalf("persisted batch work: %+v", got)
	}
}

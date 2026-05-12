package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func boolPtrLocal(b bool) *bool { return &b }

func writeSeedFile(t *testing.T, seeds []SeedFindingTemplate) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.json")
	data, err := json.Marshal(seeds)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSyncFindingTemplates_InsertsNew(t *testing.T) {
	q := New(openTestDB(t))
	path := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git", Title: "暴露 .git", Severity: "high", Summary: "S", Remediation: "R", Enabled: boolPtrLocal(true)},
	})
	res, err := q.SyncFindingTemplatesFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inserted != 1 || res.Updated != 0 || res.Preserved != 0 || res.Deleted != 0 {
		t.Fatalf("unexpected result %+v", res)
	}
	rows, _ := q.ListBuiltinFindingTemplates()
	if len(rows) != 1 || !rows[0].IsBuiltin || rows[0].UserModified {
		t.Fatalf("unexpected row %+v", rows[0])
	}
	if rows[0].BuiltinPayload == "" {
		t.Fatal("builtin_payload should be set")
	}
}

func TestSyncFindingTemplates_UpdatesUntouchedRow(t *testing.T) {
	q := New(openTestDB(t))
	pathA := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git", Title: "v1", Severity: "high", Summary: "old", Remediation: "old"},
	})
	if _, err := q.SyncFindingTemplatesFromFile(pathA); err != nil {
		t.Fatal(err)
	}

	// Upstream updates title and severity; row was never touched in UI.
	pathB := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git", Title: "v2", Severity: "critical", Summary: "new", Remediation: "new"},
	})
	res, err := q.SyncFindingTemplatesFromFile(pathB)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 || res.Preserved != 0 {
		t.Fatalf("expected one update, got %+v", res)
	}
	rows, _ := q.ListBuiltinFindingTemplates()
	if rows[0].Title != "v2" || rows[0].Severity != "critical" {
		t.Fatalf("upstream content not adopted: %+v", rows[0])
	}
}

func TestSyncFindingTemplates_PreservesUserModified(t *testing.T) {
	q := New(openTestDB(t))
	path := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git", Title: "v1", Summary: "old", Remediation: "old"},
	})
	if _, err := q.SyncFindingTemplatesFromFile(path); err != nil {
		t.Fatal(err)
	}

	// User edits the row in UI.
	rows, _ := q.ListBuiltinFindingTemplates()
	row := rows[0]
	row.Title = "local edit"
	row.Summary = "user content"
	row.UserModified = true
	if err := q.UpdateFindingTemplate(row); err != nil {
		t.Fatal(err)
	}

	// Upstream pushes new version.
	pathB := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git", Title: "v2", Summary: "new", Remediation: "new"},
	})
	res, err := q.SyncFindingTemplatesFromFile(pathB)
	if err != nil {
		t.Fatal(err)
	}
	if res.Preserved != 1 || res.Updated != 0 {
		t.Fatalf("expected preservation, got %+v", res)
	}

	got, _ := q.ListBuiltinFindingTemplates()
	if got[0].Title != "local edit" || got[0].Summary != "user content" {
		t.Fatalf("user edits were overwritten: %+v", got[0])
	}
	// builtin_payload should reflect the new upstream version so UI can offer "accept upstream".
	var seed SeedFindingTemplate
	if err := json.Unmarshal([]byte(got[0].BuiltinPayload), &seed); err != nil {
		t.Fatal(err)
	}
	if seed.Title != "v2" || seed.Summary != "new" {
		t.Fatalf("builtin_payload not refreshed: %+v", seed)
	}
}

func TestSyncFindingTemplates_DeletesUntouchedWhenUpstreamRemoved(t *testing.T) {
	q := New(openTestDB(t))
	pathA := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git"},
		{SourceTool: "nuclei", MatchKey: "csrf-missing"},
	})
	if _, err := q.SyncFindingTemplatesFromFile(pathA); err != nil {
		t.Fatal(err)
	}

	// Upstream drops csrf-missing.
	pathB := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git"},
	})
	res, err := q.SyncFindingTemplatesFromFile(pathB)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 1 {
		t.Fatalf("expected deletion, got %+v", res)
	}
	rows, _ := q.ListBuiltinFindingTemplates()
	if len(rows) != 1 || rows[0].MatchKey != "exposed-git" {
		t.Fatalf("unexpected remaining rows: %+v", rows)
	}
}

func TestSyncFindingTemplates_KeepsUserModifiedWhenUpstreamRemoved(t *testing.T) {
	q := New(openTestDB(t))
	pathA := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "csrf-missing", Title: "v1"},
	})
	if _, err := q.SyncFindingTemplatesFromFile(pathA); err != nil {
		t.Fatal(err)
	}
	rows, _ := q.ListBuiltinFindingTemplates()
	row := rows[0]
	row.UserModified = true
	row.Title = "kept by user"
	if err := q.UpdateFindingTemplate(row); err != nil {
		t.Fatal(err)
	}

	// Upstream removes the entry.
	pathB := writeSeedFile(t, []SeedFindingTemplate{})
	res, err := q.SyncFindingTemplatesFromFile(pathB)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 1 || res.Deleted != 0 {
		t.Fatalf("expected to skip the user-modified row, got %+v", res)
	}
	got, _ := q.ListBuiltinFindingTemplates()
	if len(got) != 1 || got[0].Title != "kept by user" {
		t.Fatalf("user-modified row was wrongly removed: %+v", got)
	}
}

func TestSyncFindingTemplates_LeavesUserOwnedRowsAlone(t *testing.T) {
	q := New(openTestDB(t))
	// User creates a non-builtin entry.
	userOwned := makeTpl("user-1", "sqlmap", "boolean-blind", true)
	if err := q.CreateFindingTemplate(userOwned); err != nil {
		t.Fatal(err)
	}

	pathA := writeSeedFile(t, []SeedFindingTemplate{
		{SourceTool: "nuclei", MatchKey: "exposed-git"},
	})
	if _, err := q.SyncFindingTemplatesFromFile(pathA); err != nil {
		t.Fatal(err)
	}

	// User template must survive — sync only operates on builtin rows.
	all, _ := q.ListFindingTemplates("")
	if len(all) != 2 {
		t.Fatalf("expected 2 rows (user + builtin), got %d", len(all))
	}
	var found bool
	for _, t := range all {
		if t.ID == "user-1" {
			found = true
			if t.IsBuiltin {
				t2 := t // avoid shadow warning
				_ = t2
				t2.IsBuiltin = false
			}
		}
	}
	if !found {
		t.Fatal("user-owned row went missing")
	}
}

func TestSyncFindingTemplates_MissingFileNoOp(t *testing.T) {
	q := New(openTestDB(t))
	res, err := q.SyncFindingTemplatesFromFile("/does/not/exist.json")
	if err != nil {
		t.Fatal(err)
	}
	if res.Inserted != 0 || res.Updated != 0 || res.Deleted != 0 {
		t.Fatalf("missing file should be no-op, got %+v", res)
	}
}

func TestSyncFindingTemplates_EmptyPathNoOp(t *testing.T) {
	q := New(openTestDB(t))
	res, err := q.SyncFindingTemplatesFromFile("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Inserted != 0 {
		t.Fatalf("empty path should be no-op, got %+v", res)
	}
}

// silence unused import warnings when models is referenced indirectly.
var _ = models.FindingTemplate{}

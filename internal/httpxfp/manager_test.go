package httpxfp

import (
	"bytes"
	"context"
	"database/sql"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { rawDB.Close() })
	if err := db.Migrate(rawDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return rawDB
}

func setupManager(t *testing.T) (*Manager, *sql.DB) {
	t.Helper()
	rawDB := openTestDB(t)
	q := db.New(rawDB)
	dataDir := t.TempDir()
	m := NewManager(q, dataDir)
	if err := m.EnsureLayout(); err != nil {
		t.Fatalf("EnsureLayout: %v", err)
	}
	return m, rawDB
}

func TestLayout_FilePath(t *testing.T) {
	l := NewLayout("/data")
	got := l.FilePath("fp-1")
	if got != "/data/httpx/fingerprints/fp-1.json" {
		t.Errorf("FilePath = %q, want /data/httpx/fingerprints/fp-1.json", got)
	}
}

func TestManager_CreateAndRead(t *testing.T) {
	m, _ := setupManager(t)

	content := []byte(`{"matchers":["favicon-hash"]}`)
	f, err := m.Create("test-fp", "test fingerprint", models.HttpxFingerprintTypeFavicon, content)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if f.Name != "test-fp" {
		t.Errorf("name = %q, want test-fp", f.Name)
	}
	if f.Type != models.HttpxFingerprintTypeFavicon {
		t.Errorf("type = %q, want favicon", f.Type)
	}
	if !f.Enabled {
		t.Error("expected enabled=true")
	}

	got, err := m.ReadContent(f.ID)
	if err != nil {
		t.Fatalf("ReadContent: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestManager_List(t *testing.T) {
	m, _ := setupManager(t)

	m.Create("fp1", "", models.HttpxFingerprintTypeFavicon, []byte("a"))
	m.Create("fp2", "", models.HttpxFingerprintTypeTechDetect, []byte("b"))
	m.Create("fp3", "", models.HttpxFingerprintTypeFavicon, []byte("c"))

	// List all
	all, err := m.List("")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all len = %d, want 3", len(all))
	}

	// List by type
	favs, err := m.List("favicon")
	if err != nil {
		t.Fatalf("List favicon: %v", err)
	}
	if len(favs) != 2 {
		t.Errorf("favicon len = %d, want 2", len(favs))
	}
}

func TestManager_ListEnabled(t *testing.T) {
	m, _ := setupManager(t)

	f1, _ := m.Create("fp1", "", models.HttpxFingerprintTypeFavicon, []byte("a"))
	m.Create("fp2", "", models.HttpxFingerprintTypeFavicon, []byte("b"))

	// Disable fp1
	m.Update(f1.ID, "fp1", "", models.HttpxFingerprintTypeFavicon, false)

	enabled, err := m.ListEnabled("favicon")
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(enabled) != 1 {
		t.Errorf("enabled len = %d, want 1", len(enabled))
	}
}

func TestManager_Get(t *testing.T) {
	m, _ := setupManager(t)

	f, _ := m.Create("test", "desc", models.HttpxFingerprintTypeTechDetect, []byte("x"))

	got, err := m.Get(f.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("name = %q, want test", got.Name)
	}
}

func TestManager_Update(t *testing.T) {
	m, _ := setupManager(t)

	f, _ := m.Create("old", "old desc", models.HttpxFingerprintTypeFavicon, []byte("x"))

	updated, err := m.Update(f.ID, "new", "new desc", models.HttpxFingerprintTypeFavicon, false)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "new" {
		t.Errorf("name = %q, want new", updated.Name)
	}
	if updated.Enabled {
		t.Error("expected enabled=false")
	}
}

func TestManager_UpdateContent(t *testing.T) {
	m, _ := setupManager(t)

	f, _ := m.Create("test", "", models.HttpxFingerprintTypeFavicon, []byte("old"))

	newContent := []byte(`{"new":"data"}`)
	updated, err := m.UpdateContent(f.ID, newContent)
	if err != nil {
		t.Fatalf("UpdateContent: %v", err)
	}

	got, _ := m.ReadContent(updated.ID)
	if !bytes.Equal(got, newContent) {
		t.Errorf("content mismatch")
	}
}

func TestManager_Delete(t *testing.T) {
	m, _ := setupManager(t)

	f, _ := m.Create("test", "", models.HttpxFingerprintTypeFavicon, []byte("x"))

	if err := m.Delete(context.Background(), f.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, _ := m.Get(f.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestManager_Update_NotFound(t *testing.T) {
	m, _ := setupManager(t)

	_, err := m.Update("nonexistent", "x", "y", models.HttpxFingerprintTypeFavicon, true)
	if err == nil {
		t.Error("expected error for non-existent fingerprint")
	}
}

func TestManager_UpdateContent_NotFound(t *testing.T) {
	m, _ := setupManager(t)

	_, err := m.UpdateContent("nonexistent", []byte("x"))
	if err == nil {
		t.Error("expected error for non-existent fingerprint")
	}
}

func TestManager_ReadContent_NotFound(t *testing.T) {
	m, _ := setupManager(t)

	_, err := m.ReadContent("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent fingerprint")
	}
}

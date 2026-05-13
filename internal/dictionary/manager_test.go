package dictionary

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

func TestCountLines(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want int
	}{
		{"empty", []byte{}, 0},
		{"single line no newline", []byte("hello"), 1},
		{"single line with newline", []byte("hello\n"), 1},
		{"two lines", []byte("a\nb\n"), 2},
		{"two lines no trailing", []byte("a\nb"), 2},
		{"three lines mixed", []byte("a\nb\nc"), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLines(tt.data)
			if got != tt.want {
				t.Errorf("countLines(%q) = %d, want %d", tt.data, got, tt.want)
			}
		})
	}
}

func TestLayout_FilePath(t *testing.T) {
	l := NewLayout("/data")
	got := l.FilePath("abc-123")
	if got != "/data/dictionaries/abc-123.txt" {
		t.Errorf("FilePath = %q, want /data/dictionaries/abc-123.txt", got)
	}
}

func TestManager_CreateAndRead(t *testing.T) {
	m, _ := setupManager(t)
	ctx := context.Background()
	_ = ctx

	content := []byte("admin\nroot\ntest\n")
	d, err := m.Create("common", "common paths", models.DictionaryCategoryDirscan, content)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if d.Name != "common" {
		t.Errorf("name = %q, want common", d.Name)
	}
	if d.Category != models.DictionaryCategoryDirscan {
		t.Errorf("category = %q, want dirscan", d.Category)
	}
	if d.LineCount != 3 {
		t.Errorf("line_count = %d, want 3", d.LineCount)
	}
	if d.SizeBytes != int64(len(content)) {
		t.Errorf("size_bytes = %d, want %d", d.SizeBytes, len(content))
	}

	got, err := m.ReadContent(d.ID)
	if err != nil {
		t.Fatalf("ReadContent: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestManager_List(t *testing.T) {
	m, _ := setupManager(t)

	m.Create("d1", "", models.DictionaryCategoryDirscan, []byte("a\n"))
	m.Create("d2", "", models.DictionaryCategorySubdomain, []byte("b\n"))
	m.Create("d3", "", models.DictionaryCategoryDirscan, []byte("c\n"))

	// List all
	all, err := m.List("")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all len = %d, want 3", len(all))
	}

	// List by category
	dirscan, err := m.List("dirscan")
	if err != nil {
		t.Fatalf("List dirscan: %v", err)
	}
	if len(dirscan) != 2 {
		t.Errorf("dirscan len = %d, want 2", len(dirscan))
	}
}

func TestManager_Get(t *testing.T) {
	m, _ := setupManager(t)

	d, _ := m.Create("test", "desc", models.DictionaryCategoryCustom, []byte("x\n"))

	got, err := m.Get(d.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("name = %q, want test", got.Name)
	}
}

func TestManager_Update(t *testing.T) {
	m, _ := setupManager(t)

	d, _ := m.Create("old", "old desc", models.DictionaryCategoryDirscan, []byte("x\n"))

	updated, err := m.Update(d.ID, "new", "new desc", models.DictionaryCategorySubdomain)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "new" {
		t.Errorf("name = %q, want new", updated.Name)
	}
	if updated.Category != models.DictionaryCategorySubdomain {
		t.Errorf("category = %q, want subdomain", updated.Category)
	}
}

func TestManager_UpdateContent(t *testing.T) {
	m, _ := setupManager(t)

	d, _ := m.Create("test", "", models.DictionaryCategoryDirscan, []byte("old\n"))

	newContent := []byte("new1\nnew2\nnew3\n")
	updated, err := m.UpdateContent(d.ID, newContent)
	if err != nil {
		t.Fatalf("UpdateContent: %v", err)
	}
	if updated.LineCount != 3 {
		t.Errorf("line_count = %d, want 3", updated.LineCount)
	}

	got, _ := m.ReadContent(d.ID)
	if !bytes.Equal(got, newContent) {
		t.Errorf("content mismatch")
	}
}

func TestManager_Delete(t *testing.T) {
	m, _ := setupManager(t)

	d, _ := m.Create("test", "", models.DictionaryCategoryDirscan, []byte("x\n"))

	if err := m.Delete(context.Background(), d.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, _ := m.Get(d.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestManager_Update_NotFound(t *testing.T) {
	m, _ := setupManager(t)

	_, err := m.Update("nonexistent", "x", "y", models.DictionaryCategoryDirscan)
	if err == nil {
		t.Error("expected error for non-existent dictionary")
	}
}

func TestManager_UpdateContent_NotFound(t *testing.T) {
	m, _ := setupManager(t)

	_, err := m.UpdateContent("nonexistent", []byte("x"))
	if err == nil {
		t.Error("expected error for non-existent dictionary")
	}
}

package dictionary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestSeedBuiltin_PreservesDisabledOnReseed(t *testing.T) {
	m, _ := setupManager(t)

	root := t.TempDir()
	dictFile := filepath.Join(root, "path", "common.txt")
	if err := os.MkdirAll(filepath.Dir(dictFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dictFile, []byte("admin\nroot\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := m.SeedBuiltin(root); err != nil {
		t.Fatalf("SeedBuiltin: %v", err)
	}

	id := "builtin:path/common.txt"
	got, err := m.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected seeded dictionary")
	}
	if !got.Enabled {
		t.Fatal("new builtin dictionary should be enabled")
	}
	if !got.Builtin {
		t.Fatal("expected builtin flag")
	}

	if _, err := m.UpdateEnabled(id, false); err != nil {
		t.Fatalf("UpdateEnabled: %v", err)
	}

	if err := m.SeedBuiltin(root); err != nil {
		t.Fatalf("SeedBuiltin re-seed: %v", err)
	}

	after, err := m.Get(id)
	if err != nil {
		t.Fatalf("Get after re-seed: %v", err)
	}
	if after.Enabled {
		t.Error("re-seed should preserve user-disabled enabled=false")
	}
	if after.LineCount != 2 {
		t.Errorf("line_count = %d, want 2 after content refresh", after.LineCount)
	}
}

func TestManager_ListEnabled(t *testing.T) {
	m, _ := setupManager(t)

	root := t.TempDir()
	onPath := filepath.Join(root, "path", "on.txt")
	offPath := filepath.Join(root, "path", "off.txt")
	for _, p := range []string{onPath, offPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.SeedBuiltin(root); err != nil {
		t.Fatalf("SeedBuiltin: %v", err)
	}
	if _, err := m.UpdateEnabled("builtin:path/off.txt", false); err != nil {
		t.Fatalf("UpdateEnabled: %v", err)
	}

	enabled, err := m.ListEnabled("dirscan")
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("ListEnabled len = %d, want 1", len(enabled))
	}
	if enabled[0].ID != "builtin:path/on.txt" {
		t.Errorf("id = %q, want builtin:path/on.txt", enabled[0].ID)
	}

	m.Create("user", "", models.DictionaryCategoryDirscan, []byte("u\n"))
	allEnabled, err := m.ListEnabled("dirscan")
	if err != nil {
		t.Fatalf("ListEnabled after user create: %v", err)
	}
	if len(allEnabled) != 2 {
		t.Fatalf("ListEnabled len = %d, want 2 (builtin on + user)", len(allEnabled))
	}
}

func TestManager_UpdateEnabled_NonBuiltinRejected(t *testing.T) {
	m, _ := setupManager(t)

	d, err := m.Create("user", "", models.DictionaryCategoryDirscan, []byte("x\n"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = m.UpdateEnabled(d.ID, false)
	if err == nil {
		t.Fatal("expected error toggling enabled on user dictionary")
	}
}

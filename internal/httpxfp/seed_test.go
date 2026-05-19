package httpxfp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestSeedBuiltin_InsertAndPreserveEnabled(t *testing.T) {
	m, _ := setupManager(t)

	root := t.TempDir()
	fpPath := filepath.Join(root, "finger.json")
	if err := os.WriteFile(fpPath, []byte(`[{"name":"test"}]`), 0o640); err != nil {
		t.Fatalf("write finger.json: %v", err)
	}
	if err := m.SeedBuiltin(root); err != nil {
		t.Fatalf("SeedBuiltin insert: %v", err)
	}

	got, err := m.Get(builtinHttpxID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected builtin row")
	}
	if !got.Builtin || !got.Enabled {
		t.Fatalf("builtin=%v enabled=%v, want both true", got.Builtin, got.Enabled)
	}
	if got.Type != models.HttpxFingerprintTypeTechDetect {
		t.Errorf("type = %q, want tech_detect", got.Type)
	}
	wantPath, _ := filepath.Abs(fpPath)
	if got.FilePath != wantPath {
		t.Errorf("file_path = %q, want %q", got.FilePath, wantPath)
	}
	if got.Description != "RBKD-SEC/finger" {
		t.Errorf("description = %q, want RBKD-SEC/finger", got.Description)
	}

	if _, err := m.UpdateEnabled(builtinHttpxID, false); err != nil {
		t.Fatalf("UpdateEnabled false: %v", err)
	}

	if err := m.SeedBuiltin(root); err != nil {
		t.Fatalf("SeedBuiltin re-seed: %v", err)
	}
	got, _ = m.Get(builtinHttpxID)
	if got.Enabled {
		t.Error("expected enabled=false preserved after re-seed")
	}
	if got.Description != "RBKD-SEC/finger" {
		t.Errorf("description after re-seed = %q", got.Description)
	}
}

func TestSeedBuiltin_MissingFileSkips(t *testing.T) {
	m, _ := setupManager(t)
	if err := m.SeedBuiltin(t.TempDir()); err != nil {
		t.Fatalf("SeedBuiltin missing: %v", err)
	}
	got, _ := m.Get(builtinHttpxID)
	if got != nil {
		t.Fatal("expected no row when finger.json missing")
	}
}

func TestManager_BuiltinReadOnly(t *testing.T) {
	m, _ := setupManager(t)
	root := t.TempDir()
	fpPath := filepath.Join(root, "finger.json")
	_ = os.WriteFile(fpPath, []byte("{}"), 0o640)
	if err := m.SeedBuiltin(root); err != nil {
		t.Fatalf("SeedBuiltin: %v", err)
	}

	if err := m.Delete(context.Background(), builtinHttpxID); err != ErrBuiltinReadOnly {
		t.Fatalf("Delete builtin: got %v, want ErrBuiltinReadOnly", err)
	}
	if _, err := m.UpdateContent(builtinHttpxID, []byte("x")); err != ErrBuiltinReadOnly {
		t.Fatalf("UpdateContent builtin: got %v, want ErrBuiltinReadOnly", err)
	}
	if _, err := m.Update(builtinHttpxID, "x", "", models.HttpxFingerprintTypeTechDetect, true); err != ErrBuiltinReadOnly {
		t.Fatalf("Update builtin: got %v, want ErrBuiltinReadOnly", err)
	}
}

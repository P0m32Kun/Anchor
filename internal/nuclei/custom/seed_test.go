package custom

import (
	"database/sql"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func newTestManager(t *testing.T) *Manager {
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
	return NewManager(db.New(rawDB))
}

func TestManager_SeedBuiltin_CreatesAndPreservesEnabled(t *testing.T) {
	m := newTestManager(t)

	if err := m.SeedBuiltin(); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := m.GetByID(builtinNucleiID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("builtin row missing")
	}
	if got.Name != "RBKD Templates" || got.InstallPath != "RBKD-templates" {
		t.Errorf("unexpected row: %+v", got)
	}
	if !got.Builtin || !got.Enabled {
		t.Errorf("want builtin+enabled, got builtin=%v enabled=%v", got.Builtin, got.Enabled)
	}

	if _, err := m.UpdateEnabled(builtinNucleiID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if err := m.SeedBuiltin(); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	got, err = m.GetByID(builtinNucleiID)
	if err != nil {
		t.Fatalf("get after re-seed: %v", err)
	}
	if got.Enabled {
		t.Error("re-seed should preserve user-disabled enabled=false")
	}
}

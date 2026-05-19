package custom

import (
	"context"
	"errors"
	"os"
	"testing"

	apperrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestManager_SeedBuiltin_CreatesAndPreservesEnabled(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})

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
	if got.URI == nil || *got.URI != "https://github.com/RBKD-SEC/RBKD-templates" {
		t.Errorf("uri: %v", got.URI)
	}
	if got.Branch == nil || *got.Branch != "main" {
		t.Errorf("branch: %v", got.Branch)
	}
	if got.Status != models.NucleiCustomSourceStatusReady {
		t.Errorf("status: %q", got.Status)
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

func TestManager_SeedBuiltin_NoLayoutDir(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	if err := m.SeedBuiltin(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	srcDir := m.Layout().SourceDir(builtinNucleiID)
	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Fatalf("expected no on-disk source dir at %s, stat err=%v", srcDir, err)
	}
}

func TestManager_BuiltinSource_ReadOnly(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	if err := m.SeedBuiltin(); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := m.Delete(context.Background(), builtinNucleiID)
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrForbidden {
		t.Fatalf("delete builtin: want forbidden, got %v", err)
	}

	_, err = m.Refresh(context.Background(), builtinNucleiID)
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrForbidden {
		t.Fatalf("refresh builtin: want forbidden, got %v", err)
	}

	_, err = m.ListFiles(builtinNucleiID)
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrForbidden {
		t.Fatalf("list files builtin: want forbidden, got %v", err)
	}
}

func TestManager_SeedBuiltin_DisablesDuplicateInstallPath(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})

	dup, err := m.CreateFromGit(context.Background(), "legacy", "RBKD-templates", "https://example.com/x.git", "main", "manual")
	if err != nil {
		t.Fatalf("create duplicate: %v", err)
	}
	if err := m.SeedBuiltin(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := m.GetByID(dup.ID)
	if err != nil {
		t.Fatalf("get duplicate: %v", err)
	}
	if got.Enabled {
		t.Error("duplicate install_path source should be disabled after builtin seed")
	}
}

func TestManager_BuildBundle_SkipsBuiltin(t *testing.T) {
	cloner := &fakeCloner{files: map[string]string{"templates/a.yaml": "id: a\n"}}
	m, _ := newTestManager(t, cloner)

	if err := m.SeedBuiltin(); err != nil {
		t.Fatalf("seed builtin: %v", err)
	}
	if _, err := m.CreateFromGit(context.Background(), "user", "user", "https://example.com/x.git", "main", "manual"); err != nil {
		t.Fatalf("create user source: %v", err)
	}

	version, _, err := m.BuildBundle()
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}
	if version == "" {
		t.Fatal("expected bundle version")
	}

	manifest, err := m.GetBundleManifest(version)
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	for _, e := range manifest.Sources {
		if e.ID == builtinNucleiID {
			t.Fatal("builtin source must not appear in bundle manifest")
		}
	}
}

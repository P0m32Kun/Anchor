package custom

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/db"
	apperrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// fakeCloner writes a hard-coded set of files into dest to simulate a git clone
// without shelling out. Each entry is a relative path → contents map.
type fakeCloner struct {
	files map[string]string
	err   error
	calls int
	mu    sync.Mutex
}

func (f *fakeCloner) Clone(_ context.Context, _ string, _ string, dest string) error {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for rel, body := range f.files {
		full := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func newTestManager(t *testing.T, cloner Cloner) (*Manager, string) {
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
	dataDir := t.TempDir()
	m := NewManager(db.New(rawDB), rawDB, dataDir, cloner)
	if err := m.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	return m, dataDir
}

func TestManager_CreateFromGit_Happy(t *testing.T) {
	cloner := &fakeCloner{files: map[string]string{
		"templates/a.yaml": "id: a\n",
		"templates/b.yml":  "id: b\n",
	}}
	m, _ := newTestManager(t, cloner)

	src, err := m.CreateFromGit(context.Background(), "demo", "https://example.com/x.git", "main", "manual")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if src.Status != models.NucleiCustomSourceStatusReady {
		t.Errorf("status: want ready, got %q", src.Status)
	}
	if src.Type != models.NucleiCustomSourceTypeGit {
		t.Errorf("type: want git, got %q", src.Type)
	}
	if src.URI == nil || *src.URI != "https://example.com/x.git" {
		t.Errorf("uri: %v", src.URI)
	}
	if src.LastSyncAt == nil {
		t.Error("last_sync_at: should be set")
	}

	files, err := m.ListFiles(src.ID)
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("file count: want 2, got %d", len(files))
	}
}

func TestManager_CreateFromGit_RollbackOnCloneFailure(t *testing.T) {
	cloner := &fakeCloner{err: errors.New("clone refused")}
	m, _ := newTestManager(t, cloner)

	_, err := m.CreateFromGit(context.Background(), "demo", "https://example.com/x.git", "", "manual")
	if err == nil {
		t.Fatal("expected error")
	}
	list, err := m.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("source rollback failed: %d rows remain", len(list))
	}
}

func TestManager_CreateFromGit_RejectsNonHTTPS(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	_, err := m.CreateFromGit(context.Background(), "demo", "git@github.com:foo/bar.git", "", "manual")
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrBadRequest {
		t.Errorf("want BadRequest AppError, got %v", err)
	}
}

func TestManager_CreateFromUpload_YAML(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	body := bytes.NewReader([]byte("id: hello\n"))
	src, err := m.CreateFromUpload(context.Background(), "demo", "manual", "hello.yaml", body)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if src.Status != models.NucleiCustomSourceStatusReady {
		t.Errorf("status: %q", src.Status)
	}
	files, err := m.ListFiles(src.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(files) != 1 || files[0].Path != "templates/hello.yaml" {
		t.Errorf("files: %+v", files)
	}
}

func TestManager_CreateFromUpload_ZipWithTraversalEntryRejected(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h, err := zw.Create("../escape.yaml")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := h.Write([]byte("id: x\n")); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	_, err = m.CreateFromUpload(context.Background(), "demo", "manual", "evil.zip", bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for traversal entry")
	}
	if list, _ := m.List(); len(list) != 0 {
		t.Errorf("rollback failed: %d sources remain", len(list))
	}
}

func TestManager_Patch_NameAndEnabled(t *testing.T) {
	cloner := &fakeCloner{files: map[string]string{"templates/a.yaml": "id: a\n"}}
	m, _ := newTestManager(t, cloner)
	src, err := m.CreateFromGit(context.Background(), "demo", "https://example.com/x.git", "", "manual")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newName := "renamed"
	newEnabled := false
	updated, err := m.Patch(src.ID, SourcePatch{Name: &newName, Enabled: &newEnabled})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if updated.Name != "renamed" || updated.Enabled {
		t.Errorf("patch did not apply: %+v", updated)
	}
}

func TestManager_Patch_DoesNotTouchTypeOrURI(t *testing.T) {
	// SourcePatch struct has no Type/URI fields, so even forging cannot mutate them.
	// This test confirms that a no-op patch leaves immutable fields untouched.
	cloner := &fakeCloner{files: map[string]string{"templates/a.yaml": "id: a\n"}}
	m, _ := newTestManager(t, cloner)
	src, _ := m.CreateFromGit(context.Background(), "demo", "https://example.com/x.git", "main", "manual")

	updated, err := m.Patch(src.ID, SourcePatch{})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if updated.Type != models.NucleiCustomSourceTypeGit {
		t.Errorf("type changed: %q", updated.Type)
	}
	if updated.URI == nil || *updated.URI != "https://example.com/x.git" {
		t.Errorf("uri changed: %v", updated.URI)
	}
}

func TestManager_Delete_RemovesRowAndDisk(t *testing.T) {
	cloner := &fakeCloner{files: map[string]string{"templates/a.yaml": "id: a\n"}}
	m, _ := newTestManager(t, cloner)
	src, _ := m.CreateFromGit(context.Background(), "demo", "https://example.com/x.git", "", "manual")

	dir := m.Layout().SourceDir(src.ID)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("source dir should exist: %v", err)
	}

	if err := m.Delete(context.Background(), src.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if list, _ := m.List(); len(list) != 0 {
		t.Errorf("rows remain: %d", len(list))
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("source dir should be gone, got err=%v", err)
	}
}

func TestManager_FileCRUD_RoundTrip(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	src, err := m.CreateFromUpload(context.Background(), "demo", "manual", "hello.yaml", bytes.NewReader([]byte("id: hello\n")))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Write a new template.
	newBody := []byte("id: extra\n")
	if err := m.WriteFile(src.ID, "templates/extra.yaml", newBody); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := m.ReadFile(src.ID, "templates/extra.yaml")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, newBody) {
		t.Errorf("read mismatch: %q", got)
	}

	files, err := m.ListFiles(src.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	wantPaths := map[string]bool{
		"templates/hello.yaml": false,
		"templates/extra.yaml": false,
	}
	for _, f := range files {
		if _, ok := wantPaths[f.Path]; ok {
			wantPaths[f.Path] = true
		}
	}
	for p, seen := range wantPaths {
		if !seen {
			t.Errorf("missing file %q", p)
		}
	}

	if err := m.DeleteFile(src.ID, "templates/extra.yaml"); err != nil {
		t.Fatalf("delete file: %v", err)
	}
	if _, err := m.ReadFile(src.ID, "templates/extra.yaml"); err == nil {
		t.Error("read after delete: expected error")
	}
}

func TestManager_WriteFile_RejectsDisallowedExtension(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	src, _ := m.CreateFromUpload(context.Background(), "demo", "manual", "hello.yaml", bytes.NewReader([]byte("id: x\n")))

	err := m.WriteFile(src.ID, "templates/evil.sh", []byte("#!/bin/sh\nrm -rf /\n"))
	if err == nil {
		t.Fatal("expected rejection")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrValidation {
		t.Errorf("want Validation AppError, got %v", err)
	}
}

func TestManager_WriteFile_RejectsTraversal(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	src, _ := m.CreateFromUpload(context.Background(), "demo", "manual", "hello.yaml", bytes.NewReader([]byte("id: x\n")))

	err := m.WriteFile(src.ID, "../escape.yaml", []byte("id: x\n"))
	if err == nil {
		t.Fatal("expected rejection")
	}
}

func TestManager_WriteFile_OversizeRejected(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	src, _ := m.CreateFromUpload(context.Background(), "demo", "manual", "hello.yaml", bytes.NewReader([]byte("id: x\n")))

	big := make([]byte, MaxWriteFileBytes+1)
	err := m.WriteFile(src.ID, "templates/huge.yaml", big)
	if err == nil {
		t.Fatal("expected size limit error")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrValidation {
		t.Errorf("want Validation AppError, got %v", err)
	}
}

func TestManager_WriteFile_ConcurrentWritersDontRace(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	src, _ := m.CreateFromUpload(context.Background(), "demo", "manual", "hello.yaml", bytes.NewReader([]byte("id: x\n")))

	const N = 16
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			rel := "templates/c.yaml"
			body := []byte("id: c-" + strings.Repeat("x", i+1) + "\n")
			if err := m.WriteFile(src.ID, rel, body); err != nil {
				t.Errorf("write %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	if _, err := m.ReadFile(src.ID, "templates/c.yaml"); err != nil {
		t.Fatalf("read after concurrent writes: %v", err)
	}
}

func TestManager_Refresh_OnlyValidForGitSources(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	src, _ := m.CreateFromUpload(context.Background(), "demo", "manual", "hello.yaml", bytes.NewReader([]byte("id: x\n")))

	_, err := m.Refresh(context.Background(), src.ID)
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrBadRequest {
		t.Errorf("want BadRequest AppError, got %v", err)
	}
}

func TestManager_Refresh_GitSwapsFilesAtomically(t *testing.T) {
	cloner := &fakeCloner{files: map[string]string{"templates/v1.yaml": "id: v1\n"}}
	m, _ := newTestManager(t, cloner)

	src, err := m.CreateFromGit(context.Background(), "demo", "https://example.com/x.git", "", "manual")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Second clone returns a different set of files.
	cloner.files = map[string]string{"templates/v2.yaml": "id: v2\n"}
	if _, err := m.Refresh(context.Background(), src.ID); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	files, err := m.ListFiles(src.ID)
	if err != nil {
		t.Fatalf("list after refresh: %v", err)
	}
	if len(files) != 1 || files[0].Path != "templates/v2.yaml" {
		t.Errorf("after refresh expected only v2.yaml, got %+v", files)
	}
}

func TestManager_GetByID_NotFound(t *testing.T) {
	m, _ := newTestManager(t, &fakeCloner{})
	_, err := m.GetByID("missing")
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.ErrNotFound {
		t.Errorf("want NotFound AppError, got %v", err)
	}
}

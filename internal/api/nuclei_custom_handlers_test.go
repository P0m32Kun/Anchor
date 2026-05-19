package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei/custom"
)

// nuclueCustomFakeCloner mirrors the custom-package fake but lives here so
// the test can inject it into the Server's manager. Each call writes the
// configured files into dest.
type nuclueCustomFakeCloner struct {
	files map[string]string
	mu    sync.Mutex
	calls int
}

func (f *nuclueCustomFakeCloner) Clone(_ context.Context, _ string, _ string, dest string) error {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
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

// nucleiCustomTestEnv ties together a Server, its mux, and the fake cloner so
// each test can drive realistic HTTP roundtrips.
type nucleiCustomTestEnv struct {
	server *Server
	mux    *http.ServeMux
	cloner *nuclueCustomFakeCloner
}

func newNucleiCustomTestEnv(t *testing.T, files map[string]string) *nucleiCustomTestEnv {
	t.Helper()
	server, _, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	cloner := &nuclueCustomFakeCloner{files: files}
	server.nucleiCustomMgr = custom.NewManager(server.queries, server.rawDB, server.dataDir, cloner)
	if err := server.nucleiCustomMgr.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	mux := http.NewServeMux()
	server.Register(mux)
	return &nucleiCustomTestEnv{server: server, mux: mux, cloner: cloner}
}

func countCustomSources(sources []*models.NucleiCustomSource) int {
	n := 0
	for _, s := range sources {
		if !s.Builtin {
			n++
		}
	}
	return n
}

func (e *nucleiCustomTestEnv) do(t *testing.T, method, path string, body io.Reader, contentType string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+e.server.apiToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, req)
	return w.Result()
}

func TestNucleiCustom_FullRoundTrip(t *testing.T) {
	env := newNucleiCustomTestEnv(t, map[string]string{
		"templates/seed.yaml": "id: seed\n",
	})

	// list — only team builtin from NewServer SeedBuiltin
	resp := env.do(t, http.MethodGet, "/nuclei/custom/sources", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("initial list status: %d", resp.StatusCode)
	}
	var initial []*models.NucleiCustomSource
	if err := json.NewDecoder(resp.Body).Decode(&initial); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resp.Body.Close()
	if countCustomSources(initial) != 0 {
		t.Errorf("initial custom sources want 0, got %d (total %d)", countCustomSources(initial), len(initial))
	}

	// create-git
	createBody, _ := json.Marshal(map[string]string{
		"name":           "demo",
		"install_path":  "demo",
		"uri":            "https://example.com/x.git",
		"branch":         "main",
		"routing_policy": "manual",
	})
	resp = env.do(t, http.MethodPost, "/nuclei/custom/sources/git", bytes.NewReader(createBody), "application/json")
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create git status: %d body=%s", resp.StatusCode, string(b))
	}
	var created models.NucleiCustomSource
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	resp.Body.Close()
	if created.Status != models.NucleiCustomSourceStatusReady {
		t.Errorf("status: %q", created.Status)
	}
	id := created.ID

	// list — 1
	resp = env.do(t, http.MethodGet, "/nuclei/custom/sources", nil, "")
	var afterCreate []*models.NucleiCustomSource
	json.NewDecoder(resp.Body).Decode(&afterCreate)
	resp.Body.Close()
	if countCustomSources(afterCreate) != 1 {
		t.Errorf("after-create custom sources want 1, got %d (total %d)", countCustomSources(afterCreate), len(afterCreate))
	}

	// list-files
	resp = env.do(t, http.MethodGet, "/nuclei/custom/sources/"+id+"/files", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("list files status: %d", resp.StatusCode)
	}
	var files []models.NucleiCustomFileEntry
	json.NewDecoder(resp.Body).Decode(&files)
	resp.Body.Close()
	if len(files) != 1 || files[0].Path != "templates/seed.yaml" {
		t.Errorf("files: %+v", files)
	}

	// write-file
	newBody := []byte("id: new\n")
	resp = env.do(t, http.MethodPut, "/nuclei/custom/sources/"+id+"/files/templates/new.yaml", bytes.NewReader(newBody), "application/octet-stream")
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("write status: %d body=%s", resp.StatusCode, string(b))
	}
	resp.Body.Close()

	// read-file
	resp = env.do(t, http.MethodGet, "/nuclei/custom/sources/"+id+"/files/templates/new.yaml", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("read status: %d", resp.StatusCode)
	}
	got, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Equal(got, newBody) {
		t.Errorf("read body mismatch: %q", got)
	}

	// delete-file
	resp = env.do(t, http.MethodDelete, "/nuclei/custom/sources/"+id+"/files/templates/new.yaml", nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete file status: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// patch
	patchBody, _ := json.Marshal(map[string]interface{}{"name": "renamed", "enabled": false})
	resp = env.do(t, http.MethodPatch, "/nuclei/custom/sources/"+id, bytes.NewReader(patchBody), "application/json")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("patch status: %d", resp.StatusCode)
	}
	var patched models.NucleiCustomSource
	json.NewDecoder(resp.Body).Decode(&patched)
	resp.Body.Close()
	if patched.Name != "renamed" || patched.Enabled {
		t.Errorf("patch did not apply: %+v", patched)
	}

	// delete-source
	resp = env.do(t, http.MethodDelete, "/nuclei/custom/sources/"+id, nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete source status: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// list — team builtin remains
	resp = env.do(t, http.MethodGet, "/nuclei/custom/sources", nil, "")
	var final []*models.NucleiCustomSource
	json.NewDecoder(resp.Body).Decode(&final)
	resp.Body.Close()
	if countCustomSources(final) != 0 {
		t.Errorf("final custom sources want 0, got %d (total %d)", countCustomSources(final), len(final))
	}
	foundBuiltin := false
	for _, s := range final {
		if s.Builtin {
			foundBuiltin = true
			break
		}
	}
	if !foundBuiltin {
		t.Errorf("final list missing team builtin source: %+v", final)
	}
}

func TestNucleiCustom_DisallowedExtensionRejected(t *testing.T) {
	env := newNucleiCustomTestEnv(t, map[string]string{
		"templates/seed.yaml": "id: seed\n",
	})

	createBody, _ := json.Marshal(map[string]string{
		"name":           "demo",
		"install_path":  "demo",
		"uri":            "https://example.com/x.git",
		"routing_policy": "manual",
	})
	resp := env.do(t, http.MethodPost, "/nuclei/custom/sources/git", bytes.NewReader(createBody), "application/json")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d", resp.StatusCode)
	}
	var created models.NucleiCustomSource
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	// Writing a .sh file is rejected by the extension policy at the Manager
	// layer; surfaces as 422.
	body := bytes.NewReader([]byte("#!/bin/sh\nrm -rf /\n"))
	resp = env.do(t, http.MethodPut, "/nuclei/custom/sources/"+created.ID+"/files/templates/evil.sh", body, "application/octet-stream")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("disallowed extension status: want 422, got %d body=%s", resp.StatusCode, string(b))
	}
	resp.Body.Close()
}

func TestNucleiCustom_PatchUnknownFieldRejected(t *testing.T) {
	env := newNucleiCustomTestEnv(t, map[string]string{
		"templates/seed.yaml": "id: seed\n",
	})
	createBody, _ := json.Marshal(map[string]string{
		"name":           "demo",
		"install_path":  "demo",
		"uri":            "https://example.com/x.git",
		"routing_policy": "manual",
	})
	resp := env.do(t, http.MethodPost, "/nuclei/custom/sources/git", bytes.NewReader(createBody), "application/json")
	var created models.NucleiCustomSource
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	patch, _ := json.Marshal(map[string]string{"uri": "https://attacker.example.com/evil.git"})
	resp = env.do(t, http.MethodPatch, "/nuclei/custom/sources/"+created.ID, bytes.NewReader(patch), "application/json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("patch with unknown field status: want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestNucleiCustom_RefreshNonGitReturns400(t *testing.T) {
	env := newNucleiCustomTestEnv(t, nil)

	// Create an upload source to test refresh rejection.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("name", "uploaded")
	_ = mw.WriteField("install_path", "uploaded")
	_ = mw.WriteField("routing_policy", "manual")
	fw, _ := mw.CreateFormFile("file", "x.yaml")
	_, _ = fw.Write([]byte("id: x\n"))
	_ = mw.Close()

	resp := env.do(t, http.MethodPost, "/nuclei/custom/sources/upload", &buf, mw.FormDataContentType())
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload create status: %d body=%s", resp.StatusCode, string(b))
	}
	var created models.NucleiCustomSource
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	resp = env.do(t, http.MethodPost, "/nuclei/custom/sources/"+created.ID+"/refresh", nil, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("refresh non-git status: want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/httpxfp"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- helpers ---

func createTestHttpxFingerprint(t *testing.T, server *Server, name string, fpType models.HttpxFingerprintType, builtin bool) *models.HttpxFingerprint {
	t.Helper()
	now := time.Now().UTC()
	fp := &models.HttpxFingerprint{
		ID:          util.GenerateID(),
		Name:        name,
		Description: "test fingerprint",
		Type:        fpType,
		Enabled:     true,
		Builtin:     builtin,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// Insert directly via rawDB since the manager's Create requires file content
	enabledInt := 0
	if fp.Enabled {
		enabledInt = 1
	}
	builtinInt := 0
	if builtin {
		builtinInt = 1
	}
	_, err := server.rawDB.Exec(`INSERT INTO httpx_fingerprints (id, name, description, type, file_path, enabled, builtin, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fp.ID, fp.Name, fp.Description, string(fp.Type), "/tmp/test_fp.yaml", enabledInt, builtinInt, fp.CreatedAt, fp.UpdatedAt)
	if err != nil {
		t.Fatalf("create httpx fingerprint: %v", err)
	}
	return fp
}

// --- handleListHttpxFingerprints ---

func TestHandleListHttpxFingerprints_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, false)
	createTestHttpxFingerprint(t, server, "apache", models.HttpxFingerprintTypeTechDetect, false)

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints", nil)
	w := httptest.NewRecorder()

	server.handleListHttpxFingerprints(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleListHttpxFingerprints_FilterByType(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, false)
	createTestHttpxFingerprint(t, server, "favicon", models.HttpxFingerprintTypeFavicon, false)

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints?type=tech", nil)
	w := httptest.NewRecorder()

	server.handleListHttpxFingerprints(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleListHttpxFingerprints_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.httpxFpMgr = nil

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints", nil)
	w := httptest.NewRecorder()

	server.handleListHttpxFingerprints(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

// --- handleGetHttpxFingerprint ---

func TestHandleGetHttpxFingerprint_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, false)

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints/"+fp.ID, nil)
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handleGetHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got models.HttpxFingerprint
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != fp.ID {
		t.Errorf("id = %q, want %q", got.ID, fp.ID)
	}
}

func TestHandleGetHttpxFingerprint_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleGetHttpxFingerprint_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.httpxFpMgr = nil

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	server.handleGetHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

// --- handleCreateHttpxFingerprint ---

func TestHandleCreateHttpxFingerprint_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("name", "custom-fp")
	writer.WriteField("type", "tech_detect")
	writer.WriteField("description", "my fingerprint")
	part, _ := writer.CreateFormFile("file", "fingerprint.yaml")
	part.Write([]byte("matchers:\n  - name: test\n"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/httpx-fingerprints", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	server.handleCreateHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := json.MarshalIndent(map[string]string{"body": w.Body.String()}, "", "  ")
		t.Fatalf("status = %d, want %d; body=%s", resp.StatusCode, http.StatusCreated, body)
	}
}

func TestHandleCreateHttpxFingerprint_MissingName(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("type", "tech_detect")
	part, _ := writer.CreateFormFile("file", "fingerprint.yaml")
	part.Write([]byte("matchers:\n  - name: test\n"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/httpx-fingerprints", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	server.handleCreateHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateHttpxFingerprint_MissingType(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("name", "custom-fp")
	part, _ := writer.CreateFormFile("file", "fingerprint.yaml")
	part.Write([]byte("matchers:\n  - name: test\n"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/httpx-fingerprints", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	server.handleCreateHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateHttpxFingerprint_MissingFile(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("name", "custom-fp")
	writer.WriteField("type", "tech_detect")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/httpx-fingerprints", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	server.handleCreateHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCreateHttpxFingerprint_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.httpxFpMgr = nil

	req := httptest.NewRequest(http.MethodPost, "/httpx-fingerprints", nil)
	w := httptest.NewRecorder()

	server.handleCreateHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

// --- handlePatchHttpxFingerprint ---

func TestHandlePatchHttpxFingerprint_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, false)

	newName := "updated-nginx"
	body, _ := json.Marshal(map[string]interface{}{"name": newName})

	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/"+fp.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.HttpxFingerprint
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Name != "updated-nginx" {
		t.Errorf("name = %q, want updated-nginx", updated.Name)
	}
}

func TestHandlePatchHttpxFingerprint_BuiltinReadOnly(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "builtin-fp", models.HttpxFingerprintTypeTechDetect, true)

	newName := "hacked"
	body, _ := json.Marshal(map[string]interface{}{"name": newName})

	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/"+fp.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestHandlePatchHttpxFingerprint_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandlePatchHttpxFingerprint_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, false)

	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/"+fp.ID, bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchHttpxFingerprint_ToggleEnabled(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, false)

	body, _ := json.Marshal(map[string]interface{}{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/"+fp.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.HttpxFingerprint
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Enabled {
		t.Error("expected enabled=false")
	}
}

// --- handlePatchHttpxFingerprintEnabled ---

func TestHandlePatchHttpxFingerprintEnabled_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, true)

	body, _ := json.Marshal(map[string]interface{}{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/"+fp.ID+"/enabled", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprintEnabled(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandlePatchHttpxFingerprintEnabled_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/httpx-fingerprints/abc/enabled", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	server.handlePatchHttpxFingerprintEnabled(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// --- handleDeleteHttpxFingerprint ---

func TestHandleDeleteHttpxFingerprint_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "nginx", models.HttpxFingerprintTypeTechDetect, false)

	req := httptest.NewRequest(http.MethodDelete, "/httpx-fingerprints/"+fp.ID, nil)
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handleDeleteHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleDeleteHttpxFingerprint_BuiltinReadOnly(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "builtin-fp", models.HttpxFingerprintTypeTechDetect, true)

	req := httptest.NewRequest(http.MethodDelete, "/httpx-fingerprints/"+fp.ID, nil)
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handleDeleteHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestHandleDeleteHttpxFingerprint_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/httpx-fingerprints/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleDeleteHttpxFingerprint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- handleReadHttpxFingerprintContent ---

func TestHandleReadHttpxFingerprintContent_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Use the manager to create a fingerprint with actual content
	fp, err := server.httpxFpMgr.Create("test-fp", "test desc", models.HttpxFingerprintTypeTechDetect, []byte("test content"))
	if err != nil {
		t.Fatalf("create fp via manager: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints/"+fp.ID+"/content", nil)
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handleReadHttpxFingerprintContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
}

func TestHandleReadHttpxFingerprintContent_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.httpxFpMgr = nil

	req := httptest.NewRequest(http.MethodGet, "/httpx-fingerprints/abc/content", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	server.handleReadHttpxFingerprintContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

// --- handleWriteHttpxFingerprintContent ---

func TestHandleWriteHttpxFingerprintContent_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create via manager first
	fp, err := server.httpxFpMgr.Create("test-fp", "test desc", models.HttpxFingerprintTypeTechDetect, []byte("original"))
	if err != nil {
		t.Fatalf("create fp: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/httpx-fingerprints/"+fp.ID+"/content", bytes.NewReader([]byte("updated content")))
	req.Header.Set("Content-Type", "text/plain")
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handleWriteHttpxFingerprintContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleWriteHttpxFingerprintContent_BuiltinReadOnly(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "builtin-fp", models.HttpxFingerprintTypeTechDetect, true)

	req := httptest.NewRequest(http.MethodPut, "/httpx-fingerprints/"+fp.ID+"/content", bytes.NewReader([]byte("hacked")))
	req.Header.Set("Content-Type", "text/plain")
	req.SetPathValue("id", fp.ID)
	w := httptest.NewRecorder()

	server.handleWriteHttpxFingerprintContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestHandleWriteHttpxFingerprintContent_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	server.httpxFpMgr = nil

	req := httptest.NewRequest(http.MethodPut, "/httpx-fingerprints/abc/content", bytes.NewReader([]byte("x")))
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	server.handleWriteHttpxFingerprintContent(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

// --- requireUserHttpxFingerprint ---

func TestRequireUserHttpxFingerprint_BuiltinBlocked(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "builtin-fp", models.HttpxFingerprintTypeTechDetect, true)

	w := httptest.NewRecorder()
	result := server.requireUserHttpxFingerprint(w, fp.ID)

	if result {
		t.Error("expected false for builtin fingerprint")
	}
}

func TestRequireUserHttpxFingerprint_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	result := server.requireUserHttpxFingerprint(w, "nonexistent")

	if result {
		t.Error("expected false for nonexistent fingerprint")
	}
}

func TestRequireUserHttpxFingerprint_UserCreated(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	fp := createTestHttpxFingerprint(t, server, "user-fp", models.HttpxFingerprintTypeTechDetect, false)

	w := httptest.NewRecorder()
	result := server.requireUserHttpxFingerprint(w, fp.ID)

	if !result {
		t.Error("expected true for user-created fingerprint")
	}
}

// --- writeHttpxFpMgrError ---

func TestWriteHttpxFpMgrError_BuiltinReadOnly(t *testing.T) {
	w := httptest.NewRecorder()
	writeHttpxFpMgrError(w, httpxfp.ErrBuiltinReadOnly, "test")

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestWriteHttpxFpMgrError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeHttpxFpMgrError(w, httpxfp.ErrNotBuiltin, "test")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWriteHttpxFpMgrError_GenericNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeHttpxFpMgrError(w, httpxfp.ErrNotBuiltin, "test")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

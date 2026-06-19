package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- helpers ---

func createTestDictionary(t *testing.T, server *Server, name string, category models.DictionaryCategory) *models.Dictionary {
	t.Helper()
	d, err := server.dictMgr.Create(name, "test desc", category, []byte("line1\nline2\n"))
	if err != nil {
		t.Fatalf("create dictionary: %v", err)
	}
	return d
}

func buildMultipartDict(name, category, description, fileContent string) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("name", name)
	_ = writer.WriteField("category", category)
	_ = writer.WriteField("description", description)
	part, _ := writer.CreateFormFile("file", "test.txt")
	_, _ = part.Write([]byte(fileContent))
	writer.Close()
	return &buf, writer.FormDataContentType()
}

// --- handleListDictionaries ---

func TestHandleListDictionaries_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	createTestDictionary(t, server, "dict1", models.DictionaryCategoryDirscan)
	createTestDictionary(t, server, "dict2", models.DictionaryCategorySubdomain)

	req := httptest.NewRequest(http.MethodGet, "/dictionaries", nil)
	w := httptest.NewRecorder()

	server.handleListDictionaries(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var list []*models.Dictionary
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("len = %d, want >= 2", len(list))
	}
}

func TestHandleListDictionaries_FilterByCategory(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	createTestDictionary(t, server, "dir1", models.DictionaryCategoryDirscan)
	createTestDictionary(t, server, "sub1", models.DictionaryCategorySubdomain)

	req := httptest.NewRequest(http.MethodGet, "/dictionaries?category=dirscan", nil)
	w := httptest.NewRecorder()

	server.handleListDictionaries(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var list []*models.Dictionary
	json.NewDecoder(w.Body).Decode(&list)
	for _, d := range list {
		if d.Category != models.DictionaryCategoryDirscan {
			t.Errorf("category = %q, want dirscan", d.Category)
		}
	}
}

func TestHandleListDictionaries_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	server.dictMgr = nil

	req := httptest.NewRequest(http.MethodGet, "/dictionaries", nil)
	w := httptest.NewRecorder()

	server.handleListDictionaries(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// --- handleCreateDictionary ---

func TestHandleCreateDictionary_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	buf, contentType := buildMultipartDict("newdict", "dirscan", "my dict", "word1\nword2\n")
	req := httptest.NewRequest(http.MethodPost, "/dictionaries", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	server.handleCreateDictionary(w, req)

	if w.Code != http.StatusCreated {
		body, _ := io.ReadAll(w.Body)
		t.Fatalf("status = %d, body = %s", w.Code, string(body))
	}

	var d models.Dictionary
	if err := json.NewDecoder(w.Body).Decode(&d); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if d.Name != "newdict" {
		t.Errorf("name = %q, want newdict", d.Name)
	}
	if d.Category != models.DictionaryCategoryDirscan {
		t.Errorf("category = %q, want dirscan", d.Category)
	}
	if d.LineCount != 2 {
		t.Errorf("line_count = %d, want 2", d.LineCount)
	}
}

func TestHandleCreateDictionary_MissingName(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	buf, contentType := buildMultipartDict("", "dirscan", "", "content")
	req := httptest.NewRequest(http.MethodPost, "/dictionaries", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	server.handleCreateDictionary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateDictionary_MissingCategory(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	buf, contentType := buildMultipartDict("test", "", "", "content")
	req := httptest.NewRequest(http.MethodPost, "/dictionaries", buf)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	server.handleCreateDictionary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateDictionary_MissingFile(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("name", "test")
	_ = writer.WriteField("category", "dirscan")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/dictionaries", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	server.handleCreateDictionary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateDictionary_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	server.dictMgr = nil

	req := httptest.NewRequest(http.MethodPost, "/dictionaries", nil)
	w := httptest.NewRecorder()

	server.handleCreateDictionary(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// --- handleGetDictionary ---

func TestHandleGetDictionary_Found(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "findme", models.DictionaryCategoryDirscan)

	req := httptest.NewRequest(http.MethodGet, "/dictionaries/"+d.ID, nil)
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handleGetDictionary(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var got models.Dictionary
	json.NewDecoder(w.Body).Decode(&got)
	if got.ID != d.ID {
		t.Errorf("id = %q, want %q", got.ID, d.ID)
	}
}

func TestHandleGetDictionary_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dictionaries/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleGetDictionary(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- handlePatchDictionary ---

func TestHandlePatchDictionary_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "original", models.DictionaryCategoryDirscan)

	body, _ := json.Marshal(map[string]string{
		"name":     "updated",
		"category": "subdomain",
	})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+d.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionary(w, req)

	if w.Code != http.StatusOK {
		body, _ := io.ReadAll(w.Body)
		t.Fatalf("status = %d, body = %s", w.Code, string(body))
	}

	var got models.Dictionary
	json.NewDecoder(w.Body).Decode(&got)
	if got.Name != "updated" {
		t.Errorf("name = %q, want updated", got.Name)
	}
}

func TestHandlePatchDictionary_MissingName(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "original", models.DictionaryCategoryDirscan)

	body, _ := json.Marshal(map[string]string{"category": "subdomain"})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+d.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePatchDictionary_MissingCategory(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "original", models.DictionaryCategoryDirscan)

	body, _ := json.Marshal(map[string]string{"name": "updated"})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+d.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePatchDictionary_BuiltinBlocked(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Builtin dicts are seeded by EnsureLayout+SeedBuiltin in NewServer.
	// Get the first builtin one from the list.
	list, _ := server.dictMgr.List("")
	var builtin *models.Dictionary
	for _, dd := range list {
		if dd.Builtin {
			builtin = dd
			break
		}
	}
	if builtin == nil {
		t.Skip("no builtin dictionary found in test setup")
	}

	body, _ := json.Marshal(map[string]string{
		"name":     "hacked",
		"category": "dirscan",
	})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+builtin.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", builtin.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionary(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandlePatchDictionary_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{
		"name":     "x",
		"category": "dirscan",
	})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handlePatchDictionary(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlePatchDictionary_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "test", models.DictionaryCategoryDirscan)

	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+d.ID, strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionary(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- handlePatchDictionaryEnabled ---

func TestHandlePatchDictionaryEnabled_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Find a builtin dictionary.
	list, _ := server.dictMgr.List("")
	var builtin *models.Dictionary
	for _, d := range list {
		if d.Builtin {
			builtin = d
			break
		}
	}
	if builtin == nil {
		t.Skip("no builtin dictionary found")
	}

	body, _ := json.Marshal(map[string]bool{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+builtin.ID+"/enabled", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", builtin.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionaryEnabled(w, req)

	if w.Code != http.StatusOK {
		b, _ := io.ReadAll(w.Body)
		t.Fatalf("status = %d, body = %s", w.Code, string(b))
	}

	var got models.Dictionary
	json.NewDecoder(w.Body).Decode(&got)
	if got.Enabled {
		t.Error("expected enabled=false after patch")
	}
}

func TestHandlePatchDictionaryEnabled_NonBuiltinBlocked(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "userdict", models.DictionaryCategoryDirscan)

	body, _ := json.Marshal(map[string]bool{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+d.ID+"/enabled", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionaryEnabled(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandlePatchDictionaryEnabled_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]bool{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/nonexistent/enabled", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handlePatchDictionaryEnabled(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlePatchDictionaryEnabled_InvalidJSON(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/x/enabled", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "x")
	w := httptest.NewRecorder()

	server.handlePatchDictionaryEnabled(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- handleDeleteDictionary ---

func TestHandleDeleteDictionary_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "todelete", models.DictionaryCategoryDirscan)

	req := httptest.NewRequest(http.MethodDelete, "/dictionaries/"+d.ID, nil)
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handleDeleteDictionary(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "deleted" {
		t.Errorf("status = %q, want deleted", resp["status"])
	}
}

func TestHandleDeleteDictionary_BuiltinBlocked(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	list, _ := server.dictMgr.List("")
	var builtin *models.Dictionary
	for _, d := range list {
		if d.Builtin {
			builtin = d
			break
		}
	}
	if builtin == nil {
		t.Skip("no builtin dictionary found")
	}

	req := httptest.NewRequest(http.MethodDelete, "/dictionaries/"+builtin.ID, nil)
	req.SetPathValue("id", builtin.ID)
	w := httptest.NewRecorder()

	server.handleDeleteDictionary(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleDeleteDictionary_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/dictionaries/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleDeleteDictionary(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- handleReadDictionaryContent ---

func TestHandleReadDictionaryContent_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "contenttest", models.DictionaryCategoryDirscan)

	req := httptest.NewRequest(http.MethodGet, "/dictionaries/"+d.ID+"/content", nil)
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handleReadDictionaryContent(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("content-type = %q, want text/plain", ct)
	}

	body := w.Body.String()
	if body != "line1\nline2\n" {
		t.Errorf("body = %q, want line1\\nline2\\n", body)
	}
}

func TestHandleReadDictionaryContent_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dictionaries/nonexistent/content", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleReadDictionaryContent(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- handleWriteDictionaryContent ---

func TestHandleWriteDictionaryContent_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "writetest", models.DictionaryCategoryDirscan)

	newContent := "new1\nnew2\nnew3\n"
	req := httptest.NewRequest(http.MethodPut, "/dictionaries/"+d.ID+"/content", strings.NewReader(newContent))
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handleWriteDictionaryContent(w, req)

	if w.Code != http.StatusOK {
		b, _ := io.ReadAll(w.Body)
		t.Fatalf("status = %d, body = %s", w.Code, string(b))
	}

	var got models.Dictionary
	json.NewDecoder(w.Body).Decode(&got)
	if got.LineCount != 3 {
		t.Errorf("line_count = %d, want 3", got.LineCount)
	}
}

func TestHandleWriteDictionaryContent_BuiltinBlocked(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	list, _ := server.dictMgr.List("")
	var builtin *models.Dictionary
	for _, d := range list {
		if d.Builtin {
			builtin = d
			break
		}
	}
	if builtin == nil {
		t.Skip("no builtin dictionary found")
	}

	req := httptest.NewRequest(http.MethodPut, "/dictionaries/"+builtin.ID+"/content", strings.NewReader("hack"))
	req.SetPathValue("id", builtin.ID)
	w := httptest.NewRecorder()

	server.handleWriteDictionaryContent(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- requireUserDictionary ---

func TestRequireUserDictionary_BuiltinReturnsFalse(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	list, _ := server.dictMgr.List("")
	var builtin *models.Dictionary
	for _, d := range list {
		if d.Builtin {
			builtin = d
			break
		}
	}
	if builtin == nil {
		t.Skip("no builtin dictionary found")
	}

	w := httptest.NewRecorder()
	ok := server.requireUserDictionary(w, builtin.ID)
	if ok {
		t.Error("expected false for builtin dictionary")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireUserDictionary_UserDictReturnsTrue(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "user", models.DictionaryCategoryDirscan)

	w := httptest.NewRecorder()
	ok := server.requireUserDictionary(w, d.ID)
	if !ok {
		t.Error("expected true for user dictionary")
	}
}

func TestRequireUserDictionary_NotFoundReturnsFalse(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	ok := server.requireUserDictionary(w, "nonexistent")
	if ok {
		t.Error("expected false for nonexistent dictionary")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

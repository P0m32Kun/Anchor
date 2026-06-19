package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	apperrors "github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei/custom"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// errGeneric is a plain error for testing generic error paths.
var errGeneric = &plainError{"something went wrong"}

type plainError struct{ msg string }

func (e *plainError) Error() string { return e.msg }

// ==================== report_handlers.go ====================

func TestHandleExportReportMD_ProjectNotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects/nonexistent/report", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handleExportReportMD(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleExportReportMD_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/report", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleExportReportMD(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/markdown; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/markdown", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, p.ID) {
		t.Errorf("Content-Disposition = %q, should contain project ID", cd)
	}
}

// ==================== nuclei_custom_handlers.go ====================

func TestWriteNucleiCustomError_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	err := apperrors.New(apperrors.ErrNotFound, "source not found")
	writeNucleiCustomError(w, err)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestWriteNucleiCustomError_NotBuiltin(t *testing.T) {
	w := httptest.NewRecorder()
	writeNucleiCustomError(w, custom.ErrNotBuiltin)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestWriteNucleiCustomError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	writeNucleiCustomError(w, errGeneric)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleListNucleiCustomSources_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.nucleiCustomMgr
	server.nucleiCustomMgr = nil
	defer func() { server.nucleiCustomMgr = orig }()

	req := httptest.NewRequest(http.MethodGet, "/nuclei-custom/sources", nil)
	w := httptest.NewRecorder()

	server.handleListNucleiCustomSources(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandleListNucleiCustomSources_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/nuclei-custom/sources", nil)
	w := httptest.NewRecorder()

	server.handleListNucleiCustomSources(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandlePatchNucleiCustomSourceEnabled_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	orig := server.nucleiCustomMgr
	server.nucleiCustomMgr = nil
	defer func() { server.nucleiCustomMgr = orig }()

	body := `{"enabled": false}`
	req := httptest.NewRequest(http.MethodPatch, "/nuclei-custom/sources/s1", strings.NewReader(body))
	req.SetPathValue("id", "s1")
	w := httptest.NewRecorder()

	server.handlePatchNucleiCustomSourceEnabled(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandlePatchNucleiCustomSourceEnabled_MissingID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"enabled": false}`
	req := httptest.NewRequest(http.MethodPatch, "/nuclei-custom/sources/", strings.NewReader(body))
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handlePatchNucleiCustomSourceEnabled(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchNucleiCustomSourceEnabled_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPatch, "/nuclei-custom/sources/s1", strings.NewReader("not json"))
	req.SetPathValue("id", "s1")
	w := httptest.NewRecorder()

	server.handlePatchNucleiCustomSourceEnabled(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandlePatchNucleiCustomSourceEnabled_NotFound(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"enabled": false}`
	req := httptest.NewRequest(http.MethodPatch, "/nuclei-custom/sources/nonexistent", strings.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	server.handlePatchNucleiCustomSourceEnabled(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 status for nonexistent source")
	}
}

// ==================== engine_handlers.go — handleSearchEngine (extra paths) ====================

func TestHandleSearchEngine_MissingEngine(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engine/search?query=test", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleSearchEngine_MissingQuery(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/engine/search?engine=fofa", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleSearchEngine_WithPagination(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "fofa", APIKey: "test-key",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/engine/search?engine=fofa&query=test&page=2&size=50", nil)
	w := httptest.NewRecorder()

	server.handleSearchEngine(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 200 or 500", resp.StatusCode)
	}
}

// ==================== engine_handlers.go — handleGetEngineQuota (extra paths) ====================

func TestHandleGetEngineQuota_WithCredential(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "fofa", APIKey: "test-key",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/engine/quota?engine=fofa", nil)
	w := httptest.NewRecorder()

	server.handleGetEngineQuota(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 200 or 500", resp.StatusCode)
	}
}

// ==================== worker_handlers.go — handleTaskResult ====================

func TestHandleTaskResult_FailedWithError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskRunning)

	body := `{"status": "failed", "error": "timeout exceeded"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID+"/result", strings.NewReader(body))
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleTaskResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleTaskResult_WithArtifacts(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	task := createTestTask(t, queries, p.ID, models.TaskRunning)

	body := `{
		"status": "completed",
		"artifacts": [
			{
				"type": "stdout",
				"name": "nuclei_output.txt",
				"data": "c2NhbiByZXN1bHQgZGF0YQ=="
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/tasks/"+task.ID+"/result", strings.NewReader(body))
	req.SetPathValue("id", task.ID)
	w := httptest.NewRecorder()

	server.handleTaskResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}

	// Verify artifact was saved
	artifacts, err := queries.ListRawArtifactsByTask(task.ID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) == 0 {
		t.Error("expected at least 1 artifact to be saved")
	}
}

// ==================== pipeline_handlers.go — presetDefaults ====================

func TestPresetDefaults_AllModes(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		noiseLevel  string
		expectTools bool
	}{
		{"external_standard", "external", "standard", true},
		{"external_low", "external", "low", true},
		{"external_empty_noise", "external", "", true},
		{"internal", "internal", "", false},
		{"default", "", "", false},
		{"unknown", "unknown", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := presetDefaults(tt.mode, tt.noiseLevel)
			if tt.expectTools && !cfg.EnableSubfinder {
				t.Error("expected EnableSubfinder=true for external mode")
			}
			// Verify it returns a reasonable config
			if cfg.NucleiConcurrency < 0 {
				t.Errorf("NucleiConcurrency = %d, want >= 0", cfg.NucleiConcurrency)
			}
		})
	}
}

// ==================== finding_template_handlers.go — more branches ====================

func TestHandlePatchFindingTemplate_UpdateSummaryAndRemediation(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	summary := "updated summary"
	remediation := "updated remediation"
	payload := findingTemplatePayload{Summary: &summary, Remediation: &remediation}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.FindingTemplate
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Summary != "updated summary" {
		t.Errorf("summary = %q, want 'updated summary'", updated.Summary)
	}
	if updated.Remediation != "updated remediation" {
		t.Errorf("remediation = %q, want 'updated remediation'", updated.Remediation)
	}
}

func TestHandlePatchFindingTemplate_UpdateSeverity(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	severity := "critical"
	payload := findingTemplatePayload{Severity: &severity}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandlePatchFindingTemplate_UpdateTitle(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	title := "New Title"
	payload := findingTemplatePayload{Title: &title}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated models.FindingTemplate
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Title != "New Title" {
		t.Errorf("title = %q, want 'New Title'", updated.Title)
	}
}

func TestHandlePatchFindingTemplate_UpdateMatchKeys(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	keys := []string{"CVE-2024-9999"}
	payload := findingTemplatePayload{MatchKeys: &keys}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}

	var updated models.FindingTemplate
	json.NewDecoder(resp.Body).Decode(&updated)
	if len(updated.MatchKeys) != 1 || updated.MatchKeys[0] != "CVE-2024-9999" {
		t.Errorf("match_keys = %v, want [CVE-2024-9999]", updated.MatchKeys)
	}
}

func TestHandlePatchFindingTemplate_DuplicateMatchKey(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")
	ft2 := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0002")

	keys := []string{"CVE-2024-0001"}
	payload := findingTemplatePayload{MatchKeys: &keys}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/finding-templates/"+ft2.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", ft2.ID)
	w := httptest.NewRecorder()

	server.handlePatchFindingTemplate(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestHandleFindingTemplate_BuiltinAcceptUpstream(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)

	builtinPayload := `{"source_tool":"nuclei","match_key":"CVE-2024-0099","match_keys":["CVE-2024-0099"],"title":"Builtin Vuln","severity":"high","summary":"builtin summary","remediation":"fix it","enabled":true}`
	now := time.Now().UTC()
	ft := &models.FindingTemplate{
		ID:             util.GenerateID(),
		SourceTool:     "nuclei",
		MatchKey:       "CVE-2024-0099",
		MatchKeys:      []string{"CVE-2024-0099"},
		Title:          "Modified Title",
		Severity:       "low",
		Summary:        "user modified",
		Remediation:    "user fix",
		Enabled:        true,
		IsBuiltin:      true,
		UserModified:   true,
		BuiltinPayload: builtinPayload,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := queries.CreateFindingTemplate(ft); err != nil {
		t.Fatalf("create finding template: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/finding-templates/"+ft.ID+"/accept-upstream", nil)
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handleAcceptFindingTemplateUpstream(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}

	var updated models.FindingTemplate
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.UserModified {
		t.Error("expected user_modified=false after accepting upstream")
	}
	if updated.Title != "Builtin Vuln" {
		t.Errorf("title = %q, want 'Builtin Vuln'", updated.Title)
	}
}

func TestHandleExportFindingTemplates_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/finding-templates/export", nil)
	w := httptest.NewRecorder()

	server.handleExportFindingTemplates(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "vuln-templates.json") {
		t.Errorf("Content-Disposition = %q, should contain vuln-templates.json", cd)
	}
}

func TestHandleExportFindingTemplates_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-0001")

	req := httptest.NewRequest(http.MethodGet, "/finding-templates/export", nil)
	w := httptest.NewRecorder()

	server.handleExportFindingTemplates(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// ==================== notification_handlers.go ====================

func TestHandleListNotificationChannels_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/projects//notifications", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleListNotificationChannels(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ==================== scope_handlers.go ====================

func TestHandleListScopeRules_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/scope-rules", nil)
	w := httptest.NewRecorder()

	server.handleListScopeRules(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ==================== asset_handlers.go ====================

func TestHandleListServicePorts_Empty(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/service-ports", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleListAssetsFiltered_StatusParam(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	now := time.Now().UTC()
	a := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID, Type: models.AssetTypeDomain,
		Value: "example.com", FirstSeen: now, LastSeen: now,
	}
	if err := queries.CreateAsset(a); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?status=active", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleListAssetsFiltered_TypeParam(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	now := time.Now().UTC()
	a := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID, Type: models.AssetTypeDomain,
		Value: "example.com", FirstSeen: now, LastSeen: now,
	}
	if err := queries.CreateAsset(a); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?type=domain", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleListAssetsFiltered_SearchParam(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	now := time.Now().UTC()
	a := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID, Type: models.AssetTypeDomain,
		Value: "example.com", FirstSeen: now, LastSeen: now,
	}
	if err := queries.CreateAsset(a); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?search=example", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// ==================== pipeline_handlers.go — more paths ====================

func TestHandleGetPipelineConfig_WithSavedConfig(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	cfgJSON := `{"enable_subfinder":false,"enable_naabu":true,"enable_nuclei":true,"nuclei_concurrency":5}`
	if err := queries.UpdateProjectPipelineConfig(p.ID, cfgJSON); err != nil {
		t.Fatalf("update project pipeline config: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline/config", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleGetPipelineConfig(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}
}

func TestHandleListPipelineRuns_Empty(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline-runs", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListPipelineRuns(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleGetPipelineRunStages(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p.ID, "running")

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/pipeline/runs/"+run.ID+"/stages", nil)
	req.SetPathValue("id", p.ID)
	req.SetPathValue("runId", run.ID)
	w := httptest.NewRecorder()

	server.handleGetPipelineRunStages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusOK, w.Body.String())
	}
}

func TestHandleCreateScan_MissingProjectID(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"mode":"external","config":{}}`
	req := httptest.NewRequest(http.MethodPost, "/projects//scan", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	server.handleCreateScan(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ==================== server.go — markAllWorkersOffline ====================

func TestMarkAllWorkersOffline(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)

	w1 := createTestWorker(t, queries, models.WorkerStatusOnline)
	_ = createTestWorker(t, queries, models.WorkerStatusBusy)

	server.mu.Lock()
	server.taskQueue[w1.ID] = make(chan *models.ScanTask, 10)
	server.taskResults[w1.ID] = make(chan map[string]interface{}, 10)
	server.mu.Unlock()

	server.markAllWorkersOffline()

	workers, err := queries.ListWorkerNodes()
	if err != nil {
		t.Fatalf("list workers: %v", err)
	}
	for _, w := range workers {
		if w.Status != models.WorkerStatusOffline {
			t.Errorf("worker %s status = %s, want offline", w.ID, w.Status)
		}
	}

	server.mu.Lock()
	_, exists := server.taskQueue[w1.ID]
	server.mu.Unlock()
	if exists {
		t.Error("expected task queue to be cleaned up for worker")
	}
}

// ==================== run_handlers.go — dispatchTasksToWorkers / enqueueToWorker ====================

func TestEnqueueToWorker_NoWorkers(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	task := &models.ScanTask{
		ID: util.GenerateID(), ProjectID: "p1", Tool: "nuclei",
	}
	ok := server.enqueueToWorker(task)
	if ok {
		t.Error("expected false when no workers exist")
	}
}

func TestEnqueueToWorker_WorkerOffline(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	w := createTestWorker(t, queries, models.WorkerStatusOffline)

	server.mu.Lock()
	server.taskQueue[w.ID] = make(chan *models.ScanTask, 10)
	server.mu.Unlock()

	task := &models.ScanTask{
		ID: util.GenerateID(), ProjectID: "p1", Tool: "nuclei",
	}
	ok := server.enqueueToWorker(task)
	if ok {
		t.Error("expected false when worker is offline")
	}
}

func TestDispatchTasksToWorkers_NoAvailableWorker(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	task := &models.ScanTask{
		ID: util.GenerateID(), ProjectID: "p1", Tool: "nuclei",
	}
	server.dispatchTasksToWorkers([]*models.ScanTask{task})
}

func TestDispatchTasksToWorkers_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	w := createTestWorker(t, queries, models.WorkerStatusOnline)

	server.mu.Lock()
	server.taskQueue[w.ID] = make(chan *models.ScanTask, 10)
	server.taskResults[w.ID] = make(chan map[string]interface{}, 10)
	server.mu.Unlock()

	task := &models.ScanTask{
		ID: util.GenerateID(), ProjectID: "p1", Tool: "nuclei",
		CommandTemplate: "nuclei -t test.yaml",
	}
	server.dispatchTasksToWorkers([]*models.ScanTask{task})

	server.mu.Lock()
	ch := server.taskQueue[w.ID]
	server.mu.Unlock()

	select {
	case received := <-ch:
		if received.ID != task.ID {
			t.Errorf("task ID = %s, want %s", received.ID, task.ID)
		}
	default:
		t.Error("expected task to be enqueued")
	}
}

// ==================== handlers.go — handleListWorkers extra data ====================

func TestHandleListWorkers_WithBusyWorker(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestWorker(t, queries, models.WorkerStatusBusy)

	req := httptest.NewRequest(http.MethodGet, "/workers", nil)
	w := httptest.NewRecorder()

	server.handleListWorkers(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var workers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(workers) != 1 {
		t.Fatalf("len(workers) = %d, want 1", len(workers))
	}
	if busy, ok := workers[0]["busy"].(bool); !ok || !busy {
		t.Error("expected busy=true for busy worker")
	}
}

func TestHandleListWorkers_WithOfflineWorker(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestWorker(t, queries, models.WorkerStatusOffline)

	req := httptest.NewRequest(http.MethodGet, "/workers", nil)
	w := httptest.NewRecorder()

	server.handleListWorkers(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var workers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(workers) != 1 {
		t.Fatalf("len(workers) = %d, want 1", len(workers))
	}
	if busy, ok := workers[0]["busy"].(bool); ok && busy {
		t.Error("expected busy=false for offline worker")
	}
}

func TestHandleListWorkers_WithMetrics(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()
	cpu := 55.5
	mem := 72.3
	disk := 40.0
	w := &models.WorkerNode{
		ID:               util.GenerateID(),
		Name:             "test-worker-metrics",
		Endpoint:         "http://localhost:9999",
		Mode:             models.WorkerModeRemote,
		Status:           models.WorkerStatusOnline,
		TrustLevel:       "standard",
		MaxConcurrency:   10,
		CPUPercent:       &cpu,
		MemPercent:       &mem,
		DiskPercent:      &disk,
		MetricsUpdatedAt: &now,
		LastSeen:         &now,
		CreatedAt:        now,
	}
	if err := queries.CreateWorkerNode(w); err != nil {
		t.Fatalf("create worker: %v", err)
	}
	// CreateWorkerNode doesn't persist metrics; set them via UpdateWorkerNodeMetrics
	if err := queries.UpdateWorkerNodeMetrics(w.ID, &cpu, &mem, &disk, now); err != nil {
		t.Fatalf("update worker metrics: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/workers", nil)
	rec := httptest.NewRecorder()

	server.handleListWorkers(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var workers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(workers) != 1 {
		t.Fatalf("len(workers) = %d, want 1", len(workers))
	}
	if _, ok := workers[0]["cpu_percent"]; !ok {
		t.Error("expected cpu_percent in worker data")
	}
	if _, ok := workers[0]["mem_percent"]; !ok {
		t.Error("expected mem_percent in worker data")
	}
	if _, ok := workers[0]["disk_percent"]; !ok {
		t.Error("expected disk_percent in worker data")
	}
}

// ==================== excluded domain — additional paths ====================

func TestHandleListDefaultDomains(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains/defaults", nil)
	w := httptest.NewRecorder()

	server.handleListDefaultDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := result["domains"]; !ok {
		t.Error("expected 'domains' key in response")
	}
	if _, ok := result["total"]; !ok {
		t.Error("expected 'total' key in response")
	}
}

func TestHandleAddExcludedDomain_WithProtocol(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	_ = db.New(rawDB)

	body := `{"domain": "https://tracker.example.com", "reason": "analytics"}`
	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, http.StatusCreated, w.Body.String())
	}

	var result models.ExcludedDomain
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Domain != "tracker.example.com" {
		t.Errorf("domain = %q, want tracker.example.com (protocol stripped)", result.Domain)
	}
}

// ==================== scan_run_auth.go ====================

func TestRequireRunInProject_MissingIDs(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	run, ok := server.requireRunInProject(w, nil, "", "")
	if ok {
		t.Error("expected false for missing IDs")
	}
	if run != nil {
		t.Error("expected nil run")
	}
}

func TestRequireRunInProject_RunNotFound(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	w := httptest.NewRecorder()
	run, ok := server.requireRunInProject(w, nil, p.ID, "nonexistent")
	if ok {
		t.Error("expected false for nonexistent run")
	}
	if run != nil {
		t.Error("expected nil run")
	}
}

func TestRequireRunInProject_ProjectMismatch(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p1 := createTestProject(t, queries)
	p2 := createTestProject(t, queries)
	run := createTestPipelineRun(t, queries, p1.ID, "running")

	w := httptest.NewRecorder()
	_, ok := server.requireRunInProject(w, nil, p2.ID, run.ID)
	if ok {
		t.Error("expected false for project mismatch")
	}
}

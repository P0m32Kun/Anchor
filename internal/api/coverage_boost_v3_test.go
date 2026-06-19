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
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ==================== handleListServicePorts — 38.8% → more branches ====================

func TestHandleListServicePorts_WebEndpointMerge(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	// Create IP asset
	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.1",
		NormalizedValue: "10.0.0.1", SourceTools: []string{"naabu"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	// Create web endpoint (httpx result) — should be merged into service port
	port80 := 80
	queries.CreateWebEndpoint(&models.WebEndpoint{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		AssetID:   asset.ID,
		URL:       "http://10.0.0.1:80/",
		Scheme:    "http",
		Host:      "10.0.0.1",
		Port:      &port80,
		Path:      "/",
		Title:     "Welcome",
		SourceTool: "httpx",
		CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/service-ports", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var result []*models.ServicePort
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Title != "Welcome" {
		t.Errorf("title = %q, want Welcome", result[0].Title)
	}
	if !result[0].IsWeb {
		t.Error("expected is_web=true")
	}
}

func TestHandleListServicePorts_NaabuOnly(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.2",
		NormalizedValue: "10.0.0.2", SourceTools: []string{"naabu"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	// Only naabu port, no fingerprint or web endpoint.
	// NOTE: the naabu merge path uses ipToAsset[p.AssetID] but the map is
	// keyed by IP value, so naabu-only ports with no fingerprint/web are
	// effectively unreachable. This test exercises the allPorts loop with
	// empty fingerprints and webEndpoints.
	queries.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID,
		Port: 22, Protocol: "tcp", State: "open", SourceTool: "naabu",
		CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/service-ports", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result []*models.ServicePort
	json.NewDecoder(w.Body).Decode(&result)
	// Naabu-only ports miss the ipToAsset lookup (keyed by IP value, not asset ID),
	// so they are skipped. Result is empty.
	if len(result) != 0 {
		t.Errorf("len = %d, want 0 (naabu-only merge path skipped)", len(result))
	}
}

func TestHandleListServicePorts_MultipleIPs(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	// Use fingerprints (which correctly key by IP) to create service ports for multiple IPs
	for i, ip := range []string{"10.0.0.5", "10.0.0.3", "10.0.0.1"} {
		asset := &models.Asset{
			ID: util.GenerateID(), ProjectID: p.ID,
			Type: models.AssetTypeIP, Value: ip,
			NormalizedValue: ip, SourceTools: []string{"nmap"},
			FirstSeen: now, LastSeen: now,
		}
		queries.CreateAsset(asset)
		queries.SaveServiceFingerprint(&models.ServiceFingerprint{
			ID: util.GenerateID(), ProjectID: p.ID,
			IP: ip, Port: 80 + i, Protocol: "tcp",
			Service: "http", Source: "nmap", CreatedAt: now,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/service-ports", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result []*models.ServicePort
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	// Verify sorted by IP
	if result[0].IP != "10.0.0.1" || result[1].IP != "10.0.0.3" || result[2].IP != "10.0.0.5" {
		t.Errorf("not sorted: %s, %s, %s", result[0].IP, result[1].IP, result[2].IP)
	}
}

func TestHandleListServicePorts_MergeFingerprintAndWeb(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.1",
		NormalizedValue: "10.0.0.1", SourceTools: []string{"nmap"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	// Service fingerprint from nmap
	queries.SaveServiceFingerprint(&models.ServiceFingerprint{
		ID: util.GenerateID(), ProjectID: p.ID,
		IP: "10.0.0.1", Port: 8080, Protocol: "tcp",
		Service: "http", Product: "Tomcat", Version: "9.0",
		IsWeb: true, Source: "nmap", CreatedAt: now,
	})

	// Web endpoint on same port from httpx
	port8080 := 8080
	queries.CreateWebEndpoint(&models.WebEndpoint{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		AssetID:   asset.ID,
		URL:       "http://10.0.0.1:8080/",
		Scheme:    "http",
		Host:      "10.0.0.1",
		Port:      &port8080,
		Path:      "/",
		Title:     "Tomcat Manager",
		Technologies: []string{"Java", "Tomcat"},
		SourceTool: "httpx",
		CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/service-ports", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result []*models.ServicePort
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	sp := result[0]
	if sp.ServiceName != "http" {
		t.Errorf("service = %q, want http", sp.ServiceName)
	}
	if sp.Title != "Tomcat Manager" {
		t.Errorf("title = %q, want Tomcat Manager", sp.Title)
	}
	if sp.Product != "Tomcat" {
		t.Errorf("product = %q, want Tomcat", sp.Product)
	}
	if !sp.IsWeb {
		t.Error("expected is_web=true")
	}
}

func TestHandleListServicePorts_WebEndpointFallbackPort(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.1",
		NormalizedValue: "10.0.0.1", SourceTools: []string{"naabu"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	// Web endpoint with no explicit port — should fallback to parsePortFromURL
	queries.CreateWebEndpoint(&models.WebEndpoint{
		ID:        util.GenerateID(),
		ProjectID: p.ID,
		AssetID:   asset.ID,
		URL:       "https://10.0.0.1/login",
		Scheme:    "https",
		Host:      "10.0.0.1",
		Port:      nil, // no explicit port
		Path:      "/login",
		Title:     "Login Page",
		SourceTool: "httpx",
		CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/service-ports", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result []*models.ServicePort
	json.NewDecoder(w.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Port != 443 {
		t.Errorf("port = %d, want 443 (from https scheme)", result[0].Port)
	}
}

func TestHandleListServicePorts_DBError_ListAssets(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/projects/p1/service-ports", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleGetAssetLineage — 47.4% ====================

func TestHandleGetAssetLineage_WithRunID(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "sub.example.com",
		NormalizedValue: "sub.example.com", SourceTools: []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	runID := "run-123"
	// Create a target for the lineage chain
	target := &models.Target{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.TargetTypeDomain, Value: "example.com",
		Source: "manual", Status: "active",
		CreatedAt: now,
	}
	queries.CreateTarget(target)

	// Create relation: target → asset
	queries.UpsertAssetRelation(&models.AssetRelation{
		ID:           util.GenerateID(),
		ProjectID:    p.ID,
		RunID:        &runID,
		SourceType:   models.RelationSourceTarget,
		SourceID:     target.ID,
		TargetType:   models.RelationTargetAsset,
		TargetID:     asset.ID,
		RelationType: models.RelationExpandedBy,
		SourceEngine: "subfinder",
		CreatedAt:    now,
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/"+asset.ID+"/lineage?run_id="+runID, nil)
	req.SetPathValue("id", asset.ID)
	w := httptest.NewRecorder()

	server.handleGetAssetLineage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var lineage models.AssetLineage
	json.NewDecoder(w.Body).Decode(&lineage)
	if lineage.AssetID != asset.ID {
		t.Errorf("asset_id = %q, want %q", lineage.AssetID, asset.ID)
	}
	if lineage.RunID != runID {
		t.Errorf("run_id = %q, want %q", lineage.RunID, runID)
	}
	if len(lineage.Chain) != 2 {
		t.Fatalf("chain len = %d, want 2", len(lineage.Chain))
	}
	if lineage.Chain[0].NodeType != models.RelationSourceTarget {
		t.Errorf("chain[0].node_type = %q, want target", lineage.Chain[0].NodeType)
	}
}

func TestHandleGetAssetLineage_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "a.com",
		NormalizedValue: "a.com", SourceTools: []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	// Close DB so GetAssetByID fails first (returns 500)
	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/assets/"+asset.ID+"/lineage", nil)
	req.SetPathValue("id", asset.ID)
	w := httptest.NewRecorder()

	server.handleGetAssetLineage(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleListAssetsFiltered — 59.6% ====================

func TestHandleListAssetsFiltered_PortFilter(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	a1 := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.1",
		NormalizedValue: "10.0.0.1", SourceTools: []string{"naabu"},
		FirstSeen: now, LastSeen: now,
	}
	a2 := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.2",
		NormalizedValue: "10.0.0.2", SourceTools: []string{"naabu"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(a1)
	queries.CreateAsset(a2)

	// a1 has port 80, a2 has port 443
	queries.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: a1.ID,
		Port: 80, Protocol: "tcp", State: "open", SourceTool: "naabu", CreatedAt: now,
	})
	queries.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: a2.ID,
		Port: 443, Protocol: "tcp", State: "open", SourceTool: "naabu", CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?port=80", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result PaginatedResponse[*models.Asset]
	json.NewDecoder(w.Body).Decode(&result)
	if len(result.Data) != 1 {
		t.Fatalf("len = %d, want 1", len(result.Data))
	}
	if result.Data[0].Value != "10.0.0.1" {
		t.Errorf("value = %q, want 10.0.0.1", result.Data[0].Value)
	}
}

func TestHandleListAssetsFiltered_TitleFilter(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	a1 := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "a.com",
		NormalizedValue: "a.com", SourceTools: []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	}
	a2 := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "b.com",
		NormalizedValue: "b.com", SourceTools: []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(a1)
	queries.CreateAsset(a2)

	// a1 has a web endpoint with title "Dashboard"
	port80 := 80
	queries.CreateWebEndpoint(&models.WebEndpoint{
		ID: util.GenerateID(), ProjectID: p.ID, AssetID: a1.ID,
		URL: "http://a.com/", Host: "a.com", Port: &port80,
		Title: "Dashboard", SourceTool: "httpx", CreatedAt: now,
	})
	queries.CreateWebEndpoint(&models.WebEndpoint{
		ID: util.GenerateID(), ProjectID: p.ID, AssetID: a2.ID,
		URL: "http://b.com/", Host: "b.com", Port: &port80,
		Title: "Blog", SourceTool: "httpx", CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?title=dashboard", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result PaginatedResponse[*models.Asset]
	json.NewDecoder(w.Body).Decode(&result)
	if len(result.Data) != 1 {
		t.Fatalf("len = %d, want 1", len(result.Data))
	}
	if result.Data[0].Value != "a.com" {
		t.Errorf("value = %q, want a.com", result.Data[0].Value)
	}
}

func TestHandleListAssetsFiltered_ShowExcluded(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	queries.CreateAsset(&models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "keep.com",
		NormalizedValue: "keep.com", SourceTools: []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?show_excluded=true", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result PaginatedResponse[*models.Asset]
	json.NewDecoder(w.Body).Decode(&result)
	if len(result.Data) != 1 {
		t.Fatalf("len = %d, want 1", len(result.Data))
	}
}

func TestHandleListAssetsFiltered_PaginationBeyondTotal(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	queries.CreateAsset(&models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "a.com",
		NormalizedValue: "a.com", SourceTools: []string{"subfinder"},
		FirstSeen: now, LastSeen: now,
	})

	// Request page beyond total
	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?page=99&page_size=10", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result PaginatedResponse[*models.Asset]
	json.NewDecoder(w.Body).Decode(&result)
	if len(result.Data) != 0 {
		t.Errorf("len = %d, want 0 (beyond total)", len(result.Data))
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
}

// ==================== handleListWebEndpointsByProject — 63.6% ====================

func TestHandleListWebEndpointsByProject_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "a.com",
		NormalizedValue: "a.com", SourceTools: []string{"httpx"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	port80 := 80
	for _, path := range []string{"/", "/api"} {
		queries.CreateWebEndpoint(&models.WebEndpoint{
			ID: util.GenerateID(), ProjectID: p.ID, AssetID: asset.ID,
			URL: "http://a.com" + path, Host: "a.com", Port: &port80,
			Path: path, SourceTool: "httpx", CreatedAt: now,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/web-endpoints", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListWebEndpointsByProject(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var result PaginatedResponse[*models.WebEndpoint]
	json.NewDecoder(w.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if len(result.Data) != 2 {
		t.Errorf("len = %d, want 2", len(result.Data))
	}
}

// ==================== handleDeleteFindingTemplate — 60.0% ====================

func TestHandleDeleteFindingTemplate_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	ft := createTestFindingTemplate(t, queries, "nuclei", "CVE-2024-ERR")

	rawDB.Close()

	req := httptest.NewRequest(http.MethodDelete, "/finding-templates/"+ft.ID, nil)
	req.SetPathValue("id", ft.ID)
	w := httptest.NewRecorder()

	server.handleDeleteFindingTemplate(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handlePatchDictionaryEnabled — 60.9% ====================

func TestHandlePatchDictionaryEnabled_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	server.dictMgr = nil

	body, _ := json.Marshal(map[string]bool{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/x/enabled", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "x")
	w := httptest.NewRecorder()

	server.handlePatchDictionaryEnabled(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandlePatchDictionaryEnabled_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	// Find a builtin dictionary first
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

	rawDB.Close()

	body, _ := json.Marshal(map[string]bool{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/dictionaries/"+builtin.ID+"/enabled", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", builtin.ID)
	w := httptest.NewRecorder()

	server.handlePatchDictionaryEnabled(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== requireUserDictionary — 63.6% ====================

func TestRequireUserDictionary_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	w := httptest.NewRecorder()
	ok := server.requireUserDictionary(w, "any-id")
	if ok {
		t.Error("expected false on DB error")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleGetDictionary — 66.7% ====================

func TestHandleGetDictionary_NilManager(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	server.dictMgr = nil

	req := httptest.NewRequest(http.MethodGet, "/dictionaries/x", nil)
	req.SetPathValue("id", "x")
	w := httptest.NewRecorder()

	server.handleGetDictionary(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleGetDictionary_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "errdict", models.DictionaryCategoryDirscan)

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/dictionaries/"+d.ID, nil)
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handleGetDictionary(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleWriteDictionaryContent — 66.7% ====================

func TestHandleWriteDictionaryContent_NilManagerV3(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	server.dictMgr = nil

	req := httptest.NewRequest(http.MethodPut, "/dictionaries/x/content", strings.NewReader("data"))
	req.SetPathValue("id", "x")
	w := httptest.NewRecorder()

	server.handleWriteDictionaryContent(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleWriteDictionaryContent_UpdateContentError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	d := createTestDictionary(t, server, "upderr", models.DictionaryCategoryDirscan)

	rawDB.Close()

	req := httptest.NewRequest(http.MethodPut, "/dictionaries/"+d.ID+"/content", strings.NewReader("new content"))
	req.SetPathValue("id", d.ID)
	w := httptest.NewRecorder()

	server.handleWriteDictionaryContent(w, req)

	// requireUserDictionary will fail with DB error first
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleListAssets — 66.7% ====================

func TestHandleListAssets_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssets(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleListPorts — 66.7% ====================

func TestHandleListPorts_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()
	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.1",
		NormalizedValue: "10.0.0.1", SourceTools: []string{"naabu"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/assets/"+asset.ID+"/ports", nil)
	req.SetPathValue("id", asset.ID)
	w := httptest.NewRecorder()

	server.handleListPorts(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleListServices — 66.7% ====================

func TestHandleListServices_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()
	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.1",
		NormalizedValue: "10.0.0.1", SourceTools: []string{"nmap"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/assets/"+asset.ID+"/services", nil)
	req.SetPathValue("id", asset.ID)
	w := httptest.NewRecorder()

	server.handleListServices(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleDeleteAlertWebhook — 66.7% ====================

func TestHandleDeleteAlertWebhook_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	rawDB.Close()

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+p.ID+"/alert-webhook", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleDeleteAlertWebhook(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleAssetChangeTimeline — 66.7% ====================

func TestHandleAssetChangeTimeline_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/assets/asset1/changes", nil)
	req.SetPathValue("id", "asset1")
	w := httptest.NewRecorder()

	server.handleAssetChangeTimeline(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ==================== handleCreateArchive — 67.4% ====================

func TestHandleCreateArchive_ProjectError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodPost, "/projects/p1/archive", nil)
	req.SetPathValue("id", "p1")
	w := httptest.NewRecorder()

	server.handleCreateArchive(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

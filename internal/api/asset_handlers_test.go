package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- parsePortFromURL (pure function) ---

func TestParsePortFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want int
	}{
		{"explicit port", "http://example.com:8080/path", 8080},
		{"https default", "https://example.com/path", 443},
		{"http default", "http://example.com/path", 80},
		{"no scheme", "example.com:3000", 0},
		{"invalid url", "://bad", 0},
		{"empty", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePortFromURL(tt.url)
			if got != tt.want {
				t.Errorf("parsePortFromURL(%q) = %d, want %d", tt.url, got, tt.want)
			}
		})
	}
}

// --- appendUnique (pure function) ---

func TestAppendUnique(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
		want  int
	}{
		{"empty", []string{}, "a", 1},
		{"new", []string{"a", "b"}, "c", 3},
		{"duplicate", []string{"a", "b"}, "b", 2},
		{"nil slice", nil, "a", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendUnique(tt.slice, tt.s)
			if len(got) != tt.want {
				t.Errorf("len = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestAppendUnique_PreservesOrder(t *testing.T) {
	got := appendUnique([]string{"a", "b"}, "c")
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("order wrong: %v", got)
	}
}

// --- handleListAssets ---

func TestHandleListAssets_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	for _, v := range []string{"a.com", "b.com"} {
		queries.CreateAsset(&models.Asset{
			ID: util.GenerateID(), ProjectID: p.ID,
			Type: models.AssetTypeDomain, Value: v,
			NormalizedValue: v, SourceTools: []string{"subfinder"},
			FirstSeen: now, LastSeen: now,
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssets(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result []*models.Asset
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

func TestHandleListAssets_Empty(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssets(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// --- handleListPorts ---

func TestHandleListPorts_Success(t *testing.T) {
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

	// Create port records
	queries.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID,
		Port: 80, Protocol: "tcp", State: "open", SourceTool: "naabu",
		CreatedAt: now,
	})
	queries.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID,
		Port: 443, Protocol: "tcp", State: "open", SourceTool: "naabu",
		CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/"+asset.ID+"/ports", nil)
	req.SetPathValue("id", asset.ID)
	w := httptest.NewRecorder()

	server.handleListPorts(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result []*models.Port
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

// --- handleListServices ---

func TestHandleListServices_Success(t *testing.T) {
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

	queries.CreateService(&models.Service{
		ID: util.GenerateID(), AssetID: asset.ID,
		Name: "http", Product: "Apache", Version: "2.4",
		Confidence: 95, SourceTool: "nmap", CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/"+asset.ID+"/services", nil)
	req.SetPathValue("id", asset.ID)
	w := httptest.NewRecorder()

	server.handleListServices(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result []*models.Service
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len = %d, want 1", len(result))
	}
}

// --- handleListServicePorts ---

func TestHandleListServicePorts_Merge(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	// Create IP asset
	asset := &models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeIP, Value: "10.0.0.1",
		NormalizedValue: "10.0.0.1", SourceTools: []string{"nmap"},
		FirstSeen: now, LastSeen: now,
	}
	queries.CreateAsset(asset)

	// Create port from naabu
	queries.CreatePort(&models.Port{
		ID: util.GenerateID(), AssetID: asset.ID,
		Port: 8080, Protocol: "tcp", State: "open", SourceTool: "naabu",
		CreatedAt: now,
	})

	// Create service fingerprint from nmap (same port)
	queries.SaveServiceFingerprint(&models.ServiceFingerprint{
		ID: util.GenerateID(), ProjectID: p.ID,
		IP: "10.0.0.1", Port: 8080, Protocol: "tcp",
		Service: "http", Product: "Tomcat", Version: "9.0",
		IsWeb: true, Source: "nmap", CreatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/service-ports", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListServicePorts(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result []*models.ServicePort
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1 (merged)", len(result))
	}
	sp := result[0]
	if sp.ServiceName != "http" {
		t.Errorf("service = %q, want http", sp.ServiceName)
	}
	if sp.Product != "Tomcat" {
		t.Errorf("product = %q, want Tomcat", sp.Product)
	}
}

// --- handleListAssetsFiltered ---

func TestHandleListAssetsFiltered_TechnologyFilter(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	p := createTestProject(t, queries)
	now := time.Now().UTC()

	queries.CreateAsset(&models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "a.com",
		NormalizedValue: "a.com", SourceTools: []string{"subfinder"},
		Tags: map[string]string{"server": "nginx"},
		FirstSeen: now, LastSeen: now,
	})
	queries.CreateAsset(&models.Asset{
		ID: util.GenerateID(), ProjectID: p.ID,
		Type: models.AssetTypeDomain, Value: "b.com",
		NormalizedValue: "b.com", SourceTools: []string{"subfinder"},
		Tags: map[string]string{"server": "apache"},
		FirstSeen: now, LastSeen: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+p.ID+"/assets?technology=nginx", nil)
	req.SetPathValue("id", p.ID)
	w := httptest.NewRecorder()

	server.handleListAssetsFiltered(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result PaginatedResponse[*models.Asset]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("len = %d, want 1", len(result.Data))
	}
}

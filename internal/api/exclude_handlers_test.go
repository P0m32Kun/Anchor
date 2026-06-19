package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/exclude"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- helpers ---

func createTestExcludedDomain(t *testing.T, queries *db.Queries, domain string, builtin bool) *models.ExcludedDomain {
	t.Helper()
	d := &models.ExcludedDomain{
		ID:        util.GenerateID(),
		Domain:    domain,
		Reason:    "test reason",
		Builtin:   builtin,
		CreatedAt: time.Now().UTC(),
	}
	if err := queries.CreateExcludedDomain(d); err != nil {
		t.Fatalf("create excluded domain: %v", err)
	}
	return d
}

// ============================================================
// Pure function tests
// ============================================================

func TestIndexOf(t *testing.T) {
	tests := []struct {
		s    string
		c    byte
		want int
	}{
		{"hello", 'l', 2},
		{"hello", 'o', 4},
		{"hello", 'h', 0},
		{"hello", 'x', -1},
		{"", 'a', -1},
		{"a", 'a', 0},
		{":8080", ':', 0},
		{"/path", '/', 0},
	}
	for _, tt := range tests {
		t.Run(string(tt.s)+"/"+string(tt.c), func(t *testing.T) {
			got := indexOf(tt.s, tt.c)
			if got != tt.want {
				t.Errorf("indexOf(%q, %q) = %d, want %d", tt.s, tt.c, got, tt.want)
			}
		})
	}
}

func TestExtractHostFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https simple", "https://example.com", "example.com"},
		{"http simple", "http://example.com", "example.com"},
		{"https with path", "https://example.com/path/to/resource", "example.com"},
		{"http with path", "http://example.com/path", "example.com"},
		{"https with port", "https://example.com:8443", "example.com"},
		{"http with port", "http://example.com:8080", "example.com"},
		{"https with port and path", "https://example.com:443/path", "example.com"},
		{"plain domain (no scheme)", "example.com", "example.com"},
		{"plain domain with port", "example.com:8080", "example.com"},
		{"plain domain with path", "example.com/path", "example.com"},
		{"subdomain https", "https://api.example.com", "api.example.com"},
		{"subdomain with port", "https://api.example.com:443/v1", "api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHostFromURL(tt.url)
			if got != tt.want {
				t.Errorf("extractHostFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{"plain domain", "example.com", "example.com"},
		{"uppercase", "EXAMPLE.COM", "example.com"},
		{"mixed case", "ExAmPlE.CoM", "example.com"},
		{"leading/trailing spaces", "  example.com  ", "example.com"},
		{"empty string", "", ""},
		{"only spaces", "   ", ""},
		{"https url", "https://Example.COM", "example.com"},
		{"http url", "http://Example.COM", "example.com"},
		{"https url with path", "HTTPS://Example.COM/path", "example.com"},
		{"https url with port", "https://example.com:443", "example.com"},
		{"tab and newline", "\texample.com\n", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDomain(tt.domain)
			if got != tt.want {
				t.Errorf("normalizeDomain(%q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestGetExclusionReason(t *testing.T) {
	// A known built-in domain
	builtinDomain := exclude.DefaultDomains[0]

	tests := []struct {
		name    string
		domain  string
		excluded bool
		want    string
	}{
		{"not excluded", "safe.example.com", false, "not excluded"},
		{"builtin excluded", builtinDomain, true, "built-in default"},
		{"custom excluded", "custom.example.com", true, "custom exclusion"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExclusionReason(tt.domain, tt.excluded)
			if got != tt.want {
				t.Errorf("getExclusionReason(%q, %v) = %q, want %q", tt.domain, tt.excluded, got, tt.want)
			}
		})
	}
}

// ============================================================
// handleListExcludedDomains
// ============================================================

func TestHandleListExcludedDomains_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "custom.example.com", false)

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains", nil)
	w := httptest.NewRecorder()

	server.handleListExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := result["builtin"]; !ok {
		t.Error("missing 'builtin' key")
	}
	if _, ok := result["custom"]; !ok {
		t.Error("missing 'custom' key")
	}
	if _, ok := result["total"]; !ok {
		t.Error("missing 'total' key")
	}

	// Verify custom list contains our domain
	custom, ok := result["custom"].([]interface{})
	if !ok {
		t.Fatal("custom is not an array")
	}
	if len(custom) != 1 {
		t.Errorf("len(custom) = %d, want 1", len(custom))
	}
}

func TestHandleListExcludedDomains_NoCustom(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains", nil)
	w := httptest.NewRecorder()

	server.handleListExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Custom list should be empty or nil (no custom domains added)
	custom := result["custom"]
	if custom != nil {
		arr, ok := custom.([]interface{})
		if ok && len(arr) > 0 {
			t.Errorf("expected no custom domains, got %d", len(arr))
		}
	}

	// Total should be > 0 because builtin domains are seeded by migration
	total, ok := result["total"].(float64)
	if !ok || total <= 0 {
		t.Errorf("total = %v, want > 0 (builtins seeded by migration)", total)
	}
}

// ============================================================
// handleListDefaultDomains
// ============================================================

func TestHandleListDefaultDomains_Success(t *testing.T) {
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
		t.Error("missing 'domains' key")
	}
	if _, ok := result["total"]; !ok {
		t.Error("missing 'total' key")
	}

	// Verify total matches default domains count
	total, _ := result["total"].(float64)
	if int(total) != len(exclude.DefaultDomains) {
		t.Errorf("total = %d, want %d", int(total), len(exclude.DefaultDomains))
	}
}

// ============================================================
// handleAddExcludedDomain
// ============================================================

func TestHandleAddExcludedDomain_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{
		"domain": "evil.example.com",
		"reason": "malware C2",
	})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created models.ExcludedDomain
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Domain != "evil.example.com" {
		t.Errorf("domain = %q, want evil.example.com", created.Domain)
	}
	if created.Builtin {
		t.Error("expected builtin=false")
	}
	if created.Reason != "malware C2" {
		t.Errorf("reason = %q, want malware C2", created.Reason)
	}
	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestHandleAddExcludedDomain_EmptyDomain(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"domain": ""})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleAddExcludedDomain_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleAddExcludedDomain_Duplicate(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "dup.example.com", false)

	body, _ := json.Marshal(map[string]string{"domain": "dup.example.com"})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestHandleAddExcludedDomain_NormalizesDomain(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"domain": "  HTTPS://Example.COM/Path  "})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created models.ExcludedDomain
	json.NewDecoder(resp.Body).Decode(&created)
	if created.Domain != "example.com" {
		t.Errorf("domain = %q, want example.com", created.Domain)
	}
}

func TestHandleAddExcludedDomain_WhitespaceOnly(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"domain": "   \t  "})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ============================================================
// handleDeleteExcludedDomain
// ============================================================

func TestHandleDeleteExcludedDomain_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "deleteme.example.com", false)

	req := httptest.NewRequest(http.MethodDelete, "/excluded-domains/deleteme.example.com", nil)
	req.SetPathValue("domain", "deleteme.example.com")
	w := httptest.NewRecorder()

	server.handleDeleteExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "deleted" {
		t.Errorf("status = %q, want deleted", result["status"])
	}
}

func TestHandleDeleteExcludedDomain_EmptyDomain(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/excluded-domains/", nil)
	req.SetPathValue("domain", "")
	w := httptest.NewRecorder()

	server.handleDeleteExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleDeleteExcludedDomain_BuiltinForbidden(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Use a known built-in domain
	if len(exclude.DefaultDomains) == 0 {
		t.Skip("no default domains configured")
	}
	builtinDomain := exclude.DefaultDomains[0]

	req := httptest.NewRequest(http.MethodDelete, "/excluded-domains/"+builtinDomain, nil)
	req.SetPathValue("domain", builtinDomain)
	w := httptest.NewRecorder()

	server.handleDeleteExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestHandleDeleteExcludedDomain_NormalizesDomain(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "normalize-me.example.com", false)

	req := httptest.NewRequest(http.MethodDelete, "/excluded-domains/Normalize-Me.EXAMPLE.COM", nil)
	req.SetPathValue("domain", "Normalize-Me.EXAMPLE.COM")
	w := httptest.NewRecorder()

	server.handleDeleteExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// ============================================================
// handleBatchAddExcludedDomains
// ============================================================

func TestHandleBatchAddExcludedDomains_Success(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"domains": []map[string]string{
			{"domain": "batch1.example.com", "reason": "reason1"},
			{"domain": "batch2.example.com", "reason": "reason2"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchAddExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created, ok := result["created"].(float64); !ok || created != 2 {
		t.Errorf("created = %v, want 2", result["created"])
	}
}

func TestHandleBatchAddExcludedDomains_InvalidBody(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/batch", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchAddExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleBatchAddExcludedDomains_SkipsDuplicates(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "existing.example.com", false)

	body, _ := json.Marshal(map[string]interface{}{
		"domains": []map[string]string{
			{"domain": "existing.example.com"},
			{"domain": "new.example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchAddExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if created, ok := result["created"].(float64); !ok || created != 1 {
		t.Errorf("created = %v, want 1", result["created"])
	}
}

func TestHandleBatchAddExcludedDomains_SkipsEmptyDomains(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"domains": []map[string]string{
			{"domain": ""},
			{"domain": "   "},
			{"domain": "valid.example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchAddExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if created, ok := result["created"].(float64); !ok || created != 1 {
		t.Errorf("created = %v, want 1", result["created"])
	}
}

func TestHandleBatchAddExcludedDomains_EmptyList(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"domains": []map[string]string{},
	})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchAddExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if created, ok := result["created"].(float64); !ok || created != 0 {
		t.Errorf("created = %v, want 0", result["created"])
	}
}

func TestHandleBatchAddExcludedDomains_NormalizesDomains(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"domains": []map[string]string{
			{"domain": "HTTPS://BATCH-NORM.COM/path"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchAddExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	domains, ok := result["domains"].([]interface{})
	if !ok || len(domains) == 0 {
		t.Fatal("domains is empty or not an array")
	}
	first := domains[0].(map[string]interface{})
	if first["domain"] != "batch-norm.com" {
		t.Errorf("domain = %v, want batch-norm.com", first["domain"])
	}
}

// ============================================================
// handleResetExcludedDomains
// ============================================================

func TestHandleResetExcludedDomains_Success(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "custom1.example.com", false)
	createTestExcludedDomain(t, queries, "custom2.example.com", false)

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/reset", nil)
	w := httptest.NewRecorder()

	server.handleResetExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "reset" {
		t.Errorf("status = %q, want reset", result["status"])
	}

	// Verify custom domains are deleted
	domains, _ := queries.ListExcludedDomains()
	for _, d := range domains {
		if !d.Builtin {
			t.Errorf("custom domain %q still exists after reset", d.Domain)
		}
	}
}

func TestHandleResetExcludedDomains_Idempotent(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Reset when there are no custom domains should succeed
	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/reset", nil)
	w := httptest.NewRecorder()

	server.handleResetExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// ============================================================
// handleCheckExcludedDomain
// ============================================================

func TestHandleCheckExcludedDomain_Excluded(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "blocked.example.com", false)

	// Reload exclude manager
	queries.LoadCustomExcludedDomains(server.excludeMgr)

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains/check?domain=blocked.example.com", nil)
	w := httptest.NewRecorder()

	server.handleCheckExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["excluded"] != true {
		t.Errorf("excluded = %v, want true", result["excluded"])
	}
	if result["reason"] != "custom exclusion" {
		t.Errorf("reason = %v, want custom exclusion", result["reason"])
	}
}

func TestHandleCheckExcludedDomain_NotExcluded(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains/check?domain=safe.example.com", nil)
	w := httptest.NewRecorder()

	server.handleCheckExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["excluded"] != false {
		t.Errorf("excluded = %v, want false", result["excluded"])
	}
	if result["reason"] != "not excluded" {
		t.Errorf("reason = %v, want not excluded", result["reason"])
	}
}

func TestHandleCheckExcludedDomain_MissingParam(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains/check", nil)
	w := httptest.NewRecorder()

	server.handleCheckExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleCheckExcludedDomain_BuiltinDomain(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	if len(exclude.DefaultDomains) == 0 {
		t.Skip("no default domains configured")
	}
	builtinDomain := exclude.DefaultDomains[0]

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains/check?domain="+builtinDomain, nil)
	w := httptest.NewRecorder()

	server.handleCheckExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["excluded"] != true {
		t.Errorf("excluded = %v, want true for builtin domain %q", result["excluded"], builtinDomain)
	}
	if result["reason"] != "built-in default" {
		t.Errorf("reason = %v, want built-in default", result["reason"])
	}
}

func TestHandleCheckExcludedDomain_SubdomainOfBuiltin(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	if len(exclude.DefaultDomains) == 0 {
		t.Skip("no default domains configured")
	}
	// Use a subdomain of a known builtin — should be excluded via subdomain matching
	builtinDomain := exclude.DefaultDomains[0]
	subdomain := "sub." + builtinDomain

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains/check?domain="+subdomain, nil)
	w := httptest.NewRecorder()

	server.handleCheckExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["excluded"] != true {
		t.Errorf("excluded = %v, want true for subdomain of builtin %q", result["excluded"], builtinDomain)
	}
}

// ============================================================
// Error paths (DB failures)
// ============================================================

func TestHandleListExcludedDomains_Error(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodGet, "/excluded-domains", nil)
	w := httptest.NewRecorder()

	server.handleListExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleAddExcludedDomain_ExistsCheckError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	body, _ := json.Marshal(map[string]string{"domain": "test.example.com"})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAddExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleDeleteExcludedDomain_Error(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	createTestExcludedDomain(t, queries, "custom-del.example.com", false)

	rawDB.Close()

	req := httptest.NewRequest(http.MethodDelete, "/excluded-domains/custom-del.example.com", nil)
	req.SetPathValue("domain", "custom-del.example.com")
	w := httptest.NewRecorder()

	server.handleDeleteExcludedDomain(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleBatchAddExcludedDomains_DBError(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	body, _ := json.Marshal(map[string]interface{}{
		"domains": []map[string]string{
			{"domain": "batch-err.example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBatchAddExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Batch handler continues on error, returns 201 with created=0
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestHandleResetExcludedDomains_Error(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	rawDB.Close()

	req := httptest.NewRequest(http.MethodPost, "/excluded-domains/reset", nil)
	w := httptest.NewRecorder()

	server.handleResetExcludedDomains(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

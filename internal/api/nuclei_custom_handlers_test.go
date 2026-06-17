package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/nuclei/custom"
)

func TestNucleiCustom_ListBuiltinOnly(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	server.nucleiCustomMgr = custom.NewManager(server.queries)
	mux := http.NewServeMux()
	server.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/nuclei/custom/sources", nil)
	req.Header.Set("Authorization", "Bearer "+server.apiToken)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list status: %d body=%s", w.Code, w.Body.String())
	}
	var list []*models.NucleiCustomSource
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, s := range list {
		if s.Builtin {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected builtin RBKD source in list: %+v", list)
	}
}

func TestNucleiCustom_PatchEnabledBuiltin(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	t.Cleanup(cleanup)

	server.nucleiCustomMgr = custom.NewManager(server.queries)
	mux := http.NewServeMux()
	server.Register(mux)

	body, _ := json.Marshal(map[string]bool{"enabled": false})
	req := httptest.NewRequest(http.MethodPatch, "/nuclei/custom/sources/builtin:rbkd-templates/enabled", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+server.apiToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		b, _ := io.ReadAll(w.Body)
		t.Fatalf("patch enabled status: %d body=%s", w.Code, string(b))
	}
	var updated models.NucleiCustomSource
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Enabled {
		t.Error("expected enabled=false after patch")
	}
}

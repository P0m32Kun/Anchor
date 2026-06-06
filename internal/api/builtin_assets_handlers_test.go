package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func writeFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func decodeJSON[T any](t *testing.T, rr *httptest.ResponseRecorder) []T {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var out []T
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode json: %v body=%s", err, rr.Body.String())
	}
	return out
}

func TestBuiltinAssetsSeededAndListable(t *testing.T) {
	t.Setenv("ANCHOR_BUILTIN_SYNC", "off")

	dictRoot := t.TempDir()
	// dictionary seed expects: <root>/<category>/*.txt
	writeFile(t, filepath.Join(dictRoot, "path", "test.txt"), []byte("alpha\nbeta\n"))

	fingerRoot := t.TempDir()
	// httpx seed expects: <root>/finger.json (content is not parsed, only existence)
	writeFile(t, filepath.Join(fingerRoot, "finger.json"), []byte("{}"))

	templatesRoot := t.TempDir()

	t.Setenv("ANCHOR_BUILTIN_DICT_ROOT", dictRoot)
	t.Setenv("ANCHOR_BUILTIN_FINGER_ROOT", fingerRoot)
	t.Setenv("ANCHOR_BUILTIN_TEMPLATES_ROOT", templatesRoot)

	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("nuclei custom sources includes enabled builtin", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/nuclei/custom/sources", nil)
		server.handleListNucleiCustomSources(rr, req)

		sources := decodeJSON[*models.NucleiCustomSource](t, rr)
		found := false
		for _, s := range sources {
			if s != nil && s.ID == "builtin:rbkd-templates" {
				found = true
				if !s.Builtin {
					t.Fatalf("builtin template expected Builtin=true id=%s", s.ID)
				}
				if !s.Enabled {
					t.Fatalf("builtin template expected Enabled=true id=%s", s.ID)
				}
				break
			}
		}
		if !found {
			t.Fatalf("missing builtin nuclei custom source id=builtin:rbkd-templates total=%d", len(sources))
		}
	})

	t.Run("httpx fingerprints includes enabled builtin", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/httpx/fingerprints", nil)
		server.handleListHttpxFingerprints(rr, req)

		fps := decodeJSON[*models.HttpxFingerprint](t, rr)
		found := false
		for _, fp := range fps {
			if fp != nil && fp.ID == "builtin:rbkd-finger" {
				found = true
				if !fp.Builtin {
					t.Fatalf("builtin fingerprint expected Builtin=true id=%s", fp.ID)
				}
				if !fp.Enabled {
					t.Fatalf("builtin fingerprint expected Enabled=true id=%s", fp.ID)
				}
				break
			}
		}
		if !found {
			t.Fatalf("missing builtin httpx fingerprint id=builtin:rbkd-finger total=%d", len(fps))
		}
	})

	t.Run("dictionaries includes enabled builtin", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/dictionaries", nil)
		server.handleListDictionaries(rr, req)

		ds := decodeJSON[*models.Dictionary](t, rr)
		found := false
		wantID := "builtin:path/test.txt"
		for _, d := range ds {
			if d != nil && d.ID == wantID {
				found = true
				if !d.Builtin {
					t.Fatalf("builtin dictionary expected Builtin=true id=%s", d.ID)
				}
				if !d.Enabled {
					t.Fatalf("builtin dictionary expected Enabled=true id=%s", d.ID)
				}
				break
			}
		}
		if !found {
			t.Fatalf("missing builtin dictionary id=%s total=%d", wantID, len(ds))
		}
	})
}

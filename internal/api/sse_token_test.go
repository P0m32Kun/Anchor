package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestSSEToken_IssueAndValidate(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	projectID := "proj-1"
	token, err := server.GenerateSSEToken(projectID)
	if err != nil {
		t.Fatalf("GenerateSSEToken: %v", err)
	}

	claims, err := server.ValidateSSEToken(token)
	if err != nil {
		t.Fatalf("ValidateSSEToken: %v", err)
	}
	if claims == nil {
		t.Fatalf("ValidateSSEToken returned nil claims")
	}
	if claims.ProjectID != projectID {
		t.Fatalf("ProjectID=%q want %q", claims.ProjectID, projectID)
	}
	if claims.Type != sseTokenTypeValue {
		t.Fatalf("Type=%q want %q", claims.Type, sseTokenTypeValue)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now().Add(30*time.Minute)) {
		t.Fatalf("ExpiresAt=%v expected to be in the near future", claims.ExpiresAt)
	}
}

func TestSSEAuthMiddleware_ProjectMismatch(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	projectA := "proj-a"
	projectB := "proj-b"

	tokenA, err := server.GenerateSSEToken(projectA)
	if err != nil {
		t.Fatalf("GenerateSSEToken: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := server.SSEAuthMiddleware(next)

	t.Run("token project matches path id", func(t *testing.T) {
		rr := httptest.NewRecorder()
		q := url.Values{}
		q.Set("token", tokenA)
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectA+"/events?"+q.Encode(), nil)
		req.SetPathValue("id", projectA)

		mw.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
	})

	t.Run("token project mismatches path id", func(t *testing.T) {
		rr := httptest.NewRecorder()
		q := url.Values{}
		q.Set("token", tokenA)
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectB+"/events?"+q.Encode(), nil)
		req.SetPathValue("id", projectB)

		mw.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d want %d body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
		}
	})
}

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/errors"
)

// --- Error helpers ---

func writeError(w http.ResponseWriter, status int, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// --- CORS middleware ---

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TokenAuthMiddleware verifies the Bearer token for all protected routes.
func (s *Server) TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) || auth[len(prefix):] != s.apiToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{"message": "Unauthorized: invalid or missing token"},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SSEAuthMiddleware verifies authentication for SSE endpoints.
// It accepts two authentication methods:
//  1. Standard Bearer token (Authorization header)
//  2. Short-lived SSE JWT token (query parameter "token")
//
// The JWT path also validates that the token's project_id claim matches
// the URL path parameter to prevent cross-project SSE access.
func (s *Server) SSEAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Try standard Bearer token first
		authHeader := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(authHeader, prefix) && authHeader[len(prefix):] == s.apiToken {
			next.ServeHTTP(w, r)
			return
		}

		// Try SSE JWT token from query parameter
		if tokenStr := r.URL.Query().Get("token"); tokenStr != "" {
			claims, err := s.ValidateSSEToken(tokenStr)
			if err == nil && claims.Type == sseTokenTypeValue {
				// Verify project_id matches URL path to prevent cross-project access
				projectID := r.PathValue("id")
				if projectID == "" || claims.ProjectID == projectID {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		// Authentication failed
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "Unauthorized: invalid or missing token"},
		})
	})
}

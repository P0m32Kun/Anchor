package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/golang-jwt/jwt/v5"
)

// SSE token configuration
const (
	sseTokenExpiry    = 1 * time.Hour // Short-lived token
	sseTokenIssuer    = "anchor-sse"
	sseTokenTypeValue = "sse"
)

// SSEClaims represents the JWT claims for SSE tokens
type SSEClaims struct {
	jwt.RegisteredClaims
	ProjectID string `json:"project_id"`
	Type      string `json:"type"` // "sse"
}

// sseTokenSecret generates a deterministic secret from the API token
// This avoids storing an additional secret while maintaining security
func (s *Server) sseTokenSecret() []byte {
	// Use HMAC-SHA256 to derive a secret from the API token
	h := hmac.New(sha256.New, []byte("anchor-sse-secret-key"))
	h.Write([]byte(s.apiToken))
	return h.Sum(nil)
}

// GenerateSSEToken creates a short-lived JWT token for SSE connections
func (s *Server) GenerateSSEToken(projectID string) (string, error) {
	// Generate a random token ID for revocation support
	tokenIDBytes := make([]byte, 16)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return "", fmt.Errorf("failed to generate token ID: %w", err)
	}
	tokenID := hex.EncodeToString(tokenIDBytes)

	now := time.Now()
	claims := SSEClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    sseTokenIssuer,
			Subject:   projectID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(sseTokenExpiry)),
			ID:        tokenID,
		},
		ProjectID: projectID,
		Type:      sseTokenTypeValue,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.sseTokenSecret())
	if err != nil {
		return "", fmt.Errorf("failed to sign SSE token: %w", err)
	}

	return tokenString, nil
}

// ValidateSSEToken validates an SSE token and returns the claims
func (s *Server) ValidateSSEToken(tokenString string) (*SSEClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SSEClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.sseTokenSecret(), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse SSE token: %w", err)
	}

	claims, ok := token.Claims.(*SSEClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid SSE token")
	}

	// Verify token type
	if claims.Type != sseTokenTypeValue {
		return nil, fmt.Errorf("invalid token type: %s", claims.Type)
	}

	return claims, nil
}

// handleIssueSSEToken handles POST /projects/{id}/sse-token
func (s *Server) handleIssueSSEToken(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing project id"))
		return
	}

	// Verify project exists
	_, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	// Generate SSE token
	token, err := s.GenerateSSEToken(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New(errors.ErrInternal, "failed to generate SSE token"))
		return
	}

	// Return token
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"expires_in": int(sseTokenExpiry.Seconds()),
		"project_id": projectID,
	})
}

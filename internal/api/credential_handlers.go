package api

import (
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/credentials"
	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/sources"
)

// handleListDiscoveredCredentials 列出所有已发现的凭证
func (s *Server) handleListDiscoveredCredentials(w http.ResponseWriter, r *http.Request) {
	config := credentials.DefaultDiscoveryConfig()
	discoverer := credentials.NewDiscoverer(config)

	creds, err := discoverer.DiscoverAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DISCOVERY_ERROR", err.Error()))
		return
	}

	// 脱敏处理：不返回实际的 token/key
	type safeCredential struct {
		Platform string `json:"platform"`
		Username string `json:"username,omitempty"`
		HasAPI   bool   `json:"has_api"`
		HasToken bool   `json:"has_token"`
		Source   string `json:"source"`
	}

	result := make([]safeCredential, 0, len(creds))
	for _, cred := range creds {
		result = append(result, safeCredential{
			Platform: cred.Platform,
			Username: cred.Username,
			HasAPI:   cred.APIKey != "",
			HasToken: cred.Token != "",
			Source:   cred.Source,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"credentials": result,
		"count":       len(result),
	})
}

// handleListSources 列出所有支持的 SRC 平台
func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	registry := sources.NewRegistry()

	platforms := registry.List()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sources": platforms,
		"count":   len(platforms),
	})
}

// handleListSourcesByType 按类型列出 SRC 平台
func (s *Server) handleListSourcesByType(w http.ResponseWriter, r *http.Request) {
	platformType := r.URL.Query().Get("type")
	if platformType == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAM", "type parameter is required"))
		return
	}

	registry := sources.NewRegistry()
	platforms := registry.ListByType(sources.PlatformType(platformType))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sources": platforms,
		"count":   len(platforms),
	})
}

// handleGetSource 获取指定平台信息
func (s *Server) handleGetSource(w http.ResponseWriter, r *http.Request) {
	platformID := r.PathValue("id")
	if platformID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAM", "Platform ID is required"))
		return
	}

	registry := sources.NewRegistry()
	platform, ok := registry.Get(platformID)
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Platform not found"))
		return
	}

	// 检查是否有凭证
	config := credentials.DefaultDiscoveryConfig()
	discoverer := credentials.NewDiscoverer(config)
	hasCred := discoverer.HasCredential(platformID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"source":         platform,
		"has_credential": hasCred,
	})
}

// handleCheckCredential 检查指定平台是否有凭证
func (s *Server) handleCheckCredential(w http.ResponseWriter, r *http.Request) {
	platformID := r.PathValue("id")
	if platformID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PARAM", "Platform ID is required"))
		return
	}

	config := credentials.DefaultDiscoveryConfig()
	discoverer := credentials.NewDiscoverer(config)

	cred, err := discoverer.GetCredentialForPlatform(platformID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"platform": platformID,
			"found":    false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"platform": platformID,
		"found":    true,
		"username": cred.Username,
		"source":   cred.Source,
	})
}

// handleListCredentialPlatforms 列出所有有凭证的平台
func (s *Server) handleListCredentialPlatforms(w http.ResponseWriter, r *http.Request) {
	config := credentials.DefaultDiscoveryConfig()
	discoverer := credentials.NewDiscoverer(config)

	platforms := discoverer.ListPlatforms()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"platforms": platforms,
		"count":     len(platforms),
	})
}

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/exclude"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// handleListExcludedDomains returns all excluded domains (built-in + custom).
func (s *Server) handleListExcludedDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := s.queries.ListExcludedDomains()
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list excluded domains failed: %v", err))
		return
	}

	// Separate built-in and custom
	var builtin []*models.ExcludedDomain
	var custom []*models.ExcludedDomain
	for _, d := range domains {
		if d.Builtin {
			builtin = append(builtin, d)
		} else {
			custom = append(custom, d)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"builtin": builtin,
		"custom":  custom,
		"total":   len(domains),
	})
}

// handleListDefaultDomains returns the built-in default exclusion list.
func (s *Server) handleListDefaultDomains(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domains": exclude.DefaultDomains,
		"total":   len(exclude.DefaultDomains),
	})
}

// handleAddExcludedDomain adds a custom domain to the exclusion list.
func (s *Server) handleAddExcludedDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	domain := normalizeDomain(req.Domain)
	if domain == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "domain is required"))
		return
	}

	// Check if already exists
	exists, err := s.queries.ExcludedDomainExists(domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "check domain failed: %v", err))
		return
	}
	if exists {
		writeError(w, http.StatusConflict, errors.New(errors.ErrConflict, "domain already in exclusion list"))
		return
	}

	d := &models.ExcludedDomain{
		ID:        util.GenerateID(),
		Domain:    domain,
		Reason:    req.Reason,
		Builtin:   false,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.queries.CreateExcludedDomain(d); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create excluded domain failed: %v", err))
		return
	}

	// Reload manager
	if err := s.queries.LoadCustomExcludedDomains(s.excludeMgr); err != nil {
		// Log but don't fail
		_ = err
	}

	writeJSON(w, http.StatusCreated, d)
}

// handleDeleteExcludedDomain removes a custom domain from the exclusion list.
func (s *Server) handleDeleteExcludedDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.PathValue("domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "domain is required"))
		return
	}

	domain = normalizeDomain(domain)

	// Check if it's a built-in domain
	if exclude.IsDefaultDomain(domain) {
		writeError(w, http.StatusForbidden, errors.New(errors.ErrForbidden, "cannot delete built-in domains"))
		return
	}

	if err := s.queries.DeleteExcludedDomain(domain); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete excluded domain failed: %v", err))
		return
	}

	// Reload manager
	if err := s.queries.LoadCustomExcludedDomains(s.excludeMgr); err != nil {
		_ = err
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleBatchAddExcludedDomains adds multiple custom domains at once.
func (s *Server) handleBatchAddExcludedDomains(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domains []struct {
			Domain string `json:"domain"`
			Reason string `json:"reason"`
		} `json:"domains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "invalid request body").WithDetail(err.Error()))
		return
	}

	now := time.Now().UTC()
	var created []*models.ExcludedDomain
	for _, item := range req.Domains {
		domain := normalizeDomain(item.Domain)
		if domain == "" {
			continue
		}

		exists, err := s.queries.ExcludedDomainExists(domain)
		if err != nil || exists {
			continue
		}

		d := &models.ExcludedDomain{
			ID:        util.GenerateID(),
			Domain:    domain,
			Reason:    item.Reason,
			Builtin:   false,
			CreatedAt: now,
		}
		if err := s.queries.CreateExcludedDomain(d); err != nil {
			continue
		}
		created = append(created, d)
	}

	// Reload manager
	if err := s.queries.LoadCustomExcludedDomains(s.excludeMgr); err != nil {
		_ = err
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"created": len(created),
		"domains": created,
	})
}

// handleResetExcludedDomains resets custom exclusions (keeps built-in).
func (s *Server) handleResetExcludedDomains(w http.ResponseWriter, r *http.Request) {
	if err := s.queries.DeleteAllCustomExcludedDomains(); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "reset excluded domains failed: %v", err))
		return
	}

	// Clear manager custom list
	s.excludeMgr.SetCustomList(make(map[string]string))

	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

// handleCheckExcludedDomain checks if a domain would be excluded.
func (s *Server) handleCheckExcludedDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "domain parameter is required"))
		return
	}

	excluded := s.excludeMgr.IsExcluded(domain)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":    domain,
		"excluded":  excluded,
		"reason":    getExclusionReason(domain, excluded),
	})
}

// getExclusionReason returns a human-readable reason for the exclusion.
func getExclusionReason(domain string, excluded bool) string {
	if !excluded {
		return "not excluded"
	}
	if exclude.IsDefaultDomain(domain) {
		return "built-in default"
	}
	return "custom exclusion"
}

// normalizeDomain normalizes a domain string.
func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return ""
	}
	// Remove protocol if present
	if len(domain) > 8 && (domain[:7] == "http://" || domain[:8] == "https://") {
		domain = extractHostFromURL(domain)
	}
	return domain
}

// extractHostFromURL extracts host from URL.
func extractHostFromURL(u string) string {
	// Simple extraction
	if len(u) > 8 && u[:8] == "https://" {
		u = u[8:]
	} else if len(u) > 7 && u[:7] == "http://" {
		u = u[7:]
	}
	if idx := indexOf(u, '/'); idx >= 0 {
		u = u[:idx]
	}
	if idx := indexOf(u, ':'); idx >= 0 {
		u = u[:idx]
	}
	return u
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

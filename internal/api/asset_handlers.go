package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// GET /projects/{id}/assets (with filtering)
func (s *Server) handleListAssetsFiltered(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	// Parse filter params
	statusCode := r.URL.Query().Get("status_code")
	portStr := r.URL.Query().Get("port")
	titleQuery := r.URL.Query().Get("title")
	techQuery := r.URL.Query().Get("technology")

	assets, err := s.queries.ListAssetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list assets: %v", err))
		return
	}

	// Apply filters in memory (for MVP; optimize with SQL later)
	var filtered []*models.Asset
	for _, a := range assets {
		// Status code filter applies to web_endpoints, not assets directly
		// Skip for now - needs JOIN query
		_ = statusCode

		// Port filter
		if portStr != "" {
			port, _ := strconv.Atoi(portStr)
			ports, _ := s.queries.ListPortsByAsset(a.ID)
			found := false
			for _, p := range ports {
				if p.Port == port {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Title filter (on web_endpoints)
		if titleQuery != "" {
			eps, _ := s.queries.ListWebEndpointsByAsset(a.ID)
			found := false
			for _, ep := range eps {
				if strings.Contains(strings.ToLower(ep.Title), strings.ToLower(titleQuery)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Technology filter
		if techQuery != "" {
			found := false
			for k, v := range a.Tags {
				if strings.Contains(strings.ToLower(k), strings.ToLower(techQuery)) || strings.Contains(strings.ToLower(v), strings.ToLower(techQuery)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, a)
	}

	page := parsePagination(r)
	total := len(filtered)
	start := page.Offset()
	end := start + page.PageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	writePaginatedJSON(w, filtered[start:end], total, page)
}

func (s *Server) handleListAssets(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	assets, err := s.queries.ListAssetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list assets failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, assets)
}

func (s *Server) handleListPorts(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	ports, err := s.queries.ListPortsByAsset(assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list ports failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, ports)
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	services, err := s.queries.ListServicesByAsset(assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list services failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, services)
}

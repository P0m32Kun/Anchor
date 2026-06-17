package api

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// GET /projects/{id}/web-endpoints
func (s *Server) handleListWebEndpointsByProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	page := parsePagination(r)
	total, err := s.queries.CountWebEndpointsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "count web endpoints failed: %v", err))
		return
	}
	endpoints, err := s.queries.ListWebEndpointsByProjectPaginated(projectID, page.PageSize, page.Offset())
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list web endpoints failed: %v", err))
		return
	}
	writePaginatedJSON(w, endpoints, total, page)
}

// GET /projects/{id}/assets (with filtering)
func (s *Server) handleListAssetsFiltered(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	// Parse filter params
	statusCode := r.URL.Query().Get("status_code")
	portStr := r.URL.Query().Get("port")
	titleQuery := r.URL.Query().Get("title")
	techQuery := r.URL.Query().Get("technology")
	showExcluded := r.URL.Query().Get("show_excluded") == "true"

	assets, err := s.queries.ListAssetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list assets: %v", err))
		return
	}

	// Apply filters in memory (for MVP; optimize with SQL later)
	var filtered []*models.Asset
	for _, a := range assets {
		// Exclusion filter: skip excluded domains unless explicitly requested
		if !showExcluded && s.excludeMgr != nil && s.excludeMgr.IsExcluded(a.Value) {
			continue
		}

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

// GET /projects/{id}/service-ports
// Aggregates ports, service fingerprints, and web endpoints into a unified view.
func (s *Server) handleListServicePorts(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	// 1. Get all IP assets for this project to map ip -> asset_id
	assets, err := s.queries.ListAssetsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list assets: %v", err))
		return
	}
	ipToAsset := make(map[string]*models.Asset)
	for _, a := range assets {
		if a.Type == "ip" {
			ipToAsset[a.Value] = a
		}
	}

	// 2. Get service fingerprints (nmap -sV results)
	fingerprints, err := s.queries.ListServiceFingerprintsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list fingerprints: %v", err))
		return
	}

	// 3. Get web endpoints (httpx results)
	webEndpoints, err := s.queries.ListWebEndpointsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list web endpoints: %v", err))
		return
	}

	// 4. Get all ports for this project in one query
	allPorts, err := s.queries.ListPortsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list ports: %v", err))
		return
	}

	// Aggregate by (IP, Port) key
	type spKey struct {
		IP   string
		Port int
	}
	spMap := make(map[spKey]*models.ServicePort)

	// Merge service fingerprints
	for _, fp := range fingerprints {
		asset, ok := ipToAsset[fp.IP]
		if !ok {
			continue
		}
		k := spKey{IP: fp.IP, Port: fp.Port}
		if _, exists := spMap[k]; !exists {
			spMap[k] = &models.ServicePort{
				ID:          util.GenerateID(),
				ProjectID:   projectID,
				AssetID:     asset.ID,
				IP:          fp.IP,
				Port:        fp.Port,
				Protocol:    fp.Protocol,
				State:       "open",
				ServiceName: fp.Service,
				Product:     fp.Product,
				Version:     fp.Version,
				SourceTools: []string{fp.Source},
				IsWeb:       fp.IsWeb,
				CreatedAt:   fp.CreatedAt,
			}
		} else {
			sp := spMap[k]
			if sp.ServiceName == "" || sp.ServiceName == "未知服务" {
				sp.ServiceName = fp.Service
			}
			if sp.Product == "" {
				sp.Product = fp.Product
			}
			if sp.Version == "" {
				sp.Version = fp.Version
			}
			sp.SourceTools = appendUnique(sp.SourceTools, fp.Source)
			if fp.IsWeb {
				sp.IsWeb = true
			}
		}
	}

	// Merge web endpoints
	for _, we := range webEndpoints {
		if we.Host == "" {
			continue
		}
		port := 0
		if we.Port != nil {
			port = *we.Port
		}
		// Fallback: parse port from URL
		if port == 0 && we.URL != "" {
			port = parsePortFromURL(we.URL)
		}
		if port == 0 {
			continue
		}

		asset, ok := ipToAsset[we.Host]
		if !ok {
			continue
		}

		k := spKey{IP: we.Host, Port: port}
		if _, exists := spMap[k]; !exists {
			spMap[k] = &models.ServicePort{
				ID:           util.GenerateID(),
				ProjectID:    projectID,
				AssetID:      asset.ID,
				IP:           we.Host,
				Port:         port,
				Protocol:     "tcp",
				State:        "open",
				ServiceName:  "未知服务",
				Title:        we.Title,
				Technologies: we.Technologies,
				URL:          we.URL,
				SourceTools:  []string{we.SourceTool},
				IsWeb:        true,
				CreatedAt:    we.CreatedAt,
			}
		} else {
			sp := spMap[k]
			sp.Title = we.Title
			sp.Technologies = we.Technologies
			sp.URL = we.URL
			sp.IsWeb = true
			sp.SourceTools = appendUnique(sp.SourceTools, we.SourceTool)
		}
	}

	// Merge naabu ports (only add if not already present from other sources)
	for _, p := range allPorts {
		asset, ok := ipToAsset[p.AssetID]
		if !ok {
			continue
		}
		k := spKey{IP: asset.Value, Port: p.Port}
		if _, exists := spMap[k]; !exists {
			spMap[k] = &models.ServicePort{
				ID:          util.GenerateID(),
				ProjectID:   projectID,
				AssetID:     asset.ID,
				IP:          asset.Value,
				Port:        p.Port,
				Protocol:    p.Protocol,
				State:       p.State,
				ServiceName: "未知服务",
				SourceTools: []string{p.SourceTool},
				CreatedAt:   p.CreatedAt,
			}
		} else {
			sp := spMap[k]
			sp.SourceTools = appendUnique(sp.SourceTools, p.SourceTool)
			if sp.State == "" {
				sp.State = p.State
			}
		}
	}

	// Convert map to slice
	result := make([]*models.ServicePort, 0, len(spMap))
	for _, sp := range spMap {
		if sp.ServiceName == "" {
			sp.ServiceName = "未知服务"
		}
		if sp.State == "" {
			sp.State = "open"
		}
		result = append(result, sp)
	}

	// Sort by IP then port
	sort.Slice(result, func(i, j int) bool {
		if result[i].IP != result[j].IP {
			return result[i].IP < result[j].IP
		}
		return result[i].Port < result[j].Port
	})

	writeJSON(w, http.StatusOK, result)
}

func parsePortFromURL(raw string) int {
	u, err := url.Parse(raw)
	if err != nil {
		return 0
	}
	if u.Port() != "" {
		p, _ := strconv.Atoi(u.Port())
		return p
	}
	if u.Scheme == "https" {
		return 443
	}
	if u.Scheme == "http" {
		return 80
	}
	return 0
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// GET /assets/{id}/lineage?run_id=
func (s *Server) handleGetAssetLineage(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	if assetID == "" {
		writeError(w, http.StatusBadRequest, errors.Newf(errors.ErrBadRequest, "missing asset id"))
		return
	}

	asset, err := s.queries.GetAssetByID(assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get asset: %v", err))
		return
	}
	if asset == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "asset not found"))
		return
	}

	var runID *string
	if rid := strings.TrimSpace(r.URL.Query().Get("run_id")); rid != "" {
		runID = &rid
	}

	lineage, err := s.queries.BuildAssetLineage(asset.ProjectID, assetID, runID)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, lineage)
}

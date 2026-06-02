package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/workflows"
)

func (s *Server) handleStartAssetDiscovery(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	wf := workflows.NewAssetDiscoveryWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		result, err := wf.Run(ctx, projectID)
		if err != nil {
			log.Printf("asset discovery workflow failed for project %s: %v", projectID, err)
		}
		s.broadcastProjectSSE(projectID, map[string]interface{}{
			"event":      "asset_discovery_complete",
			"project_id": projectID,
			"result":     result,
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

func (s *Server) handleStartWebScreening(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project failed: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	wf := workflows.NewWebScreeningWorkflow(s.queries, s.worker, s.scopeEng, s.dataDir)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		result, err := wf.Run(ctx, projectID)
		if err != nil {
			log.Printf("web screening workflow failed for project %s: %v", projectID, err)
		}
		s.broadcastProjectSSE(projectID, map[string]interface{}{
			"event":      "web_screening_complete",
			"project_id": projectID,
			"result":     result,
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

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

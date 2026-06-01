package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/evaluator"
)

// handleGetEvaluation returns the evaluation report for a run.
// GET /projects/{id}/runs/{runId}/evaluation
func (s *Server) handleGetEvaluation(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")

	reportPath := filepath.Join(s.dataDir, "projects", projectID, "reports",
		runID+"_evaluation.md")

	// Check if report exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "评估报告不存在，可能扫描尚未完成"))
		return
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("READ_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"content": string(content),
	})
}

// handleRetryEvaluation manually triggers evaluation for a run.
// POST /projects/{id}/runs/{runId}/evaluation/retry
func (s *Server) handleRetryEvaluation(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runID := r.PathValue("runId")

	go func() {
		eval := evaluator.NewEvaluator(s.queries, s.dataDir, projectID, runID)
		_, err := eval.Evaluate(context.Background())
		if err != nil {
			log.Printf("[evaluation] manual retry failed for run %s: %v", runID, err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "评估已触发，稍后刷新查看结果",
	})
}

// handleListEvaluations returns a list of evaluation reports for a project.
// GET /projects/{id}/evaluations
func (s *Server) handleListEvaluations(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	reportsDir := filepath.Join(s.dataDir, "projects", projectID, "reports")

	// Check if directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, []map[string]string{})
		return
	}

	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("READ_DIR_ERROR", err.Error()))
		return
	}

	var evaluations []map[string]string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "_evaluation.md") {
			runID := strings.TrimSuffix(entry.Name(), "_evaluation.md")
			info, err := entry.Info()
			if err != nil {
				continue
			}
			evaluations = append(evaluations, map[string]string{
				"run_id":     runID,
				"filename":   entry.Name(),
				"created_at": info.ModTime().Format("2006-01-02T15:04:05Z"),
			})
		}
	}

	writeJSON(w, http.StatusOK, evaluations)
}

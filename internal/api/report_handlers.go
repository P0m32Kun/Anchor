package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/report"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (s *Server) handleExportReportMD(w http.ResponseWriter, r *http.Request) {
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

	data, err := report.Aggregate(r.Context(), s.queries, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "report aggregation failed: %v", err))
		return
	}

	md := report.GenerateMarkdown(data)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.md", projectID))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(md))
}

func (s *Server) handleExportReportJSON(w http.ResponseWriter, r *http.Request) {
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

	data, err := report.Aggregate(r.Context(), s.queries, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "report aggregation failed: %v", err))
		return
	}

	jsonData, err := report.GenerateJSON(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "json generation failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.json", projectID))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// --- New Report Endpoints ---

func (s *Server) handleCreateReport(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")

	// Parse optional body.
	var body struct {
		Title string `json:"title"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}

	// Check disk space (500MB minimum).
	if ok, avail := checkDiskSpace(s.dataDir); !ok {
		writeError(w, http.StatusInsufficientStorage, errors.Newf(errors.ErrInternal, "磁盘空间不足（剩余 %dMB），请清理后重试", avail/1024/1024))
		return
	}

	// Check if report already exists for this run.
	existing, err := s.queries.GetReportByRunID(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "check existing report: %v", err))
		return
	}
	if existing != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}

	// Fetch run to validate.
	run, err := s.queries.GetPipelineRun(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get run: %v", err))
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "运行不存在"))
		return
	}

	// Create report record with "generating" status.
	now := time.Now().UTC()
	title := body.Title
	if title == "" {
		title = fmt.Sprintf("%s 扫描安全评估报告", run.Mode)
	}
	rpt := &models.Report{
		ID:        util.GenerateID(),
		RunID:     runID,
		Status:    models.ReportGenerating,
		Title:     title,
		CreatedAt: now,
	}
	if err := s.queries.CreateReport(rpt); err != nil {
		writeError(w, http.StatusConflict, errors.Newf(errors.ErrInternal, "创建报告失败（该运行可能已有报告）: %v", err))
		return
	}

	// Generate report asynchronously.
	go s.generateReport(rpt, runID)

	writeJSON(w, http.StatusAccepted, rpt)
}

func (s *Server) generateReport(rpt *models.Report, runID string) {
	now := time.Now().UTC()
	ctx := context.Background()

	// Aggregate data with batch evidence.
	data, _, err := report.AggregateByRunWithBatchEvidence(ctx, s.queries, runID)
	if err != nil {
		rpt.Status = models.ReportFailed
		rpt.ErrorMessage = fmt.Sprintf("数据聚合失败: %v", err)
		rpt.CompletedAt = &now
		s.queries.UpdateReport(rpt)
		s.broadcastReportProgress(rpt)
		return
	}

	// Determine status: partial if run failed but has findings.
	if len(data.Findings) > 0 {
		rpt.FindingCount = len(data.Findings)
		for _, rf := range data.Findings {
			rpt.EvidenceCount += len(rf.EvidenceList)
		}
	}

	// Generate HTML.
	html, err := report.GenerateHTML(data)
	if err != nil {
		rpt.Status = models.ReportFailed
		rpt.ErrorMessage = fmt.Sprintf("HTML 生成失败: %v", err)
		rpt.CompletedAt = &now
		s.queries.UpdateReport(rpt)
		s.broadcastReportProgress(rpt)
		return
	}

	// Write HTML file.
	reportDir := filepath.Join(s.dataDir, "reports")
	if err := os.MkdirAll(reportDir, 0750); err != nil {
		rpt.Status = models.ReportFailed
		rpt.ErrorMessage = fmt.Sprintf("创建报告目录失败: %v", err)
		rpt.CompletedAt = &now
		s.queries.UpdateReport(rpt)
		s.broadcastReportProgress(rpt)
		return
	}

	filePath := filepath.Join(reportDir, rpt.ID+".html")
	if err := os.WriteFile(filePath, []byte(html), 0640); err != nil {
		rpt.Status = models.ReportFailed
		rpt.ErrorMessage = fmt.Sprintf("写入报告文件失败: %v", err)
		rpt.CompletedAt = &now
		s.queries.UpdateReport(rpt)
		s.broadcastReportProgress(rpt)
		return
	}

	// Update report with success.
	info, _ := os.Stat(filePath)
	rpt.Status = models.ReportComplete
	rpt.FilePath = filePath
	if info != nil {
		rpt.FileSizeBytes = info.Size()
	}
	rpt.CompletedAt = &now
	if err := s.queries.UpdateReport(rpt); err != nil {
		log.Printf("[report] update report: %v", err)
	}

	s.broadcastReportProgress(rpt)
}

func (s *Server) broadcastReportProgress(rpt *models.Report) {
	// Get run to find project ID for SSE routing.
	run, err := s.queries.GetPipelineRun(rpt.RunID)
	if err != nil || run == nil {
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"event":     "report_progress",
		"report_id": rpt.ID,
		"run_id":    rpt.RunID,
		"status":    string(rpt.Status),
		"title":     rpt.Title,
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if clients, ok := s.sseClients[run.ProjectID]; ok {
		for _, ch := range clients {
			select {
			case ch <- data:
			default:
			}
		}
	}
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")

	rpt, err := s.queries.GetReport(reportID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get report: %v", err))
		return
	}
	if rpt == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "报告不存在"))
		return
	}

	writeJSON(w, http.StatusOK, rpt)
}

func (s *Server) handleDownloadReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")

	rpt, err := s.queries.GetReport(reportID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get report: %v", err))
		return
	}
	if rpt == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "报告不存在"))
		return
	}
	if rpt.Status != models.ReportComplete {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrValidation, "报告尚未生成完成"))
		return
	}

	if rpt.FilePath == "" {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "报告文件不存在"))
		return
	}

	data, err := os.ReadFile(rpt.FilePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "读取报告文件失败: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.html", rpt.ID))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")

	rpt, err := s.queries.GetReport(reportID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get report: %v", err))
		return
	}
	if rpt == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "报告不存在"))
		return
	}

	// Delete file if exists.
	if rpt.FilePath != "" {
		os.Remove(rpt.FilePath)
	}

	if err := s.queries.DeleteReport(reportID); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "delete report: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := 20

	reports, hasMore, err := s.queries.ListReports(cursor, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "list reports: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":    reports,
		"has_more": hasMore,
	})
}

// checkDiskSpace returns true if the data directory has at least 500MB free.
// Uses syscall.Statfs on darwin/linux; always returns true on unsupported platforms.
func checkDiskSpace(dataDir string) (bool, int64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dataDir, &stat); err != nil {
		return true, 0 // Can't check, allow.
	}
	avail := int64(stat.Bavail) * int64(stat.Bsize)
	return avail > 500*1024*1024, avail
}

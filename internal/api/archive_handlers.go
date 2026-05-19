package api

import (
	"archive/zip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/report"
)

// POST /projects/{id}/archive
func (s *Server) handleCreateArchive(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "get project: %v", err))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "project not found"))
		return
	}

	// Create archive directory
	archiveDir := filepath.Join(s.dataDir, "projects", projectID, "archives")
	if err := os.MkdirAll(archiveDir, 0750); err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create archive dir: %v", err))
		return
	}

	archiveName := fmt.Sprintf("archive_%s_%s.zip", projectID, time.Now().Format("20060102_150405"))
	archivePath := filepath.Join(archiveDir, archiveName)

	// Create zip file
	zipFile, err := os.Create(archivePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "create zip: %v", err))
		return
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// 1. Add Markdown report
	reportData, err := report.Aggregate(r.Context(), s.queries, project)
	if err == nil {
		mdData := report.GenerateMarkdown(reportData)
		fw, _ := zw.Create("report.md")
		fw.Write([]byte(mdData))
	}

	// 2. Add screenshots
	screenshots, _ := s.queries.ListScreenshotsByProject(projectID)
	for _, sc := range screenshots {
		data, err := os.ReadFile(sc.OriginalPath)
		if err != nil {
			continue
		}
		fw, _ := zw.Create(filepath.Join("screenshots", filepath.Base(sc.OriginalPath)))
		fw.Write(data)
	}

	// 3. Add scope snapshot
	scopeRules, _ := s.queries.ListScopeRulesByProject(projectID)
	if len(scopeRules) > 0 {
		fw, _ := zw.Create("scope_rules.txt")
		for _, rule := range scopeRules {
			fmt.Fprintf(fw, "%s %s %s: %s\n", rule.Action, rule.Type, rule.Value, rule.Reason)
		}
	}

	// 4. Add tool versions snapshot
	fw, _ := zw.Create("tool_versions.txt")
	fmt.Fprintln(fw, "Tool Versions Snapshot")
	fmt.Fprintln(fw, "=====================")
	fmt.Fprintln(fw, time.Now().Format(time.RFC3339))

	writeJSON(w, http.StatusCreated, map[string]string{
		"archive_path": archivePath,
		"archive_name": archiveName,
	})
}

// GET /projects/{id}/archive/download
func (s *Server) handleDownloadArchive(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	archiveName := r.URL.Query().Get("name")
	if archiveName == "" {
		writeError(w, http.StatusBadRequest, errors.New(errors.ErrBadRequest, "missing name"))
		return
	}

	archivePath := filepath.Join(s.dataDir, "projects", projectID, "archives", archiveName)
	if _, err := os.Stat(archivePath); err != nil {
		writeError(w, http.StatusNotFound, errors.New(errors.ErrNotFound, "archive not found"))
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", archiveName))
	http.ServeFile(w, r, archivePath)
}

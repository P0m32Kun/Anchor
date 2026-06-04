package api

import (
	"encoding/json"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// handleCreateSRCProgram 创建 SRC 程序
func (s *Server) handleCreateSRCProgram(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	// 检查项目是否存在
	project, err := s.queries.GetProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Project not found"))
		return
	}

	// 检查是否已存在 SRC 程序
	existing, err := s.queries.GetSRCProgram(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, errors.New("ALREADY_EXISTS", "SRC program already exists for this project"))
		return
	}

	var req models.CreateSRCProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", err.Error()))
		return
	}

	// 验证平台
	if req.Platform != "" && !models.IsSupportedPlatform(req.Platform) {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_PLATFORM", "Unsupported platform: "+req.Platform))
		return
	}

	// 创建程序
	program := &models.SRCProgram{
		ID:                     util.GenerateID(),
		ProjectID:              projectID,
		Name:                   req.Name,
		Platform:               req.Platform,
		ProgramURL:             req.ProgramURL,
		RulesURL:               req.RulesURL,
		AllowAutomation:        req.AllowAutomation,
		AllowDirBrute:          req.AllowDirBrute,
		AllowWeakPassword:      req.AllowWeakPassword,
		AllowAuthenticatedTest: req.AllowAuthenticatedTest,
		MaxRPS:                 req.MaxRPS,
		MaxConcurrency:         req.MaxConcurrency,
		PreferredVulnTypes:     req.PreferredVulnTypes,
		PayoutHint:             req.PayoutHint,
		Notes:                  req.Notes,
	}

	// 设置默认值
	if program.Platform == "" {
		program.Platform = "other"
	}
	if program.MaxRPS == 0 {
		program.MaxRPS = 5
	}
	if program.MaxConcurrency == 0 {
		program.MaxConcurrency = 3
	}
	if program.PreferredVulnTypes == nil {
		program.PreferredVulnTypes = []string{}
	}
	if program.PayoutHint == nil {
		program.PayoutHint = map[string]any{}
	}

	if err := s.queries.CreateSRCProgram(program); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, program)
}

// handleGetSRCProgram 获取 SRC 程序
func (s *Server) handleGetSRCProgram(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	program, err := s.queries.GetSRCProgram(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if program == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "SRC program not found"))
		return
	}

	writeJSON(w, http.StatusOK, program)
}

// handleUpdateSRCProgram 更新 SRC 程序
func (s *Server) handleUpdateSRCProgram(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	// 获取现有程序
	program, err := s.queries.GetSRCProgram(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if program == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "SRC program not found"))
		return
	}

	var req models.UpdateSRCProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", err.Error()))
		return
	}

	// 验证平台
	if req.Platform != nil && !models.IsSupportedPlatform(*req.Platform) {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_PLATFORM", "Unsupported platform: "+*req.Platform))
		return
	}

	// 更新字段
	if req.Name != nil {
		program.Name = *req.Name
	}
	if req.Platform != nil {
		program.Platform = *req.Platform
	}
	if req.ProgramURL != nil {
		program.ProgramURL = *req.ProgramURL
	}
	if req.RulesURL != nil {
		program.RulesURL = *req.RulesURL
	}
	if req.AllowAutomation != nil {
		program.AllowAutomation = *req.AllowAutomation
	}
	if req.AllowDirBrute != nil {
		program.AllowDirBrute = *req.AllowDirBrute
	}
	if req.AllowWeakPassword != nil {
		program.AllowWeakPassword = *req.AllowWeakPassword
	}
	if req.AllowAuthenticatedTest != nil {
		program.AllowAuthenticatedTest = *req.AllowAuthenticatedTest
	}
	if req.MaxRPS != nil {
		program.MaxRPS = *req.MaxRPS
	}
	if req.MaxConcurrency != nil {
		program.MaxConcurrency = *req.MaxConcurrency
	}
	if req.PreferredVulnTypes != nil {
		program.PreferredVulnTypes = req.PreferredVulnTypes
	}
	if req.PayoutHint != nil {
		program.PayoutHint = req.PayoutHint
	}
	if req.Notes != nil {
		program.Notes = *req.Notes
	}

	if err := s.queries.UpdateSRCProgram(program); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, program)
}

// handleDeleteSRCProgram 删除 SRC 程序
func (s *Server) handleDeleteSRCProgram(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_PROJECT_ID", "Project ID is required"))
		return
	}

	// 检查是否存在
	program, err := s.queries.GetSRCProgram(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if program == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "SRC program not found"))
		return
	}

	if err := s.queries.DeleteSRCProgram(projectID); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleListSRCPrograms 列出所有 SRC 程序
func (s *Server) handleListSRCPrograms(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")

	var programs []*models.SRCProgram
	var err error

	if platform != "" {
		programs, err = s.queries.ListSRCProgramsByPlatform(platform)
	} else {
		programs, err = s.queries.ListSRCPrograms()
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"programs": programs,
		"count":    len(programs),
	})
}

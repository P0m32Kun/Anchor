package api

import (
	"encoding/json"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/bounty"
	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// handleListBountyCandidates 列出项目的所有赏金候选
func (s *Server) handleListBountyCandidates(w http.ResponseWriter, r *http.Request) {
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

	// 获取筛选参数
	verifyStatus := r.URL.Query().Get("verify_status")
	submissionStatus := r.URL.Query().Get("submission_status")

	var candidates []*models.BountyCandidate
	if verifyStatus != "" {
		candidates, err = s.queries.ListBountyCandidatesByStatus(projectID, verifyStatus)
	} else {
		candidates, err = s.queries.ListBountyCandidatesByProject(projectID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	// 按提交状态筛选
	if submissionStatus != "" {
		var filtered []*models.BountyCandidate
		for _, c := range candidates {
			if c.SubmissionStatus == submissionStatus {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
	}

	// 获取统计信息
	stats, err := s.queries.GetBountyCandidateStats(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"candidates": candidates,
		"count":      len(candidates),
		"stats":      stats,
	})
}

// handleGetBountyCandidate 获取指定赏金候选
func (s *Server) handleGetBountyCandidate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Candidate ID is required"))
		return
	}

	candidate, err := s.queries.GetBountyCandidate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if candidate == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Bounty candidate not found"))
		return
	}

	writeJSON(w, http.StatusOK, candidate)
}

// handleUpdateBountyCandidate 更新赏金候选
func (s *Server) handleUpdateBountyCandidate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Candidate ID is required"))
		return
	}

	candidate, err := s.queries.GetBountyCandidate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if candidate == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Bounty candidate not found"))
		return
	}

	var req models.UpdateBountyCandidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", err.Error()))
		return
	}

	// 更新字段
	if req.VerifyStatus != nil {
		if !models.IsValidVerifyStatus(*req.VerifyStatus) {
			writeError(w, http.StatusBadRequest, errors.New("INVALID_STATUS", "Invalid verify status"))
			return
		}
		candidate.VerifyStatus = *req.VerifyStatus
	}
	if req.SubmissionStatus != nil {
		if !models.IsValidSubmissionStatus(*req.SubmissionStatus) {
			writeError(w, http.StatusBadRequest, errors.New("INVALID_STATUS", "Invalid submission status"))
			return
		}
		candidate.SubmissionStatus = *req.SubmissionStatus
	}
	if req.SubmissionURL != nil {
		candidate.SubmissionURL = *req.SubmissionURL
	}
	if req.SubmissionID != nil {
		candidate.SubmissionID = *req.SubmissionID
	}
	if req.PaidAmount != nil {
		candidate.PaidAmount = *req.PaidAmount
	}
	if req.Notes != nil {
		candidate.Notes = *req.Notes
	}

	if err := s.queries.UpdateBountyCandidate(candidate); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, candidate)
}

// handleDeleteBountyCandidate 删除赏金候选
func (s *Server) handleDeleteBountyCandidate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Candidate ID is required"))
		return
	}

	candidate, err := s.queries.GetBountyCandidate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if candidate == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Bounty candidate not found"))
		return
	}

	if err := s.queries.DeleteBountyCandidate(id); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleRefreshBountyCandidates 刷新项目的赏金候选
func (s *Server) handleRefreshBountyCandidates(w http.ResponseWriter, r *http.Request) {
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

	// 获取项目的 SRC 程序
	program, err := s.queries.GetSRCProgram(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	// 获取项目的 findings
	findings, err := s.queries.ListFindingsByProject(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	// 创建评分器和重复风险评估器
	scorer := bounty.NewScorer()
	riskAssessor := bounty.NewDuplicateRiskAssessor()

	// 刷新候选
	var created int
	for _, finding := range findings {
		// 检查是否已存在候选
		existing, err := s.queries.GetBountyCandidateByFinding(projectID, finding.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
			return
		}
		if existing != nil {
			// 更新现有候选的分数
			scorer.Score(existing)
			existing.DuplicateRisk = riskAssessor.Assess(existing)
			existing.RankingReason = scorer.GenerateRankingReason(existing)
			if err := s.queries.UpdateBountyCandidate(existing); err != nil {
				writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
				return
			}
			continue
		}

		// 检查是否应该创建候选
		if !shouldCreateCandidate(finding) {
			continue
		}

		// 创建新候选
		candidate := &models.BountyCandidate{
			ID:               util.GenerateID(),
			ProjectID:        projectID,
			FindingID:        &finding.ID,
			SourceKind:       models.SourceKindFinding,
			Title:            finding.Title,
			VulnType:         string(finding.Severity), // 简化处理
			Severity:         string(finding.Severity),
			Confidence:       "medium",
			VerifyStatus:     models.VerifyStatusPending,
			SubmissionStatus: models.SubmissionStatusNotReady,
		}

		if program != nil {
			candidate.ProgramID = &program.ID
		}

		// 评分
		scorer.Score(candidate)
		candidate.DuplicateRisk = riskAssessor.Assess(candidate)
		candidate.RankingReason = scorer.GenerateRankingReason(candidate)

		if err := s.queries.CreateBountyCandidate(candidate); err != nil {
			writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
			return
		}
		created++
	}

	// 获取更新后的统计
	stats, err := s.queries.GetBountyCandidateStats(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "refreshed",
		"created": created,
		"stats":   stats,
	})
}

// shouldCreateCandidate 判断是否应该创建候选
func shouldCreateCandidate(finding *models.Finding) bool {
	// 排除 info 级别
	if finding.Severity == "info" {
		return false
	}

	// 排除低置信度的 banner/version 发现
	if finding.Severity == "low" && (finding.Title == "Version Detection" || finding.Title == "Banner Grab") {
		return false
	}

	// 包含 high/critical 级别
	if finding.Severity == "critical" || finding.Severity == "high" {
		return true
	}

	// 包含 medium 级别的高信号发现
	if finding.Severity == "medium" {
		highSignalTypes := []string{
			"exposed_config", "unauthenticated_api", "cloud_key",
			"actuator", "file_read", "secret_leak",
		}
		for _, t := range highSignalTypes {
			if finding.Title == t {
				return true
			}
		}
	}

	return false
}

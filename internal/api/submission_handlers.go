package api

import (
	"encoding/json"
	"net/http"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/submission"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// handleCreateSubmissionPack 创建提交包
func (s *Server) handleCreateSubmissionPack(w http.ResponseWriter, r *http.Request) {
	candidateID := r.PathValue("id")
	if candidateID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Candidate ID is required"))
		return
	}

	// 检查候选是否存在
	candidate, err := s.queries.GetBountyCandidate(candidateID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if candidate == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Bounty candidate not found"))
		return
	}

	var req models.CreateSubmissionPackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", err.Error()))
		return
	}

	// 验证模板
	if req.Template != "" && !models.IsValidSubmissionTemplate(req.Template) {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_TEMPLATE", "Invalid submission template"))
		return
	}
	if req.Template == "" {
		req.Template = models.SubmissionTemplateGeneric
	}

	// 获取 finding（如果有）
	var finding *models.Finding
	if candidate.FindingID != nil {
		finding, err = s.queries.GetFinding(*candidate.FindingID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
			return
		}
	}

	// 生成提交包
	generator := submission.NewPackGenerator()
	pack, err := generator.Generate(candidate, finding, req.Template)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("GENERATE_ERROR", err.Error()))
		return
	}
	pack.ID = util.GenerateID()

	if err := s.queries.CreateSubmissionPack(pack); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, pack)
}

// handleGetSubmissionPack 获取提交包
func (s *Server) handleGetSubmissionPack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Pack ID is required"))
		return
	}

	pack, err := s.queries.GetSubmissionPack(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if pack == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Submission pack not found"))
		return
	}

	writeJSON(w, http.StatusOK, pack)
}

// handleListSubmissionPacks 列出候选的所有提交包
func (s *Server) handleListSubmissionPacks(w http.ResponseWriter, r *http.Request) {
	candidateID := r.PathValue("id")
	if candidateID == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Candidate ID is required"))
		return
	}

	// 检查候选是否存在
	candidate, err := s.queries.GetBountyCandidate(candidateID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if candidate == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Bounty candidate not found"))
		return
	}

	packs, err := s.queries.ListSubmissionPacksByCandidate(candidateID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"packs": packs,
		"count": len(packs),
	})
}

// handleUpdateSubmissionPack 更新提交包
func (s *Server) handleUpdateSubmissionPack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Pack ID is required"))
		return
	}

	pack, err := s.queries.GetSubmissionPack(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if pack == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Submission pack not found"))
		return
	}

	var req models.UpdateSubmissionPackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("INVALID_BODY", err.Error()))
		return
	}

	// 更新字段
	if req.Content != nil {
		pack.Content = *req.Content
	}
	if req.ChecklistJSON != nil {
		pack.ChecklistJSON = *req.ChecklistJSON
	}
	if req.RedactionStatus != nil {
		if !models.IsValidRedactionStatus(*req.RedactionStatus) {
			writeError(w, http.StatusBadRequest, errors.New("INVALID_STATUS", "Invalid redaction status"))
			return
		}
		pack.RedactionStatus = *req.RedactionStatus
	}

	if err := s.queries.UpdateSubmissionPack(pack); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, pack)
}

// handleDeleteSubmissionPack 删除提交包
func (s *Server) handleDeleteSubmissionPack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Pack ID is required"))
		return
	}

	pack, err := s.queries.GetSubmissionPack(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if pack == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Submission pack not found"))
		return
	}

	if err := s.queries.DeleteSubmissionPack(id); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleRedactSubmissionPack 脱敏提交包
func (s *Server) handleRedactSubmissionPack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("MISSING_ID", "Pack ID is required"))
		return
	}

	pack, err := s.queries.GetSubmissionPack(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}
	if pack == nil {
		writeError(w, http.StatusNotFound, errors.New("NOT_FOUND", "Submission pack not found"))
		return
	}

	// 脱敏内容
	generator := submission.NewPackGenerator()
	pack.Content = generator.RedactContent(pack.Content)
	pack.RedactionStatus = models.RedactionStatusRedacted

	if err := s.queries.UpdateSubmissionPack(pack); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("DB_ERROR", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, pack)
}

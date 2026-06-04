package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// CreateBountyCandidate 创建赏金候选
func (q *Queries) CreateBountyCandidate(candidate *models.BountyCandidate) error {
	if candidate.ID == "" {
		candidate.ID = util.GenerateID()
	}
	now := time.Now().UTC()
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	_, err := q.db.Exec(`
		INSERT INTO bounty_candidates (
			id, project_id, program_id, finding_id, endpoint_id,
			source_kind, title, vuln_type, severity, confidence,
			value_score, impact_score, novelty_score, repro_score, scope_score, safety_score,
			duplicate_risk, ranking_reason, verify_status, submission_status,
			submission_url, submission_id, paid_amount, notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		candidate.ID, candidate.ProjectID, candidate.ProgramID, candidate.FindingID, candidate.EndpointID,
		candidate.SourceKind, candidate.Title, candidate.VulnType, candidate.Severity, candidate.Confidence,
		candidate.ValueScore, candidate.ImpactScore, candidate.NoveltyScore, candidate.ReproScore,
		candidate.ScopeScore, candidate.SafetyScore,
		candidate.DuplicateRisk, candidate.RankingReason, candidate.VerifyStatus, candidate.SubmissionStatus,
		candidate.SubmissionURL, candidate.SubmissionID, candidate.PaidAmount, candidate.Notes,
		candidate.CreatedAt, candidate.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert bounty_candidate: %w", err)
	}

	return nil
}

// GetBountyCandidate 获取指定赏金候选
func (q *Queries) GetBountyCandidate(id string) (*models.BountyCandidate, error) {
	var candidate models.BountyCandidate

	err := q.db.QueryRow(`
		SELECT id, project_id, program_id, finding_id, endpoint_id,
			source_kind, title, vuln_type, severity, confidence,
			value_score, impact_score, novelty_score, repro_score, scope_score, safety_score,
			duplicate_risk, ranking_reason, verify_status, submission_status,
			submission_url, submission_id, paid_amount, notes,
			created_at, updated_at
		FROM bounty_candidates
		WHERE id = ?
	`, id).Scan(
		&candidate.ID, &candidate.ProjectID, &candidate.ProgramID, &candidate.FindingID, &candidate.EndpointID,
		&candidate.SourceKind, &candidate.Title, &candidate.VulnType, &candidate.Severity, &candidate.Confidence,
		&candidate.ValueScore, &candidate.ImpactScore, &candidate.NoveltyScore, &candidate.ReproScore,
		&candidate.ScopeScore, &candidate.SafetyScore,
		&candidate.DuplicateRisk, &candidate.RankingReason, &candidate.VerifyStatus, &candidate.SubmissionStatus,
		&candidate.SubmissionURL, &candidate.SubmissionID, &candidate.PaidAmount, &candidate.Notes,
		&candidate.CreatedAt, &candidate.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query bounty_candidate: %w", err)
	}

	return &candidate, nil
}

// UpdateBountyCandidate 更新赏金候选
func (q *Queries) UpdateBountyCandidate(candidate *models.BountyCandidate) error {
	candidate.UpdatedAt = time.Now().UTC()

	_, err := q.db.Exec(`
		UPDATE bounty_candidates SET
			program_id = ?, finding_id = ?, endpoint_id = ?,
			source_kind = ?, title = ?, vuln_type = ?, severity = ?, confidence = ?,
			value_score = ?, impact_score = ?, novelty_score = ?, repro_score = ?, scope_score = ?, safety_score = ?,
			duplicate_risk = ?, ranking_reason = ?, verify_status = ?, submission_status = ?,
			submission_url = ?, submission_id = ?, paid_amount = ?, notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		candidate.ProgramID, candidate.FindingID, candidate.EndpointID,
		candidate.SourceKind, candidate.Title, candidate.VulnType, candidate.Severity, candidate.Confidence,
		candidate.ValueScore, candidate.ImpactScore, candidate.NoveltyScore, candidate.ReproScore,
		candidate.ScopeScore, candidate.SafetyScore,
		candidate.DuplicateRisk, candidate.RankingReason, candidate.VerifyStatus, candidate.SubmissionStatus,
		candidate.SubmissionURL, candidate.SubmissionID, candidate.PaidAmount, candidate.Notes,
		candidate.UpdatedAt, candidate.ID,
	)
	if err != nil {
		return fmt.Errorf("update bounty_candidate: %w", err)
	}

	return nil
}

// DeleteBountyCandidate 删除赏金候选
func (q *Queries) DeleteBountyCandidate(id string) error {
	_, err := q.db.Exec("DELETE FROM bounty_candidates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete bounty_candidate: %w", err)
	}
	return nil
}

// ListBountyCandidatesByProject 列出项目的所有赏金候选
func (q *Queries) ListBountyCandidatesByProject(projectID string) ([]*models.BountyCandidate, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, program_id, finding_id, endpoint_id,
			source_kind, title, vuln_type, severity, confidence,
			value_score, impact_score, novelty_score, repro_score, scope_score, safety_score,
			duplicate_risk, ranking_reason, verify_status, submission_status,
			submission_url, submission_id, paid_amount, notes,
			created_at, updated_at
		FROM bounty_candidates
		WHERE project_id = ?
		ORDER BY value_score DESC, duplicate_risk ASC, created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query bounty_candidates: %w", err)
	}
	defer rows.Close()

	return scanBountyCandidates(rows)
}

// ListBountyCandidatesByProgram 列出程序的所有赏金候选
func (q *Queries) ListBountyCandidatesByProgram(programID string) ([]*models.BountyCandidate, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, program_id, finding_id, endpoint_id,
			source_kind, title, vuln_type, severity, confidence,
			value_score, impact_score, novelty_score, repro_score, scope_score, safety_score,
			duplicate_risk, ranking_reason, verify_status, submission_status,
			submission_url, submission_id, paid_amount, notes,
			created_at, updated_at
		FROM bounty_candidates
		WHERE program_id = ?
		ORDER BY value_score DESC, duplicate_risk ASC, created_at DESC
	`, programID)
	if err != nil {
		return nil, fmt.Errorf("query bounty_candidates: %w", err)
	}
	defer rows.Close()

	return scanBountyCandidates(rows)
}

// ListBountyCandidatesByStatus 按状态列出赏金候选
func (q *Queries) ListBountyCandidatesByStatus(projectID, verifyStatus string) ([]*models.BountyCandidate, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, program_id, finding_id, endpoint_id,
			source_kind, title, vuln_type, severity, confidence,
			value_score, impact_score, novelty_score, repro_score, scope_score, safety_score,
			duplicate_risk, ranking_reason, verify_status, submission_status,
			submission_url, submission_id, paid_amount, notes,
			created_at, updated_at
		FROM bounty_candidates
		WHERE project_id = ? AND verify_status = ?
		ORDER BY value_score DESC, created_at DESC
	`, projectID, verifyStatus)
	if err != nil {
		return nil, fmt.Errorf("query bounty_candidates: %w", err)
	}
	defer rows.Close()

	return scanBountyCandidates(rows)
}

// GetBountyCandidateStats 获取赏金候选统计
func (q *Queries) GetBountyCandidateStats(projectID string) (*models.BountyCandidateStats, error) {
	var stats models.BountyCandidateStats

	err := q.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN verify_status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN verify_status = 'confirmed' THEN 1 ELSE 0 END), 0) as verified,
			COALESCE(SUM(CASE WHEN submission_status = 'submitted' THEN 1 ELSE 0 END), 0) as submitted,
			COALESCE(SUM(CASE WHEN submission_status = 'accepted' THEN 1 ELSE 0 END), 0) as accepted,
			COALESCE(SUM(CASE WHEN submission_status = 'paid' THEN 1 ELSE 0 END), 0) as paid,
			COALESCE(SUM(value_score), 0) as total_value,
			COALESCE(AVG(value_score), 0) as average_value
		FROM bounty_candidates
		WHERE project_id = ?
	`, projectID).Scan(
		&stats.Total, &stats.Pending, &stats.Verified,
		&stats.Submitted, &stats.Accepted, &stats.Paid,
		&stats.TotalValue, &stats.AverageValue,
	)
	if err != nil {
		return nil, fmt.Errorf("query bounty_candidate stats: %w", err)
	}

	return &stats, nil
}

// GetBountyCandidateByFinding 通过 finding_id 获取赏金候选
func (q *Queries) GetBountyCandidateByFinding(projectID, findingID string) (*models.BountyCandidate, error) {
	var candidate models.BountyCandidate

	err := q.db.QueryRow(`
		SELECT id, project_id, program_id, finding_id, endpoint_id,
			source_kind, title, vuln_type, severity, confidence,
			value_score, impact_score, novelty_score, repro_score, scope_score, safety_score,
			duplicate_risk, ranking_reason, verify_status, submission_status,
			submission_url, submission_id, paid_amount, notes,
			created_at, updated_at
		FROM bounty_candidates
		WHERE project_id = ? AND finding_id = ?
	`, projectID, findingID).Scan(
		&candidate.ID, &candidate.ProjectID, &candidate.ProgramID, &candidate.FindingID, &candidate.EndpointID,
		&candidate.SourceKind, &candidate.Title, &candidate.VulnType, &candidate.Severity, &candidate.Confidence,
		&candidate.ValueScore, &candidate.ImpactScore, &candidate.NoveltyScore, &candidate.ReproScore,
		&candidate.ScopeScore, &candidate.SafetyScore,
		&candidate.DuplicateRisk, &candidate.RankingReason, &candidate.VerifyStatus, &candidate.SubmissionStatus,
		&candidate.SubmissionURL, &candidate.SubmissionID, &candidate.PaidAmount, &candidate.Notes,
		&candidate.CreatedAt, &candidate.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query bounty_candidate: %w", err)
	}

	return &candidate, nil
}

// DeleteBountyCandidatesByProject 删除项目的所有赏金候选
func (q *Queries) DeleteBountyCandidatesByProject(projectID string) error {
	_, err := q.db.Exec("DELETE FROM bounty_candidates WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("delete bounty_candidates: %w", err)
	}
	return nil
}

// scanBountyCandidates 扫描赏金候选结果集
func scanBountyCandidates(rows *sql.Rows) ([]*models.BountyCandidate, error) {
	var candidates []*models.BountyCandidate
	for rows.Next() {
		var candidate models.BountyCandidate
		if err := rows.Scan(
			&candidate.ID, &candidate.ProjectID, &candidate.ProgramID, &candidate.FindingID, &candidate.EndpointID,
			&candidate.SourceKind, &candidate.Title, &candidate.VulnType, &candidate.Severity, &candidate.Confidence,
			&candidate.ValueScore, &candidate.ImpactScore, &candidate.NoveltyScore, &candidate.ReproScore,
			&candidate.ScopeScore, &candidate.SafetyScore,
			&candidate.DuplicateRisk, &candidate.RankingReason, &candidate.VerifyStatus, &candidate.SubmissionStatus,
			&candidate.SubmissionURL, &candidate.SubmissionID, &candidate.PaidAmount, &candidate.Notes,
			&candidate.CreatedAt, &candidate.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan bounty_candidate: %w", err)
		}
		candidates = append(candidates, &candidate)
	}
	return candidates, nil
}

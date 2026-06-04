package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// CreateSubmissionPack 创建提交包
func (q *Queries) CreateSubmissionPack(pack *models.SubmissionPack) error {
	if pack.ID == "" {
		pack.ID = util.GenerateID()
	}
	now := time.Now().UTC()
	pack.CreatedAt = now
	pack.UpdatedAt = now

	_, err := q.db.Exec(`
		INSERT INTO submission_packs (
			id, candidate_id, format, template, content,
			checklist_json, redaction_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		pack.ID, pack.CandidateID, pack.Format, pack.Template, pack.Content,
		pack.ChecklistJSON, pack.RedactionStatus, pack.CreatedAt, pack.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert submission_pack: %w", err)
	}

	return nil
}

// GetSubmissionPack 获取指定提交包
func (q *Queries) GetSubmissionPack(id string) (*models.SubmissionPack, error) {
	var pack models.SubmissionPack

	err := q.db.QueryRow(`
		SELECT id, candidate_id, format, template, content,
			checklist_json, redaction_status, created_at, updated_at
		FROM submission_packs
		WHERE id = ?
	`, id).Scan(
		&pack.ID, &pack.CandidateID, &pack.Format, &pack.Template, &pack.Content,
		&pack.ChecklistJSON, &pack.RedactionStatus, &pack.CreatedAt, &pack.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query submission_pack: %w", err)
	}

	return &pack, nil
}

// GetSubmissionPackByCandidate 通过候选 ID 获取提交包
func (q *Queries) GetSubmissionPackByCandidate(candidateID string) (*models.SubmissionPack, error) {
	var pack models.SubmissionPack

	err := q.db.QueryRow(`
		SELECT id, candidate_id, format, template, content,
			checklist_json, redaction_status, created_at, updated_at
		FROM submission_packs
		WHERE candidate_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, candidateID).Scan(
		&pack.ID, &pack.CandidateID, &pack.Format, &pack.Template, &pack.Content,
		&pack.ChecklistJSON, &pack.RedactionStatus, &pack.CreatedAt, &pack.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query submission_pack: %w", err)
	}

	return &pack, nil
}

// UpdateSubmissionPack 更新提交包
func (q *Queries) UpdateSubmissionPack(pack *models.SubmissionPack) error {
	pack.UpdatedAt = time.Now().UTC()

	_, err := q.db.Exec(`
		UPDATE submission_packs SET
			content = ?, checklist_json = ?, redaction_status = ?,
			updated_at = ?
		WHERE id = ?
	`,
		pack.Content, pack.ChecklistJSON, pack.RedactionStatus,
		pack.UpdatedAt, pack.ID,
	)
	if err != nil {
		return fmt.Errorf("update submission_pack: %w", err)
	}

	return nil
}

// DeleteSubmissionPack 删除提交包
func (q *Queries) DeleteSubmissionPack(id string) error {
	_, err := q.db.Exec("DELETE FROM submission_packs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete submission_pack: %w", err)
	}
	return nil
}

// ListSubmissionPacksByCandidate 列出候选的所有提交包
func (q *Queries) ListSubmissionPacksByCandidate(candidateID string) ([]*models.SubmissionPack, error) {
	rows, err := q.db.Query(`
		SELECT id, candidate_id, format, template, content,
			checklist_json, redaction_status, created_at, updated_at
		FROM submission_packs
		WHERE candidate_id = ?
		ORDER BY created_at DESC
	`, candidateID)
	if err != nil {
		return nil, fmt.Errorf("query submission_packs: %w", err)
	}
	defer rows.Close()

	var packs []*models.SubmissionPack
	for rows.Next() {
		var pack models.SubmissionPack
		if err := rows.Scan(
			&pack.ID, &pack.CandidateID, &pack.Format, &pack.Template, &pack.Content,
			&pack.ChecklistJSON, &pack.RedactionStatus, &pack.CreatedAt, &pack.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan submission_pack: %w", err)
		}
		packs = append(packs, &pack)
	}

	return packs, nil
}

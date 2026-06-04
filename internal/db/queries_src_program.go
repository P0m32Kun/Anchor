package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// CreateSRCProgram 创建 SRC 程序
func (q *Queries) CreateSRCProgram(program *models.SRCProgram) error {
	if program.ID == "" {
		program.ID = util.GenerateID()
	}
	now := time.Now().UTC()
	program.CreatedAt = now
	program.UpdatedAt = now

	preferredVulnTypes, err := program.MarshalPreferredVulnTypes()
	if err != nil {
		return fmt.Errorf("marshal preferred_vuln_types: %w", err)
	}

	payoutHint, err := program.MarshalPayoutHint()
	if err != nil {
		return fmt.Errorf("marshal payout_hint: %w", err)
	}

	_, err = q.db.Exec(`
		INSERT INTO src_programs (
			id, project_id, name, platform, program_url, rules_url,
			allow_automation, allow_dir_brute, allow_weak_password, allow_authenticated_test,
			max_rps, max_concurrency, preferred_vuln_types, payout_hint, notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		program.ID, program.ProjectID, program.Name, program.Platform,
		program.ProgramURL, program.RulesURL,
		boolToInt(program.AllowAutomation), boolToInt(program.AllowDirBrute),
		boolToInt(program.AllowWeakPassword), boolToInt(program.AllowAuthenticatedTest),
		program.MaxRPS, program.MaxConcurrency,
		preferredVulnTypes, payoutHint, program.Notes,
		program.CreatedAt, program.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert src_program: %w", err)
	}

	return nil
}

// GetSRCProgram 获取指定项目的 SRC 程序
func (q *Queries) GetSRCProgram(projectID string) (*models.SRCProgram, error) {
	var program models.SRCProgram
	var preferredVulnTypes, payoutHint string
	var allowAutomation, allowDirBrute, allowWeakPassword, allowAuthenticatedTest int

	err := q.db.QueryRow(`
		SELECT id, project_id, name, platform, program_url, rules_url,
			allow_automation, allow_dir_brute, allow_weak_password, allow_authenticated_test,
			max_rps, max_concurrency, preferred_vuln_types, payout_hint, notes,
			created_at, updated_at
		FROM src_programs
		WHERE project_id = ?
	`, projectID).Scan(
		&program.ID, &program.ProjectID, &program.Name, &program.Platform,
		&program.ProgramURL, &program.RulesURL,
		&allowAutomation, &allowDirBrute, &allowWeakPassword, &allowAuthenticatedTest,
		&program.MaxRPS, &program.MaxConcurrency,
		&preferredVulnTypes, &payoutHint, &program.Notes,
		&program.CreatedAt, &program.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query src_program: %w", err)
	}

	program.AllowAutomation = intToBool(allowAutomation)
	program.AllowDirBrute = intToBool(allowDirBrute)
	program.AllowWeakPassword = intToBool(allowWeakPassword)
	program.AllowAuthenticatedTest = intToBool(allowAuthenticatedTest)

	if err := program.UnmarshalPreferredVulnTypes(preferredVulnTypes); err != nil {
		return nil, fmt.Errorf("unmarshal preferred_vuln_types: %w", err)
	}
	if err := program.UnmarshalPayoutHint(payoutHint); err != nil {
		return nil, fmt.Errorf("unmarshal payout_hint: %w", err)
	}

	return &program, nil
}

// GetSRCProgramByID 通过 ID 获取 SRC 程序
func (q *Queries) GetSRCProgramByID(id string) (*models.SRCProgram, error) {
	var program models.SRCProgram
	var preferredVulnTypes, payoutHint string
	var allowAutomation, allowDirBrute, allowWeakPassword, allowAuthenticatedTest int

	err := q.db.QueryRow(`
		SELECT id, project_id, name, platform, program_url, rules_url,
			allow_automation, allow_dir_brute, allow_weak_password, allow_authenticated_test,
			max_rps, max_concurrency, preferred_vuln_types, payout_hint, notes,
			created_at, updated_at
		FROM src_programs
		WHERE id = ?
	`, id).Scan(
		&program.ID, &program.ProjectID, &program.Name, &program.Platform,
		&program.ProgramURL, &program.RulesURL,
		&allowAutomation, &allowDirBrute, &allowWeakPassword, &allowAuthenticatedTest,
		&program.MaxRPS, &program.MaxConcurrency,
		&preferredVulnTypes, &payoutHint, &program.Notes,
		&program.CreatedAt, &program.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query src_program: %w", err)
	}

	program.AllowAutomation = intToBool(allowAutomation)
	program.AllowDirBrute = intToBool(allowDirBrute)
	program.AllowWeakPassword = intToBool(allowWeakPassword)
	program.AllowAuthenticatedTest = intToBool(allowAuthenticatedTest)

	if err := program.UnmarshalPreferredVulnTypes(preferredVulnTypes); err != nil {
		return nil, fmt.Errorf("unmarshal preferred_vuln_types: %w", err)
	}
	if err := program.UnmarshalPayoutHint(payoutHint); err != nil {
		return nil, fmt.Errorf("unmarshal payout_hint: %w", err)
	}

	return &program, nil
}

// UpdateSRCProgram 更新 SRC 程序
func (q *Queries) UpdateSRCProgram(program *models.SRCProgram) error {
	program.UpdatedAt = time.Now().UTC()

	preferredVulnTypes, err := program.MarshalPreferredVulnTypes()
	if err != nil {
		return fmt.Errorf("marshal preferred_vuln_types: %w", err)
	}

	payoutHint, err := program.MarshalPayoutHint()
	if err != nil {
		return fmt.Errorf("marshal payout_hint: %w", err)
	}

	_, err = q.db.Exec(`
		UPDATE src_programs SET
			name = ?, platform = ?, program_url = ?, rules_url = ?,
			allow_automation = ?, allow_dir_brute = ?, allow_weak_password = ?, allow_authenticated_test = ?,
			max_rps = ?, max_concurrency = ?, preferred_vuln_types = ?, payout_hint = ?, notes = ?,
			updated_at = ?
		WHERE id = ?
	`,
		program.Name, program.Platform, program.ProgramURL, program.RulesURL,
		boolToInt(program.AllowAutomation), boolToInt(program.AllowDirBrute),
		boolToInt(program.AllowWeakPassword), boolToInt(program.AllowAuthenticatedTest),
		program.MaxRPS, program.MaxConcurrency,
		preferredVulnTypes, payoutHint, program.Notes,
		program.UpdatedAt, program.ID,
	)
	if err != nil {
		return fmt.Errorf("update src_program: %w", err)
	}

	return nil
}

// DeleteSRCProgram 删除 SRC 程序
func (q *Queries) DeleteSRCProgram(projectID string) error {
	_, err := q.db.Exec("DELETE FROM src_programs WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("delete src_program: %w", err)
	}
	return nil
}

// DeleteSRCProgramByID 通过 ID 删除 SRC 程序
func (q *Queries) DeleteSRCProgramByID(id string) error {
	_, err := q.db.Exec("DELETE FROM src_programs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete src_program: %w", err)
	}
	return nil
}

// ListSRCPrograms 列出所有 SRC 程序
func (q *Queries) ListSRCPrograms() ([]*models.SRCProgram, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, name, platform, program_url, rules_url,
			allow_automation, allow_dir_brute, allow_weak_password, allow_authenticated_test,
			max_rps, max_concurrency, preferred_vuln_types, payout_hint, notes,
			created_at, updated_at
		FROM src_programs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query src_programs: %w", err)
	}
	defer rows.Close()

	var programs []*models.SRCProgram
	for rows.Next() {
		var program models.SRCProgram
		var preferredVulnTypes, payoutHint string
		var allowAutomation, allowDirBrute, allowWeakPassword, allowAuthenticatedTest int

		if err := rows.Scan(
			&program.ID, &program.ProjectID, &program.Name, &program.Platform,
			&program.ProgramURL, &program.RulesURL,
			&allowAutomation, &allowDirBrute, &allowWeakPassword, &allowAuthenticatedTest,
			&program.MaxRPS, &program.MaxConcurrency,
			&preferredVulnTypes, &payoutHint, &program.Notes,
			&program.CreatedAt, &program.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan src_program: %w", err)
		}

		program.AllowAutomation = intToBool(allowAutomation)
		program.AllowDirBrute = intToBool(allowDirBrute)
		program.AllowWeakPassword = intToBool(allowWeakPassword)
		program.AllowAuthenticatedTest = intToBool(allowAuthenticatedTest)

		if err := program.UnmarshalPreferredVulnTypes(preferredVulnTypes); err != nil {
			return nil, fmt.Errorf("unmarshal preferred_vuln_types: %w", err)
		}
		if err := program.UnmarshalPayoutHint(payoutHint); err != nil {
			return nil, fmt.Errorf("unmarshal payout_hint: %w", err)
		}

		programs = append(programs, &program)
	}

	return programs, nil
}

// ListSRCProgramsByPlatform 按平台列出 SRC 程序
func (q *Queries) ListSRCProgramsByPlatform(platform string) ([]*models.SRCProgram, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, name, platform, program_url, rules_url,
			allow_automation, allow_dir_brute, allow_weak_password, allow_authenticated_test,
			max_rps, max_concurrency, preferred_vuln_types, payout_hint, notes,
			created_at, updated_at
		FROM src_programs
		WHERE platform = ?
		ORDER BY created_at DESC
	`, platform)
	if err != nil {
		return nil, fmt.Errorf("query src_programs: %w", err)
	}
	defer rows.Close()

	var programs []*models.SRCProgram
	for rows.Next() {
		var program models.SRCProgram
		var preferredVulnTypes, payoutHint string
		var allowAutomation, allowDirBrute, allowWeakPassword, allowAuthenticatedTest int

		if err := rows.Scan(
			&program.ID, &program.ProjectID, &program.Name, &program.Platform,
			&program.ProgramURL, &program.RulesURL,
			&allowAutomation, &allowDirBrute, &allowWeakPassword, &allowAuthenticatedTest,
			&program.MaxRPS, &program.MaxConcurrency,
			&preferredVulnTypes, &payoutHint, &program.Notes,
			&program.CreatedAt, &program.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan src_program: %w", err)
		}

		program.AllowAutomation = intToBool(allowAutomation)
		program.AllowDirBrute = intToBool(allowDirBrute)
		program.AllowWeakPassword = intToBool(allowWeakPassword)
		program.AllowAuthenticatedTest = intToBool(allowAuthenticatedTest)

		if err := program.UnmarshalPreferredVulnTypes(preferredVulnTypes); err != nil {
			return nil, fmt.Errorf("unmarshal preferred_vuln_types: %w", err)
		}
		if err := program.UnmarshalPayoutHint(payoutHint); err != nil {
			return nil, fmt.Errorf("unmarshal payout_hint: %w", err)
		}

		programs = append(programs, &program)
	}

	return programs, nil
}

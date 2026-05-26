package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
)

type targetService struct {
	queries  *db.Queries
	rawDB    *sql.DB
	scopeEng *scope.Engine
}

// NewTargetService creates a new TargetService.
func NewTargetService(queries *db.Queries, rawDB *sql.DB, scopeEng *scope.Engine) TargetService {
	return &targetService{
		queries:  queries,
		rawDB:    rawDB,
		scopeEng: scopeEng,
	}
}

func (s *targetService) Create(ctx context.Context, projectID string, req CreateTargetRequest) (*TargetResponse, error) {
	resolvedType := req.Type
	if resolvedType == "auto" {
		resolvedType = string(scope.DetectType(req.Value))
	}

	rules, err := s.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "list scope rules failed: %v", err)
	}

	if len(rules) == 0 {
		suggestedType := resolvedType
		suggestedValue := req.Value
		// 对 IP 目标建议使用 CIDR /32 规则，确保 scope check 能精确覆盖
		if resolvedType == string(models.TargetTypeIP) {
			suggestedType = string(models.TargetTypeCIDR)
			suggestedValue = req.Value + "/32"
		}
		return &TargetResponse{
			NeedsScopeConfirmation: true,
			Message:                "当前项目未设置授权范围，是否将此目标自动加入授权范围？",
			SuggestedRule: &ScopeRuleSuggestion{
				Action: "include",
				Type:   suggestedType,
				Value:  suggestedValue,
			},
		}, nil
	}

	t := &models.Target{
		ID:        util.GenerateID(),
		ProjectID: projectID,
		Type:      models.TargetType(resolvedType),
		Value:     req.Value,
		Source:    "manual",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}

	if err := s.queries.CreateTarget(t); err != nil {
		return nil, errors.Newf(errors.ErrInternal, "create target failed: %v", err)
	}

	return &TargetResponse{Target: t}, nil
}

func (s *targetService) List(ctx context.Context, projectID string) ([]*models.Target, error) {
	targets, err := s.queries.ListTargetsByProject(projectID)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "list targets failed: %v", err)
	}
	return targets, nil
}

func (s *targetService) ListPaginated(ctx context.Context, projectID string, p PaginationParams) (*PaginatedList[*models.Target], error) {
	total, err := s.queries.CountTargetsByProject(projectID)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "count targets failed: %v", err)
	}
	targets, err := s.queries.ListTargetsByProjectPaginated(projectID, p.PageSize, p.Offset())
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "list targets failed: %v", err)
	}
	return &PaginatedList[*models.Target]{
		Data:     targets,
		Total:    total,
		Page:     p.Page,
		PageSize: p.PageSize,
	}, nil
}

func (s *targetService) Import(ctx context.Context, projectID string, targets []ImportTarget) (*ImportResult, error) {
	rules, err := s.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternal, "list scope rules failed: %v", err)
	}

	if len(rules) == 0 {
		suggested := buildSuggestedScopeRules(targets)
		if len(suggested) > 0 {
			return &ImportResult{
				NeedsScopeConfirmation: true,
				Message:                "当前项目未设置授权范围，是否将导入的目标自动加入授权范围？",
				SuggestedRules:         suggested,
			}, nil
		}
	}

	// 与 Create 一致：导入阶段不做 scope 校验，授权边界在 dry-run / 扫描流水线 enforced。
	return s.importTargets(projectID, targets)
}

func buildSuggestedScopeRules(targets []ImportTarget) []ScopeRuleSuggestion {
	var suggested []ScopeRuleSuggestion
	seen := make(map[string]bool)
	for _, t := range targets {
		// company 目标在流水线中直接透传，不需要 include scope 规则。
		if t.Type == models.TargetTypeCompany {
			continue
		}
		key := string(t.Type) + ":" + t.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		suggestedType := string(t.Type)
		suggestedValue := t.Value
		if t.Type == models.TargetTypeIP {
			suggestedType = string(models.TargetTypeCIDR)
			suggestedValue = t.Value + "/32"
		}
		suggested = append(suggested, ScopeRuleSuggestion{
			Action: "include",
			Type:   suggestedType,
			Value:  suggestedValue,
		})
	}
	return suggested
}

func (s *targetService) importTargets(projectID string, targets []ImportTarget) (*ImportResult, error) {
	now := time.Now().UTC()
	result := &ImportResult{}
	var toInsert []*models.Target

	for _, pt := range targets {
		if pt.Value == "" {
			result.Errors++
			continue
		}
		exists, dbErr := s.queries.TargetExistsByValue(projectID, pt.Value)
		if dbErr != nil {
			result.Errors++
			continue
		}
		if exists {
			result.Duplicates++
			continue
		}

		t := &models.Target{
			ID:        util.GenerateID(),
			ProjectID: projectID,
			Type:      pt.Type,
			Value:     pt.Value,
			Source:    "import",
			Status:    "active",
			CreatedAt: now,
		}
		toInsert = append(toInsert, t)
		result.Targets = append(result.Targets, t)
		result.Imported++
	}

	if len(toInsert) > 0 {
		txErr := db.WithTx(s.rawDB, func(tx *db.Queries) error {
			return tx.BulkCreateTargets(toInsert)
		})
		if txErr != nil {
			return nil, errors.Newf(errors.ErrInternal, "bulk insert failed: %v", txErr)
		}
	}

	return result, nil
}

func (s *targetService) Delete(ctx context.Context, targetID string) error {
	if err := s.queries.DeleteTarget(targetID); err != nil {
		return errors.Newf(errors.ErrInternal, "delete target failed: %v", err)
	}
	return nil
}

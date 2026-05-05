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
		return &TargetResponse{
			NeedsScopeConfirmation: true,
			Message:                "当前项目未设置授权范围，是否将此目标自动加入授权范围？",
			SuggestedRule: &ScopeRuleSuggestion{
				Action: "include",
				Type:   resolvedType,
				Value:  req.Value,
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
		var suggested []ScopeRuleSuggestion
		seen := make(map[string]bool)
		for _, t := range targets {
			key := string(t.Type) + ":" + t.Value
			if seen[key] {
				continue
			}
			seen[key] = true
			suggested = append(suggested, ScopeRuleSuggestion{
				Action: "include",
				Type:   string(t.Type),
				Value:  t.Value,
			})
		}
		return &ImportResult{
			NeedsScopeConfirmation: true,
			Message:                "当前项目未设置授权范围，是否将导入的目标自动加入授权范围？",
			SuggestedRules:         suggested,
		}, nil
	}

	now := time.Now().UTC()
	result := &ImportResult{}
	var toInsert []*models.Target

	for _, pt := range targets {
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

		decision, chkErr := s.scopeEng.Check(ctx, projectID, t)
		if chkErr != nil {
			result.Errors++
			continue
		}

		if decision.Decision == models.ScopeDeny {
			result.Denied++
			result.DeniedTargets = append(result.DeniedTargets, DeniedTarget{Value: t.Value, Reason: decision.Reason})
			continue
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

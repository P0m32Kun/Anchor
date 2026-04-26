package scope

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"secbench/internal/db"
	"secbench/internal/models"
	"secbench/internal/util"
)

// Engine performs scope checks against targets.
type Engine struct {
	queries *db.Queries
}

func NewEngine(q *db.Queries) *Engine {
	return &Engine{queries: q}
}

// Check evaluates a target against the project's scope rules.
// It persists the decision and returns it.
func (e *Engine) Check(ctx context.Context, projectID string, target *models.Target) (*models.ScopeDecision, error) {
	rules, err := e.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("list scope rules: %w", err)
	}

	decision, matchedRule, reason := e.evaluate(target, rules)

	d := &models.ScopeDecision{
		ID:          util.GenerateID(),
		ProjectID:   projectID,
		TargetValue: target.Value,
		Decision:    decision,
		Reason:      reason,
		CreatedAt:   time.Now().UTC(),
	}
	if matchedRule != nil {
		d.MatchedRuleID = &matchedRule.ID
	}

	if err := e.queries.CreateScopeDecision(d); err != nil {
		return nil, fmt.Errorf("persist scope decision: %w", err)
	}
	return d, nil
}

// ValidateBeforeRun performs TOCTOU check: re-validates scope decision freshness.
func (e *Engine) ValidateBeforeRun(ctx context.Context, projectID string, target *models.Target, taskID string) (*models.ScopeDecision, error) {
	latest, err := e.queries.GetLatestScopeDecision(projectID, target.Value)
	if err != nil {
		return nil, fmt.Errorf("get latest scope decision: %w", err)
	}
	if latest == nil {
		return e.Check(ctx, projectID, target)
	}

	maxUpdated, err := e.queries.GetMaxScopeRuleUpdatedAt(projectID)
	if err != nil {
		return nil, fmt.Errorf("get max scope rule updated at: %w", err)
	}

	// If scope rules changed after the last decision, re-check.
	if maxUpdated.After(latest.CreatedAt) {
		return e.Check(ctx, projectID, target)
	}

	// Attach task ID if not already set.
	if latest.TaskID == nil {
		// In real implementation we'd update the record; for MVP just return.
		latest.TaskID = &taskID
	}
	return latest, nil
}

func (e *Engine) evaluate(target *models.Target, rules []*models.ScopeRule) (models.ScopeDecisionResult, *models.ScopeRule, string) {
	var includeMatched, excludeMatched *models.ScopeRule

	for _, rule := range rules {
		matched := e.matchRule(target, rule)
		if !matched {
			continue
		}
		switch rule.Action {
		case models.ScopeActionInclude:
			if includeMatched == nil {
				includeMatched = rule
			}
		case models.ScopeActionExclude:
			if excludeMatched == nil {
				excludeMatched = rule
			}
		}
	}

	// Exclude has priority over include.
	if excludeMatched != nil {
		return models.ScopeDeny, excludeMatched, fmt.Sprintf("命中排除规则: %s", excludeMatched.Value)
	}
	if includeMatched != nil {
		return models.ScopeAllow, includeMatched, fmt.Sprintf("命中包含规则: %s", includeMatched.Value)
	}
	// No rule matched → deny by default (whitelist mode).
	return models.ScopeDeny, nil, "未命中任何包含规则，默认拒绝"
}

func (e *Engine) matchRule(target *models.Target, rule *models.ScopeRule) bool {
	switch target.Type {
	case models.TargetTypeDomain:
		return e.matchDomain(target.Value, rule)
	case models.TargetTypeURL:
		return e.matchURL(target.Value, rule)
	case models.TargetTypeIP, models.TargetTypeCIDR:
		return e.matchIP(target.Value, rule)
	}
	return false
}

func (e *Engine) matchDomain(domain string, rule *models.ScopeRule) bool {
	switch rule.Type {
	case models.TargetTypeDomain:
		return matchDomainRule(domain, rule.Value)
	case models.TargetTypeURL:
		// URL rule can match domain if the URL's host matches.
		u, err := url.Parse(rule.Value)
		if err != nil {
			return false
		}
		return matchDomainRule(domain, u.Host)
	}
	return false
}

func matchDomainRule(domain, ruleValue string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	ruleValue = strings.ToLower(strings.TrimSpace(ruleValue))

	// Exact match.
	if domain == ruleValue {
		return true
	}

	// Wildcard prefix: *.example.com
	if strings.HasPrefix(ruleValue, "*.") {
		suffix := ruleValue[2:]
		// Must end with .example.com and have exactly one prefix part.
		if strings.HasSuffix(domain, "."+suffix) {
			prefix := strings.TrimSuffix(domain, "."+suffix)
			// a.example.com -> prefix="a", no dots -> match
			// a.b.example.com -> prefix="a.b", has dot -> no match
			if prefix != "" && !strings.Contains(prefix, ".") {
				return true
			}
		}
		return false
	}

	// Subdomain match: example.com matches sub.example.com, a.b.example.com.
	if strings.HasSuffix(domain, "."+ruleValue) {
		return true
	}

	return false
}

func (e *Engine) matchURL(targetURL string, rule *models.ScopeRule) bool {
	switch rule.Type {
	case models.TargetTypeURL:
		return strings.HasPrefix(targetURL, rule.Value)
	case models.TargetTypeDomain:
		u, err := url.Parse(targetURL)
		if err != nil {
			return false
		}
		return matchDomainRule(u.Host, rule.Value)
	}
	return false
}

func (e *Engine) matchIP(ipStr string, rule *models.ScopeRule) bool {
	switch rule.Type {
	case models.TargetTypeIP:
		return ipStr == rule.Value
	case models.TargetTypeCIDR:
		_, cidr, err := net.ParseCIDR(rule.Value)
		if err != nil {
			return false
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		return cidr.Contains(ip)
	}
	return false
}

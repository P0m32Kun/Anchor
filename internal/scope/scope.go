package scope

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// Engine performs scope checks against targets.
type Engine struct {
	queries *db.Queries
}

func NewEngine(q *db.Queries) *Engine {
	return &Engine{queries: q}
}

// Check evaluates a target against the project's scope rules and time window.
// It persists the decision and returns it.
func (e *Engine) Check(ctx context.Context, projectID string, target *models.Target) (*models.ScopeDecision, error) {
	// Fetch project for time window and rate limit checks.
	project, err := e.queries.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}


	// Rate limit validation.
	if project.RateLimit < 0 {
		d := &models.ScopeDecision{
			ID:          util.GenerateID(),
			ProjectID:   projectID,
			TargetValue: target.Value,
			Decision:    models.ScopeDeny,
			Reason:      fmt.Sprintf("无效的速率限制配置: %d", project.RateLimit),
			CreatedAt:   time.Now().UTC(),
		}
		if err := e.queries.CreateScopeDecision(d); err != nil {
			return nil, fmt.Errorf("persist scope decision: %w", err)
		}
		return d, nil
	}

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

// checkTimeWindow returns a deny reason if the current time is outside
// the project's configured time window. Returns empty string if in-window or unconfigured.

// ValidateBeforeRun performs TOCTOU check: re-validates scope decision freshness.
// It always re-checks if the project time window or rate limit has changed,
// because these can expire or be modified at any time.
func (e *Engine) ValidateBeforeRun(ctx context.Context, projectID string, target *models.Target, taskID string) (*models.ScopeDecision, error) {
	project, err := e.queries.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	// If time window or rate limit would currently deny, force a fresh Check.
	if project.RateLimit < 0 {
		return e.Check(ctx, projectID, target)
	}

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

// ExpandCIDR expands a CIDR notation into all usable host IP addresses.
// It skips the network and broadcast addresses for subnets /30 and larger.
func ExpandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("parse CIDR %q: %w", cidr, err)
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network address and broadcast address for subnets larger than /30.
	ones, bits := ipnet.Mask.Size()
	if ones <= bits-2 && len(ips) >= 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

// inc increments an IP address.
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// CheckIP is a convenience wrapper around Check for IP targets.
func (e *Engine) CheckIP(ctx context.Context, projectID, ip string) (*models.ScopeDecision, error) {
	target := &models.Target{
		Type:  models.TargetTypeIP,
		Value: ip,
	}
	return e.Check(ctx, projectID, target)
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
	case models.TargetTypeIP:
		u, err := url.Parse(targetURL)
		if err != nil {
			return false
		}
		host := u.Host
		if idx := strings.Index(host, ":"); idx >= 0 {
			host = host[:idx]
		}
		return host == rule.Value
	case models.TargetTypeCIDR:
		u, err := url.Parse(targetURL)
		if err != nil {
			return false
		}
		host := u.Host
		if idx := strings.Index(host, ":"); idx >= 0 {
			host = host[:idx]
		}
		return e.matchIP(host, rule)
	}
	return false
}

func (e *Engine) matchIP(ipStr string, rule *models.ScopeRule) bool {
	switch rule.Type {
	case models.TargetTypeIP:
		// 精确匹配单个 IP
		if ipStr == rule.Value {
			return true
		}
		// 如果目标是 CIDR（如 /32），检查其网络地址是否匹配
		if strings.Contains(ipStr, "/") {
			_, cidr, err := net.ParseCIDR(ipStr)
			if err == nil && cidr.IP.String() == rule.Value {
				return true
			}
		}
		return false
	case models.TargetTypeCIDR:
		_, ruleCIDR, err := net.ParseCIDR(rule.Value)
		if err != nil {
			return false
		}
		// 如果目标也是 CIDR，检查是否完全相同
		if strings.Contains(ipStr, "/") {
			_, targetCIDR, err := net.ParseCIDR(ipStr)
			if err != nil {
				return false
			}
			return ruleCIDR.IP.Equal(targetCIDR.IP) && ruleCIDR.Mask.String() == targetCIDR.Mask.String()
		}
		// 目标是单个 IP，检查是否在 CIDR 范围内
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false
		}
		return ruleCIDR.Contains(ip)
	}
	return false
}

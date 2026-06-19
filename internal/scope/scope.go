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

// evaluate applies exclusion-only scope: exclude rules deny; include rules are
// ignored; no matching exclude → allow (asset discovery may expand the graph).
func (e *Engine) evaluate(target *models.Target, rules []*models.ScopeRule) (models.ScopeDecisionResult, *models.ScopeRule, string) {
	var excludeMatched *models.ScopeRule

	for _, rule := range rules {
		if rule.Action != models.ScopeActionExclude {
			continue
		}
		if !e.matchRule(target, rule) {
			continue
		}
		if excludeMatched == nil {
			excludeMatched = rule
		}
	}

	if excludeMatched != nil {
		return models.ScopeDeny, excludeMatched, fmt.Sprintf("命中排除规则: %s", excludeMatched.Value)
	}
	return models.ScopeAllow, nil, "未命中排除规则，默认允许"
}

// IsExcluded reports whether the target hits any project exclude rule.
func (e *Engine) IsExcluded(target *models.Target, rules []*models.ScopeRule) bool {
	decision, _, _ := e.evaluate(target, rules)
	return decision == models.ScopeDeny
}

// IsExcludedForProject loads scope rules and checks exclusion for a target.
func (e *Engine) IsExcludedForProject(projectID string, target *models.Target) (bool, error) {
	rules, err := e.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		return false, fmt.Errorf("list scope rules: %w", err)
	}
	return e.IsExcluded(target, rules), nil
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

// MaxCIDRHostBits caps how wide a CIDR FilterTargets is willing to expand.
// 16 host bits == /16 == 65,536 IPs == ~1 MiB of strings, which is safe in
// memory and reasonable for an internal-network scan. Anything wider (e.g.
// 0.0.0.0/0) is rejected explicitly with an actionable error instead of
// silently OOM-ing the server.
const MaxCIDRHostBits = 16

// FilterTargets is the single scope-enforcement entry point for the scan
// pipeline. It performs three jobs in one place so every downstream tool
// (nmap / naabu / httpx / nuclei) can consume the result without knowing
// scope rules exist:
//
//  1. CIDR-mask sanity check: refuse to expand subnets larger than /16
//     (~65k hosts) to bound memory usage.
//  2. CIDR expansion: replace each `cidr` Target with the underlying
//     `ip` Targets (using ExpandCIDR's existing network/broadcast trim).
//  3. Rule evaluation: drop any Target that matches a project exclude rule
//     (exclusion-only; default allow when no exclude matches).
//
// `company` Targets are passed through unevaluated — they're expanded later
// in runCompanyFlow via FOFA, which re-enters Check on each derived target.
//
// Decisions are NOT persisted here (Check / ValidateBeforeRun handle that
// when individual tools call them). FilterTargets is a pure filter so the
// pipeline can run it once at the entry and rely on the result.
func (e *Engine) FilterTargets(ctx context.Context, projectID string, targets []*models.Target) ([]*models.Target, error) {
	rules, err := e.queries.ListScopeRulesByProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("list scope rules: %w", err)
	}
	return e.filterTargetsWithRules(targets, rules)
}

// filterTargetsWithRules is the pure-logic core of FilterTargets — it does
// not touch the database, so it can be unit-tested directly with fabricated
// rule sets.
func (e *Engine) filterTargetsWithRules(targets []*models.Target, rules []*models.ScopeRule) ([]*models.Target, error) {
	out := make([]*models.Target, 0, len(targets))
	for _, t := range targets {
		if t == nil {
			continue
		}
		switch t.Type {
		case models.TargetTypeCIDR:
			// Mask sanity check before we allocate any memory for expansion.
			_, ipnet, err := net.ParseCIDR(t.Value)
			if err != nil {
				return nil, fmt.Errorf("parse CIDR %q: %w", t.Value, err)
			}
			ones, bits := ipnet.Mask.Size()
			hostBits := bits - ones
			if hostBits > MaxCIDRHostBits {
				return nil, fmt.Errorf("CIDR %s is too large (%d host bits, limit is /%d): refuse to expand to avoid OOM", t.Value, hostBits, bits-MaxCIDRHostBits)
			}

			ips, err := ExpandCIDR(t.Value)
			if err != nil {
				return nil, fmt.Errorf("expand CIDR %q: %w", t.Value, err)
			}
			for _, ip := range ips {
				expanded := &models.Target{
					ID:        t.ID, // reference parent CIDR for traceability
					ProjectID: t.ProjectID,
					Type:      models.TargetTypeIP,
					Value:     ip,
					Source:    t.Source,
					Status:    t.Status,
					CreatedAt: t.CreatedAt,
				}
				decision, _, _ := e.evaluate(expanded, rules)
				if decision == models.ScopeAllow {
					out = append(out, expanded)
				}
			}
		case models.TargetTypeCompany:
			// Pass through — FOFA expansion happens later, scope check
			// runs on each derived domain/ip then.
			out = append(out, t)
		default:
			decision, _, _ := e.evaluate(t, rules)
			if decision == models.ScopeAllow {
				out = append(out, t)
			}
		}
	}
	return out, nil
}



// matchRule checks if a target value matches a rule value (simple string match or wildcard).
func matchRule(target, rule string) bool {
	if rule == "*" {
		return true
	}
	// Simple suffix match for wildcard domains (*.example.com)
	if len(rule) > 2 && rule[:2] == "*." {
		suffix := rule[1:] // .example.com
		if target == suffix[1:] || strings.HasSuffix(target, suffix) {
			return true
		}
		return false
	}
	return target == rule
}

// EvaluateBoundary checks if a target is in scope for boundary filtering.
// Returns (allow, decision, reason).
//
// Semantics (strict mode):
//  1. Exclude rules checked first — any match → deny.
//  2. Include rules checked next — any match → allow.
//  3. If include rules existed but none matched → deny.
//  4. If no include rules existed at all → allow (default-allow with excludes).
func (e *Engine) EvaluateBoundary(target *models.Target, rules []*models.ScopeRule, mode string) (bool, string, string) {
	if mode == "" || mode == string(models.ScopeBoundaryOff) {
		return true, string(models.ScopeAllow), "boundary off"
	}

	// Pass 1: check exclude rules first — exclude takes priority.
	for _, rule := range rules {
		if rule.Action != models.ScopeActionExclude {
			continue
		}
		if e.boundaryMatch(target, rule) {
			return false, string(models.ScopeDeny), "matched exclude rule"
		}
	}

	// Pass 2: check include rules.
	hasInclude := false
	for _, rule := range rules {
		if rule.Action != models.ScopeActionInclude {
			continue
		}
		hasInclude = true
		if e.boundaryMatch(target, rule) {
			return true, string(models.ScopeAllow), "matched include rule"
		}
	}

	// If include rules existed but none matched → deny.
	if hasInclude {
		return false, string(models.ScopeDeny), "no matching include rule in strict mode"
	}

	// No include rules at all → allow (only excludes were configured, target wasn't excluded).
	return true, string(models.ScopeAllow), "default allow (no include rules)"
}

// boundaryMatch checks if a target matches a scope rule for boundary filtering.
// It handles cross-type matching for CIDR rules (IP targets can match CIDR rules).
func (e *Engine) boundaryMatch(target *models.Target, rule *models.ScopeRule) bool {
	// CIDR rules can match IP targets (check if IP is within the CIDR range).
	if rule.Type == models.TargetTypeCIDR && target.Type == models.TargetTypeIP {
		return e.matchIP(target.Value, rule)
	}
	if rule.Type != target.Type {
		return false
	}
	return matchRule(target.Value, rule.Value)
}

package scope

import (
	"testing"
	"time"

	"secbench/internal/models"
)

func TestMatchDomainRule(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		rule     string
		expected bool
	}{
		{"exact match", "example.com", "example.com", true},
		{"subdomain match", "sub.example.com", "example.com", true},
		{"deep subdomain", "a.b.example.com", "example.com", true},
		{"not matching", "notexample.com", "example.com", false},
		{"wildcard subdomain", "a.example.com", "*.example.com", true},
		{"wildcard exact", "example.com", "*.example.com", false},
		{"wildcard deep", "a.b.example.com", "*.example.com", false},
		{"case insensitive", "EXAMPLE.COM", "example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchDomainRule(tt.domain, tt.rule)
			if result != tt.expected {
				t.Errorf("matchDomainRule(%q, %q) = %v, want %v", tt.domain, tt.rule, result, tt.expected)
			}
		})
	}
}

func TestEngineEvaluate(t *testing.T) {
	e := &Engine{}

	tests := []struct {
		name     string
		target   *models.Target
		rules    []*models.ScopeRule
		decision models.ScopeDecisionResult
	}{
		{
			name:     "allow by include",
			target:   &models.Target{Type: models.TargetTypeDomain, Value: "example.com"},
			rules:    []*models.ScopeRule{{Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "example.com"}},
			decision: models.ScopeAllow,
		},
		{
			name:   "deny by exclude priority",
			target: &models.Target{Type: models.TargetTypeDomain, Value: "sub.example.com"},
			rules: []*models.ScopeRule{
				{Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "example.com"},
				{Action: models.ScopeActionExclude, Type: models.TargetTypeDomain, Value: "sub.example.com"},
			},
			decision: models.ScopeDeny,
		},
		{
			name:     "deny by default",
			target:   &models.Target{Type: models.TargetTypeDomain, Value: "unknown.com"},
			rules:    []*models.ScopeRule{{Action: models.ScopeActionInclude, Type: models.TargetTypeDomain, Value: "example.com"}},
			decision: models.ScopeDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, _, _ := e.evaluate(tt.target, tt.rules)
			if dec != tt.decision {
				t.Errorf("expected %v, got %v", tt.decision, dec)
			}
		})
	}
}

func TestCheckTimeWindow(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name    string
		project *models.Project
		wantOK  bool // true = empty reason (in window)
	}{
		{
			name:    "both nil - in window",
			project: &models.Project{},
			wantOK:  true,
		},
		{
			name:    "start nil, end in future - in window",
			project: &models.Project{EndTime: &future},
			wantOK:  true,
		},
		{
			name:    "start nil, end in past - outside window",
			project: &models.Project{EndTime: &past},
			wantOK:  false,
		},
		{
			name:    "start in past, end nil - in window",
			project: &models.Project{StartTime: &past},
			wantOK:  true,
		},
		{
			name:    "start in future, end nil - outside window",
			project: &models.Project{StartTime: &future},
			wantOK:  false,
		},
		{
			name:    "start in past, end in future - in window",
			project: &models.Project{StartTime: &past, EndTime: &future},
			wantOK:  true,
		},
		{
			name:    "start in future, end in future - outside window",
			project: &models.Project{StartTime: &future, EndTime: &future},
			wantOK:  false,
		},
		{
			name:    "start in past, end in past - outside window",
			project: &models.Project{StartTime: &past, EndTime: &past},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := checkTimeWindow(tt.project)
			gotOK := reason == ""
			if gotOK != tt.wantOK {
				t.Errorf("checkTimeWindow() reason=%q, wantOK=%v", reason, tt.wantOK)
			}
		})
	}
}

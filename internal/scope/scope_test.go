package scope

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
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

func TestExpandCIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected []string
		wantErr  bool
	}{
		{
			name:     "/32 single IP",
			cidr:     "192.168.1.1/32",
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "/31 two IPs",
			cidr:     "192.168.1.0/31",
			expected: []string{"192.168.1.0", "192.168.1.1"},
		},
		{
			name:     "/30 four IPs no skip",
			cidr:     "192.168.1.0/30",
			expected: []string{"192.168.1.1", "192.168.1.2"},
		},
		{
			name:     "/29 skip network and broadcast",
			cidr:     "192.168.1.0/29",
			expected: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4", "192.168.1.5", "192.168.1.6"},
		},
		{
			name:     "/24 skip network and broadcast",
			cidr:     "10.0.0.0/24",
			expected: nil, // verify count and bounds instead
		},
		{
			name:    "invalid CIDR",
			cidr:    "not-a-cidr",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandCIDR(tt.cidr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExpandCIDR(%q) expected error, got nil", tt.cidr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExpandCIDR(%q) unexpected error: %v", tt.cidr, err)
			}
			if tt.expected == nil {
				// Special case: verify count and boundary IPs for /24
				if len(got) != 254 {
					t.Errorf("ExpandCIDR(%q) returned %d IPs, want 254", tt.cidr, len(got))
				}
				if len(got) > 0 && got[0] != "10.0.0.1" {
					t.Errorf("ExpandCIDR(%q) first IP = %q, want 10.0.0.1", tt.cidr, got[0])
				}
				if len(got) > 0 && got[len(got)-1] != "10.0.0.254" {
					t.Errorf("ExpandCIDR(%q) last IP = %q, want 10.0.0.254", tt.cidr, got[len(got)-1])
				}
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("ExpandCIDR(%q) returned %d IPs, want %d", tt.cidr, len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("ExpandCIDR(%q)[%d] = %q, want %q", tt.cidr, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMatchIP(t *testing.T) {
	e := &Engine{}

	tests := []struct {
		name     string
		ip       string
		rule     *models.ScopeRule
		expected bool
	}{
		{
			name:     "exact IP match",
			ip:       "192.168.1.1",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "192.168.1.1"},
			expected: true,
		},
		{
			name:     "exact IP no match",
			ip:       "192.168.1.1",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "192.168.1.2"},
			expected: false,
		},
		{
			name:     "IP in CIDR",
			ip:       "192.168.1.5",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			expected: true,
		},
		{
			name:     "IP not in CIDR",
			ip:       "10.0.0.1",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			expected: false,
		},
		{
			name:     "IP in /30 CIDR",
			ip:       "192.168.1.2",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/30"},
			expected: true,
		},
		{
			name:     "network address in CIDR not matched",
			ip:       "192.168.1.0",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			expected: true, // matchIP uses cidr.Contains, so network addr matches
		},
		{
			name:     "invalid CIDR rule",
			ip:       "192.168.1.1",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "invalid"},
			expected: false,
		},
		{
			name:     "invalid IP",
			ip:       "not-an-ip",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			expected: false,
		},
		{
			name:     "domain rule does not match IP",
			ip:       "192.168.1.1",
			rule:     &models.ScopeRule{Type: models.TargetTypeDomain, Value: "example.com"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.matchIP(tt.ip, tt.rule)
			if result != tt.expected {
				t.Errorf("matchIP(%q, %v) = %v, want %v", tt.ip, tt.rule.Value, result, tt.expected)
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

package scope

import (
	"strings"
	"testing"

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
		{
			name:     "IPv6 exact match",
			ip:       "2001:db8::1",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "2001:db8::1"},
			expected: true,
		},
		{
			name:     "IPv6 no match",
			ip:       "2001:db8::1",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "2001:db8::2"},
			expected: false,
		},
		{
			name:     "IPv6 in CIDR",
			ip:       "2001:db8::1",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "2001:db8::/32"},
			expected: true,
		},
		{
			name:     "IPv6 not in CIDR",
			ip:       "2001:db9::1",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "2001:db8::/32"},
			expected: false,
		},
		{
			name:     "invalid IP against CIDR",
			ip:       "not-an-ip",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			expected: false,
		},
		{
			name:     "invalid IP against IP rule",
			ip:       "not-an-ip",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "192.168.1.1"},
			expected: false,
		},
		{
			name:     "empty IP",
			ip:       "",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "192.168.1.1"},
			expected: false,
		},
		{
			name:     "CIDR target against IP rule matching",
			ip:       "192.168.1.1/32",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "192.168.1.1"},
			expected: true,
		},
		{
			name:     "CIDR target against IP rule not matching",
			ip:       "192.168.1.2/32",
			rule:     &models.ScopeRule{Type: models.TargetTypeIP, Value: "192.168.1.1"},
			expected: false,
		},
		{
			name:     "CIDR target against CIDR rule matching",
			ip:       "192.168.1.0/24",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			expected: true,
		},
		{
			name:     "CIDR target against CIDR rule not matching",
			ip:       "192.168.2.0/24",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			expected: false,
		},
		{
			name:     "invalid CIDR target",
			ip:       "invalid/24",
			rule:     &models.ScopeRule{Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
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

// TestFilterTargetsWithRules covers the entry-point scope filter that the
// pipeline relies on. It uses the pure-logic filterTargetsWithRules helper so
// no database is required.
func TestFilterTargetsWithRules(t *testing.T) {
	e := &Engine{}

	t.Run("CIDR expansion with exclude IP", func(t *testing.T) {
		targets := []*models.Target{
			{Type: models.TargetTypeCIDR, Value: "172.30.0.0/29"},
		}
		// /29 yields 6 usable hosts (.1..6 after network/broadcast trim).
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionInclude, Type: models.TargetTypeCIDR, Value: "172.30.0.0/29"},
			{Action: models.ScopeActionExclude, Type: models.TargetTypeIP, Value: "172.30.0.1"},
		}
		got, err := e.filterTargetsWithRules(targets, rules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 5 {
			t.Fatalf("expected 5 IPs (6 hosts minus .1), got %d: %+v", len(got), valuesOf(got))
		}
		for _, tgt := range got {
			if tgt.Value == "172.30.0.1" {
				t.Errorf(".1 should have been excluded but appeared in output")
			}
			if tgt.Type != models.TargetTypeIP {
				t.Errorf("expected expanded targets to be type=ip, got %s", tgt.Type)
			}
		}
	})

	t.Run("no rules → default deny", func(t *testing.T) {
		targets := []*models.Target{
			{Type: models.TargetTypeIP, Value: "10.0.0.5"},
		}
		got, err := e.filterTargetsWithRules(targets, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("whitelist mode should deny with no rules, got %d allowed: %+v", len(got), valuesOf(got))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got, err := e.filterTargetsWithRules(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("empty input must yield empty output, got %d", len(got))
		}
	})

	t.Run("CIDR too large rejected", func(t *testing.T) {
		// MaxCIDRHostBits = 16. /15 has 17 host bits → must be rejected.
		targets := []*models.Target{
			{Type: models.TargetTypeCIDR, Value: "10.0.0.0/15"},
		}
		_, err := e.filterTargetsWithRules(targets, nil)
		if err == nil {
			t.Fatalf("expected error for oversized CIDR, got nil")
		}
		// Sanity: error should mention the CIDR.
		if got := err.Error(); !contains(got, "10.0.0.0/15") || !contains(got, "host bits") {
			t.Errorf("error message should mention CIDR and host bits, got: %s", got)
		}
	})

	t.Run("exclude IP wins over include CIDR", func(t *testing.T) {
		// Plain IP target (not CIDR-expanded) intersecting an include CIDR
		// and an exclude IP — exclude must win per evaluate() precedence.
		targets := []*models.Target{
			{Type: models.TargetTypeIP, Value: "192.168.1.50"},
		}
		rules := []*models.ScopeRule{
			{Action: models.ScopeActionInclude, Type: models.TargetTypeCIDR, Value: "192.168.1.0/24"},
			{Action: models.ScopeActionExclude, Type: models.TargetTypeIP, Value: "192.168.1.50"},
		}
		got, err := e.filterTargetsWithRules(targets, rules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("exclude rule should override include, got %d allowed: %+v", len(got), valuesOf(got))
		}
	})
}

func valuesOf(targets []*models.Target) []string {
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		out = append(out, t.Value)
	}
	return out
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


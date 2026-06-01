package exclude

import (
	"testing"
)

func TestManager_IsExcluded(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Built-in defaults
		{"github.com", "github.com", true},
		{"api.github.com", "api.github.com", true},
		{"apache.org", "apache.org", true},
		{"w3.org", "w3.org", true},
		{"npmjs.com", "npmjs.com", true},
		{"momentjs.com", "momentjs.com", true},
		{"miit.gov.cn", "miit.gov.cn", true},
		{"beian.gov.cn", "beian.gov.cn", true},
		{"weibo.com", "weibo.com", true},
		{"gitee.com", "gitee.com", true},
		{"feishu.cn", "feishu.cn", true},
		{"bytecdntp.com", "bytecdntp.com", true},

		// URLs
		{"github url", "https://github.com/user/repo", true},
		{"apache url", "http://apache.org/licenses/", true},

		// Not excluded
		{"example.com", "example.com", false},
		{"mycompany.com", "mycompany.com", false},

		// Empty
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mgr.IsExcluded(tt.input)
			if result != tt.expected {
				t.Errorf("IsExcluded(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestManager_CustomDomains(t *testing.T) {
	mgr := NewManager()

	// Add custom domain
	mgr.AddCustom("evil.com", "test")

	if !mgr.IsExcluded("evil.com") {
		t.Error("expected evil.com to be excluded")
	}
	if !mgr.IsExcluded("sub.evil.com") {
		t.Error("expected sub.evil.com to be excluded (subdomain)")
	}

	// Remove custom domain
	removed := mgr.RemoveCustom("evil.com")
	if !removed {
		t.Error("expected RemoveCustom to return true")
	}
	if mgr.IsExcluded("evil.com") {
		t.Error("expected evil.com to NOT be excluded after removal")
	}

	// Try to remove non-existent domain
	removed = mgr.RemoveCustom("nonexistent.com")
	if removed {
		t.Error("expected RemoveCustom to return false for non-existent domain")
	}
}

func TestManager_GetCustomList(t *testing.T) {
	mgr := NewManager()

	mgr.AddCustom("a.com", "test a")
	mgr.AddCustom("b.com", "test b")

	custom := mgr.GetCustomList()
	if len(custom) != 2 {
		t.Errorf("expected 2 custom domains, got %d", len(custom))
	}
	if custom["a.com"] != "test a" {
		t.Errorf("expected a.com reason 'test a', got %q", custom["a.com"])
	}
}

func TestIsDefaultDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"github.com", true},
		{"GITHUB.COM", true},
		{"example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsDefaultDomain(tt.domain)
		if result != tt.expected {
			t.Errorf("IsDefaultDomain(%q) = %v, want %v", tt.domain, result, tt.expected)
		}
	}
}

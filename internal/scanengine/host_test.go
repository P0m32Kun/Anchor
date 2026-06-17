package scanengine

import "testing"

func TestHostWithoutPort(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"1.14.236.216:443", "1.14.236.216"},
		{"10.0.0.1", "10.0.0.1"},
		{"example.com:443", "example.com"},
	}
	for _, tt := range tests {
		if got := hostWithoutPort(tt.in); got != tt.want {
			t.Errorf("hostWithoutPort(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDefaultEngineConfig_NoAbsoluteTimeout(t *testing.T) {
	cfg := DefaultEngineConfig()
	if cfg.AbsoluteTimeout != 0 {
		t.Fatalf("AbsoluteTimeout = %v, want 0 (disabled)", cfg.AbsoluteTimeout)
	}
}

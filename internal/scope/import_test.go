package scope

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ImportTarget
	}{
		{"domain prefix", "domain:example.com", []ImportTarget{{Type: models.TargetTypeDomain, Value: "example.com"}}},
		{"url prefix", "url:https://example.com/path", []ImportTarget{{Type: models.TargetTypeURL, Value: "https://example.com/path"}}},
		{"ip prefix", "ip:192.168.1.1", []ImportTarget{{Type: models.TargetTypeIP, Value: "192.168.1.1"}}},
		{"cidr prefix", "cidr:10.0.0.0/8", []ImportTarget{{Type: models.TargetTypeCIDR, Value: "10.0.0.0/8"}}},
		{"no prefix defaults to domain", "example.com", []ImportTarget{{Type: models.TargetTypeDomain, Value: "example.com"}}},
		{"unknown type defaults to domain", "unknown:foo:bar", []ImportTarget{{Type: models.TargetTypeDomain, Value: "unknown:foo:bar"}}},
		{"empty value after colon", "domain:", []ImportTarget{{Type: models.TargetTypeDomain, Value: "domain:"}}},
		{"empty string", "", nil},
		{"whitespace trimmed", "  domain:example.com  ", []ImportTarget{{Type: models.TargetTypeDomain, Value: "example.com"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLine(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLine(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTXT(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "single domain",
			input:   "example.com",
			wantLen: 1,
		},
		{
			name:    "multiple lines",
			input:   "domain:example.com\nurl:https://test.com\n192.168.1.1",
			wantLen: 3,
		},
		{
			name:    "skip empty lines",
			input:   "example.com\n\n\nsub.example.com\n",
			wantLen: 2,
		},
		{
			name:    "empty input",
			input:   "",
			wantLen: 0,
		},
		{
			name: "mixed types",
			input: `domain:example.com
url:https://api.example.com
ip:10.0.0.1
cidr:192.168.0.0/24
bare-domain.com`,
			wantLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := ParseTXT(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTXT() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if len(targets) != tt.wantLen {
				t.Errorf("ParseTXT() len = %d, want %d", len(targets), tt.wantLen)
			}
		})
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "header with type,value",
			input:   "type,value\ndomain,example.com\nurl,https://test.com",
			wantLen: 2,
		},
		{
			name:    "single column defaults to domain",
			input:   "value\nexample.com\nsub.example.com",
			wantLen: 2,
		},
		{
			name:    "no header, two columns",
			input:   "domain,example.com\nip,10.0.0.1",
			wantLen: 2,
		},
		{
			name:    "skip empty rows",
			input:   "type,value\ndomain,example.com\n\n\nip,192.168.1.1",
			wantLen: 2,
		},
		{
			name:    "empty input",
			input:   "",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := ParseCSV(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("ParseCSV() error = %v", err)
			}
			if len(targets) != tt.wantLen {
				t.Errorf("ParseCSV() len = %d, want %d", len(targets), tt.wantLen)
			}
		})
	}
}

func TestParseCSVTypeMapping(t *testing.T) {
	input := "type,value\ndomain,example.com\nurl,https://api.example.com\nip,10.0.0.1\ncidr,192.168.0.0/24"
	targets, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseCSV() error = %v", err)
	}

	expected := []struct {
		typ models.TargetType
		val string
	}{
		{models.TargetTypeDomain, "example.com"},
		{models.TargetTypeURL, "https://api.example.com"},
		{models.TargetTypeIP, "10.0.0.1"},
		{models.TargetTypeCIDR, "192.168.0.0/24"},
	}

	if len(targets) != len(expected) {
		t.Fatalf("got %d targets, want %d", len(targets), len(expected))
	}

	for i, exp := range expected {
		if targets[i].Type != exp.typ {
			t.Errorf("target[%d].Type = %q, want %q", i, targets[i].Type, exp.typ)
		}
		if targets[i].Value != exp.val {
			t.Errorf("target[%d].Value = %q, want %q", i, targets[i].Value, exp.val)
		}
	}
}

func TestIsValidTargetType(t *testing.T) {
	tests := []struct {
		typ     models.TargetType
		isValid bool
	}{
		{models.TargetTypeDomain, true},
		{models.TargetTypeURL, true},
		{models.TargetTypeIP, true},
		{models.TargetTypeCIDR, true},
		{models.TargetType("invalid"), false},
		{models.TargetType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			got := isValidTargetType(tt.typ)
			if got != tt.isValid {
				t.Errorf("isValidTargetType(%q) = %v, want %v", tt.typ, got, tt.isValid)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// New tests for extended parsing
// ---------------------------------------------------------------------------

func TestDetectType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  models.TargetType
	}{
		{"http URL", "http://example.com", models.TargetTypeURL},
		{"https URL", "https://example.com/path", models.TargetTypeURL},
		{"valid CIDR", "10.0.0.0/8", models.TargetTypeCIDR},
		{"valid IPv4", "192.168.1.1", models.TargetTypeIP},
		{"IPv4 range", "192.168.0.1-10", models.TargetTypeIP},
		{"domain", "example.com", models.TargetTypeDomain},
		{"domain with dots", "sub.sub.example.com", models.TargetTypeDomain},
		{"invalid CIDR syntax", "10.0.0.0/33", models.TargetTypeDomain},
		{"not enough octets", "192.168.1", models.TargetTypeDomain},
		{"too many octets", "192.168.1.1.1", models.TargetTypeDomain},
		{"bare string", "hello", models.TargetTypeDomain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectType(tt.input)
			if got != tt.want {
				t.Errorf("detectType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandCommas(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"three items", "a,b,c", []string{"a", "b", "c"}},
		{"with spaces", "a, b , c", []string{"a", "b", "c"}},
		{"empty segments", "a,,b", []string{"a", "b"}},
		{"single item", "a", []string{"a"}},
		{"all empty", ", ,", nil},
		{"empty string", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandCommas(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expandCommas(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandIPRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "simple range",
			input: "192.168.0.1-3",
			want:  []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"},
		},
		{
			name:  "single IP no hyphen",
			input: "192.168.0.1",
			want:  []string{"192.168.0.1"},
		},
		{
			name:  "single value range",
			input: "192.168.0.5-5",
			want:  []string{"192.168.0.5"},
		},
		{
			name:    "start > end",
			input:   "192.168.0.10-1",
			wantErr: true,
		},
		{
			name:    "end out of bounds",
			input:   "192.168.0.1-256",
			wantErr: true,
		},
		{
			name:    "negative start",
			input:   "192.168.0.-1-5",
			wantErr: true,
		},
		{
			name:  "full octet range",
			input: "10.0.0.0-255",
			want:  generateIPRange("10.0.0", 0, 255),
		},
		{
			name:    "range too large (>256)",
			input:   "192.168.0.0-256",
			wantErr: true,
		},
		{
			name:    "invalid prefix",
			input:   "192.168.1-10",
			wantErr: true,
		},
		{
			name:    "no dots",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "non-numeric range",
			input:   "192.168.0.a-b",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandIPRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandIPRange(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expandIPRange(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func generateIPRange(prefix string, start, end int) []string {
	var out []string
	for i := start; i <= end; i++ {
		out = append(out, fmt.Sprintf("%s.%d", prefix, i))
	}
	return out
}

func TestParseLineCommaExpansion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ImportTarget
	}{
		{
			name:  "domain with commas",
			input: "domain:example.com,sub.example.com",
			want: []ImportTarget{
				{Type: models.TargetTypeDomain, Value: "example.com"},
				{Type: models.TargetTypeDomain, Value: "sub.example.com"},
			},
		},
		{
			name:  "mixed auto-detect commas",
			input: "example.com,192.168.1.1,10.0.0.0/8",
			want: []ImportTarget{
				{Type: models.TargetTypeDomain, Value: "example.com"},
				{Type: models.TargetTypeIP, Value: "192.168.1.1"},
				{Type: models.TargetTypeCIDR, Value: "10.0.0.0/8"},
			},
		},
		{
			name:  "ip prefix shared across commas",
			input: "ip:192.168.1.1,192.168.1.2",
			want: []ImportTarget{
				{Type: models.TargetTypeIP, Value: "192.168.1.1"},
				{Type: models.TargetTypeIP, Value: "192.168.1.2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLine(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLine(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLineIPRangeExpansion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ImportTarget
	}{
		{
			name:  "explicit ip prefix with range",
			input: "ip:192.168.0.1-3",
			want: []ImportTarget{
				{Type: models.TargetTypeIP, Value: "192.168.0.1"},
				{Type: models.TargetTypeIP, Value: "192.168.0.2"},
				{Type: models.TargetTypeIP, Value: "192.168.0.3"},
			},
		},
		{
			name:  "auto-detected ip with range",
			input: "192.168.0.10-12",
			want: []ImportTarget{
				{Type: models.TargetTypeIP, Value: "192.168.0.10"},
				{Type: models.TargetTypeIP, Value: "192.168.0.11"},
				{Type: models.TargetTypeIP, Value: "192.168.0.12"},
			},
		},
		{
			name:  "single IP stays single",
			input: "192.168.0.1",
			want: []ImportTarget{
				{Type: models.TargetTypeIP, Value: "192.168.0.1"},
			},
		},
		{
			name:  "mixed line with range and domain",
			input: "192.168.0.1-2,example.com",
			want: []ImportTarget{
				{Type: models.TargetTypeIP, Value: "192.168.0.1"},
				{Type: models.TargetTypeIP, Value: "192.168.0.2"},
				{Type: models.TargetTypeDomain, Value: "example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLine(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLine(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLineAutoDetect(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType models.TargetType
		wantVal  string
	}{
		{"bare IPv4", "127.0.0.1", models.TargetTypeIP, "127.0.0.1"},
		{"bare CIDR", "192.168.0.0/24", models.TargetTypeCIDR, "192.168.0.0/24"},
		{"bare URL", "https://api.example.com", models.TargetTypeURL, "https://api.example.com"},
		{"bare domain", "example.com", models.TargetTypeDomain, "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLine(tt.input)
			if len(got) != 1 {
				t.Fatalf("parseLine(%q) returned %d targets, want 1", tt.input, len(got))
			}
			if got[0].Type != tt.wantType {
				t.Errorf("parseLine(%q).Type = %q, want %q", tt.input, got[0].Type, tt.wantType)
			}
			if got[0].Value != tt.wantVal {
				t.Errorf("parseLine(%q).Value = %q, want %q", tt.input, got[0].Value, tt.wantVal)
			}
		})
	}
}

func TestParseTXTRangeLimits(t *testing.T) {
	// Range exceeding 256 should cause expandIPRange to fail, but since
	// parseLine falls back to treating the raw string as an ip target when
	// expansion fails, the invalid range stays as a single (invalid) target.
	// Therefore we test the expandIPRange function directly for the 256
	// limit, and test ParseTXT for the 10000 total limit.

	t.Run("expandIPRange 256 limit", func(t *testing.T) {
		// 0-255 is exactly 256 and should succeed.
		_, err := expandIPRange("10.0.0.0-255")
		if err != nil {
			t.Errorf("0-255 should succeed, got error: %v", err)
		}
		// 0-256 is 257 and should fail.
		_, err = expandIPRange("10.0.0.0-256")
		if err == nil {
			t.Error("0-256 should fail, got nil error")
		}
	})

	t.Run("ParseTXT 10000 total limit", func(t *testing.T) {
		// Build an input with a single range that expands to 10001 IPs.
		// 10001 = 10001, but max octet range is 256, so we need many lines.
		// Use 40 lines of "10.x.0.0-255" = 40 * 256 = 10240 > 10000.
		var lines []string
		for i := 0; i < 40; i++ {
			lines = append(lines, fmt.Sprintf("10.%d.0.0-255", i))
		}
		input := strings.Join(lines, "\n")
		_, err := ParseTXT(strings.NewReader(input))
		if err == nil {
			t.Error("expected error for >10000 targets, got nil")
		}
	})

	t.Run("ParseTXT just under limit", func(t *testing.T) {
		// 39 lines * 256 = 9984 < 10000, should succeed.
		var lines []string
		for i := 0; i < 39; i++ {
			lines = append(lines, fmt.Sprintf("10.%d.0.0-255", i))
		}
		input := strings.Join(lines, "\n")
		targets, err := ParseTXT(strings.NewReader(input))
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(targets) != 9984 {
			t.Errorf("expected 9984 targets, got %d", len(targets))
		}
	})
}

func TestParseCSVRangesAndCommas(t *testing.T) {
	input := "type,value\nip,192.168.0.1-3\ndomain,\"example.com,sub.example.com\""
	targets, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseCSV() error = %v", err)
	}

	expected := []ImportTarget{
		{Type: models.TargetTypeIP, Value: "192.168.0.1"},
		{Type: models.TargetTypeIP, Value: "192.168.0.2"},
		{Type: models.TargetTypeIP, Value: "192.168.0.3"},
		{Type: models.TargetTypeDomain, Value: "example.com"},
		{Type: models.TargetTypeDomain, Value: "sub.example.com"},
	}

	if len(targets) != len(expected) {
		t.Fatalf("got %d targets, want %d", len(targets), len(expected))
	}
	for i, exp := range expected {
		if targets[i].Type != exp.Type {
			t.Errorf("target[%d].Type = %q, want %q", i, targets[i].Type, exp.Type)
		}
		if targets[i].Value != exp.Value {
			t.Errorf("target[%d].Value = %q, want %q", i, targets[i].Value, exp.Value)
		}
	}
}

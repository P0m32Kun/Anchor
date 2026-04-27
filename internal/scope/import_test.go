package scope

import (
	"strings"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType models.TargetType
		wantVal  string
	}{
		{"domain prefix", "domain:example.com", models.TargetTypeDomain, "example.com"},
		{"url prefix", "url:https://example.com/path", models.TargetTypeURL, "https://example.com/path"},
		{"ip prefix", "ip:192.168.1.1", models.TargetTypeIP, "192.168.1.1"},
		{"cidr prefix", "cidr:10.0.0.0/8", models.TargetTypeCIDR, "10.0.0.0/8"},
		{"no prefix defaults to domain", "example.com", models.TargetTypeDomain, "example.com"},
		{"unknown type defaults to domain", "unknown:foo:bar", models.TargetTypeDomain, "unknown:foo:bar"},
		{"empty value after colon", "domain:", models.TargetTypeDomain, "domain:"},
		{"empty string", "", models.TargetTypeDomain, ""},
		{"whitespace trimmed", "  domain:example.com  ", models.TargetTypeDomain, "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLine(tt.input)
			if got.Type != tt.wantType {
				t.Errorf("parseLine(%q).Type = %q, want %q", tt.input, got.Type, tt.wantType)
			}
			if got.Value != tt.wantVal {
				t.Errorf("parseLine(%q).Value = %q, want %q", tt.input, got.Value, tt.wantVal)
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

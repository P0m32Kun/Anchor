package parser

import (
	"strings"
	"testing"
)

func TestParseNmapAlive_BasicUpDown(t *testing.T) {
	out := strings.Join([]string{
		"# Nmap 7.94 scan initiated Thu May  7 15:00:00 2026 as: nmap -sn -n -T4 -oG - -iL hosts.txt",
		"Host: 172.30.0.10 (rangefield-target-1)\tStatus: Up",
		"Host: 172.30.0.11 (rangefield-target-2)\tStatus: Up",
		"Host: 172.30.0.5 ()\tStatus: Down",
		"Host: 172.30.0.99 ()\tStatus: Down",
		"# Nmap done at Thu May  7 15:00:01 2026 -- 4 IP addresses scanned",
	}, "\n")

	got := ParseNmapAlive(strings.NewReader(out))
	want := []string{"172.30.0.10", "172.30.0.11"}

	if len(got) != len(want) {
		t.Fatalf("expected %d IPs, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ip[%d]: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestParseNmapAlive_Empty(t *testing.T) {
	got := ParseNmapAlive(strings.NewReader(""))
	if len(got) != 0 {
		t.Errorf("expected no IPs, got %v", got)
	}
}

func TestParseNmapAlive_AllDown(t *testing.T) {
	out := strings.Join([]string{
		"Host: 10.0.0.1 ()\tStatus: Down",
		"Host: 10.0.0.2 ()\tStatus: Down",
	}, "\n")
	got := ParseNmapAlive(strings.NewReader(out))
	if len(got) != 0 {
		t.Errorf("expected no IPs, got %v", got)
	}
}

func TestParseNmapAlive_DedupesRepeats(t *testing.T) {
	out := strings.Join([]string{
		"Host: 192.168.1.10 ()\tStatus: Up",
		"Host: 192.168.1.10 ()\tStatus: Up",
		"Host: 192.168.1.11 ()\tStatus: Up",
	}, "\n")
	got := ParseNmapAlive(strings.NewReader(out))
	if len(got) != 2 {
		t.Errorf("expected 2 unique IPs, got %v", got)
	}
}

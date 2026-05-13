package fingerprint

import (
	"strings"
	"testing"
)

// --- IsWebService ---

func TestIsWebService(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    bool
	}{
		{"http", "http", true},
		{"https", "https", true},
		{"http-proxy", "http-proxy", true},
		{"HTTP uppercase", "HTTP", true},
		{"ssh", "ssh", false},
		{"mysql", "mysql", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NmapServiceResult{Service: tt.service}
			if got := IsWebService(r); got != tt.want {
				t.Errorf("IsWebService(%q) = %v, want %v", tt.service, got, tt.want)
			}
		})
	}
}

// --- BuildNmapServiceScanCommand ---

func TestBuildNmapServiceScanCommand(t *testing.T) {
	cmd := BuildNmapServiceScanCommand("/tmp/hosts.txt", []int{80, 443, 8080})

	expected := []string{"nmap", "-sV", "-p", "80,443,8080", "-iL", "/tmp/hosts.txt", "-oX", "-", "-T4", "-n", "--open"}
	if len(cmd) != len(expected) {
		t.Fatalf("len = %d, want %d", len(cmd), len(expected))
	}
	for i, v := range expected {
		if cmd[i] != v {
			t.Errorf("cmd[%d] = %q, want %q", i, cmd[i], v)
		}
	}
}

func TestBuildNmapServiceScanCommand_SinglePort(t *testing.T) {
	cmd := BuildNmapServiceScanCommand("hosts.txt", []int{22})
	if cmd[3] != "22" {
		t.Errorf("port arg = %q, want %q", cmd[3], "22")
	}
}

// --- ParseNmapXMLOutput ---

func TestParseNmapXMLOutput_ValidXML(t *testing.T) {
	xml := `<?xml version="1.0"?>
<nmaprun>
  <host>
    <address addr="10.0.0.1" addrtype="ipv4"/>
    <ports>
      <port protocol="tcp" portid="80">
        <state state="open"/>
        <service name="http" product="Apache" version="2.4.49">
          <cpe>cpe:/a:apache:http_server:2.4.49</cpe>
        </service>
      </port>
      <port protocol="tcp" portid="22">
        <state state="open"/>
        <service name="ssh" product="OpenSSH" version="8.2"/>
      </port>
    </ports>
  </host>
</nmaprun>`

	results := ParseNmapXMLOutput(xml)
	if len(results) != 2 {
		t.Fatalf("len = %d, want 2", len(results))
	}

	r0 := results[0]
	if r0.IP != "10.0.0.1" {
		t.Errorf("ip = %q, want 10.0.0.1", r0.IP)
	}
	if r0.Port != 80 {
		t.Errorf("port = %d, want 80", r0.Port)
	}
	if r0.Service != "http" {
		t.Errorf("service = %q, want http", r0.Service)
	}
	if r0.Product != "Apache" {
		t.Errorf("product = %q, want Apache", r0.Product)
	}
	if r0.Version != "2.4.49" {
		t.Errorf("version = %q, want 2.4.49", r0.Version)
	}
	if r0.CPE != "cpe:/a:apache:http_server:2.4.49" {
		t.Errorf("cpe = %q, want cpe:/a:apache:http_server:2.4.49", r0.CPE)
	}

	r1 := results[1]
	if r1.Port != 22 {
		t.Errorf("port = %d, want 22", r1.Port)
	}
	if r1.Service != "ssh" {
		t.Errorf("service = %q, want ssh", r1.Service)
	}
}

func TestParseNmapXMLOutput_ClosedPortFiltered(t *testing.T) {
	xml := `<?xml version="1.0"?>
<nmaprun>
  <host>
    <address addr="10.0.0.1" addrtype="ipv4"/>
    <ports>
      <port protocol="tcp" portid="80">
        <state state="closed"/>
        <service name="http"/>
      </port>
    </ports>
  </host>
</nmaprun>`

	results := ParseNmapXMLOutput(xml)
	if len(results) != 0 {
		t.Errorf("len = %d, want 0 (closed port should be filtered)", len(results))
	}
}

func TestParseNmapXMLOutput_InvalidXML(t *testing.T) {
	results := ParseNmapXMLOutput("not xml")
	if results != nil {
		t.Errorf("expected nil for invalid XML, got %v", results)
	}
}

func TestParseNmapXMLOutput_EmptyXML(t *testing.T) {
	results := ParseNmapXMLOutput(`<nmaprun></nmaprun>`)
	if len(results) != 0 {
		t.Errorf("len = %d, want 0", len(results))
	}
}

func TestParseNmapXMLOutput_MultipleCPEs(t *testing.T) {
	xml := `<?xml version="1.0"?>
<nmaprun>
  <host>
    <address addr="10.0.0.1" addrtype="ipv4"/>
    <ports>
      <port protocol="tcp" portid="443">
        <state state="open"/>
        <service name="https" product="nginx" version="1.18">
          <cpe>cpe:/a:nginx:nginx:1.18</cpe>
          <cpe>cpe:/o:linux:kernel</cpe>
        </service>
      </port>
    </ports>
  </host>
</nmaprun>`

	results := ParseNmapXMLOutput(xml)
	if len(results) != 1 {
		t.Fatalf("len = %d, want 1", len(results))
	}
	if !strings.Contains(results[0].CPE, "cpe:/a:nginx:nginx:1.18") {
		t.Errorf("cpe should contain nginx cpe, got %q", results[0].CPE)
	}
	if !strings.Contains(results[0].CPE, "cpe:/o:linux:kernel") {
		t.Errorf("cpe should contain linux cpe, got %q", results[0].CPE)
	}
}

// --- ConvertToServiceFingerprints ---

func TestConvertToServiceFingerprints(t *testing.T) {
	results := []NmapServiceResult{
		{IP: "10.0.0.1", Port: 80, Protocol: "tcp", Service: "http", Product: "Apache", Version: "2.4", CPE: "cpe:/a"},
		{IP: "10.0.0.1", Port: 22, Protocol: "tcp", Service: "ssh", Product: "OpenSSH", Version: "8.2"},
	}

	fps := ConvertToServiceFingerprints("proj-1", results)
	if len(fps) != 2 {
		t.Fatalf("len = %d, want 2", len(fps))
	}

	fp0 := fps[0]
	if fp0.IP != "10.0.0.1" {
		t.Errorf("ip = %q, want 10.0.0.1", fp0.IP)
	}
	if !fp0.IsWeb {
		t.Error("http should be IsWeb=true")
	}
	if fp0.Product != "Apache" {
		t.Errorf("product = %q, want Apache", fp0.Product)
	}
	if fp0.Version != "2.4" {
		t.Errorf("version = %q, want 2.4", fp0.Version)
	}
	if fp0.Source != "nmap" {
		t.Errorf("source = %q, want nmap", fp0.Source)
	}
	if fp0.Metadata["cpe"] != "cpe:/a" {
		t.Errorf("metadata cpe = %v, want cpe:/a", fp0.Metadata["cpe"])
	}

	fp1 := fps[1]
	if fp1.IsWeb {
		t.Error("ssh should be IsWeb=false")
	}
	if fp1.Metadata["cpe"] != nil {
		t.Errorf("ssh should have no cpe metadata, got %v", fp1.Metadata["cpe"])
	}
}

func TestConvertToServiceFingerprints_Empty(t *testing.T) {
	fps := ConvertToServiceFingerprints("proj-1", nil)
	if fps != nil {
		t.Errorf("expected nil for nil input, got %v", fps)
	}
}

// --- SplitByServiceType ---

func TestSplitByServiceType(t *testing.T) {
	results := []NmapServiceResult{
		{IP: "10.0.0.1", Port: 80, Service: "http"},
		{IP: "10.0.0.1", Port: 22, Service: "ssh"},
		{IP: "10.0.0.1", Port: 443, Service: "https"},
		{IP: "10.0.0.1", Port: 3306, Service: "mysql"},
	}

	web, nonWeb := SplitByServiceType(results)

	if len(web) != 2 {
		t.Errorf("web len = %d, want 2", len(web))
	}
	if len(nonWeb) != 2 {
		t.Errorf("nonWeb len = %d, want 2", len(nonWeb))
	}
}

func TestSplitByServiceType_AllWeb(t *testing.T) {
	results := []NmapServiceResult{
		{Service: "http"},
		{Service: "https"},
		{Service: "http-proxy"},
	}

	web, nonWeb := SplitByServiceType(results)
	if len(web) != 3 {
		t.Errorf("web len = %d, want 3", len(web))
	}
	if len(nonWeb) != 0 {
		t.Errorf("nonWeb len = %d, want 0", len(nonWeb))
	}
}

func TestSplitByServiceType_Empty(t *testing.T) {
	web, nonWeb := SplitByServiceType(nil)
	if web != nil || nonWeb != nil {
		t.Errorf("expected nil,nil for nil input")
	}
}

// --- MakeHTTPTargets ---

func TestMakeHTTPTargets(t *testing.T) {
	results := []NmapServiceResult{
		{IP: "10.0.0.1", Port: 80, Service: "http"},
		{IP: "10.0.0.1", Port: 443, Service: "https"},
		{IP: "10.0.0.1", Port: 8080, Service: "http"},
	}

	targets := MakeHTTPTargets(results)
	if len(targets) != 3 {
		t.Fatalf("len = %d, want 3", len(targets))
	}
	if targets[0] != "http://10.0.0.1:80" {
		t.Errorf("targets[0] = %q, want http://10.0.0.1:80", targets[0])
	}
	if targets[1] != "https://10.0.0.1" {
		t.Errorf("targets[1] = %q, want https://10.0.0.1", targets[1])
	}
	if targets[2] != "http://10.0.0.1:8080" {
		t.Errorf("targets[2] = %q, want http://10.0.0.1:8080", targets[2])
	}
}

func TestMakeHTTPTargets_Empty(t *testing.T) {
	targets := MakeHTTPTargets(nil)
	if targets != nil {
		t.Errorf("expected nil for nil input, got %v", targets)
	}
}

func TestMakeHTTPTargets_Only443(t *testing.T) {
	results := []NmapServiceResult{
		{IP: "10.0.0.1", Port: 443, Service: "https"},
	}
	targets := MakeHTTPTargets(results)
	if len(targets) != 1 || targets[0] != "https://10.0.0.1" {
		t.Errorf("got %v, want [https://10.0.0.1]", targets)
	}
}

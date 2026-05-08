package fingerprint

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// NmapServiceResult represents a single service fingerprint from nmap -sV.
type NmapServiceResult struct {
	IP       string
	Port     int
	Protocol string // tcp | udp
	Service  string // http, ssh, mysql, etc.
	Product  string
	Version  string
	CPE      string
}

// IsWebService checks if the detected service is a web service.
func IsWebService(result NmapServiceResult) bool {
	s := strings.ToLower(result.Service)
	return s == "http" || s == "https" || s == "http-proxy"
}

// BuildNmapServiceScanCommand builds an nmap -sV command for service fingerprinting.
// hostFile should contain one host per line; ports is the list of ports to scan.
// Output is XML to stdout for reliable parsing.
func BuildNmapServiceScanCommand(hostFile string, ports []int) []string {
	portStrs := make([]string, len(ports))
	for i, p := range ports {
		portStrs[i] = strconv.Itoa(p)
	}
	return []string{
		"nmap", "-sV",
		"-p", strings.Join(portStrs, ","),
		"-iL", hostFile,
		"-oX", "-",
		"-T4", "-n", "--open",
	}
}

// nmapXML is the minimal XML structure we need from nmap -oX output.
type nmapXML struct {
	Hosts []nmapHost `xml:"host"`
}

type nmapHost struct {
	Addresses []nmapAddress `xml:"address"`
	Ports     nmapPorts     `xml:"ports"`
}

type nmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type nmapPorts struct {
	Ports []nmapPort `xml:"port"`
}

type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   int         `xml:"portid,attr"`
	State    nmapState   `xml:"state"`
	Service  nmapService `xml:"service"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
	CPEs    []nmapCPE `xml:"cpe"`
}

type nmapCPE struct {
	Value string `xml:",chardata"`
}

// ParseNmapXMLOutput parses nmap -oX XML output and returns service results.
func ParseNmapXMLOutput(output string) []NmapServiceResult {
	var nmapRun nmapXML
	if err := xml.Unmarshal([]byte(output), &nmapRun); err != nil {
		return nil
	}

	var results []NmapServiceResult
	for _, host := range nmapRun.Hosts {
		ip := extractHostIP(host.Addresses)
		if ip == "" {
			continue
		}
		for _, port := range host.Ports.Ports {
			if port.State.State != "open" {
				continue
			}
			var cpes []string
			for _, cpe := range port.Service.CPEs {
				cpes = append(cpes, cpe.Value)
			}
			results = append(results, NmapServiceResult{
				IP:       ip,
				Port:     port.PortID,
				Protocol: port.Protocol,
				Service:  port.Service.Name,
				Product:  port.Service.Product,
				Version:  port.Service.Version,
				CPE:      strings.Join(cpes, ","),
			})
		}
	}
	return results
}

func extractHostIP(addrs []nmapAddress) string {
	for _, a := range addrs {
		if a.AddrType == "ipv4" || a.AddrType == "ipv6" {
			return a.Addr
		}
	}
	if len(addrs) > 0 {
		return addrs[0].Addr
	}
	return ""
}

// ConvertToServiceFingerprints converts nmap results to ServiceFingerprint models.
func ConvertToServiceFingerprints(projectID string, results []NmapServiceResult) []models.ServiceFingerprint {
	var fps []models.ServiceFingerprint
	for _, r := range results {
		meta := map[string]interface{}{}
		if r.Product != "" {
			meta["product"] = r.Product
		}
		if r.Version != "" {
			meta["version"] = r.Version
		}
		if r.CPE != "" {
			meta["cpe"] = r.CPE
		}
		fp := models.ServiceFingerprint{
			IP:       r.IP,
			Port:     r.Port,
			Protocol: r.Protocol,
			IsWeb:    IsWebService(r),
			Service:  r.Service,
			Metadata: meta,
			Source:   "nmap",
		}
		fps = append(fps, fp)
	}
	return fps
}

// SplitByServiceType splits nmap results into web and non-web services.
func SplitByServiceType(results []NmapServiceResult) (web []NmapServiceResult, nonWeb []NmapServiceResult) {
	for _, r := range results {
		if IsWebService(r) {
			web = append(web, r)
		} else {
			nonWeb = append(nonWeb, r)
		}
	}
	return
}

// MakeHTTPTargets builds http:// and https:// URLs from nmap web service results.
func MakeHTTPTargets(results []NmapServiceResult) []string {
	var targets []string
	for _, r := range results {
		host := r.IP
		if r.Port == 443 {
			targets = append(targets, fmt.Sprintf("https://%s", host))
		} else {
			targets = append(targets, fmt.Sprintf("http://%s:%d", host, r.Port))
		}
	}
	return targets
}

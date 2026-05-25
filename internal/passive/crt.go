package passive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// crtEntry is a single row from crt.sh JSON output.
type crtEntry struct {
	IssuerCAID int      `json:"issuer_ca_id"`
	IssuerName string   `json:"issuer_name"`
	NameValue  string   `json:"name_value"`
	MinCertID  int      `json:"min_cert_id"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
}

var crtHTTPClient = &http.Client{Timeout: 30 * time.Second}

// FetchSubdomains queries crt.sh for SSL certificate transparency logs
// matching the given domain and returns deduplicated subdomain strings.
// Results with wildcard prefixes (*.) are kept as the bare domain.
func FetchSubdomains(ctx context.Context, domain string) ([]string, error) {
	url := fmt.Sprintf("https://crt.sh/?q=%%.%s&output=json", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("crt.sh request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := crtHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crt.sh get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh status %d", resp.StatusCode)
	}

	var entries []crtEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("crt.sh decode: %w", err)
	}

	seen := make(map[string]bool)
	var subs []string
	for _, e := range entries {
		// name_value can contain multiple newline-separated names
		for _, name := range strings.Split(e.NameValue, "\n") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			// Strip wildcard prefix
			name = strings.TrimPrefix(name, "*.")
			if seen[name] {
				continue
			}
			seen[name] = true
			subs = append(subs, name)
		}
	}
	return subs, nil
}

package models

import "time"

// --- DNS ---

type DNSRecord struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Domain    string    `json:"domain"`
	IPs       []string  `json:"ips"`
	CNAMEs    []string  `json:"cnames,omitempty"`
	TTL       uint32    `json:"ttl"`
	Resolver  string    `json:"resolver"`
	CreatedAt time.Time `json:"created_at"`
}

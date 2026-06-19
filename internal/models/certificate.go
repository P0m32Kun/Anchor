package models

import "time"

type Certificate struct {
	ID             string     `json:"id" db:"id"`
	ProjectID      string     `json:"project_id" db:"project_id"`
	AssetID        string     `json:"asset_id" db:"asset_id"`
	WebEndpointID  *string    `json:"web_endpoint_id,omitempty" db:"web_endpoint_id"`
	Host           string     `json:"host" db:"host"`
	Port           int        `json:"port" db:"port"`
	SubjectCN      string     `json:"subject_cn" db:"subject_cn"`
	IssuerOrg      string     `json:"issuer_org" db:"issuer_org"`
	NotBefore      time.Time  `json:"not_before" db:"not_before"`
	NotAfter       time.Time  `json:"not_after" db:"not_after"`
	SANs           string     `json:"sans" db:"sans"`
	FingerprintSHA256 string  `json:"fingerprint_sha256" db:"fingerprint_sha256"`
	FirstSeen      time.Time  `json:"first_seen" db:"first_seen"`
	LastSeen       time.Time  `json:"last_seen" db:"last_seen"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

func (c *Certificate) DaysUntilExpiry() int {
	if c.NotAfter.IsZero() {
		return 0
	}
	return int(time.Until(c.NotAfter).Hours() / 24)
}

func (c *Certificate) IsExpiringSoon(days int) bool {
	return c.DaysUntilExpiry() > 0 && c.DaysUntilExpiry() <= days
}

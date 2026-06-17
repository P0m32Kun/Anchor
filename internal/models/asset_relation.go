package models

import "time"

const (
	RelationSourceTarget = "target"
	RelationSourceAsset  = "asset"
	RelationTargetAsset  = "asset"

	RelationExpandedBy    = "expanded_by"
	RelationDiscoveredFrom = "discovered_from"
	RelationResolvesTo    = "resolves_to"
	RelationContains      = "contains"
)

// AssetRelation records a directed edge in the asset discovery graph.
type AssetRelation struct {
	ID           string    `json:"id" db:"id"`
	ProjectID    string    `json:"project_id" db:"project_id"`
	RunID        *string   `json:"run_id,omitempty" db:"run_id"`
	SourceType   string    `json:"source_type" db:"source_type"`
	SourceID     string    `json:"source_id" db:"source_id"`
	TargetType   string    `json:"target_type" db:"target_type"`
	TargetID     string    `json:"target_id" db:"target_id"`
	RelationType string    `json:"relation_type" db:"relation_type"`
	SourceEngine string    `json:"source_engine,omitempty" db:"source_engine"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// LineageNode is one hop in an asset lineage chain (root → current).
type LineageNode struct {
	NodeType     string `json:"node_type"` // target | asset
	ID           string `json:"id"`
	Value        string `json:"value,omitempty"`
	Relation     string `json:"relation,omitempty"`      // edge into this node from previous
	SourceEngine string `json:"source_engine,omitempty"` // engine on the incoming edge
}

// AssetLineage is the API response for GET /assets/{id}/lineage.
type AssetLineage struct {
	AssetID   string        `json:"asset_id"`
	RunID     string        `json:"run_id,omitempty"`
	ProjectID string        `json:"project_id"`
	Chain     []LineageNode `json:"chain"`
}

package asset

import (
	"slices"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// Merger handles asset deduplication and merging.
type Merger struct {
	queries *db.Queries
}

func NewMerger(queries *db.Queries) *Merger {
	return &Merger{queries: queries}
}

// MergeOrCreateAsset looks up an asset by normalized value. If found, updates last_seen
// and merges source_tools. If not found, creates a new asset.
func (m *Merger) MergeOrCreateAsset(projectID, assetType, value, sourceTool string) (*models.Asset, bool, error) {
	normalized := Normalize(assetType, value)
	existing, err := m.queries.GetAssetByNormalizedValue(projectID, normalized)
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UTC()
	if existing != nil {
		// Merge source_tools without duplicates.
		tools := existing.SourceTools
		if sourceTool != "" && !slices.Contains(tools, sourceTool) {
			tools = append(tools, sourceTool)
		}
		if err := m.queries.UpdateAssetLastSeen(existing.ID, now, tools); err != nil {
			return nil, false, err
		}
		existing.LastSeen = now
		existing.SourceTools = tools
		return existing, false, nil
	}

	a := &models.Asset{
		ID:              util.GenerateID(),
		ProjectID:       projectID,
		Type:            models.AssetType(assetType),
		Value:           value,
		NormalizedValue: normalized,
		SourceTools:     []string{},
		FirstSeen:       now,
		LastSeen:        now,
	}
	if sourceTool != "" {
		a.SourceTools = []string{sourceTool}
	}
	if err := m.queries.CreateAsset(a); err != nil {
		return nil, false, err
	}
	return a, true, nil
}

// CreatePortIfNotExists creates a port if one does not already exist for the asset.
func (m *Merger) CreatePortIfNotExists(assetID string, port int, protocol, sourceTool string) (*models.Port, bool, error) {
	if protocol == "" {
		protocol = "tcp"
	}
	exists, err := m.queries.PortExists(assetID, port)
	if err != nil {
		return nil, false, err
	}
	if exists {
		return nil, false, nil
	}
	p := &models.Port{
		ID:         util.GenerateID(),
		AssetID:    assetID,
		Port:       port,
		Protocol:   protocol,
		State:      "open",
		SourceTool: sourceTool,
		CreatedAt:  time.Now().UTC(),
	}
	if err := m.queries.CreatePort(p); err != nil {
		return nil, false, err
	}
	return p, true, nil
}

// CreateWebEndpointIfNotExists creates a web endpoint if one does not already exist for the project+url.
func (m *Merger) CreateWebEndpointIfNotExists(projectID, assetID, url, scheme, host string, port *int, path, title string, statusCode *int, technologies []string, sourceTool string) (*models.WebEndpoint, bool, error) {
	exists, err := m.queries.WebEndpointExists(projectID, url)
	if err != nil {
		return nil, false, err
	}
	if exists {
		return nil, false, nil
	}
	we := &models.WebEndpoint{
		ID:           util.GenerateID(),
		ProjectID:    projectID,
		AssetID:      assetID,
		URL:          url,
		Scheme:       scheme,
		Host:         host,
		Port:         port,
		Path:         path,
		StatusCode:   statusCode,
		Title:        title,
		Technologies: technologies,
		SourceTool:   sourceTool,
		CreatedAt:    time.Now().UTC(),
	}
	if err := m.queries.CreateWebEndpoint(we); err != nil {
		return nil, false, err
	}
	return we, true, nil
}

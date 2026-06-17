package signal

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// Service generates and manages signals from asset changes and findings.
type Service struct {
	queries *db.Queries
}

// NewService creates a new signal Service.
func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

// GenerateAssetSignals compares the current assets against the previous snapshot
// and creates signals for new, disappeared, and changed assets.
func (s *Service) GenerateAssetSignals(projectID, runID string) error {
	assets, err := s.queries.ListAssetsByProject(projectID)
	if err != nil {
		return fmt.Errorf("list assets: %w", err)
	}

	// Build current asset map by normalized value.
	current := make(map[string]*models.Asset, len(assets))
	for _, a := range assets {
		current[a.NormalizedValue] = a
	}

	// Get the previous snapshot (before this run).
	prev, err := s.queries.GetLatestAssetSnapshot(projectID, runID)
	if err != nil {
		// No previous snapshot — first scan, all assets are new.
		if assetsCount := len(assets); assetsCount > 0 {
			log.Printf("[signal] first scan for project %s, %d baseline assets (no snapshot)", projectID, assetsCount)
		}
		return nil
	}

	var prevData struct {
		AssetValues []string `json:"asset_values"`
	}
	if err := json.Unmarshal([]byte(prev.AssetChangesJSON), &prevData); err != nil {
		return fmt.Errorf("unmarshal previous snapshot: %w", err)
	}

	prevSet := make(map[string]bool, len(prevData.AssetValues))
	for _, v := range prevData.AssetValues {
		prevSet[v] = true
	}

	// Detect new assets.
	newCount := 0
	for value, a := range current {
		if !prevSet[value] {
			newCount++
			s.createOrUpdateSignal(projectID, models.SignalSourceKindAssetNew, a.ID,
				fmt.Sprintf("New asset discovered: %s", a.Value),
				models.SignalSeverityInfo, 10,
				map[string]string{"asset_value": a.Value, "asset_type": string(a.Type)})
		}
	}

	// Detect disappeared assets.
	goneCount := 0
	for value := range prevSet {
		if _, exists := current[value]; !exists {
			goneCount++
			s.createOrUpdateSignal(projectID, models.SignalSourceKindAssetGone, value,
				fmt.Sprintf("Asset disappeared: %s", value),
				models.SignalSeverityMedium, 30,
				map[string]string{"asset_value": value})
		}
	}

	if newCount > 0 || goneCount > 0 {
		log.Printf("[signal] project %s: %d new, %d disappeared assets", projectID, newCount, goneCount)
	}

	return nil
}

// CreateFindingSignal creates a signal from a finding.
func (s *Service) CreateFindingSignal(projectID, findingID, title, severity string, score int, metadata map[string]string) error {
	sig := &models.Signal{
		ID:          util.GenerateID(),
		ProjectID:   projectID,
		SourceKind:  models.SignalSourceKindFinding,
		SourceID:    findingID,
		Title:       title,
		Severity:    severity,
		Score:       score,
		ScopeStatus: "in_scope",
		Status:      models.SignalStatusNew,
	}
	if metadataJSON, err := json.Marshal(metadata); err == nil {
		sig.Metadata = string(metadataJSON)
	}
	return s.queries.CreateSignal(sig)
}

func (s *Service) createOrUpdateSignal(projectID, sourceKind, sourceID, title, severity string, score int, metadata map[string]string) {
	existing, err := s.queries.GetSignalsBySource(sourceKind, sourceID)
	if err == nil && len(existing) > 0 {
		// Update existing signal.
		_ = s.queries.UpdateSignalLastSeen(existing[0].ID, score)
		return
	}

	sig := &models.Signal{
		ID:          util.GenerateID(),
		ProjectID:   projectID,
		SourceKind:  sourceKind,
		SourceID:    sourceID,
		Title:       title,
		Severity:    severity,
		Score:       score,
		ScopeStatus: "in_scope",
		Status:      models.SignalStatusNew,
	}
	if metadataJSON, err := json.Marshal(metadata); err == nil {
		sig.Metadata = string(metadataJSON)
	}
	if err := s.queries.CreateSignal(sig); err != nil {
		log.Printf("[signal] create signal for %s/%s: %v", sourceKind, sourceID, err)
	}
}

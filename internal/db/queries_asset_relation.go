package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func (q *Queries) UpsertAssetRelation(rel *models.AssetRelation) error {
	if rel == nil {
		return fmt.Errorf("asset relation is nil")
	}
	if rel.ID == "" {
		return fmt.Errorf("asset relation id is required")
	}
	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = time.Now().UTC()
	}
	runID := ""
	if rel.RunID != nil {
		runID = *rel.RunID
	}
	_, err := q.db.Exec(`
		INSERT INTO asset_relations (
			id, project_id, run_id, source_type, source_id,
			target_type, target_id, relation_type, source_engine, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, source_type, source_id, target_type, target_id, relation_type, run_id)
		DO UPDATE SET source_engine = excluded.source_engine`,
		rel.ID, rel.ProjectID, runID, rel.SourceType, rel.SourceID,
		rel.TargetType, rel.TargetID, rel.RelationType, rel.SourceEngine, rel.CreatedAt,
	)
	return err
}

func (q *Queries) ListIncomingAssetRelations(projectID, targetID string, runID *string) ([]*models.AssetRelation, error) {
	query := `
		SELECT id, project_id, run_id, source_type, source_id,
		       target_type, target_id, relation_type, source_engine, created_at
		FROM asset_relations
		WHERE project_id = ? AND target_id = ?`
	args := []any{projectID, targetID}
	if runID != nil && *runID != "" {
		query += ` AND run_id = ?`
		args = append(args, *runID)
	}
	query += ` ORDER BY created_at`

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.AssetRelation
	for rows.Next() {
		rel, err := scanAssetRelation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}

func scanAssetRelation(rows *sql.Rows) (*models.AssetRelation, error) {
	var rel models.AssetRelation
	var runID string
	if err := rows.Scan(
		&rel.ID, &rel.ProjectID, &runID, &rel.SourceType, &rel.SourceID,
		&rel.TargetType, &rel.TargetID, &rel.RelationType, &rel.SourceEngine, &rel.CreatedAt,
	); err != nil {
		return nil, err
	}
	if runID != "" {
		rel.RunID = &runID
	}
	return &rel, nil
}

// BuildAssetLineage walks incoming edges from assetID back to tier-0 targets.
func (q *Queries) BuildAssetLineage(projectID, assetID string, runID *string) (*models.AssetLineage, error) {
	asset, err := q.GetAssetByID(assetID)
	if err != nil {
		return nil, err
	}
	if asset == nil || asset.ProjectID != projectID {
		return nil, fmt.Errorf("asset not found")
	}

	type frame struct {
		nodeType string
		id       string
		value    string
		relation string
		engine   string
	}

	chain := []frame{{
		nodeType: "asset",
		id:       asset.ID,
		value:    asset.Value,
	}}

	visited := map[string]bool{assetID: true}
	currentID := assetID

	for depth := 0; depth < 32; depth++ {
		rels, err := q.ListIncomingAssetRelations(projectID, currentID, runID)
		if err != nil {
			return nil, err
		}
		if len(rels) == 0 {
			break
		}
		rel := rels[0]
		if visited[rel.SourceID] {
			break
		}
		visited[rel.SourceID] = true

		var value string
		switch rel.SourceType {
		case models.RelationSourceTarget:
			t, err := q.GetTarget(rel.SourceID)
			if err != nil {
				return nil, err
			}
			if t != nil {
				value = t.Value
			}
		case models.RelationSourceAsset:
			src, err := q.GetAssetByID(rel.SourceID)
			if err != nil {
				return nil, err
			}
			if src != nil {
				value = src.Value
			}
		}

		chain = append([]frame{{
			nodeType: rel.SourceType,
			id:       rel.SourceID,
			value:    value,
		}}, chain...)
		if len(chain) > 1 {
			chain[1].relation = rel.RelationType
			chain[1].engine = rel.SourceEngine
		}

		if rel.SourceType == models.RelationSourceTarget {
			break
		}
		currentID = rel.SourceID
	}

	out := &models.AssetLineage{
		AssetID:   assetID,
		ProjectID: projectID,
		Chain:     make([]models.LineageNode, 0, len(chain)),
	}
	if runID != nil {
		out.RunID = *runID
	}
	for i, f := range chain {
		node := models.LineageNode{
			NodeType: f.nodeType,
			ID:       f.id,
			Value:    f.value,
		}
		if i > 0 {
			node.Relation = chain[i].relation
			node.SourceEngine = chain[i].engine
		}
		out.Chain = append(out.Chain, node)
	}
	return out, nil
}

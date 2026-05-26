package workflow

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) recordPassiveTask(tool string, summary string, data []byte) {
	taskID := util.GenerateID()
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID:              taskID,
		ProjectID:       p.projectID,
		RunID:           &p.runID,
		Tool:            tool,
		CommandTemplate: summary,
		Status:          models.TaskCompleted,
		CreatedAt:       now,
		StartedAt:       &now,
		FinishedAt:      &now,
	}
	if err := p.queries.CreateScanTask(task); err != nil {
		log.Printf("[pipeline] record passive task %s: %v", tool, err)
		return
	}

	workdir := filepath.Join(p.dataDir, "workdirs", p.projectID, taskID)
	_ = os.MkdirAll(workdir, 0750)
	filename := fmt.Sprintf("stdout_%d.json", time.Now().UnixNano())
	path := filepath.Join(workdir, filename)
	if err := os.WriteFile(path, data, 0640); err != nil {
		log.Printf("[pipeline] write passive artifact %s: %v", tool, err)
		return
	}

	sum := sha256.Sum256(data)
	a := &models.RawArtifact{
		ID:              util.GenerateID(),
		ProjectID:       p.projectID,
		TaskID:          &taskID,
		Type:            models.ArtifactStdout,
		Path:            path,
		SHA256:          fmt.Sprintf("%x", sum),
		Size:            int64(len(data)),
		RedactionStatus: "unchecked",
		CreatedAt:       now,
	}
	if err := p.queries.CreateRawArtifact(a); err != nil {
		log.Printf("[pipeline] create passive artifact %s: %v", tool, err)
	}
}

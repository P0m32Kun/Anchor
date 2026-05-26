package workflow

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func (p *Pipeline) createAndRunTask(ctx context.Context, tool string, args []string) (*models.ScanTask, []byte, error) {
	return p.legacyCreateAndRunTask(ctx, util.GenerateID(), tool, args)
}

func (p *Pipeline) legacyCreateAndRunTask(ctx context.Context, taskID, tool string, args []string) (*models.ScanTask, []byte, error) {
	now := time.Now().UTC()

	task := &models.ScanTask{
		ID:              taskID,
		ProjectID:       p.projectID,
		RunID:           &p.runID,
		Tool:            tool,
		CommandTemplate: strings.Join(args, " "),
		Status:          models.TaskCreated,
		CreatedAt:       now,
	}

	if tool == "nuclei" {
		if version, err := p.queries.GetActiveNucleiCustomBundleVersion(); err == nil && version != "" {
			task.NucleiCustomBundleVersion = &version
		}
	}

	if err := p.queries.CreateScanTask(task); err != nil {
		return nil, nil, fmt.Errorf("create scan task: %w", err)
	}

	if err := p.runner.Run(ctx, task.ID); err != nil {
		log.Printf("[pipeline] task %s (%s) run error: %v", task.ID, tool, err)
		stdout, _ := p.readTaskStdout(task.ID)
		return task, stdout, err
	}

	stdout, err := p.readTaskStdout(task.ID)
	if err != nil {
		log.Printf("[pipeline] task %s (%s) read stdout: %v", task.ID, tool, err)
	}

	return task, stdout, nil
}

func (p *Pipeline) readTaskStdout(taskID string) ([]byte, error) {
	artifacts, err := p.queries.ListRawArtifactsByTask(taskID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if a.Type == models.ArtifactStdout {
			return os.ReadFile(a.Path)
		}
	}
	return nil, fmt.Errorf("no stdout artifact found for task %s", taskID)
}

// Package toolrun provides a unified entry point for executing scanning tools.
// It bridges the tool registry (argv generation) with the worker Runner
// (subprocess execution) and enforces the allowlist gate.
package toolrun

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/toolguard"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// TaskRunner abstracts the worker.Runner for testability.
type TaskRunner interface {
	Run(ctx context.Context, taskID string) error
	Cancel(taskID string) error
}

// ScanTaskDB wraps the DB operations needed by Invoke. The caller
// (typically Pipeline via db.Queries) implements this interface.
type ScanTaskDB interface {
	CreateScanTask(t *models.ScanTask) error
	ListRawArtifactsByTask(taskID string) ([]*models.RawArtifact, error)
	GetActiveNucleiCustomBundleVersion() (string, error)
}

// InvokeInput carries all parameters for a single tool invocation.
type InvokeInput struct {
	ProjectID string
	RunID     *string
	TaskID    string // empty → generated
	ToolID    string
	Params    toolregistry.RenderParams
	// ExtraArgs allows direct argv tokens not expressible via Params
	// (e.g. nuclei workflow-file injection from customWorkflowPaths).
	ExtraArgs []string
}

// InvokeResult contains the completed ScanTask and its stdout bytes.
type InvokeResult struct {
	Task   *models.ScanTask
	Stdout []byte
	Err    error
}

// Invoke runs a registered tool with the given parameters.
// Steps:
//  1. Look up tool_id in registry
//  2. Render argv via reg.Render
//  3. Append ExtraArgs
//  4. Validate against allowlist
//  5. Create ScanTask record
//  6. Execute via runner.Run
//  7. Read stdout artifact
//  8. Return result
func Invoke(ctx context.Context, db ScanTaskDB, runner TaskRunner, reg *toolregistry.Registry, in InvokeInput) *InvokeResult {
	res := &InvokeResult{}

	// 1. Look up tool
	def := reg.Get(in.ToolID)
	if def == nil {
		res.Err = fmt.Errorf("tool %q not found in registry", in.ToolID)
		return res
	}

	// 2. Render argv from registry
	argv, err := reg.Render(in.ToolID, in.Params)
	if err != nil {
		res.Err = fmt.Errorf("tool %q render: %w", in.ToolID, err)
		return res
	}

	// 3. Append extra pipeline-side args (e.g. Nuclei custom workflow paths)
	if len(in.ExtraArgs) > 0 {
		argv = append(argv, in.ExtraArgs...)
	}

	// 4. Validate against allowlist
	binary := argv[0]
	allowlist := toolguard.NewAllowlistFromBinaries(reg.Binaries())
	if err := allowlist.Validate(binary, argv[1:]); err != nil {
		res.Err = fmt.Errorf("tool %q allowlist: %w", in.ToolID, err)
		return res
	}

	// 5. Create ScanTask
	now := time.Now().UTC()
	taskID := in.TaskID
	if taskID == "" {
		taskID = util.GenerateID()
	}

	task := &models.ScanTask{
		ID:              taskID,
		ProjectID:       in.ProjectID,
		RunID:           in.RunID,
		Tool:            in.ToolID,
		CommandTemplate: joinArgs(argv),
		Status:          models.TaskCreated,
		CreatedAt:       now,
	}

	// Record active custom bundle version for nuclei tasks
	if in.ToolID == "nuclei" {
		if version, err := db.GetActiveNucleiCustomBundleVersion(); err == nil && version != "" {
			task.NucleiCustomBundleVersion = &version
		}
	}
	// nuclei_custom tool_id also gets bundle version
	if in.ToolID == "nuclei_custom" {
		if version, err := db.GetActiveNucleiCustomBundleVersion(); err == nil && version != "" {
			task.NucleiCustomBundleVersion = &version
		}
	}

	if err := db.CreateScanTask(task); err != nil {
		res.Err = fmt.Errorf("create scan task: %w", err)
		return res
	}
	res.Task = task

	// 6. Execute via runner
	if err := runner.Run(ctx, taskID); err != nil {
		log.Printf("[toolrun] task %s (%s) run error: %v", taskID, in.ToolID, err)
		res.Err = err
		// Still try to read partial stdout
	}

	// 7. Read stdout artifact
	stdout, readErr := readTaskStdout(db, taskID)
	if readErr != nil {
		log.Printf("[toolrun] task %s (%s) read stdout: %v", taskID, in.ToolID, readErr)
	}
	res.Stdout = stdout

	return res
}

// readTaskStdout reads the stdout artifact file from disk.
func readTaskStdout(db ScanTaskDB, taskID string) ([]byte, error) {
	artifacts, err := db.ListRawArtifactsByTask(taskID)
	if err != nil {
		return nil, err
	}
	for _, a := range artifacts {
		if a.Type == models.ArtifactStdout {
			data, err := os.ReadFile(a.Path)
			if err != nil {
				return nil, fmt.Errorf("read artifact %s: %w", a.Path, err)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("no stdout artifact for task %s", taskID)
}

// joinArgs joins argv into a command template string for storage.
// The worker.Run(CommandTemplate) expects argv[0] to be the binary name.
func joinArgs(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	// Include binary name (argv[0]) so worker.Run can parse it back:
	//   args[0] → binary, args[1:] → flags
	var out string
	for i, s := range argv {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

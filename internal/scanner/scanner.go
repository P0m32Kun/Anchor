package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
	"github.com/P0m32Kun/Anchor/internal/scanengine/executor"
	"github.com/P0m32Kun/Anchor/internal/toolguard"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
	"github.com/P0m32Kun/Anchor/internal/toolrun"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// ScanResult holds normalized output and provenance.
type ScanResult struct {
	Stdout  []byte
	Receipt ExecutionReceipt
}

// ExecutionReceipt records provenance for a tool invocation.
type ExecutionReceipt struct {
	ToolID          string
	CommandTemplate string
	TaskID          string
	DurationMs      int64
	ExitCode        *int
}

// Scanner executes semantic ScanRequests via guarded adapters.
type Scanner struct {
	registry *toolregistry.Registry
	runner   toolrun.TaskRunner
	db       toolrun.ScanTaskDB
	dataDir  string
	// scopeChecker is an optional hook for scope enforcement.
	// If nil, scope checks are skipped.
	scopeChecker ScopeChecker
	// provenance creator for ToolCallLog (optional, for parity with ToolExecutor)
	queries ProvenanceQueries
}

// ScopeChecker validates that targets are within scope.
type ScopeChecker interface {
	IsExcluded(projectID string, target string) (bool, error)
}

// ProvenanceQueries is the minimal DB interface for ToolCallLog.
type ProvenanceQueries interface {
	CreateToolCallLog(log *models.ToolCallLog) error
	UpdateToolCallLogTaskID(logID, taskID string) error
	UpdateToolCallLogOnComplete(logID string, finishedAt time.Time, exitCode *int, status models.ToolCallStatus, durationMs int64, outputSummary, errorMsg string) error
}

// New creates a Scanner with required dependencies.
func New(registry *toolregistry.Registry, runner toolrun.TaskRunner, db toolrun.ScanTaskDB, dataDir string) *Scanner {
	return &Scanner{
		registry: registry,
		runner:   runner,
		db:       db,
		dataDir:  dataDir,
	}
}

// WithScopeChecker sets the scope checker.
func (s *Scanner) WithScopeChecker(sc ScopeChecker) *Scanner {
	s.scopeChecker = sc
	return s
}

// WithProvenance sets the provenance DB handle.
func (s *Scanner) WithProvenance(q ProvenanceQueries) *Scanner {
	s.queries = q
	return s
}

// Execute runs the ScanRequest and returns a ScanResult.
func (s *Scanner) Execute(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if s.scopeChecker != nil {
		for _, t := range req.Targets {
			excluded, err := s.scopeChecker.IsExcluded(req.ProjectID, t)
			if err != nil {
				return nil, fmt.Errorf("scope check: %w", err)
			}
			if excluded {
				return nil, fmt.Errorf("%w: target %s", ErrScopeDenied, t)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Action != core.ActionPortScan {
		return nil, fmt.Errorf("%w: unsupported action %s", ErrInvalidRequest, req.Action)
	}
	return s.executePortScan(ctx, req)
}

func (s *Scanner) executePortScan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	hostFile, cleanup, err := executor.WriteHostFile(s.dataDir, req.Targets)
	if err != nil {
		return nil, fmt.Errorf("write host file: %w", err)
	}
	defer cleanup()

	params := toolregistry.RenderParams{
		"host_file":  hostFile,
		"port_range": req.PortRange,
		"rate":       req.Budgets.Rate,
		"threads":    req.Budgets.Threads,
		"timeout":    req.Budgets.Timeout,
	}

	// Guarded process adapter path: render → allowlist → invoke
	toolID := core.ActionToTool[core.ActionPortScan]
	argv, err := s.registry.Render(toolID, params)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", toolID, err)
	}
	allowlist := toolguard.NewAllowlistFromBinaries(s.registry.Binaries())
	if err := allowlist.Validate(argv[0], argv[1:]); err != nil {
		return nil, fmt.Errorf("allowlist: %w", err)
	}

	// Provenance: ToolCallLog
	var logID string
	var startedAt time.Time
	if s.queries != nil {
		logID = util.GenerateID()
		startedAt = time.Now().UTC()
		paramsJSON := "{}"
		if pj, err := json.Marshal(params); err == nil {
			paramsJSON = string(pj)
		}
		callLog := &models.ToolCallLog{
			ID:         logID,
			RunID:      req.RunID,
			WorkItemID: &req.IdempotencyKey,
			Tool:       toolID,
			Action:     string(req.Action),
			AssetID:    nil,
			ParamsJSON: paramsJSON,
			StartedAt:  startedAt,
			Status:     models.ToolCallRunning,
			CreatedAt:  startedAt,
		}
		if err := s.queries.CreateToolCallLog(callLog); err != nil {
			fmt.Printf("[scanner] failed to create tool call log: %v\n", err)
		}
	}

	// Invoke
	taskID := req.IdempotencyKey
	if taskID == "" {
		taskID = util.GenerateID()
	}
	res := toolrun.Invoke(ctx, s.db, s.runner, s.registry, toolrun.InvokeInput{
		ProjectID: req.ProjectID,
		RunID:     &req.RunID,
		TaskID:    taskID,
		ToolID:    toolID,
		Params:    params,
	})

	finishedAt := time.Now().UTC()
	var durationMs int64
	if s.queries != nil {
		durationMs = finishedAt.Sub(startedAt).Milliseconds()
	}

	var exitCode *int
	var status models.ToolCallStatus
	var outputSummary, errorMsg string
	if res.Err != nil {
		status = models.ToolCallFailed
		errorMsg = res.Err.Error()
	} else {
		status = models.ToolCallCompleted
	}
	if res.Task != nil && res.Task.ExitCode != nil {
		exitCode = res.Task.ExitCode
	}
	if len(res.Stdout) > 0 {
		summary := string(res.Stdout)
		if len(summary) > 500 {
			summary = summary[:500]
		}
		outputSummary = summary
	}
	if s.queries != nil {
		if res.Task != nil {
			_ = s.queries.UpdateToolCallLogTaskID(logID, res.Task.ID)
		}
		_ = s.queries.UpdateToolCallLogOnComplete(logID, finishedAt, exitCode, status, durationMs, outputSummary, errorMsg)
	}

	if res.Err != nil {
		return nil, res.Err
	}

	receipt := ExecutionReceipt{
		ToolID:          toolID,
		CommandTemplate: "",
		TaskID:          taskID,
		DurationMs:      durationMs,
		ExitCode:        exitCode,
	}
	if res.Task != nil {
		receipt.CommandTemplate = res.Task.CommandTemplate
		receipt.TaskID = res.Task.ID
		if res.Task.ExitCode != nil {
			receipt.ExitCode = res.Task.ExitCode
		}
	}

	return &ScanResult{
		Stdout:  res.Stdout,
		Receipt: receipt,
	}, nil
}

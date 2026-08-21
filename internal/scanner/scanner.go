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
			excluded, err := s.scopeChecker.IsExcluded(req.Authorization.ProjectID, t)
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

	return s.executePortScan(ctx, req)
}

// mapNaabuParams maps a ScanRequest to the registry's RenderParams for the
// naabu tool. This is the guarded process adapter's contract seam: the CLI
// shape is owned here, in native tool units, so the engine never learns flags.
// hostFile is provided by the caller (WriteHostFile) and passed through verbatim.
func mapNaabuParams(req ScanRequest, hostFile string) toolregistry.RenderParams {
	return toolregistry.RenderParams{
		"host_file":  hostFile,
		"port_range": req.PortRange,
		"rate":       req.Budgets.Rate,
		"threads":    req.Budgets.Threads,
		"timeout":    req.Budgets.TimeoutMs,
	}
}

// renderArgs renders and allowlist-validates argv for a request.
// It is the testable seam for process-adapter parity with toolregistry.Render.
func (s *Scanner) renderArgs(toolID string, req ScanRequest, hostFile string) ([]string, error) {
	params := mapNaabuParams(req, hostFile)
	argv, err := s.registry.Render(toolID, params)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", toolID, err)
	}
	allowlist := toolguard.NewAllowlistFromBinaries(s.registry.Binaries())
	if err := allowlist.Validate(argv[0], argv[1:]); err != nil {
		return nil, fmt.Errorf("allowlist: %w", err)
	}
	return argv, nil
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

	params := mapNaabuParams(req, hostFile)

	// Guarded process adapter path: render → allowlist. toolrun.Invoke re-renders
	// and re-validates at the final boundary; this pre-check is defense in depth.
	toolID := core.ActionToTool[core.ActionPortScan]
	if _, err := s.renderArgs(toolID, req, hostFile); err != nil {
		return nil, err
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
			RunID:      req.Authorization.RunID,
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
		ProjectID: req.Authorization.ProjectID,
		RunID:     &req.Authorization.RunID,
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

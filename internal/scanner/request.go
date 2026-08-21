package scanner

import (
	"fmt"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

// Budgets carries rate and resource limits for a scan request.
// Values are in native tool units (no scaling).
// Rate is requests per second for naabu -rate;
// TimeoutMs is milliseconds for naabu -timeout.
type Budgets struct {
	Rate      int // requests per second
	Threads   int // concurrency (-c)
	TimeoutMs int // milliseconds (-timeout)
}

// Authorization identifies the owning project and run for a scan request.
type Authorization struct {
	ProjectID string
	RunID     string
}

// ScanRequest is the typed semantic request for the execution module.
// It hides CLI flags and SDK option types from callers.
// Targets are assumed normalized (IP/host strings) by the caller
// (the engine verifies asset existence before building the request).
// PolicyTier and TemplateIDs are reserved for the full P2-1 contract
// (all actions + nuclei routing) and are unused by the ActionPortScan tracer.
type ScanRequest struct {
	Action         core.TaskAction
	Targets        []string // normalized targets
	PortRange      string   // naabu port range preset: high-risk, top1000, etc.
	Budgets        Budgets
	PolicyTier     string   // reserved: policy tier for future actions
	TemplateIDs    []string // reserved: template/rule selection
	Authorization  Authorization
	IdempotencyKey string
}

// Validate checks that the request is well-formed.
func (r ScanRequest) Validate() error {
	if r.Action == "" {
		return fmt.Errorf("%w: action is required", ErrInvalidRequest)
	}
	if r.Action != core.ActionPortScan {
		return fmt.Errorf("%w: unsupported action %s", ErrInvalidRequest, r.Action)
	}
	if len(r.Targets) == 0 {
		return fmt.Errorf("%w: targets must not be empty", ErrEmptyTargets)
	}
	for i, t := range r.Targets {
		if t == "" {
			return fmt.Errorf("%w: target[%d] is empty", ErrEmptyTargets, i)
		}
	}
	if r.Authorization.ProjectID == "" {
		return fmt.Errorf("%w: project_id is required", ErrInvalidRequest)
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency_key is required", ErrInvalidRequest)
	}
	return nil
}

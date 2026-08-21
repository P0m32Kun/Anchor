package scanner

import (
	"fmt"

	"github.com/P0m32Kun/Anchor/internal/scanengine/core"
)

// Budgets carries rate and resource limits for a scan request.
// Values are in native tool units (no scaling).
type Budgets struct {
	Rate    int
	Threads int
	Timeout int
}

// ScanRequest is the typed semantic request for the execution module.
// It hides CLI flags and SDK option types from callers.
type ScanRequest struct {
	Action         core.TaskAction
	Targets        []string
	PortRange      string
	Budgets        Budgets
	ProjectID      string
	RunID          string
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
	if r.ProjectID == "" {
		return fmt.Errorf("%w: project_id is required", ErrInvalidRequest)
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency_key is required", ErrInvalidRequest)
	}
	return nil
}

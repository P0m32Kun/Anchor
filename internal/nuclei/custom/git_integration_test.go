//go:build integration

package custom

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestExecCloner_RealClone is gated behind the `integration` build tag and
// exercises the real `git` binary against a small public repository.
//
// Run with:  go test -tags=integration ./internal/nuclei/custom/...
func TestExecCloner_RealClone(t *testing.T) {
	tmp := t.TempDir()
	dest := filepath.Join(tmp, "repo")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := ExecCloner{}
	if err := c.Clone(ctx, "https://github.com/projectdiscovery/nuclei-templates.git", "main", dest); err != nil {
		t.Fatalf("clone: %v", err)
	}
}

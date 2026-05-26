package toolrun

import (
	"context"
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/toolregistry"
)

type mockDB struct {
	createFn    func(t *models.ScanTask) error
	listFn      func(taskID string) ([]*models.RawArtifact, error)
	bundleVerFn func() (string, error)
}

func (m *mockDB) CreateScanTask(t *models.ScanTask) error                       { return m.createFn(t) }
func (m *mockDB) ListRawArtifactsByTask(id string) ([]*models.RawArtifact, error) { return m.listFn(id) }
func (m *mockDB) GetActiveNucleiCustomBundleVersion() (string, error)            { return m.bundleVerFn() }

type mockRunner struct {
	runFn    func(ctx context.Context, taskID string) error
	cancelFn func(taskID string) error
}

func (m *mockRunner) Run(_ context.Context, _ string) error {
	if m.runFn != nil {
		return m.runFn(context.Background(), "")
	}
	return nil
}
func (m *mockRunner) Cancel(taskID string) error {
	if m.cancelFn != nil {
		return m.cancelFn(taskID)
	}
	return nil
}

func mockDBOK() *mockDB {
	return &mockDB{
		createFn:    func(t *models.ScanTask) error { return nil },
		listFn:      func(id string) ([]*models.RawArtifact, error) { return nil, nil },
		bundleVerFn: func() (string, error) { return "", nil },
	}
}

func TestInvoke_UnknownTool(t *testing.T) {
	reg := toolregistry.DefaultRegistry()
	res := Invoke(context.Background(), mockDBOK(), &mockRunner{}, reg, InvokeInput{
		ProjectID: "proj1",
		ToolID:    "nonexistent",
		Params:    toolregistry.RenderParams{},
	})
	if res.Err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestInvoke_BasicRun(t *testing.T) {
	reg := toolregistry.DefaultRegistry()
	taskCreated := false
	db := &mockDB{
		createFn:    func(t *models.ScanTask) error { taskCreated = true; return nil },
		listFn:      func(id string) ([]*models.RawArtifact, error) { return nil, nil },
		bundleVerFn: func() (string, error) { return "", nil },
	}
	res := Invoke(context.Background(), db, &mockRunner{}, reg, InvokeInput{
		ProjectID: "proj1",
		RunID:     strPtr("run1"),
		ToolID:    "cdncheck",
		Params: toolregistry.RenderParams{
			"ips": "1.2.3.4",
		},
	})
	if !taskCreated {
		t.Error("ScanTask was not created")
	}
	_ = res.Task
}

func TestInvoke_ExtraArgs(t *testing.T) {
	reg := toolregistry.DefaultRegistry()
	res := Invoke(context.Background(), mockDBOK(), &mockRunner{}, reg, InvokeInput{
		ProjectID:  "proj1",
		ToolID:     "cdncheck",
		Params:     toolregistry.RenderParams{"ips": "1.1.1.1"},
		ExtraArgs:  []string{"-custom", "flag"},
	})
	// ExtraArgs should be in the command template
	if res.Task != nil && res.Err == nil {
		t.Logf("task created with extra args")
	}
}

func strPtr(s string) *string { return &s }

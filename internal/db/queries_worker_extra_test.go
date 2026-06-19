package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestCreateWorkerNode_GetWorkerNode_RoundTrip(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	w := &models.WorkerNode{
		ID:               "w-1",
		Name:             "worker-alpha",
		Endpoint:         "https://worker-1.example.com:9090",
		Mode:             models.WorkerModeRemote,
		Status:           models.WorkerStatusOnline,
		TrustLevel:       "high",
		NetworkProfile:   "internal",
		Capabilities:     `["katana","nuclei"]`,
		ToolVersions:     `{"nuclei":"3.0.0"}`,
		TemplateVersions: `{"nuclei-templates":"9.7.0"}`,
		MaxConcurrency:   4,
		LastSeen:         &now,
		CreatedAt:        now,
	}

	if err := q.CreateWorkerNode(w); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := q.GetWorkerNode("w-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get returned nil for existing row")
	}
	if got.ID != "w-1" {
		t.Errorf("id: want w-1, got %q", got.ID)
	}
	if got.Name != "worker-alpha" {
		t.Errorf("name: want worker-alpha, got %q", got.Name)
	}
	if got.Endpoint != "https://worker-1.example.com:9090" {
		t.Errorf("endpoint: want https://worker-1.example.com:9090, got %q", got.Endpoint)
	}
	if got.Mode != models.WorkerModeRemote {
		t.Errorf("mode: want remote, got %q", got.Mode)
	}
	if got.Status != models.WorkerStatusOnline {
		t.Errorf("status: want online, got %q", got.Status)
	}
	if got.TrustLevel != "high" {
		t.Errorf("trust_level: want high, got %q", got.TrustLevel)
	}
	if got.MaxConcurrency != 4 {
		t.Errorf("max_concurrency: want 4, got %d", got.MaxConcurrency)
	}
	if got.LastSeen == nil || !got.LastSeen.Equal(now) {
		t.Errorf("last_seen: want %v, got %v", now, got.LastSeen)
	}
	if got.RevokedAt != nil {
		t.Errorf("revoked_at: want nil, got %v", got.RevokedAt)
	}
}

func TestGetWorkerNode_NotFoundReturnsNil(t *testing.T) {
	q := New(openTestDB(t))

	got, err := q.GetWorkerNode("nonexistent")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestListWorkerNodes_OrderByCreatedAt(t *testing.T) {
	q := New(openTestDB(t))
	base := time.Now().UTC().Truncate(time.Second)

	for i, id := range []string{"w-old", "w-mid", "w-new"} {
		w := &models.WorkerNode{
			ID:             id,
			Name:           id,
			Endpoint:       "https://" + id + ".example.com",
			Mode:           models.WorkerModeRemote,
			Status:         models.WorkerStatusOnline,
			TrustLevel:     "standard",
			NetworkProfile: "default",
			MaxConcurrency: 2,
			CreatedAt:      base.Add(time.Duration(i) * time.Minute),
		}
		if err := q.CreateWorkerNode(w); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	list, err := q.ListWorkerNodes()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len: want 3, got %d", len(list))
	}
	wantOrder := []string{"w-old", "w-mid", "w-new"}
	for i, want := range wantOrder {
		if list[i].ID != want {
			t.Errorf("position %d: want %q, got %q", i, want, list[i].ID)
		}
	}
}

func TestUpdateWorkerNodeStatus(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	w := &models.WorkerNode{
		ID:             "w-status",
		Name:           "worker",
		Endpoint:       "https://w.example.com",
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		NetworkProfile: "default",
		MaxConcurrency: 1,
		CreatedAt:      now,
	}
	if err := q.CreateWorkerNode(w); err != nil {
		t.Fatalf("create: %v", err)
	}

	newLastSeen := now.Add(5 * time.Minute)
	if err := q.UpdateWorkerNodeStatus("w-status", models.WorkerStatusBusy, newLastSeen); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, err := q.GetWorkerNode("w-status")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != models.WorkerStatusBusy {
		t.Errorf("status: want busy, got %q", got.Status)
	}
	if got.LastSeen == nil || !got.LastSeen.Equal(newLastSeen) {
		t.Errorf("last_seen: want %v, got %v", newLastSeen, got.LastSeen)
	}
}

func TestUpdateWorkerNodeTemplateVersions(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	w := &models.WorkerNode{
		ID:               "w-tpl",
		Name:             "worker",
		Endpoint:         "https://w.example.com",
		Mode:             models.WorkerModeRemote,
		Status:           models.WorkerStatusOnline,
		TrustLevel:       "standard",
		NetworkProfile:   "default",
		TemplateVersions: `{"old":"1.0"}`,
		MaxConcurrency:   1,
		CreatedAt:        now,
	}
	if err := q.CreateWorkerNode(w); err != nil {
		t.Fatalf("create: %v", err)
	}

	newLastSeen := now.Add(time.Minute)
	newVersions := `{"nuclei-templates":"10.0.0","katana":"1.1.0"}`
	if err := q.UpdateWorkerNodeTemplateVersions("w-tpl", models.WorkerStatusOnline, newLastSeen, newVersions); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := q.GetWorkerNode("w-tpl")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.TemplateVersions != newVersions {
		t.Errorf("template_versions: want %q, got %q", newVersions, got.TemplateVersions)
	}
	if got.LastSeen == nil || !got.LastSeen.Equal(newLastSeen) {
		t.Errorf("last_seen: want %v, got %v", newLastSeen, got.LastSeen)
	}
}

func TestUpdateWorkerNodeMetrics(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	w := &models.WorkerNode{
		ID:             "w-metrics",
		Name:           "worker",
		Endpoint:       "https://w.example.com",
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		NetworkProfile: "default",
		MaxConcurrency: 1,
		CreatedAt:      now,
	}
	if err := q.CreateWorkerNode(w); err != nil {
		t.Fatalf("create: %v", err)
	}

	cpu := 55.5
	mem := 72.3
	disk := 40.0
	metricsAt := now.Add(2 * time.Minute)
	if err := q.UpdateWorkerNodeMetrics("w-metrics", &cpu, &mem, &disk, metricsAt); err != nil {
		t.Fatalf("update metrics: %v", err)
	}

	list, err := q.ListWorkerNodes()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len: want 1, got %d", len(list))
	}
	got := list[0]
	if got.CPUPercent == nil || *got.CPUPercent != cpu {
		t.Errorf("cpu_percent: want %v, got %v", cpu, got.CPUPercent)
	}
	if got.MemPercent == nil || *got.MemPercent != mem {
		t.Errorf("mem_percent: want %v, got %v", mem, got.MemPercent)
	}
	if got.DiskPercent == nil || *got.DiskPercent != disk {
		t.Errorf("disk_percent: want %v, got %v", disk, got.DiskPercent)
	}
	if got.MetricsUpdatedAt == nil || !got.MetricsUpdatedAt.Equal(metricsAt) {
		t.Errorf("metrics_updated_at: want %v, got %v", metricsAt, got.MetricsUpdatedAt)
	}
}

func TestRevokeWorkerNode(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	w := &models.WorkerNode{
		ID:             "w-revoke",
		Name:           "worker",
		Endpoint:       "https://w.example.com",
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		NetworkProfile: "default",
		MaxConcurrency: 1,
		CreatedAt:      now,
	}
	if err := q.CreateWorkerNode(w); err != nil {
		t.Fatalf("create: %v", err)
	}

	revokedAt := now.Add(time.Hour)
	if err := q.RevokeWorkerNode("w-revoke", revokedAt); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	got, err := q.GetWorkerNode("w-revoke")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != models.WorkerStatusOffline {
		t.Errorf("status: want offline, got %q", got.Status)
	}
	if got.RevokedAt == nil || !got.RevokedAt.Equal(revokedAt) {
		t.Errorf("revoked_at: want %v, got %v", revokedAt, got.RevokedAt)
	}
}

func TestDeleteWorkerNode(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	w := &models.WorkerNode{
		ID:             "w-del",
		Name:           "worker",
		Endpoint:       "https://w.example.com",
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		NetworkProfile: "default",
		MaxConcurrency: 1,
		CreatedAt:      now,
	}
	if err := q.CreateWorkerNode(w); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := q.DeleteWorkerNode("w-del"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, err := q.GetWorkerNode("w-del")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if got != nil {
		t.Errorf("want nil after delete, got %+v", got)
	}
}

func TestCreateWorkerHealthCheck_ListWorkerHealthChecks(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)

	// Create parent worker first
	w := &models.WorkerNode{
		ID:             "w-hc",
		Name:           "worker",
		Endpoint:       "https://w.example.com",
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		NetworkProfile: "default",
		MaxConcurrency: 1,
		CreatedAt:      now,
	}
	if err := q.CreateWorkerNode(w); err != nil {
		t.Fatalf("create worker: %v", err)
	}

	hc1 := &models.WorkerHealthCheck{
		ID:        "hc-1",
		WorkerID:  "w-hc",
		Tool:      "nuclei",
		Status:    models.HealthCheckReady,
		Version:   "3.0.0",
		Details:   "all good",
		CheckedAt: now,
	}
	hc2 := &models.WorkerHealthCheck{
		ID:        "hc-2",
		WorkerID:  "w-hc",
		Tool:      "katana",
		Status:    models.HealthCheckMissing,
		Version:   "",
		Details:   "binary not found",
		CheckedAt: now.Add(time.Minute),
	}

	if err := q.CreateWorkerHealthCheck(hc1); err != nil {
		t.Fatalf("create hc1: %v", err)
	}
	if err := q.CreateWorkerHealthCheck(hc2); err != nil {
		t.Fatalf("create hc2: %v", err)
	}

	list, err := q.ListWorkerHealthChecks("w-hc")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len: want 2, got %d", len(list))
	}
	// Ordered by checked_at DESC
	if list[0].ID != "hc-2" {
		t.Errorf("position 0: want hc-2, got %q", list[0].ID)
	}
	if list[1].ID != "hc-1" {
		t.Errorf("position 1: want hc-1, got %q", list[1].ID)
	}
	if list[0].Tool != "katana" {
		t.Errorf("tool: want katana, got %q", list[0].Tool)
	}
	if list[0].Status != models.HealthCheckMissing {
		t.Errorf("status: want missing, got %q", list[0].Status)
	}
	if list[1].Status != models.HealthCheckReady {
		t.Errorf("status: want ready, got %q", list[1].Status)
	}
}

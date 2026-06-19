package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// --- handleGetDashboardStats ---

func TestHandleGetDashboardStats_Empty(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	w := httptest.NewRecorder()

	server.handleGetDashboardStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var stats models.DashboardStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if stats.TotalProjects != 0 {
		t.Errorf("total_projects = %d, want 0", stats.TotalProjects)
	}
}

func TestHandleGetDashboardStats_WithData(t *testing.T) {
	server, rawDB, cleanup := setupTestServer(t)
	defer cleanup()

	queries := db.New(rawDB)
	now := time.Now().UTC()

	// Create a project
	p := &models.Project{
		ID:             util.GenerateID(),
		Name:           "Dashboard Test",
		DefaultProfile: "standard",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := queries.CreateProject(p); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create a worker
	w := &models.WorkerNode{
		ID:             util.GenerateID(),
		Name:           "w1",
		Mode:           models.WorkerModeRemote,
		Status:         models.WorkerStatusOnline,
		TrustLevel:     "standard",
		MaxConcurrency: 10,
		LastSeen:       &now,
		CreatedAt:      now,
	}
	if err := queries.CreateWorkerNode(w); err != nil {
		t.Fatalf("create worker: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	w2 := httptest.NewRecorder()

	server.handleGetDashboardStats(w2, req)

	resp := w2.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var stats models.DashboardStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if stats.TotalProjects != 1 {
		t.Errorf("total_projects = %d, want 1", stats.TotalProjects)
	}
	if stats.OnlineWorkers != 1 {
		t.Errorf("online_workers = %d, want 1", stats.OnlineWorkers)
	}
}

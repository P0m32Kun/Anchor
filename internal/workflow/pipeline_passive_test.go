package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/scope"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestFofaExpandCompany_Mock(t *testing.T) {
	fofaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": false, "size": 1,
			"results": [][]string{{"sub.fofa.example", "198.51.100.1", "80", "t", "http", "nginx"}},
		})
	}))
	defer fofaSrv.Close()

	t.Setenv("FOFA_BASE_URL", fofaSrv.URL)

	dir := t.TempDir()
	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	queries := db.New(sqlDB)

	now := time.Now().UTC()
	project := &models.Project{ID: util.GenerateID(), Name: "fofa-passive", CreatedAt: now, UpdatedAt: now}
	if err := queries.CreateProject(project); err != nil {
		t.Fatal(err)
	}

	p := NewPipeline(queries, nil, scope.NewEngine(queries), dir).
		WithFOFA("k")
	p.projectID = project.ID

	domains, ips, err := p.fofaExpandCompany(context.Background(), "TestCorp")
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0] != "sub.fofa.example" {
		t.Fatalf("domains = %v", domains)
	}
	if len(ips) != 1 || ips[0] != "198.51.100.1" {
		t.Fatalf("ips = %v", ips)
	}

	targets, err := queries.ListTargetsByProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) < 2 {
		t.Fatalf("expected persisted targets, got %d", len(targets))
	}
}

func TestQuakeSearchCompany_Mock(t *testing.T) {
	quakeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/search/quake_service" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": []map[string]interface{}{
				{
					"ip": "203.0.113.10", "port": 80, "domain": "quake-only.example.com",
					"service": map[string]interface{}{"name": "http", "http": map[string]interface{}{}},
				},
			},
		})
	}))
	defer quakeSrv.Close()

	t.Setenv("QUAKE_BASE_URL", quakeSrv.URL)

	dir := t.TempDir()
	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	queries := db.New(sqlDB)

	now := time.Now().UTC()
	project := &models.Project{ID: util.GenerateID(), Name: "quake-passive", CreatedAt: now, UpdatedAt: now}
	if err := queries.CreateProject(project); err != nil {
		t.Fatal(err)
	}
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "quake", APIKey: "k", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	p := NewPipeline(queries, nil, scope.NewEngine(queries), dir).
		WithConfig(models.PipelineConfig{PassiveSearchResultLimit: 50})
	p.projectID = project.ID

	domains, ips, err := p.quakeSearchCompany(context.Background(), "TestCorp")
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0] != "quake-only.example.com" {
		t.Fatalf("domains = %v", domains)
	}
	if len(ips) != 1 || ips[0] != "203.0.113.10" {
		t.Fatalf("ips = %v", ips)
	}
}

func TestRunPassiveSearch_RecordsSearchStage(t *testing.T) {
	fofaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": false, "size": 0, "results": [][]interface{}{},
		})
	}))
	defer fofaSrv.Close()

	t.Setenv("FOFA_BASE_URL", fofaSrv.URL)

	dir := t.TempDir()
	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	queries := db.New(sqlDB)
	scopeEng := scope.NewEngine(queries)

	now := time.Now().UTC()
	project := &models.Project{ID: util.GenerateID(), Name: "search-stage", CreatedAt: now, UpdatedAt: now}
	if err := queries.CreateProject(project); err != nil {
		t.Fatal(err)
	}
	if err := queries.SaveEngineCredential(&models.EngineCredential{
		ID: util.GenerateID(), Engine: "fofa", APIKey: "k", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	runID := util.GenerateID()
	if err := queries.CreatePipelineRun(&models.PipelineRun{
		ID: runID, ProjectID: project.ID, Mode: "external", Status: "running",
		StartedAt: now, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	p := NewPipeline(queries, nil, scopeEng, dir).
		WithFOFA("k").
		WithConfig(models.PipelineConfig{
			EnablePassiveSearch: true,
			EnableFOFA:          true,
		}).
		WithRunID(runID)
	p.projectID = project.ID

	if err := p.runPassiveSearch(context.Background(), "EmptyCorp"); err != nil {
		t.Fatalf("runPassiveSearch: %v", err)
	}

	stages, err := queries.ListPipelineRunStages(runID)
	if err != nil {
		t.Fatal(err)
	}
	var searchDone bool
	for _, s := range stages {
		if s.Stage == string(StageSearch) && s.Status == "completed" {
			searchDone = true
		}
	}
	if !searchDone {
		t.Fatalf("search stage not completed: %#v", stages)
	}
}

package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// --- helpers ---

func setupFindingTestData(t *testing.T) (*Queries, string, string) {
	t.Helper()
	q := New(openTestDB(t))
	now := time.Now().UTC().Truncate(time.Second)
	if err := q.CreateProject(&models.Project{
		ID: "proj-1", Name: "test", RateLimit: 10,
		DefaultProfile: string(models.ProfileStandard),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := q.CreateScanPlan(&models.ScanPlan{
		ID: "plan-1", ProjectID: "proj-1", WorkflowType: "manual",
		Profile: models.ProfileStandard, Status: "approved", CreatedBy: "test",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scan plan: %v", err)
	}
	if err := q.CreateRun(&models.Run{
		ID: "run-1", ProjectID: "proj-1", Name: "test-run",
		Status: models.RunRunning, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return q, "proj-1", "run-1"
}

func seedFinding(t *testing.T, q *Queries, id, projectID, runID string, status models.FindingStatus, severity models.FindingSeverity) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	if err := q.CreateFinding(&models.Finding{
		ID: id, ProjectID: projectID, RunID: &runID,
		SourceTool: "nuclei", SourceRuleID: "CVE-2024-0001", DedupKey: id,
		Title: "Test Finding", Severity: severity, Confidence: 80, Priority: 100,
		Status: status, Summary: "test", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed finding %s: %v", id, err)
	}
}

// --- tests ---

func TestUpdateFindingStatus(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-1", projID, runID, models.FindingNew, models.SeverityHigh)

	now := time.Now().UTC().Truncate(time.Second)
	if err := q.UpdateFindingStatus("f-1", models.FindingConfirmed, now); err != nil {
		t.Fatalf("UpdateFindingStatus: %v", err)
	}

	got, err := q.GetFinding("f-1")
	if err != nil {
		t.Fatalf("GetFinding: %v", err)
	}
	if got == nil {
		t.Fatal("GetFinding returned nil")
	}
	if got.Status != models.FindingConfirmed {
		t.Errorf("Status = %q, want %q", got.Status, models.FindingConfirmed)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, now)
	}
}

func TestUpdateFindingEvidence(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-ev", projID, runID, models.FindingNew, models.SeverityLow)

	now := time.Now().UTC().Truncate(time.Second)
	if err := q.UpdateFindingEvidence("f-ev", models.SeverityCritical, 95, 200, "updated summary", "updated remediation", now); err != nil {
		t.Fatalf("UpdateFindingEvidence: %v", err)
	}

	got, err := q.GetFinding("f-ev")
	if err != nil {
		t.Fatalf("GetFinding: %v", err)
	}
	if got == nil {
		t.Fatal("GetFinding returned nil")
	}
	if got.Severity != models.SeverityCritical {
		t.Errorf("Severity = %q, want %q", got.Severity, models.SeverityCritical)
	}
	if got.Confidence != 95 {
		t.Errorf("Confidence = %d, want 95", got.Confidence)
	}
	if got.Priority != 200 {
		t.Errorf("Priority = %d, want 200", got.Priority)
	}
	if got.Summary != "updated summary" {
		t.Errorf("Summary = %q, want %q", got.Summary, "updated summary")
	}
	if got.Remediation != "updated remediation" {
		t.Errorf("Remediation = %q, want %q", got.Remediation, "updated remediation")
	}
}

func TestUpdateFindingWebEndpointID(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-wep", projID, runID, models.FindingNew, models.SeverityMedium)

	// Create prerequisite asset + web_endpoint for FK constraint
	now := time.Now().UTC().Truncate(time.Second)
	if err := q.CreateAsset(&models.Asset{ID: "asset-wep", ProjectID: projID, Type: "url", Value: "https://example.com", NormalizedValue: "https://example.com", FirstSeen: now, LastSeen: now}); err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := q.CreateWebEndpoint(&models.WebEndpoint{ID: "wep-123", ProjectID: projID, AssetID: "asset-wep", URL: "https://example.com/test", Scheme: "https", Host: "example.com", CreatedAt: now}); err != nil {
		t.Fatalf("create web_endpoint: %v", err)
	}

	if err := q.UpdateFindingWebEndpointID("f-wep", "wep-123"); err != nil {
		t.Fatalf("UpdateFindingWebEndpointID: %v", err)
	}

	got, err := q.GetFinding("f-wep")
	if err != nil {
		t.Fatalf("GetFinding: %v", err)
	}
	if got == nil {
		t.Fatal("GetFinding returned nil")
	}
	if got.WebEndpointID == nil || *got.WebEndpointID != "wep-123" {
		t.Errorf("WebEndpointID = %v, want %q", got.WebEndpointID, "wep-123")
	}
}

func TestListFindingsByStatus(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-new-1", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-new-2", projID, runID, models.FindingNew, models.SeverityLow)
	seedFinding(t, q, "f-conf-1", projID, runID, models.FindingConfirmed, models.SeverityCritical)

	list, err := q.ListFindingsByStatus(projID, models.FindingNew)
	if err != nil {
		t.Fatalf("ListFindingsByStatus: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(list))
	}
	for _, f := range list {
		if f.Status != models.FindingNew {
			t.Errorf("finding %s: status = %q, want %q", f.ID, f.Status, models.FindingNew)
		}
	}
}

func TestCountFindingsByProject(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-c1", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-c2", projID, runID, models.FindingConfirmed, models.SeverityMedium)
	seedFinding(t, q, "f-c3", projID, runID, models.FindingNew, models.SeverityLow)

	// Count all
	total, err := q.CountFindingsByProject(projID, "")
	if err != nil {
		t.Fatalf("CountFindingsByProject(no filter): %v", err)
	}
	if total != 3 {
		t.Errorf("total count = %d, want 3", total)
	}

	// Count by status
	newCount, err := q.CountFindingsByProject(projID, models.FindingNew)
	if err != nil {
		t.Fatalf("CountFindingsByProject(new): %v", err)
	}
	if newCount != 2 {
		t.Errorf("new count = %d, want 2", newCount)
	}

	// Count with no matches
	fpCount, err := q.CountFindingsByProject(projID, models.FindingFalsePositive)
	if err != nil {
		t.Fatalf("CountFindingsByProject(false_positive): %v", err)
	}
	if fpCount != 0 {
		t.Errorf("false_positive count = %d, want 0", fpCount)
	}
}

func TestListFindingsByStatusPaginated(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	for i := 0; i < 5; i++ {
		seedFinding(t, q, "f-pag-"+string(rune('a'+i)), projID, runID, models.FindingNew, models.SeverityInfo)
	}
	seedFinding(t, q, "f-pag-other", projID, runID, models.FindingConfirmed, models.SeverityHigh)

	// First page
	page1, err := q.ListFindingsByStatusPaginated(projID, models.FindingNew, 2, 0)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(page1))
	}

	// Second page
	page2, err := q.ListFindingsByStatusPaginated(projID, models.FindingNew, 2, 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page 2: expected 2, got %d", len(page2))
	}

	// Third page (remainder)
	page3, err := q.ListFindingsByStatusPaginated(projID, models.FindingNew, 2, 4)
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page 3: expected 1, got %d", len(page3))
	}

	// No overlap between pages
	seen := map[string]bool{}
	for _, f := range page1 {
		seen[f.ID] = true
	}
	for _, f := range page2 {
		if seen[f.ID] {
			t.Errorf("duplicate finding across pages: %s", f.ID)
		}
	}
}

func TestListFindingsForReport(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-r1", projID, runID, models.FindingNew, models.SeverityLow)
	seedFinding(t, q, "f-r2", projID, runID, models.FindingConfirmed, models.SeverityCritical)
	seedFinding(t, q, "f-r3", projID, runID, models.FindingFalsePositive, models.SeverityMedium)

	// Update priority to control ordering
	now := time.Now().UTC().Truncate(time.Second)
	q.UpdateFindingEvidence("f-r1", models.SeverityLow, 40, 50, "", "", now)
	q.UpdateFindingEvidence("f-r2", models.SeverityCritical, 99, 200, "", "", now)
	q.UpdateFindingEvidence("f-r3", models.SeverityMedium, 60, 100, "", "", now)

	list, err := q.ListFindingsForReport(projID)
	if err != nil {
		t.Fatalf("ListFindingsForReport: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(list))
	}
	// Should be ordered by priority DESC
	if list[0].ID != "f-r2" {
		t.Errorf("first finding = %s, want f-r2 (priority 200)", list[0].ID)
	}
	if list[2].ID != "f-r1" {
		t.Errorf("last finding = %s, want f-r1 (priority 50)", list[2].ID)
	}
}

func TestListFindingsByRun(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-run-1", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-run-2", projID, runID, models.FindingConfirmed, models.SeverityMedium)

	// Create a second run with its own finding
	now := time.Now().UTC().Truncate(time.Second)
	q.CreateRun(&models.Run{ID: "run-2", ProjectID: projID, Name: "run-2", Status: models.RunRunning, CreatedAt: now})
	seedFinding(t, q, "f-run-3", projID, "run-2", models.FindingNew, models.SeverityLow)

	list, err := q.ListFindingsByRun(projID, runID)
	if err != nil {
		t.Fatalf("ListFindingsByRun: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 findings for run-1, got %d", len(list))
	}
	for _, f := range list {
		if f.RunID == nil || *f.RunID != runID {
			t.Errorf("finding %s: run_id = %v, want %q", f.ID, f.RunID, runID)
		}
	}
}

func TestCreateEvidence_ListEvidenceByFinding(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-evd", projID, runID, models.FindingNew, models.SeverityHigh)

	now := time.Now().UTC().Truncate(time.Second)
	// Create prerequisite raw_artifact for FK constraint
	if err := q.CreateRawArtifact(&models.RawArtifact{ID: "art-1", ProjectID: projID, Type: "response", Path: "/tmp/art-1.txt", CreatedAt: now}); err != nil {
		t.Fatalf("create raw_artifact: %v", err)
	}
	artID := "art-1"
	evidence := []*models.Evidence{
		{ID: "ev-1", FindingID: "f-evd", Type: models.EvidenceRequest, Excerpt: "GET /api", CreatedBy: "scanner", CreatedAt: now},
		{ID: "ev-2", FindingID: "f-evd", Type: models.EvidenceResponse, ArtifactID: &artID, Excerpt: "200 OK", CreatedBy: "scanner", CreatedAt: now.Add(time.Second)},
	}
	for _, e := range evidence {
		if err := q.CreateEvidence(e); err != nil {
			t.Fatalf("CreateEvidence %s: %v", e.ID, err)
		}
	}

	list, err := q.ListEvidenceByFinding("f-evd")
	if err != nil {
		t.Fatalf("ListEvidenceByFinding: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 evidence, got %d", len(list))
	}
	if list[0].ID != "ev-1" {
		t.Errorf("first evidence = %s, want ev-1 (ordered by created_at)", list[0].ID)
	}
	if list[1].ArtifactID == nil || *list[1].ArtifactID != "art-1" {
		t.Errorf("second evidence ArtifactID = %v, want art-1", list[1].ArtifactID)
	}

	// Empty result for unknown finding
	empty, err := q.ListEvidenceByFinding("nonexistent")
	if err != nil {
		t.Fatalf("ListEvidenceByFinding(nonexistent): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 evidence, got %d", len(empty))
	}
}

func TestGetFindingStatsBySeverity(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-sev1", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-sev2", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-sev3", projID, runID, models.FindingNew, models.SeverityCritical)
	seedFinding(t, q, "f-sev4", projID, runID, models.FindingNew, models.SeverityLow)

	stats, err := q.GetFindingStatsBySeverity(runID)
	if err != nil {
		t.Fatalf("GetFindingStatsBySeverity: %v", err)
	}
	if len(stats) != 3 {
		t.Fatalf("expected 3 severity groups, got %d", len(stats))
	}

	statsMap := map[string]int{}
	for _, s := range stats {
		statsMap[s.Severity] = s.Count
	}
	if statsMap["high"] != 2 {
		t.Errorf("high count = %d, want 2", statsMap["high"])
	}
	if statsMap["critical"] != 1 {
		t.Errorf("critical count = %d, want 1", statsMap["critical"])
	}
	if statsMap["low"] != 1 {
		t.Errorf("low count = %d, want 1", statsMap["low"])
	}
}

func TestGetFindingStatsByStatus(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-st1", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-st2", projID, runID, models.FindingNew, models.SeverityMedium)
	seedFinding(t, q, "f-st3", projID, runID, models.FindingConfirmed, models.SeverityLow)

	stats, err := q.GetFindingStatsByStatus(runID)
	if err != nil {
		t.Fatalf("GetFindingStatsByStatus: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 status groups, got %d", len(stats))
	}

	statsMap := map[string]int{}
	for _, s := range stats {
		statsMap[s.Status] = s.Count
	}
	if statsMap["new"] != 2 {
		t.Errorf("new count = %d, want 2", statsMap["new"])
	}
	if statsMap["confirmed"] != 1 {
		t.Errorf("confirmed count = %d, want 1", statsMap["confirmed"])
	}
}

func TestGetFindingAvgConfidence(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)

	// Seed with known confidence values: 60, 80, 100 → avg = 80
	now := time.Now().UTC().Truncate(time.Second)
	for i, conf := range []int{60, 80, 100} {
		id := "f-avg-" + string(rune('a'+i))
		q.CreateFinding(&models.Finding{
			ID: id, ProjectID: projID, RunID: &runID,
			SourceTool: "nuclei", SourceRuleID: "CVE-2024-0001", DedupKey: id,
			Title: "Test", Severity: models.SeverityMedium, Confidence: conf, Priority: 50,
			Status: models.FindingNew, Summary: "test", CreatedAt: now, UpdatedAt: now,
		})
	}

	avg, err := q.GetFindingAvgConfidence(runID)
	if err != nil {
		t.Fatalf("GetFindingAvgConfidence: %v", err)
	}
	if avg < 79.9 || avg > 80.1 {
		t.Errorf("avg confidence = %f, want ~80", avg)
	}

	// Empty run → 0
	avg2, err := q.GetFindingAvgConfidence("nonexistent-run")
	if err != nil {
		t.Fatalf("GetFindingAvgConfidence(empty): %v", err)
	}
	if avg2 != 0 {
		t.Errorf("avg confidence for empty run = %f, want 0", avg2)
	}
}

func TestGetUnlinkedFindingCount(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)

	// seedFinding creates findings with nil AssetID (unlinked)
	seedFinding(t, q, "f-ul1", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-ul2", projID, runID, models.FindingNew, models.SeverityLow)

	// Both are unlinked
	count, err := q.GetUnlinkedFindingCount(runID)
	if err != nil {
		t.Fatalf("GetUnlinkedFindingCount: %v", err)
	}
	if count != 2 {
		t.Errorf("unlinked count = %d, want 2", count)
	}

	// Now link one
	assetID := "asset-1"
	q.CreateFinding(&models.Finding{
		ID: "f-linked", ProjectID: projID, RunID: &runID, AssetID: &assetID,
		SourceTool: "nuclei", SourceRuleID: "CVE-2024-0002", DedupKey: "f-linked",
		Title: "Linked", Severity: models.SeverityMedium, Confidence: 70, Priority: 50,
		Status: models.FindingNew, Summary: "linked", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})

	count2, err := q.GetUnlinkedFindingCount(runID)
	if err != nil {
		t.Fatalf("GetUnlinkedFindingCount after link: %v", err)
	}
	if count2 != 2 {
		t.Errorf("unlinked count after link = %d, want 2", count2)
	}
}

func TestGetTemplateHitStats(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)

	// 3 findings with same tool+rule, 1 confirmed
	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		id := "f-th-" + string(rune('a'+i))
		status := models.FindingNew
		if i == 0 {
			status = models.FindingConfirmed
		}
		q.CreateFinding(&models.Finding{
			ID: id, ProjectID: projID, RunID: &runID,
			SourceTool: "nuclei", SourceRuleID: "CVE-2024-1234", DedupKey: id,
			Title: "Template Hit", Severity: models.SeverityHigh, Confidence: 80, Priority: 100,
			Status: status, Summary: "test", CreatedAt: now, UpdatedAt: now,
		})
	}
	// 1 finding with different rule
	seedFinding(t, q, "f-th-other", projID, runID, models.FindingNew, models.SeverityLow)

	stats, err := q.GetTemplateHitStats(runID)
	if err != nil {
		t.Fatalf("GetTemplateHitStats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 template groups, got %d", len(stats))
	}

	// Find the CVE-2024-1234 group
	for _, s := range stats {
		if s.SourceRuleID == "CVE-2024-1234" {
			if s.HitCount != 3 {
				t.Errorf("hit count = %d, want 3", s.HitCount)
			}
			if s.ConfirmedCount != 1 {
				t.Errorf("confirmed count = %d, want 1", s.ConfirmedCount)
			}
		}
	}
}

func TestGetDictionaryHitStats(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)

	// No ffuf tasks → empty result
	stats, err := q.GetDictionaryHitStats(runID)
	if err != nil {
		t.Fatalf("GetDictionaryHitStats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0 stats for empty run, got %d", len(stats))
	}

	// Create an ffuf scan task
	now := time.Now().UTC().Truncate(time.Second)
	q.CreateScanTask(&models.ScanTask{
		ID: "task-ffuf-1", ProjectID: projID, PlanID: "plan-1",
		RunID: &runID, Tool: "ffuf", CommandTemplate: "ffuf -w wordlist",
		Status: models.TaskCompleted, CreatedAt: now,
	})

	stats, err = q.GetDictionaryHitStats(runID)
	if err != nil {
		t.Fatalf("GetDictionaryHitStats with task: %v", err)
	}
	// No stdout artifact → empty stats (task has no artifact path)
	if len(stats) != 0 {
		t.Errorf("expected 0 stats (no artifact), got %d", len(stats))
	}
}

func TestListEvidenceByFindingIDs(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-b1", projID, runID, models.FindingNew, models.SeverityHigh)
	seedFinding(t, q, "f-b2", projID, runID, models.FindingNew, models.SeverityMedium)

	now := time.Now().UTC().Truncate(time.Second)
	q.CreateEvidence(&models.Evidence{ID: "ev-b1-1", FindingID: "f-b1", Type: models.EvidenceNote, Excerpt: "note1", CreatedAt: now})
	q.CreateEvidence(&models.Evidence{ID: "ev-b1-2", FindingID: "f-b1", Type: models.EvidenceFile, Excerpt: "file1", CreatedAt: now})
	q.CreateEvidence(&models.Evidence{ID: "ev-b2-1", FindingID: "f-b2", Type: models.EvidenceScreenshot, Excerpt: "ss1", CreatedAt: now})

	result, err := q.ListEvidenceByFindingIDs([]string{"f-b1", "f-b2"})
	if err != nil {
		t.Fatalf("ListEvidenceByFindingIDs: %v", err)
	}
	if len(result["f-b1"]) != 2 {
		t.Errorf("f-b1 evidence count = %d, want 2", len(result["f-b1"]))
	}
	if len(result["f-b2"]) != 1 {
		t.Errorf("f-b2 evidence count = %d, want 1", len(result["f-b2"]))
	}

	// Empty input → nil result
	empty, err := q.ListEvidenceByFindingIDs([]string{})
	if err != nil {
		t.Fatalf("ListEvidenceByFindingIDs(empty): %v", err)
	}
	if empty != nil {
		t.Errorf("expected nil for empty input, got %v", empty)
	}

	// Non-existent finding ID → empty map entry
	single, err := q.ListEvidenceByFindingIDs([]string{"nonexistent"})
	if err != nil {
		t.Fatalf("ListEvidenceByFindingIDs(nonexistent): %v", err)
	}
	if len(single) != 0 {
		t.Errorf("expected 0 entries for nonexistent, got %d", len(single))
	}
}

func TestCreateRetestRun_ListRetestRunsByFinding(t *testing.T) {
	q, projID, runID := setupFindingTestData(t)
	seedFinding(t, q, "f-rt", projID, runID, models.FindingNew, models.SeverityHigh)

	now := time.Now().UTC().Truncate(time.Second)
	// Create prerequisite evidence for FK constraint
	if err := q.CreateEvidence(&models.Evidence{ID: "ev-rt-1", FindingID: "f-rt", Type: models.EvidenceNote, Excerpt: "retest evidence", CreatedBy: "tester", CreatedAt: now}); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
	evID := "ev-rt-1"
	retests := []*models.RetestRun{
		{ID: "rt-1", FindingID: "f-rt", TaskID: "task-1", Result: models.RetestStillPresent, CreatedAt: now},
		{ID: "rt-2", FindingID: "f-rt", TaskID: "task-2", Result: models.RetestFixed, EvidenceID: &evID, CreatedAt: now.Add(time.Second)},
	}
	for _, r := range retests {
		if err := q.CreateRetestRun(r); err != nil {
			t.Fatalf("CreateRetestRun %s: %v", r.ID, err)
		}
	}

	list, err := q.ListRetestRunsByFinding("f-rt")
	if err != nil {
		t.Fatalf("ListRetestRunsByFinding: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 retest runs, got %d", len(list))
	}
	// Ordered by created_at DESC → rt-2 first
	if list[0].ID != "rt-2" {
		t.Errorf("first retest = %s, want rt-2", list[0].ID)
	}
	if list[0].Result != models.RetestFixed {
		t.Errorf("first retest result = %q, want %q", list[0].Result, models.RetestFixed)
	}
	if list[0].EvidenceID == nil || *list[0].EvidenceID != "ev-rt-1" {
		t.Errorf("first retest evidence_id = %v, want ev-rt-1", list[0].EvidenceID)
	}
	if list[1].EvidenceID != nil {
		t.Errorf("second retest evidence_id should be nil, got %v", *list[1].EvidenceID)
	}

	// Empty for unknown finding
	empty, err := q.ListRetestRunsByFinding("nonexistent")
	if err != nil {
		t.Fatalf("ListRetestRunsByFinding(nonexistent): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 retest runs, got %d", len(empty))
	}
}

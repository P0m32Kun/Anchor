package db

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestCreateAndGetSRCProgram(t *testing.T) {
	q := New(openTestDB(t))

	// 创建项目
	project := &models.Project{
		ID:   "proj-1",
		Name: "Test Project",
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 创建 SRC 程序
	program := &models.SRCProgram{
		ID:                     "prog-1",
		ProjectID:              "proj-1",
		Name:                   "Test SRC",
		Platform:               "hackerone",
		ProgramURL:             "https://hackerone.com/test",
		RulesURL:               "https://hackerone.com/test/rules",
		AllowAutomation:        true,
		AllowDirBrute:          false,
		AllowWeakPassword:      false,
		AllowAuthenticatedTest: true,
		MaxRPS:                 10,
		MaxConcurrency:         5,
		PreferredVulnTypes:     []string{"rce", "ssrf"},
		PayoutHint:             map[string]any{"rce": 1000},
		Notes:                  "Test notes",
	}

	if err := q.CreateSRCProgram(program); err != nil {
		t.Fatalf("create src program: %v", err)
	}

	// 获取程序
	got, err := q.GetSRCProgram("proj-1")
	if err != nil {
		t.Fatalf("get src program: %v", err)
	}
	if got == nil {
		t.Fatal("expected program, got nil")
	}

	// 验证字段
	if got.ID != "prog-1" {
		t.Errorf("expected ID 'prog-1', got '%s'", got.ID)
	}
	if got.Name != "Test SRC" {
		t.Errorf("expected Name 'Test SRC', got '%s'", got.Name)
	}
	if got.Platform != "hackerone" {
		t.Errorf("expected Platform 'hackerone', got '%s'", got.Platform)
	}
	if !got.AllowAutomation {
		t.Error("expected AllowAutomation true")
	}
	if got.AllowDirBrute {
		t.Error("expected AllowDirBrute false")
	}
	if got.MaxRPS != 10 {
		t.Errorf("expected MaxRPS 10, got %d", got.MaxRPS)
	}
	if len(got.PreferredVulnTypes) != 2 {
		t.Errorf("expected 2 preferred vuln types, got %d", len(got.PreferredVulnTypes))
	}
}

func TestUpdateSRCProgram(t *testing.T) {
	q := New(openTestDB(t))

	// 创建项目
	project := &models.Project{
		ID:   "proj-1",
		Name: "Test Project",
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 创建 SRC 程序
	program := &models.SRCProgram{
		ID:        "prog-1",
		ProjectID: "proj-1",
		Name:      "Test SRC",
		Platform:  "hackerone",
		MaxRPS:    5,
		PreferredVulnTypes: []string{},
		PayoutHint: map[string]any{},
	}

	if err := q.CreateSRCProgram(program); err != nil {
		t.Fatalf("create src program: %v", err)
	}

	// 更新程序
	program.Name = "Updated SRC"
	program.MaxRPS = 20
	program.AllowAutomation = true

	if err := q.UpdateSRCProgram(program); err != nil {
		t.Fatalf("update src program: %v", err)
	}

	// 获取更新后的程序
	got, err := q.GetSRCProgram("proj-1")
	if err != nil {
		t.Fatalf("get src program: %v", err)
	}
	if got == nil {
		t.Fatal("expected program, got nil")
	}

	if got.Name != "Updated SRC" {
		t.Errorf("expected Name 'Updated SRC', got '%s'", got.Name)
	}
	if got.MaxRPS != 20 {
		t.Errorf("expected MaxRPS 20, got %d", got.MaxRPS)
	}
	if !got.AllowAutomation {
		t.Error("expected AllowAutomation true")
	}
}

func TestDeleteSRCProgram(t *testing.T) {
	q := New(openTestDB(t))

	// 创建项目
	project := &models.Project{
		ID:   "proj-1",
		Name: "Test Project",
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 创建 SRC 程序
	program := &models.SRCProgram{
		ID:        "prog-1",
		ProjectID: "proj-1",
		Name:      "Test SRC",
		Platform:  "hackerone",
		PreferredVulnTypes: []string{},
		PayoutHint: map[string]any{},
	}

	if err := q.CreateSRCProgram(program); err != nil {
		t.Fatalf("create src program: %v", err)
	}

	// 删除程序
	if err := q.DeleteSRCProgram("proj-1"); err != nil {
		t.Fatalf("delete src program: %v", err)
	}

	// 验证已删除
	got, err := q.GetSRCProgram("proj-1")
	if err != nil {
		t.Fatalf("get src program: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestListSRCPrograms(t *testing.T) {
	q := New(openTestDB(t))

	// 创建项目
	for i := 1; i <= 3; i++ {
		project := &models.Project{
			ID:   "proj-" + string(rune('0'+i)),
			Name: "Test Project " + string(rune('0'+i)),
		}
		if err := q.CreateProject(project); err != nil {
			t.Fatalf("create project: %v", err)
		}
	}

	// 创建 SRC 程序
	for i := 1; i <= 3; i++ {
		program := &models.SRCProgram{
			ID:        "prog-" + string(rune('0'+i)),
			ProjectID: "proj-" + string(rune('0'+i)),
			Name:      "Test SRC " + string(rune('0'+i)),
			Platform:  "hackerone",
			PreferredVulnTypes: []string{},
			PayoutHint: map[string]any{},
		}
		if err := q.CreateSRCProgram(program); err != nil {
			t.Fatalf("create src program: %v", err)
		}
	}

	// 列出所有程序
	programs, err := q.ListSRCPrograms()
	if err != nil {
		t.Fatalf("list src programs: %v", err)
	}
	if len(programs) != 3 {
		t.Errorf("expected 3 programs, got %d", len(programs))
	}
}

func TestListSRCProgramsByPlatform(t *testing.T) {
	q := New(openTestDB(t))

	// 创建项目
	for i := 1; i <= 3; i++ {
		project := &models.Project{
			ID:   "proj-" + string(rune('0'+i)),
			Name: "Test Project " + string(rune('0'+i)),
		}
		if err := q.CreateProject(project); err != nil {
			t.Fatalf("create project: %v", err)
		}
	}

	// 创建不同平台的 SRC 程序
	platforms := []string{"hackerone", "bugcrowd", "hackerone"}
	for i, platform := range platforms {
		program := &models.SRCProgram{
			ID:        "prog-" + string(rune('0'+i+1)),
			ProjectID: "proj-" + string(rune('0'+i+1)),
			Name:      "Test SRC " + string(rune('0'+i+1)),
			Platform:  platform,
			PreferredVulnTypes: []string{},
			PayoutHint: map[string]any{},
		}
		if err := q.CreateSRCProgram(program); err != nil {
			t.Fatalf("create src program: %v", err)
		}
	}

	// 按平台筛选
	programs, err := q.ListSRCProgramsByPlatform("hackerone")
	if err != nil {
		t.Fatalf("list src programs by platform: %v", err)
	}
	if len(programs) != 2 {
		t.Errorf("expected 2 hackerone programs, got %d", len(programs))
	}
}

func TestSRCProgramDuplicateProject(t *testing.T) {
	q := New(openTestDB(t))

	// 创建项目
	project := &models.Project{
		ID:   "proj-1",
		Name: "Test Project",
	}
	if err := q.CreateProject(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 创建第一个 SRC 程序
	program1 := &models.SRCProgram{
		ID:        "prog-1",
		ProjectID: "proj-1",
		Name:      "First SRC",
		Platform:  "hackerone",
		PreferredVulnTypes: []string{},
		PayoutHint: map[string]any{},
	}
	if err := q.CreateSRCProgram(program1); err != nil {
		t.Fatalf("create first src program: %v", err)
	}

	// 尝试创建第二个（应该失败）
	program2 := &models.SRCProgram{
		ID:        "prog-2",
		ProjectID: "proj-1",
		Name:      "Second SRC",
		Platform:  "bugcrowd",
		PreferredVulnTypes: []string{},
		PayoutHint: map[string]any{},
	}
	err := q.CreateSRCProgram(program2)
	if err == nil {
		t.Error("expected error for duplicate project")
	}
}

package main

import (
	"os"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/report"
)

func main() {
	data := &report.ReportData{
		Project: &models.Project{ID: "test", Name: "测试项目", Organization: "测试组织"},
		Targets: []*models.Target{{Type: "domain", Value: "example.com"}},
		Findings: []*report.ReportFinding{{
			Finding: &models.Finding{ID: "f1", Title: "SQL 注入", Severity: models.SeverityHigh, Status: models.FindingConfirmed, Summary: "在 /api/search 发现 SQL 注入"},
			EvidenceList: []*models.Evidence{
				{Type: models.EvidenceRequest, Excerpt: "GET /api/search?q=1' OR '1'='1"},
				{Type: models.EvidenceResponse, Excerpt: "HTTP/1.1 200 OK\n..."},
			},
		}},
	}
	html, _ := report.GenerateHTML(data)
	os.WriteFile("/tmp/report_test.html", []byte(html), 0644)
}

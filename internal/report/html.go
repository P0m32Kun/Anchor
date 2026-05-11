package report

import (
	"html/template"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// GenerateHTML produces an HTML security assessment report.
// Screen uses a dark theme; @media print switches to light for PDF export.
func GenerateHTML(data *ReportData) (string, error) {
	if data == nil || data.Project == nil {
		return "", nil
	}

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"severityColor": severityColor,
		"severityLabel": severityLabel,
		"statusLabel":   statusLabelHTML,
		"formatTime":    func(t time.Time) string { return t.Format("2006-01-02 15:04") },
		"countSeverity": func(findings []*ReportFinding, sev models.FindingSeverity) int {
			n := 0
			for _, rf := range findings {
				if rf.Finding.Severity == sev {
					n++
				}
			}
			return n
		},
		"filterStatus": filterByStatus,
		"truncate": func(s string, max int) string {
			if len(s) <= max {
				return s
			}
			return s[:max] + "..."
		},
	}).Parse(htmlReportTmpl)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func severityColor(sev models.FindingSeverity) string {
	switch sev {
	case models.SeverityCritical:
		return "#ff4444"
	case models.SeverityHigh:
		return "#ff8800"
	case models.SeverityMedium:
		return "#ffbb00"
	case models.SeverityLow:
		return "#44bb44"
	default:
		return "#888888"
	}
}

func severityLabel(sev models.FindingSeverity) string {
	switch sev {
	case models.SeverityCritical:
		return "严重"
	case models.SeverityHigh:
		return "高危"
	case models.SeverityMedium:
		return "中危"
	case models.SeverityLow:
		return "低危"
	default:
		return "信息"
	}
}

const htmlReportTmpl = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Project.Name}} — 安全评估报告</title>
<style>
  :root {
    --bg: #0d1117; --bg2: #161b22; --bg3: #21262d;
    --fg: #e6edf3; --fg2: #8b949e; --border: #30363d;
    --accent: #00d4ff; --red: #ff4444; --orange: #ff8800;
    --yellow: #ffbb00; --green: #44bb44; --gray: #888888;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif;
    background: var(--bg); color: var(--fg); line-height: 1.6;
    max-width: 1100px; margin: 0 auto; padding: 24px;
  }
  h1 { font-size: 28px; margin-bottom: 8px; color: var(--accent); }
  h2 { font-size: 20px; margin: 32px 0 16px; color: var(--fg); border-bottom: 1px solid var(--border); padding-bottom: 8px; }
  h3 { font-size: 16px; margin: 16px 0 8px; color: var(--fg); }
  table { width: 100%; border-collapse: collapse; margin: 12px 0; }
  th, td { padding: 8px 12px; text-align: left; border: 1px solid var(--border); }
  th { background: var(--bg2); color: var(--fg2); font-weight: 600; }
  td { background: var(--bg); }
  .badge {
    display: inline-block; padding: 2px 8px; border-radius: 4px;
    font-size: 12px; font-weight: 600; color: #fff;
  }
  .finding-card {
    background: var(--bg2); border: 1px solid var(--border); border-radius: 8px;
    margin: 16px 0; padding: 16px;
  }
  .finding-header { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
  .finding-title { font-size: 16px; font-weight: 600; }
  .meta-row { display: flex; gap: 24px; flex-wrap: wrap; color: var(--fg2); font-size: 13px; margin: 4px 0; }
  .evidence-block {
    background: var(--bg); border: 1px solid var(--border); border-radius: 4px;
    margin: 8px 0; padding: 12px; font-family: monospace; font-size: 13px;
    white-space: pre-wrap; word-break: break-all; max-height: 300px; overflow-y: auto;
  }
  .summary-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; margin: 16px 0; }
  .summary-card {
    background: var(--bg2); border: 1px solid var(--border); border-radius: 8px;
    padding: 16px; text-align: center;
  }
  .summary-count { font-size: 32px; font-weight: 700; }
  .summary-label { font-size: 13px; color: var(--fg2); margin-top: 4px; }
  .footer { margin-top: 48px; padding-top: 16px; border-top: 1px solid var(--border); color: var(--fg2); font-size: 12px; }

  @media print {
    :root {
      --bg: #fff; --bg2: #f8f9fa; --bg3: #e9ecef;
      --fg: #212529; --fg2: #6c757d; --border: #dee2e6;
      --accent: #0066cc;
    }
    body { max-width: 100%; padding: 0; }
    .finding-card { break-inside: avoid; }
    .evidence-block { max-height: none; }
  }
</style>
</head>
<body>

<h1>{{.Project.Name}} — 安全评估报告</h1>
<p style="color: var(--fg2); margin-bottom: 24px;">
  生成时间：{{formatTime .GeneratedAt}}
  {{- if .Project.Organization}} | 组织：{{.Project.Organization}}{{end}}
</p>

<h2>漏洞概览</h2>
<div class="summary-grid">
  <div class="summary-card">
    <div class="summary-count" style="color: var(--red)">{{countSeverity .Findings "critical"}}</div>
    <div class="summary-label">严重</div>
  </div>
  <div class="summary-card">
    <div class="summary-count" style="color: var(--orange)">{{countSeverity .Findings "high"}}</div>
    <div class="summary-label">高危</div>
  </div>
  <div class="summary-card">
    <div class="summary-count" style="color: var(--yellow)">{{countSeverity .Findings "medium"}}</div>
    <div class="summary-label">中危</div>
  </div>
  <div class="summary-card">
    <div class="summary-count" style="color: var(--green)">{{countSeverity .Findings "low"}}</div>
    <div class="summary-label">低危</div>
  </div>
  <div class="summary-card">
    <div class="summary-count" style="color: var(--gray)">{{countSeverity .Findings "info"}}</div>
    <div class="summary-label">信息</div>
  </div>
</div>

<h2>扫描范围</h2>
<h3>目标</h3>
{{if .Targets}}
<table>
  <tr><th>类型</th><th>目标值</th></tr>
  {{range .Targets}}<tr><td>{{.Type}}</td><td>{{.Value}}</td></tr>{{end}}
</table>
{{else}}<p style="color: var(--fg2)">未定义目标</p>{{end}}

<h3>资产清单</h3>
{{if .Assets}}
<table>
  <tr><th>类型</th><th>值</th><th>来源</th></tr>
  {{range .Assets}}<tr><td>{{.Type}}</td><td>{{.Value}}</td><td>{{range .SourceTools}}{{.}} {{end}}</td></tr>{{end}}
</table>
{{else}}<p style="color: var(--fg2)">无资产数据</p>{{end}}

<h2>漏洞详情</h2>
{{$confirmed := filterStatus .Findings "confirmed"}}
{{if $confirmed}}
{{range $i, $rf := $confirmed}}
<div class="finding-card">
  <div class="finding-header">
    <span class="badge" style="background: {{severityColor $rf.Finding.Severity}}">
      {{severityLabel $rf.Finding.Severity}}
    </span>
    <span class="finding-title">{{$rf.Finding.Title}}</span>
  </div>
  <div class="meta-row">
    {{if $rf.Asset}}<span>资产：{{$rf.Asset.Value}}</span>{{end}}
    {{if $rf.WebEndpoint}}<span>端点：{{$rf.WebEndpoint.URL}}</span>{{end}}
    <span>来源：{{$rf.Finding.SourceTool}}</span>
    <span>置信度：{{$rf.Finding.Confidence}}%</span>
  </div>
  {{if $rf.Finding.Summary}}
  <h3>描述</h3>
  <p>{{$rf.Finding.Summary}}</p>
  {{end}}
  {{if $rf.EvidenceList}}
  <h3>证据</h3>
  {{range $rf.EvidenceList}}
  <div class="evidence-block"><strong>[{{.Type}}]</strong> {{truncate .Excerpt 2000}}</div>
  {{end}}
  {{end}}
  {{if $rf.Finding.Remediation}}
  <h3>修复建议</h3>
  <p>{{$rf.Finding.Remediation}}</p>
  {{end}}
</div>
{{end}}
{{else}}<p style="color: var(--fg2)">无已确认漏洞</p>{{end}}

<h2>已接受风险</h2>
{{$accepted := filterStatus .Findings "accepted_risk"}}
{{if $accepted}}
{{range $accepted}}
<div class="finding-card">
  <div class="finding-header">
    <span class="badge" style="background: {{severityColor .Finding.Severity}}">
      {{severityLabel .Finding.Severity}}
    </span>
    <span class="finding-title">{{.Finding.Title}}</span>
  </div>
  {{if .Finding.Summary}}<p>{{.Finding.Summary}}</p>{{end}}
</div>
{{end}}
{{else}}<p style="color: var(--fg2)">无已接受风险</p>{{end}}

<h2>工具版本</h2>
{{if .ToolVersions}}
<table>
  <tr><th>工具</th><th>版本</th><th>路径</th></tr>
  {{range .ToolVersions}}<tr><td>{{.Tool}}</td><td>{{.Version}}</td><td>{{.BinaryPath}}</td></tr>{{end}}
</table>
{{else}}<p style="color: var(--fg2)">无工具记录</p>{{end}}

<div class="footer">
  <p>Anchor 安全评估平台 | 生成于 {{formatTime .GeneratedAt}} | 项目 ID: {{.Project.ID}}</p>
</div>

</body>
</html>`

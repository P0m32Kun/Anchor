package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// escapeMDTable escapes special Markdown characters in table cell values.
func escapeMDTable(v string) string {
	v = strings.ReplaceAll(v, "|", "\\|")
	v = strings.ReplaceAll(v, "\n", " ")
	return v
}

// GenerateMarkdown produces a security assessment report in Markdown format.
// All edge cases (nil data, empty slices) are handled gracefully.
func GenerateMarkdown(data *ReportData) string {
	// data.Project is guaranteed non-nil by Aggregate() — it uses the project
	// returned by GetProject(), which is validated before Aggregate() is called.
	// But we guard against direct calls with incomplete data.
	if data == nil || data.Project == nil {
		return "Error: report data is incomplete (no project)."
	}

	var sb strings.Builder

	// --- 1. Title ---
	sb.WriteString("# Security Assessment Report\n\n")

	// --- 2. Executive Summary ---
	sb.WriteString("## Executive Summary\n\n")
	fmt.Fprintf(&sb, "| Field | Value |\n")
	fmt.Fprintf(&sb, "|---|---|\n")
	fmt.Fprintf(&sb, "| **Project** | %s |\n", escapeMDTable(data.Project.Name))
	if data.Project.Organization != "" {
		fmt.Fprintf(&sb, "| **Organization** | %s |\n", escapeMDTable(data.Project.Organization))
	}
	fmt.Fprintf(&sb, "| **Report Generated** | %s |\n", data.GeneratedAt.Format(time.RFC3339))
	sb.WriteString("\n")

	// Severity counts.
	severityCounts := severityCountMap(data.Findings)
	fmt.Fprintf(&sb, "### Finding Summary\n\n")
	fmt.Fprintf(&sb, "| Severity | Count |\n")
	fmt.Fprintf(&sb, "|---|---|\n")
	for _, sev := range []models.FindingSeverity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	} {
		count := severityCounts[sev]
		fmt.Fprintf(&sb, "| %s | %d |\n", sev, count)
	}
	sb.WriteString("\n")

	totalFindings := len(data.Findings)
	if totalFindings == 0 {
		sb.WriteString("> **No vulnerabilities confirmed.**\n\n")
	}

	// --- 3. Scope ---
	sb.WriteString("## Scope\n\n")
	sb.WriteString("### Targets\n\n")
	if len(data.Targets) > 0 {
		for _, t := range data.Targets {
			fmt.Fprintf(&sb, "- `%s` (%s)\n", t.Value, t.Type)
		}
	} else {
		sb.WriteString("*No targets defined.*\n")
	}
	sb.WriteString("\n")

	sb.WriteString("### Scope Rules\n\n")
	if len(data.ScopeRules) > 0 {
		for _, r := range data.ScopeRules {
			action := "Include"
			if r.Action == models.ScopeActionExclude {
				action = "Exclude"
			}
			fmt.Fprintf(&sb, "- [%s] `%s` (%s)", action, r.Value, r.Type)
			if r.Reason != "" {
				fmt.Fprintf(&sb, " — %s", r.Reason)
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("*No scope rules defined.*\n")
	}
	sb.WriteString("\n")

	// --- 4. Methodology ---
	sb.WriteString("## Methodology\n\n")
	if len(data.ToolVersions) > 0 {
		// Deduplicate by tool name; keep latest version info.
		toolMap := make(map[string]*models.ToolInvocation)
		for _, inv := range data.ToolVersions {
			toolMap[inv.Tool] = inv
		}
		toolNames := make([]string, 0, len(toolMap))
		for name := range toolMap {
			toolNames = append(toolNames, name)
		}
		sort.Strings(toolNames)

		sb.WriteString("| Tool | Version | Binary Path |\n")
		sb.WriteString("|---|---|---|\n")
		for _, name := range toolNames {
			inv := toolMap[name]
			version := inv.Version
			if version == "" {
				version = "N/A"
			}
			fmt.Fprintf(&sb, "| %s | %s | %s |\n", escapeMDTable(name), escapeMDTable(version), escapeMDTable(inv.BinaryPath))
		}
	} else {
		sb.WriteString("*No tool invocations recorded.*\n")
	}
	sb.WriteString("\n")

	// --- 5. Risk Statistics ---
	sb.WriteString("## Risk Statistics\n\n")
	confirmed := filterByStatus(data.Findings, models.FindingConfirmed)
	acceptedRisks := filterByStatus(data.Findings, models.FindingAcceptedRisk)

	confirmedBySev := severityCountMap(confirmed)

	fmt.Fprintf(&sb, "| Severity | Confirmed |\n")
	fmt.Fprintf(&sb, "|---|---|\n")
	for _, sev := range []models.FindingSeverity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	} {
		count := confirmedBySev[sev]
		fmt.Fprintf(&sb, "| %s | %d |\n", sev, count)
	}
	sb.WriteString("\n")

	// --- 6. Vulnerability Details ---
	sb.WriteString("## Vulnerability Details\n\n")
	if len(confirmed) == 0 {
		sb.WriteString("*No confirmed vulnerabilities.*\n\n")
	} else {
		for i, rf := range confirmed {
			writeFindingDetail(&sb, i+1, rf)
		}
	}

	// --- 7. Accepted Risks ---
	sb.WriteString("## Accepted Risks\n\n")
	if len(acceptedRisks) == 0 {
		sb.WriteString("*No accepted risks.*\n\n")
	} else {
		for i, rf := range acceptedRisks {
			writeFindingDetail(&sb, i+1, rf)
		}
	}

	// --- 8. Appendix ---
	sb.WriteString("## Appendix\n\n")
	sb.WriteString("### Tool Versions\n\n")
	if len(data.ToolVersions) > 0 {
		toolMap := make(map[string]*models.ToolInvocation)
		for _, inv := range data.ToolVersions {
			toolMap[inv.Tool] = inv
		}
		toolNames := make([]string, 0, len(toolMap))
		for name := range toolMap {
			toolNames = append(toolNames, name)
		}
		sort.Strings(toolNames)

		for _, name := range toolNames {
			inv := toolMap[name]
			version := inv.Version
			if version == "" {
				version = "version unknown"
			}
			fmt.Fprintf(&sb, "- **%s**: %s (`%s`)\n", name, version, inv.BinaryPath)
		}
	} else {
		sb.WriteString("*No tool invocations recorded.*\n")
	}
	sb.WriteString("\n")

	sb.WriteString("### Report Metadata\n\n")
	fmt.Fprintf(&sb, "- **Generated by**: anchor v0.1.0\n")
	fmt.Fprintf(&sb, "- **Generated at**: %s\n", data.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&sb, "- **Project ID**: %s\n", data.Project.ID)
	sb.WriteString("\n")

	return sb.String()
}

// writeFindingDetail writes a single finding entry in the Markdown report.
func writeFindingDetail(sb *strings.Builder, num int, rf *ReportFinding) {
	f := rf.Finding

	fmt.Fprintf(sb, "### %d. %s\n\n", num, escapeMDTable(f.Title))
	fmt.Fprintf(sb, "| Field | Value |\n")
	fmt.Fprintf(sb, "|---|---|\n")

	// Asset info.
	assetVal := "N/A"
	if rf.Asset != nil {
		assetVal = fmt.Sprintf("%s (%s)", escapeMDTable(rf.Asset.Value), escapeMDTable(string(rf.Asset.Type)))
	}
	fmt.Fprintf(sb, "| **Asset** | %s |\n", assetVal)

	// Web endpoint info.
	epVal := "N/A"
	if rf.WebEndpoint != nil {
		epVal = escapeMDTable(rf.WebEndpoint.URL)
	}
	fmt.Fprintf(sb, "| **Endpoint** | %s |\n", epVal)

	fmt.Fprintf(sb, "| **Severity** | %s |\n", escapeMDTable(string(f.Severity)))
	fmt.Fprintf(sb, "| **Confidence** | %d |\n", f.Confidence)
	fmt.Fprintf(sb, "| **Status** | %s |\n", escapeMDTable(string(f.Status)))
	fmt.Fprintf(sb, "| **Source Tool** | %s |\n", escapeMDTable(f.SourceTool))
	fmt.Fprintf(sb, "\n")

	// Summary.
	sb.WriteString("#### Description\n\n")
	if f.Summary != "" {
		sb.WriteString(f.Summary)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*No description provided.*\n\n")
	}

	// Evidence.
	sb.WriteString("#### Evidence\n\n")
	if len(rf.EvidenceList) > 0 {
		for _, ev := range rf.EvidenceList {
			etype := string(ev.Type)
			fmt.Fprintf(sb, "**%s** — %s\n", etype, ev.CreatedAt.Format("2006-01-02 15:04:05"))
			if ev.Excerpt != "" {
				// Truncate very long excerpts for readability.
				excerpt := ev.Excerpt
				if len(excerpt) > 2000 {
					excerpt = excerpt[:2000] + "\n... (truncated)"
				}
				sb.WriteString("```\n")
				sb.WriteString(excerpt)
				sb.WriteString("\n```\n\n")
			}
		}
	} else {
		sb.WriteString("*No evidence recorded.*\n\n")
	}

	// Remediation.
	sb.WriteString("#### Remediation\n\n")
	if f.Remediation != "" {
		sb.WriteString(f.Remediation)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("*No remediation guidance provided.*\n\n")
	}

	sb.WriteString("---\n\n")
}

// severityCountMap counts findings by severity.
func severityCountMap(findings []*ReportFinding) map[models.FindingSeverity]int {
	counts := make(map[models.FindingSeverity]int)
	for _, rf := range findings {
		counts[rf.Finding.Severity]++
	}
	return counts
}

// filterByStatus returns findings matching a given status.
func filterByStatus(findings []*ReportFinding, status models.FindingStatus) []*ReportFinding {
	var result []*ReportFinding
	for _, rf := range findings {
		if rf.Finding.Status == status {
			result = append(result, rf)
		}
	}
	return result
}

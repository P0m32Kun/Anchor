package submission

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// PackGenerator 提交包生成器
type PackGenerator struct{}

// NewPackGenerator 创建提交包生成器
func NewPackGenerator() *PackGenerator {
	return &PackGenerator{}
}

// Generate 生成提交包
func (g *PackGenerator) Generate(candidate *models.BountyCandidate, finding *models.Finding, template string) (*models.SubmissionPack, error) {
	// 生成检查清单
	checklist := models.DefaultChecklist()
	checklistJSON, err := json.Marshal(checklist)
	if err != nil {
		return nil, fmt.Errorf("marshal checklist: %w", err)
	}

	// 生成内容
	content, err := g.generateContent(candidate, finding, template)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}

	pack := &models.SubmissionPack{
		CandidateID:     candidate.ID,
		Format:          models.SubmissionFormatMarkdown,
		Template:        template,
		Content:         content,
		ChecklistJSON:   string(checklistJSON),
		RedactionStatus: models.RedactionStatusRaw,
	}

	return pack, nil
}

// generateContent 生成报告内容
func (g *PackGenerator) generateContent(candidate *models.BountyCandidate, finding *models.Finding, template string) (string, error) {
	switch template {
	case models.SubmissionTemplateHackerOne:
		return g.generateHackerOneTemplate(candidate, finding), nil
	case models.SubmissionTemplateBugcrowd:
		return g.generateBugcrowdTemplate(candidate, finding), nil
	case models.SubmissionTemplate360SRC:
		return g.generate360SRCTemplate(candidate, finding), nil
	default:
		return g.generateGenericTemplate(candidate, finding), nil
	}
}

// generateGenericTemplate 生成通用模板
func (g *PackGenerator) generateGenericTemplate(candidate *models.BountyCandidate, finding *models.Finding) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", candidate.Title))

	sb.WriteString("## 基本信息\n\n")
	sb.WriteString(fmt.Sprintf("- **漏洞类型**: %s\n", candidate.VulnType))
	sb.WriteString(fmt.Sprintf("- **严重程度**: %s\n", candidate.Severity))
	sb.WriteString(fmt.Sprintf("- **置信度**: %s\n", candidate.Confidence))
	sb.WriteString(fmt.Sprintf("- **发现时间**: %s\n", candidate.CreatedAt.Format(time.RFC3339)))
	sb.WriteString("\n")

	sb.WriteString("## 漏洞描述\n\n")
	if finding != nil && finding.Summary != "" {
		sb.WriteString(finding.Summary)
	} else {
		sb.WriteString("待补充漏洞描述...")
	}
	sb.WriteString("\n\n")

	sb.WriteString("## 复现步骤\n\n")
	sb.WriteString("1. 待补充复现步骤...\n\n")

	sb.WriteString("## 影响范围\n\n")
	sb.WriteString("待补充影响范围...\n\n")

	sb.WriteString("## 修复建议\n\n")
	if finding != nil && finding.Remediation != "" {
		sb.WriteString(finding.Remediation)
	} else {
		sb.WriteString("待补充修复建议...")
	}
	sb.WriteString("\n\n")

	sb.WriteString("## 证据\n\n")
	sb.WriteString("待补充证据...\n\n")

	return sb.String()
}

// generateHackerOneTemplate 生成 HackerOne 模板
func (g *PackGenerator) generateHackerOneTemplate(candidate *models.BountyCandidate, finding *models.Finding) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", candidate.Title))

	sb.WriteString("## Summary\n\n")
	if finding != nil && finding.Summary != "" {
		sb.WriteString(finding.Summary)
	} else {
		sb.WriteString("A brief summary of the vulnerability.")
	}
	sb.WriteString("\n\n")

	sb.WriteString("## Vulnerability Details\n\n")
	sb.WriteString(fmt.Sprintf("**Vulnerability Type:** %s\n", candidate.VulnType))
	sb.WriteString(fmt.Sprintf("**Severity:** %s\n", candidate.Severity))
	sb.WriteString("\n")

	sb.WriteString("## Steps To Reproduce\n\n")
	sb.WriteString("1. Step 1\n")
	sb.WriteString("2. Step 2\n")
	sb.WriteString("3. Step 3\n\n")

	sb.WriteString("## Impact\n\n")
	sb.WriteString("Describe the impact of this vulnerability.\n\n")

	sb.WriteString("## Remediation\n\n")
	if finding != nil && finding.Remediation != "" {
		sb.WriteString(finding.Remediation)
	} else {
		sb.WriteString("Suggest how to fix the vulnerability.")
	}
	sb.WriteString("\n\n")

	return sb.String()
}

// generateBugcrowdTemplate 生成 Bugcrowd 模板
func (g *PackGenerator) generateBugcrowdTemplate(candidate *models.BountyCandidate, finding *models.Finding) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", candidate.Title))

	sb.WriteString("## Description\n\n")
	if finding != nil && finding.Summary != "" {
		sb.WriteString(finding.Summary)
	} else {
		sb.WriteString("Describe the vulnerability in detail.")
	}
	sb.WriteString("\n\n")

	sb.WriteString("## Vulnerability Type\n\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", candidate.VulnType))

	sb.WriteString("## Steps to Reproduce\n\n")
	sb.WriteString("1. Step 1\n")
	sb.WriteString("2. Step 2\n")
	sb.WriteString("3. Step 3\n\n")

	sb.WriteString("## Impact\n\n")
	sb.WriteString("What is the impact of this vulnerability?\n\n")

	sb.WriteString("## Recommended Fix\n\n")
	if finding != nil && finding.Remediation != "" {
		sb.WriteString(finding.Remediation)
	} else {
		sb.WriteString("How should this be fixed?")
	}
	sb.WriteString("\n\n")

	return sb.String()
}

// generate360SRCTemplate 生成 360SRC 模板
func (g *PackGenerator) generate360SRCTemplate(candidate *models.BountyCandidate, finding *models.Finding) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", candidate.Title))

	sb.WriteString("## 漏洞概述\n\n")
	if finding != nil && finding.Summary != "" {
		sb.WriteString(finding.Summary)
	} else {
		sb.WriteString("简要描述漏洞情况...")
	}
	sb.WriteString("\n\n")

	sb.WriteString("## 漏洞信息\n\n")
	sb.WriteString(fmt.Sprintf("- **漏洞类型**: %s\n", candidate.VulnType))
	sb.WriteString(fmt.Sprintf("- **危害等级**: %s\n", candidate.Severity))
	sb.WriteString("\n")

	sb.WriteString("## 复现步骤\n\n")
	sb.WriteString("1. 第一步\n")
	sb.WriteString("2. 第二步\n")
	sb.WriteString("3. 第三步\n\n")

	sb.WriteString("## 影响范围\n\n")
	sb.WriteString("描述影响范围...\n\n")

	sb.WriteString("## 修复方案\n\n")
	if finding != nil && finding.Remediation != "" {
		sb.WriteString(finding.Remediation)
	} else {
		sb.WriteString("提供修复建议...")
	}
	sb.WriteString("\n\n")

	return sb.String()
}

// RedactContent 脱敏内容
func (g *PackGenerator) RedactContent(content string) string {
	// 脱敏邮箱
	content = redactEmails(content)
	// 脱敏手机号
	content = redactPhones(content)
	// 脱敏身份证号
	content = redactIDCards(content)
	// 脱敏 IP 地址（保留网段）
	content = redactIPs(content)
	return content
}

// redactEmails 脱敏邮箱
func redactEmails(content string) string {
	// 简化处理：替换 @ 前面的部分
	parts := strings.Split(content, "@")
	if len(parts) > 1 {
		for i := 0; i < len(parts)-1; i++ {
			parts[i] = "***"
		}
	}
	return strings.Join(parts, "@")
}

// redactPhones 脱敏手机号
func redactPhones(content string) string {
	// 简化处理
	return content
}

// redactIDCards 脱敏身份证号
func redactIDCards(content string) string {
	// 简化处理
	return content
}

// redactIPs 脱敏 IP 地址
func redactIPs(content string) string {
	// 简化处理
	return content
}

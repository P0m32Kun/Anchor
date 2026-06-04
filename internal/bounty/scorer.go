package bounty

import (
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// Scorer 赏金候选评分器
type Scorer struct {
	severityWeights   map[string]int
	confidenceWeights map[string]int
	typeWeights       map[string]int
}

// NewScorer 创建评分器
func NewScorer() *Scorer {
	return &Scorer{
		severityWeights: map[string]int{
			"critical": 40,
			"high":     30,
			"medium":   20,
			"low":      10,
			"info":     0,
		},
		confidenceWeights: map[string]int{
			"high":   20,
			"medium": 10,
			"low":    5,
		},
		typeWeights: map[string]int{
			"rce":              30,
			"sqli":             25,
			"ssrf":             20,
			"idor":             20,
			"file_read":        25,
			"file_upload":      20,
			"auth_bypass":      25,
			"secret_leak":      15,
			"info_disclosure":  10,
			"xss":              10,
			"csrf":             5,
			"open_redirect":    5,
			"default_password": 20,
			"weak_password":    15,
		},
	}
}

// Score 计算候选的分数
func (s *Scorer) Score(candidate *models.BountyCandidate) {
	// 基础分数 = 严重度权重 + 置信度权重
	base := s.severityWeights[candidate.Severity] + s.confidenceWeights[candidate.Confidence]

	// 影响分数 = 漏洞类型权重
	impact := s.typeWeights[strings.ToLower(candidate.VulnType)]

	// 新颖度分数 = 来源权重
	novelty := s.getSourceWeight(candidate.SourceKind)

	// 复现分数 = 证据权重
	repro := s.getEvidenceWeight(candidate)

	// 范围分数 = 目标权重
	scope := s.getTargetWeight(candidate)

	// 安全分数 = 安全方法权重
	safety := s.getSafetyWeight(candidate)

	// 重复惩罚
	duplicatePenalty := s.getDuplicatePenalty(candidate.DuplicateRisk)

	// 计算总分
	candidate.ValueScore = clamp(base+impact+novelty+repro+scope+safety-duplicatePenalty, 0, 100)
	candidate.ImpactScore = impact
	candidate.NoveltyScore = novelty
	candidate.ReproScore = repro
	candidate.ScopeScore = scope
	candidate.SafetyScore = safety
}

// getSourceWeight 获取来源权重
func (s *Scorer) getSourceWeight(sourceKind string) int {
	switch sourceKind {
	case models.SourceKindManual:
		return 20
	case models.SourceKindEndpoint:
		return 15
	case models.SourceKindFinding:
		return 10
	case models.SourceKindAsset:
		return 5
	default:
		return 0
	}
}

// getEvidenceWeight 获取证据权重
func (s *Scorer) getEvidenceWeight(candidate *models.BountyCandidate) int {
	// 如果有请求/响应证据，增加分数
	if candidate.SubmissionURL != "" {
		return 15
	}
	// 如果有 notes，说明有手动验证
	if candidate.Notes != "" {
		return 10
	}
	return 5
}

// getTargetWeight 获取目标权重
func (s *Scorer) getTargetWeight(candidate *models.BountyCandidate) int {
	// 根据漏洞类型判断目标价值
	switch strings.ToLower(candidate.VulnType) {
	case "rce", "sqli", "file_read", "auth_bypass":
		return 15
	case "ssrf", "idor", "file_upload":
		return 10
	case "secret_leak", "default_password":
		return 10
	default:
		return 5
	}
}

// getSafetyWeight 获取安全方法权重
func (s *Scorer) getSafetyWeight(candidate *models.BountyCandidate) int {
	// 根据漏洞类型判断是否安全
	switch strings.ToLower(candidate.VulnType) {
	case "xss", "csrf", "open_redirect", "info_disclosure":
		return 15
	case "ssrf", "idor":
		return 10
	case "rce", "sqli", "file_read":
		return 5
	default:
		return 10
	}
}

// getDuplicatePenalty 获取重复惩罚
func (s *Scorer) getDuplicatePenalty(risk string) int {
	switch risk {
	case models.DuplicateRiskHigh:
		return 30
	case models.DuplicateRiskMedium:
		return 15
	case models.DuplicateRiskLow:
		return 5
	default:
		return 10
	}
}

// clamp 限制数值范围
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// GenerateRankingReason 生成排名原因
func (s *Scorer) GenerateRankingReason(candidate *models.BountyCandidate) string {
	var reasons []string

	// 严重度
	if candidate.Severity == "critical" || candidate.Severity == "high" {
		reasons = append(reasons, "高严重度漏洞")
	}

	// 漏洞类型
	switch strings.ToLower(candidate.VulnType) {
	case "rce":
		reasons = append(reasons, "远程代码执行")
	case "sqli":
		reasons = append(reasons, "SQL注入")
	case "ssrf":
		reasons = append(reasons, "服务端请求伪造")
	case "idor":
		reasons = append(reasons, "不安全的直接对象引用")
	case "file_read":
		reasons = append(reasons, "文件读取")
	case "auth_bypass":
		reasons = append(reasons, "认证绕过")
	case "secret_leak":
		reasons = append(reasons, "敏感信息泄露")
	}

	// 来源
	if candidate.SourceKind == models.SourceKindManual {
		reasons = append(reasons, "手动发现")
	}

	// 重复风险
	if candidate.DuplicateRisk == models.DuplicateRiskLow {
		reasons = append(reasons, "低重复风险")
	}

	if len(reasons) == 0 {
		return "标准评估"
	}

	return strings.Join(reasons, "; ")
}

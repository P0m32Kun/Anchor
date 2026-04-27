package scoring

import (
	"github.com/P0m32Kun/Anchor/internal/parser"
)

// ScoringEngine computes confidence and priority for parser results.
type ScoringEngine struct{}

// NewScoringEngine creates a new ScoringEngine.
func NewScoringEngine() *ScoringEngine {
	return &ScoringEngine{}
}

// ConfidenceReason explains one confidence bonus.
type ConfidenceReason struct {
	Points  int    `json:"points"`
	Reason  string `json:"reason"`
}

// ScoreFinding computes confidence and priority for a Nuclei result.
// It also returns a list of human-readable confidence reasons.
func (e *ScoringEngine) ScoreFinding(result *parser.NucleiResult) (confidence, priority int, reasons []ConfidenceReason) {
	confidence = 50
	reasons = []ConfidenceReason{}

	addConfidence := func(points int, reason string) {
		confidence += points
		reasons = append(reasons, ConfidenceReason{Points: points, Reason: reason})
	}

	if result.MatcherName != "" {
		addConfidence(15, "明确 matcher 证据")
	}
	if result.Request != "" && result.Response != "" {
		addConfidence(15, "存在请求响应")
	}
	if len(result.ExtractedResults) > 0 {
		addConfidence(10, "命中具体版本/内容")
	}

	if confidence > 100 {
		confidence = 100
	}

	priority = 50

	switch result.Severity {
	case "critical":
		priority += 40
	case "high":
		priority += 30
	case "medium":
		priority += 20
	case "low":
		priority += 10
	}

	if confidence >= 80 {
		priority += 10
	}

	if priority > 100 {
		priority = 100
	}

	return
}

package scoring

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/parser"
)

func TestScoreFinding_BaseConfidence(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{
		Severity: "low",
	}

	conf, pri, reasons := e.ScoreFinding(result)

	if conf != 50 {
		t.Errorf("confidence = %d, want 50", conf)
	}
	if pri != 60 { // 50 base + 10 low
		t.Errorf("priority = %d, want 60", pri)
	}
	if len(reasons) != 0 {
		t.Errorf("reasons len = %d, want 0", len(reasons))
	}
}

func TestScoreFinding_MatcherBonus(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{
		Severity:    "medium",
		MatcherName: "status-matcher",
	}

	conf, _, reasons := e.ScoreFinding(result)

	if conf != 65 { // 50 + 15
		t.Errorf("confidence = %d, want 65", conf)
	}
	if len(reasons) != 1 || reasons[0].Points != 15 {
		t.Errorf("reasons = %v, want 1 reason with 15 points", reasons)
	}
}

func TestScoreFinding_RequestResponseBonus(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{
		Severity: "high",
		Request:  "GET / HTTP/1.1",
		Response: "HTTP/1.1 200 OK",
	}

	conf, _, reasons := e.ScoreFinding(result)

	if conf != 65 { // 50 + 15
		t.Errorf("confidence = %d, want 65", conf)
	}
	if len(reasons) != 1 || reasons[0].Points != 15 {
		t.Errorf("reasons = %v, want 1 reason with 15 points", reasons)
	}
}

func TestScoreFinding_ExtractedResultsBonus(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{
		Severity:         "critical",
		ExtractedResults: []string{"Apache/2.4.49"},
	}

	conf, _, reasons := e.ScoreFinding(result)

	if conf != 60 { // 50 + 10
		t.Errorf("confidence = %d, want 60", conf)
	}
	if len(reasons) != 1 || reasons[0].Points != 10 {
		t.Errorf("reasons = %v, want 1 reason with 10 points", reasons)
	}
}

func TestScoreFinding_AllBonuses(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{
		Severity:         "critical",
		MatcherName:      "status-matcher",
		Request:          "GET / HTTP/1.1",
		Response:         "HTTP/1.1 200 OK",
		ExtractedResults: []string{"v1.0"},
	}

	conf, pri, reasons := e.ScoreFinding(result)

	// 50 + 15 (matcher) + 15 (req/resp) + 10 (extracted) = 90
	if conf != 90 {
		t.Errorf("confidence = %d, want 90", conf)
	}
	// 50 + 40 (critical) + 10 (conf >= 80) = 100
	if pri != 100 {
		t.Errorf("priority = %d, want 100", pri)
	}
	if len(reasons) != 3 {
		t.Errorf("reasons len = %d, want 3", len(reasons))
	}
}

func TestScoreFinding_ConfidenceClamped100(t *testing.T) {
	e := NewScoringEngine()
	// All bonuses: 50 + 15 + 15 + 10 = 90, not over 100
	// But verify the clamp path works
	result := &parser.NucleiResult{
		Severity:         "medium",
		MatcherName:      "m",
		Request:          "r",
		Response:         "s",
		ExtractedResults: []string{"e"},
	}

	conf, _, _ := e.ScoreFinding(result)
	if conf > 100 {
		t.Errorf("confidence = %d, should be clamped to <= 100", conf)
	}
	if conf != 90 {
		t.Errorf("confidence = %d, want 90", conf)
	}
}

func TestScoreFinding_PriorityClamped100(t *testing.T) {
	e := NewScoringEngine()
	// critical: 50 + 40 = 90, + 10 (conf >= 80) = 100
	// Even with high confidence, should not exceed 100
	result := &parser.NucleiResult{
		Severity:         "critical",
		MatcherName:      "m",
		Request:          "r",
		Response:         "s",
		ExtractedResults: []string{"e"},
	}

	_, pri, _ := e.ScoreFinding(result)
	if pri > 100 {
		t.Errorf("priority = %d, should be clamped to <= 100", pri)
	}
}

func TestScoreFinding_SeverityBonusCritical(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{Severity: "critical"}

	_, pri, _ := e.ScoreFinding(result)
	if pri != 90 { // 50 + 40
		t.Errorf("priority = %d, want 90", pri)
	}
}

func TestScoreFinding_SeverityBonusHigh(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{Severity: "high"}

	_, pri, _ := e.ScoreFinding(result)
	if pri != 80 { // 50 + 30
		t.Errorf("priority = %d, want 80", pri)
	}
}

func TestScoreFinding_SeverityBonusMedium(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{Severity: "medium"}

	_, pri, _ := e.ScoreFinding(result)
	if pri != 70 { // 50 + 20
		t.Errorf("priority = %d, want 70", pri)
	}
}

func TestScoreFinding_SeverityBonusUnknown(t *testing.T) {
	e := NewScoringEngine()
	result := &parser.NucleiResult{Severity: "info"}

	_, pri, _ := e.ScoreFinding(result)
	if pri != 50 { // 50 + 0
		t.Errorf("priority = %d, want 50", pri)
	}
}

func TestScoreFinding_HighConfidencePriorityBonus(t *testing.T) {
	e := NewScoringEngine()
	// Matcher + req/resp = 50 + 15 + 15 = 80, triggers conf >= 80 bonus
	result := &parser.NucleiResult{
		Severity:    "low",
		MatcherName: "m",
		Request:     "r",
		Response:    "s",
	}

	conf, pri, _ := e.ScoreFinding(result)
	if conf != 80 {
		t.Errorf("confidence = %d, want 80", conf)
	}
	// 50 + 10 (low) + 10 (conf >= 80) = 70
	if pri != 70 {
		t.Errorf("priority = %d, want 70", pri)
	}
}

func TestScoreFinding_LowConfidenceNoPriorityBonus(t *testing.T) {
	e := NewScoringEngine()
	// No bonuses, confidence = 50 < 80
	result := &parser.NucleiResult{
		Severity: "high",
	}

	conf, pri, _ := e.ScoreFinding(result)
	if conf != 50 {
		t.Errorf("confidence = %d, want 50", conf)
	}
	// 50 + 30 (high), no conf bonus
	if pri != 80 {
		t.Errorf("priority = %d, want 80", pri)
	}
}

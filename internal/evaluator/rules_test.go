package evaluator

import (
	"testing"
	"time"
)

func TestRuleEngine_ToolReliabilityLow(t *testing.T) {
	engine := NewRuleEngine()

	// Test case: tool with low success rate
	metrics := &ScanMetrics{
		ToolStats: map[string]*ToolStat{
			"subfinder": {
				ToolName:    "subfinder",
				TotalCalls:  10,
				SuccessRate: 0.6, // 60% success rate
			},
		},
	}

	issues := engine.Evaluate(metrics)

	found := false
	for _, issue := range issues {
		if issue.RuleID == "tool_reliability_low" {
			found = true
			if issue.Severity != "high" {
				t.Errorf("Expected severity 'high', got '%s'", issue.Severity)
			}
		}
	}

	if !found {
		t.Error("Expected tool_reliability_low issue to be triggered")
	}
}

func TestRuleEngine_ToolReliabilityHigh(t *testing.T) {
	engine := NewRuleEngine()

	// Test case: tool with high success rate (should not trigger)
	metrics := &ScanMetrics{
		ToolStats: map[string]*ToolStat{
			"subfinder": {
				ToolName:    "subfinder",
				TotalCalls:  10,
				SuccessRate: 0.9, // 90% success rate
			},
		},
	}

	issues := engine.Evaluate(metrics)

	for _, issue := range issues {
		if issue.RuleID == "tool_reliability_low" {
			t.Error("tool_reliability_low should not be triggered for 90% success rate")
		}
	}
}

func TestRuleEngine_StageBottleneck(t *testing.T) {
	engine := NewRuleEngine()

	// Test case: stage taking > 50% of total duration
	metrics := &ScanMetrics{
		TotalDuration: 10 * time.Minute,
		StageDurations: map[string]time.Duration{
			"nuclei":   6 * time.Minute, // 60%
			"portscan": 2 * time.Minute,
			"httpx":    2 * time.Minute,
		},
	}

	issues := engine.Evaluate(metrics)

	found := false
	for _, issue := range issues {
		if issue.RuleID == "stage_bottleneck" {
			found = true
			if issue.Severity != "high" {
				t.Errorf("Expected severity 'high', got '%s'", issue.Severity)
			}
		}
	}

	if !found {
		t.Error("Expected stage_bottleneck issue to be triggered")
	}
}

func TestRuleEngine_FindingConfidenceLow(t *testing.T) {
	engine := NewRuleEngine()

	// Test case: low average confidence
	metrics := &ScanMetrics{
		TotalFindings: 10,
		AvgConfidence: 45, // Below 60
	}

	issues := engine.Evaluate(metrics)

	found := false
	for _, issue := range issues {
		if issue.RuleID == "finding_confidence_low" {
			found = true
			if issue.Severity != "medium" {
				t.Errorf("Expected severity 'medium', got '%s'", issue.Severity)
			}
		}
	}

	if !found {
		t.Error("Expected finding_confidence_low issue to be triggered")
	}
}

func TestRuleEngine_NoIssues(t *testing.T) {
	engine := NewRuleEngine()

	// Test case: all metrics are good
	metrics := &ScanMetrics{
		ToolStats: map[string]*ToolStat{
			"subfinder": {
				ToolName:    "subfinder",
				TotalCalls:  10,
				SuccessRate: 0.95,
				AvgDuration: 30 * time.Second,
			},
		},
		TotalDuration: 10 * time.Minute,
		StageDurations: map[string]time.Duration{
			"nuclei":   3 * time.Minute,
			"portscan": 3 * time.Minute,
			"httpx":    4 * time.Minute,
		},
		StageStatuses: map[string]string{
			"nuclei":   "completed",
			"portscan": "completed",
			"httpx":    "completed",
		},
		TotalFindings:    10,
		AvgConfidence:    85,
		UnlinkedFindings: 1,
	}

	issues := engine.Evaluate(metrics)

	if len(issues) > 0 {
		t.Errorf("Expected no issues, got %d: %v", len(issues), issues)
	}
}

func TestCalculateTrend(t *testing.T) {
	tests := []struct {
		name      string
		values    []float64
		threshold float64
		expected  TrendDirection
	}{
		{
			name:      "upward trend",
			values:    []float64{10, 20, 30, 40, 50},
			threshold: 0.1,
			expected:  TrendUp,
		},
		{
			name:      "downward trend",
			values:    []float64{50, 40, 30, 20, 10},
			threshold: 0.1,
			expected:  TrendDown,
		},
		{
			name:      "stable trend",
			values:    []float64{30, 30, 30, 30, 30},
			threshold: 0.1,
			expected:  TrendStable,
		},
		{
			name:      "single value",
			values:    []float64{30},
			threshold: 0.1,
			expected:  TrendStable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTrend(tt.values, tt.threshold)
			if result != tt.expected {
				t.Errorf("CalculateTrend() = %v, want %v", result, tt.expected)
			}
		})
	}
}

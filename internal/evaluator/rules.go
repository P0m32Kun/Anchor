package evaluator

// Rule defines an evaluation rule.
type Rule struct {
	ID          string
	Category    string
	Name        string
	Description string
	Condition   func(metrics *ScanMetrics) bool
	Severity    string // high/medium/low
	Suggestion  func(metrics *ScanMetrics) string
}

// RuleEngine evaluates rules against metrics.
type RuleEngine struct {
	rules []Rule
}

// NewRuleEngine creates a new rule engine with predefined rules.
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: DefaultRules(),
	}
}

// Evaluate runs all rules against the metrics and returns triggered issues.
func (e *RuleEngine) Evaluate(metrics *ScanMetrics) []Issue {
	var issues []Issue

	for _, rule := range e.rules {
		if rule.Condition(metrics) {
			issues = append(issues, Issue{
				RuleID:      rule.ID,
				Category:    rule.Category,
				Severity:    rule.Severity,
				Description: rule.Description,
				Suggestion:  rule.Suggestion(metrics),
			})
		}
	}

	return issues
}

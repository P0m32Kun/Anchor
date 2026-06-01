package evaluator

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
)

// Evaluator is the main entry point for scan quality evaluation.
type Evaluator struct {
	queries   *db.Queries
	dataDir   string
	projectID string
	runID     string

	// Sub-evaluators
	toolEval       *ToolEffectivenessEvaluator
	templateEval   *TemplateEffectivenessEvaluator
	efficiencyEval *EfficiencyEvaluator
	findingEval    *FindingQualityEvaluator
	ruleEngine     *RuleEngine
	trendAnalyzer  *TrendAnalyzer
	reportGen      *ReportGenerator
}

// NewEvaluator creates a new evaluator instance.
func NewEvaluator(queries *db.Queries, dataDir, projectID, runID string) *Evaluator {
	return &Evaluator{
		queries:        queries,
		dataDir:        dataDir,
		projectID:      projectID,
		runID:          runID,
		toolEval:       NewToolEffectivenessEvaluator(queries),
		templateEval:   NewTemplateEffectivenessEvaluator(queries),
		efficiencyEval: NewEfficiencyEvaluator(queries),
		findingEval:    NewFindingQualityEvaluator(queries),
		ruleEngine:     NewRuleEngine(),
		trendAnalyzer:  NewTrendAnalyzer(queries),
		reportGen:      NewReportGenerator(),
	}
}

// Evaluate executes the full evaluation pipeline.
func (e *Evaluator) Evaluate(ctx context.Context) (*EvaluationReport, error) {
	// 1. Collect metrics
	metrics, err := e.collectMetrics()
	if err != nil {
		return nil, err
	}

	// 2. Run rule engine
	issues := e.ruleEngine.Evaluate(metrics)

	// 3. Analyze trends
	history, err := e.trendAnalyzer.QueryHistory(ctx, e.projectID, 10)
	if err != nil {
		// Non-fatal: continue without trends
		history = []*TrendData{}
	}
	trends := e.trendAnalyzer.Analyze(history, metrics)

	// 4. Generate report
	content := e.reportGen.Generate(metrics, issues, trends)

	// 5. Save report
	reportPath := e.GetReportPath()
	if err := e.saveReport(reportPath, content); err != nil {
		return nil, err
	}

	return &EvaluationReport{
		Path:        reportPath,
		GeneratedAt: time.Now(),
		Metrics:     metrics,
		Issues:      issues,
		Trends:      trends,
		Content:     content,
	}, nil
}

// GetReportPath returns the path where the evaluation report should be saved.
func (e *Evaluator) GetReportPath() string {
	return filepath.Join(e.dataDir, "projects", e.projectID, "reports",
		e.runID+"_evaluation.md")
}

// collectMetrics collects all metrics from the database.
func (e *Evaluator) collectMetrics() (*ScanMetrics, error) {
	metrics := &ScanMetrics{
		RunID:              e.runID,
		ProjectID:          e.projectID,
		FindingsBySeverity: make(map[string]int),
		FindingsByStatus:   make(map[string]int),
		StageDurations:     make(map[string]time.Duration),
		StageStatuses:      make(map[string]string),
	}

	// Get run info
	run, err := e.queries.GetPipelineRun(e.runID)
	if err != nil {
		return nil, err
	}
	if run != nil {
		metrics.StartedAt = run.StartedAt
		if run.CompletedAt != nil {
			metrics.CompletedAt = *run.CompletedAt
			metrics.TotalDuration = run.CompletedAt.Sub(run.StartedAt)
		}
	}

	// Collect tool effectiveness
	toolStats, err := e.toolEval.Evaluate(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.ToolStats = toolStats

	// Collect template effectiveness
	templateStats, err := e.templateEval.EvaluateTemplateStats(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.TemplateStats = templateStats

	// Collect dictionary stats
	dictStats, err := e.templateEval.EvaluateDictionaryStats(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.DictionaryStats = dictStats

	// Collect efficiency
	efficiency, err := e.efficiencyEval.Evaluate(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.TotalDuration = efficiency.TotalDuration
	metrics.StageDurations = efficiency.StageDurations
	metrics.StageStatuses = efficiency.StageStatuses

	// Collect finding quality
	findingQuality, err := e.findingEval.Evaluate(e.runID)
	if err != nil {
		return nil, err
	}
	metrics.FindingsBySeverity = findingQuality.FindingsBySeverity
	metrics.FindingsByStatus = findingQuality.FindingsByStatus
	metrics.AvgConfidence = findingQuality.AvgConfidence
	metrics.UnlinkedFindings = findingQuality.UnlinkedFindings
	metrics.TotalFindings = findingQuality.TotalFindings

	return metrics, nil
}

// saveReport saves the report content to a file.
func (e *Evaluator) saveReport(path, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

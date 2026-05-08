package models

import "time"

// --- Dashboard ---

type DashboardStats struct {
	TotalProjects   int                    `json:"total_projects"`
	ActiveRuns      int                    `json:"active_runs"`
	PendingFindings int                    `json:"pending_findings"`
	OnlineWorkers   int                    `json:"online_workers"`
	RecentRuns      []*DashboardRunItem    `json:"recent_runs"`
	RecentFindings  []*DashboardFindingItem `json:"recent_findings"`
}

type DashboardRunItem struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	ProjectName string     `json:"project_name"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type DashboardFindingItem struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	CreatedAt   time.Time `json:"created_at"`
}

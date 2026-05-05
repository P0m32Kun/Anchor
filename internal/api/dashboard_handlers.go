package api

import (
	"net/http"
	"sync"

	"github.com/P0m32Kun/Anchor/internal/errors"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func (s *Server) handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	var stats models.DashboardStats
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	run := func(fn func() error) {
		defer wg.Done()
		if err := fn(); err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}
	}

	wg.Add(6)
	go run(func() error {
		n, err := s.queries.CountProjects()
		stats.TotalProjects = n
		return err
	})
	go run(func() error {
		n, err := s.queries.CountActiveRuns()
		stats.ActiveRuns = n
		return err
	})
	go run(func() error {
		n, err := s.queries.CountPendingFindings()
		stats.PendingFindings = n
		return err
	})
	go run(func() error {
		n, err := s.queries.CountOnlineWorkers()
		stats.OnlineWorkers = n
		return err
	})
	go run(func() error {
		runs, err := s.queries.ListRecentRuns(5)
		stats.RecentRuns = runs
		return err
	})
	go run(func() error {
		findings, err := s.queries.ListRecentFindingsByStatus(models.FindingPendingReview, 5)
		stats.RecentFindings = findings
		return err
	})

	wg.Wait()
	if firstErr != nil {
		writeError(w, http.StatusInternalServerError, errors.Newf(errors.ErrInternal, "dashboard stats: %v", firstErr))
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

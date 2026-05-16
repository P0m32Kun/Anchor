package api

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
)

const (
	// workdirRetentionDays is how long completed task workdirs and host-list
	// files are kept on disk before being cleaned up by the periodic scanner.
	workdirRetentionDays = 30

	// workdirCleanupInterval is how often the periodic scanner runs.
	workdirCleanupInterval = 24 * time.Hour
)

// cleanupProjectWorkdir removes the entire workdir tree for a project.
// Called synchronously when a project is deleted via the API.
func (s *Server) cleanupProjectWorkdir(projectID string) {
	dir := filepath.Join(s.dataDir, "workdirs", projectID)
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("[server] cleanup workdir for project %s: %v", projectID, err)
	} else {
		log.Printf("[server] cleaned up workdir for deleted project %s", projectID)
	}
}

// startWorkdirCleanup runs a periodic background scanner that removes stale
// workdir files: orphaned project directories, terminal task workdirs older
// than the retention window, and host-list files past their mtime cutoff.
func (s *Server) startWorkdirCleanup() {
	// Run once at startup to catch leftovers from before this feature existed.
	go s.cleanupStaleWorkdirs()

	ticker := time.NewTicker(workdirCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupStaleWorkdirs()
	}
}

func (s *Server) cleanupStaleWorkdirs() {
	workdirsRoot := filepath.Join(s.dataDir, "workdirs")
	cutoff := time.Now().Add(-workdirRetentionDays * 24 * time.Hour)

	entries, err := os.ReadDir(workdirsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("[server] workdir cleanup: read root %s: %v", workdirsRoot, err)
		return
	}

	var removedDirs, removedFiles int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectID := entry.Name()
		projectDir := filepath.Join(workdirsRoot, projectID)

		// Check whether the project still exists in the database.
		project, err := s.queries.GetProject(projectID)
		if err != nil {
			log.Printf("[server] workdir cleanup: get project %s: %v", projectID, err)
			continue
		}

		if project == nil {
			// Orphan: project was deleted from DB but workdir never cleaned.
			if err := os.RemoveAll(projectDir); err != nil {
				log.Printf("[server] workdir cleanup: remove orphan %s: %v", projectID, err)
			} else {
				removedDirs++
				log.Printf("[server] workdir cleanup: removed orphan workdir %s", projectID)
			}
			continue
		}

		// Project still exists — scan its contents for stale task workdirs
		// and host-list files.
		taskEntries, err := os.ReadDir(projectDir)
		if err != nil {
			log.Printf("[server] workdir cleanup: read project dir %s: %v", projectID, err)
			continue
		}

		for _, te := range taskEntries {
			name := te.Name()
			fullPath := filepath.Join(projectDir, name)

			if te.IsDir() {
				// Task workdir: {projectID}/{taskID}/
				task, err := s.queries.GetScanTask(name)
				if err != nil {
					// DB error — skip this entry to be safe.
					continue
				}
				if task == nil {
					// Task record gone from DB, safe to remove.
					if err := os.RemoveAll(fullPath); err != nil {
						log.Printf("[server] workdir cleanup: remove stale task dir %s: %v", fullPath, err)
					} else {
						removedDirs++
					}
					continue
				}
				// Task exists but is in a terminal state and old enough.
				if isTaskTerminal(task.Status) &&
					task.FinishedAt != nil &&
					task.FinishedAt.Before(cutoff) {
					if err := os.RemoveAll(fullPath); err != nil {
						log.Printf("[server] workdir cleanup: remove old task dir %s: %v", fullPath, err)
					} else {
						removedDirs++
					}
				}
			} else {
				// Host-list / temp files (e.g. nmap-xxx.txt, naabu-xxx.txt).
				// These have no DB record; clean based on mtime.
				info, err := te.Info()
				if err != nil {
					continue
				}
				if info.ModTime().Before(cutoff) {
					if err := os.Remove(fullPath); err != nil {
						log.Printf("[server] workdir cleanup: remove old file %s: %v", fullPath, err)
					} else {
						removedFiles++
					}
				}
			}
		}
	}

	if removedDirs > 0 || removedFiles > 0 {
		log.Printf("[server] workdir cleanup: removed %d directories, %d files", removedDirs, removedFiles)
	}
}

// isTaskTerminal reports whether a task status means the task will never
// produce more output and its workdir is safe to delete.
func isTaskTerminal(status models.TaskStatus) bool {
	switch status {
	case models.TaskCompleted,
		models.TaskFailed,
		models.TaskCancelled,
		models.TaskPartialSuccess,
		models.TaskScopeDenied:
		return true
	default:
		return false
	}
}

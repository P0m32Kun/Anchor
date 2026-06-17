package screenshot

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

const (
	screenshotsDir = "screenshots"
	thumbnailsDir  = "thumbnails"
)

// Manager handles screenshot capture, storage, and evidence association.
type Manager struct {
	queries *db.Queries
	dataDir string
}

// NewManager creates a new screenshot Manager.
func NewManager(queries *db.Queries, dataDir string) *Manager {
	return &Manager{
		queries: queries,
		dataDir: dataDir,
	}
}

// ProjectScreenshotDir returns the directory for storing screenshots of
// a specific project.
func (m *Manager) ProjectScreenshotDir(projectID string) string {
	return filepath.Join(m.dataDir, "projects", projectID, screenshotsDir)
}

// ProjectThumbnailDir returns the thumbnail directory for a project.
func (m *Manager) ProjectThumbnailDir(projectID string) string {
	return filepath.Join(m.dataDir, "projects", projectID, screenshotsDir, thumbnailsDir)
}

// CaptureAndStore captures a screenshot of the given URL and persists it.
func (m *Manager) CaptureAndStore(ctx context.Context, projectID string, url string, assetID, taskID *string, timeout time.Duration) (*models.Screenshot, error) {
	dir := m.ProjectScreenshotDir(projectID)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create screenshot dir: %w", err)
	}

	// Add protocol if missing
	captureURL := url
	if !hasProtocol(captureURL) {
		captureURL = "https://" + captureURL
	}

	timeoutVal := timeout
	if timeoutVal <= 0 {
		timeoutVal = 30 * time.Second
	}

	result, err := Capture(ctx, captureURL, dir, timeoutVal)
	if err != nil {
		return nil, fmt.Errorf("capture %s: %w", url, err)
	}

	// Generate thumbnail
	thumbDir := m.ProjectThumbnailDir(projectID)
	if err := os.MkdirAll(thumbDir, 0750); err != nil {
		log.Printf("[screenshot] create thumbnail dir: %v", err)
	}

	thumbFilename := "thumb_" + filepath.Base(result.Path)
	thumbPath := filepath.Join(thumbDir, thumbFilename)
	if err := CreateThumbnail(result.Path, thumbPath, 320, 240); err != nil {
		log.Printf("[screenshot] create thumbnail: %v", err)
		thumbPath = ""
	}

	s := &models.Screenshot{
		ID:            util.GenerateID(),
		ProjectID:     projectID,
		AssetID:       assetID,
		TaskID:        taskID,
		URL:           captureURL,
		OriginalPath:  result.Path,
		ThumbnailPath: thumbPath,
		Width:         result.Width,
		Height:        result.Height,
		TakenAt:       time.Now().UTC(),
	}

	if err := m.queries.CreateScreenshot(s); err != nil {
		// Clean up files on DB failure
		os.Remove(result.Path)
		if thumbPath != "" {
			os.Remove(thumbPath)
		}
		return nil, fmt.Errorf("persist screenshot: %w", err)
	}

	log.Printf("[screenshot] captured %s (%dx%d, %dB)", url, result.Width, result.Height, result.Size)
	return s, nil
}

// CaptureForEndpoint captures a screenshot for a web endpoint and links it
// via the screenshot_artifact_id field.
func (m *Manager) CaptureForEndpoint(ctx context.Context, endpoint *models.WebEndpoint, timeout time.Duration) (*models.Screenshot, error) {
	s, err := m.CaptureAndStore(ctx, endpoint.ProjectID, endpoint.URL, &endpoint.AssetID, nil, timeout)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.OriginalPath)
	if err != nil {
		return s, nil
	}
	sum := sha256.Sum256(data)
	artifact := &models.RawArtifact{
		ID:              util.GenerateID(),
		ProjectID:       endpoint.ProjectID,
		TaskID:          s.TaskID,
		Type:            models.ArtifactScreenshot,
		Path:            s.OriginalPath,
		SHA256:          fmt.Sprintf("%x", sum),
		Size:            int64(len(data)),
		RedactionStatus: "raw",
		CreatedAt:       time.Now().UTC(),
	}
	if err := m.queries.CreateRawArtifact(artifact); err != nil {
		log.Printf("[screenshot] create raw artifact for %s: %v", endpoint.URL, err)
		return s, nil
	}
	if err := m.queries.UpdateWebEndpointScreenshotArtifactID(endpoint.ID, artifact.ID); err != nil {
		log.Printf("[screenshot] update web endpoint %s screenshot: %v", endpoint.ID, err)
	}
	return s, nil
}

// AddScreenshotEvidence links a screenshot as evidence for a finding.
func (m *Manager) AddScreenshotEvidence(ctx context.Context, findingID string, screenshot *models.Screenshot) (*models.Evidence, error) {
	ev := &models.Evidence{
		ID:        util.GenerateID(),
		FindingID: findingID,
		Type:      models.EvidenceScreenshot,
		Excerpt:   fmt.Sprintf("截图: %s (%dx%d)", screenshot.URL, screenshot.Width, screenshot.Height),
		CreatedBy: "screenshot_bot",
		CreatedAt: time.Now().UTC(),
	}

	if err := m.queries.CreateEvidence(ev); err != nil {
		return nil, fmt.Errorf("create evidence: %w", err)
	}
	return ev, nil
}

// ListByProject returns all screenshots for a project.
func (m *Manager) ListByProject(projectID string) ([]*models.Screenshot, error) {
	return m.queries.ListScreenshotsByProject(projectID)
}

func hasProtocol(url string) bool {
	return len(url) >= 7 && (url[:7] == "http://" || url[:8] == "https://")
}

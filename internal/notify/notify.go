package notify

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

// Dispatcher sends notification payloads to configured channels.
type Dispatcher struct {
	queries    *db.Queries
	httpClient *http.Client
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(queries *db.Queries) *Dispatcher {
	return &Dispatcher{
		queries: queries,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DispatchSignal sends a single signal to all enabled notification channels for a project.
func (d *Dispatcher) DispatchSignal(projectID string, sig *models.Signal) {
	go d.dispatchSignal(projectID, sig)
}

func (d *Dispatcher) dispatchSignal(projectID string, sig *models.Signal) {
	channels, err := d.queries.ListEnabledNotificationChannelsByProject(projectID)
	if err != nil {
		log.Printf("[notify] list channels for project %s: %v", projectID, err)
		return
	}
	if len(channels) == 0 {
		return
	}

	payload := &models.NotificationPayload{
		Event:     "signal.new",
		ProjectID: projectID,
		Signal: &models.SignalNotification{
			ID:         sig.ID,
			SourceKind: sig.SourceKind,
			Title:      sig.Title,
			Severity:   sig.Severity,
			Score:      sig.Score,
			Status:     sig.Status,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[notify] marshal payload: %v", err)
		return
	}

	for _, ch := range channels {
		if ch.ChannelType != models.NotificationChannelTypeWebhook {
			continue
		}
		d.sendWebhook(ch, body)
	}
}

// DispatchScanSummary sends a summary notification after a watch scan completes.
func (d *Dispatcher) DispatchScanSummary(projectID, runID string, summary *models.ScanSummaryNotification) {
	go func() {
		channels, err := d.queries.ListEnabledNotificationChannelsByProject(projectID)
		if err != nil {
			log.Printf("[notify] list channels for project %s: %v", projectID, err)
			return
		}
		if len(channels) == 0 {
			return
		}

		payload := &models.NotificationPayload{
			Event:     "scan.completed",
			ProjectID: projectID,
			RunID:     runID,
			Summary:   summary,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[notify] marshal summary payload: %v", err)
			return
		}

		for _, ch := range channels {
			if ch.ChannelType != models.NotificationChannelTypeWebhook {
				continue
			}
			d.sendWebhook(ch, body)
		}
	}()
}

// DispatchCertExpiry sends a certificate expiry warning to all enabled channels.
func (d *Dispatcher) DispatchCertExpiry(projectID string, sig *models.Signal) {
	d.DispatchSignal(projectID, sig)
}

func (d *Dispatcher) sendWebhook(ch *models.NotificationChannel, body []byte) {
	req, err := http.NewRequest(http.MethodPost, ch.URL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[notify] create request for %s: %v", ch.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Anchor/1.0 Notification")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		log.Printf("[notify] send to %s (%s): %v", ch.ID, ch.URL, err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("[notify] channel %s (%s) returned %d", ch.ID, ch.URL, resp.StatusCode)
	}
}

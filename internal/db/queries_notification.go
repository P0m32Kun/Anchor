package db

import (
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// CreateNotificationChannel inserts a new notification channel.
func (q *Queries) CreateNotificationChannel(ch *models.NotificationChannel) error {
	if ch.ID == "" {
		ch.ID = util.GenerateID()
	}
	now := time.Now().UTC()
	if ch.CreatedAt.IsZero() {
		ch.CreatedAt = now
	}
	if ch.UpdatedAt.IsZero() {
		ch.UpdatedAt = now
	}
	_, err := q.db.Exec(`
		INSERT INTO notification_channels (id, project_id, name, channel_type, url, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ch.ID, ch.ProjectID, ch.Name, ch.ChannelType, ch.URL, boolToInt(ch.Enabled), ch.CreatedAt, ch.UpdatedAt,
	)
	return err
}

// ListNotificationChannelsByProject returns all notification channels for a project.
func (q *Queries) ListNotificationChannelsByProject(projectID string) ([]*models.NotificationChannel, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, name, channel_type, url, enabled, created_at, updated_at
		FROM notification_channels
		WHERE project_id = ?
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.NotificationChannel
	for rows.Next() {
		ch := &models.NotificationChannel{}
		var enabled int
		if err := rows.Scan(&ch.ID, &ch.ProjectID, &ch.Name, &ch.ChannelType, &ch.URL, &enabled,
			&ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		ch.Enabled = enabled != 0
		list = append(list, ch)
	}
	return list, rows.Err()
}

// UpdateNotificationChannel updates name, url, and enabled fields.
func (q *Queries) UpdateNotificationChannel(ch *models.NotificationChannel) error {
	_, err := q.db.Exec(`
		UPDATE notification_channels SET name = ?, url = ?, enabled = ?, updated_at = ?
		WHERE id = ? AND project_id = ?`,
		ch.Name, ch.URL, boolToInt(ch.Enabled), time.Now().UTC(), ch.ID, ch.ProjectID,
	)
	return err
}

// DeleteNotificationChannel removes a notification channel.
func (q *Queries) DeleteNotificationChannel(id, projectID string) error {
	_, err := q.db.Exec(`DELETE FROM notification_channels WHERE id = ? AND project_id = ?`, id, projectID)
	return err
}

// GetNotificationChannelByID returns a single notification channel.
func (q *Queries) GetNotificationChannelByID(id, projectID string) (*models.NotificationChannel, error) {
	row := q.db.QueryRow(`
		SELECT id, project_id, name, channel_type, url, enabled, created_at, updated_at
		FROM notification_channels
		WHERE id = ? AND project_id = ?`, id, projectID)
	ch := &models.NotificationChannel{}
	var enabled int
	if err := row.Scan(&ch.ID, &ch.ProjectID, &ch.Name, &ch.ChannelType, &ch.URL, &enabled,
		&ch.CreatedAt, &ch.UpdatedAt); err != nil {
		return nil, err
	}
	ch.Enabled = enabled != 0
	return ch, nil
}

// ListEnabledNotificationChannelsByProject returns only enabled channels for a project.
func (q *Queries) ListEnabledNotificationChannelsByProject(projectID string) ([]*models.NotificationChannel, error) {
	rows, err := q.db.Query(`
		SELECT id, project_id, name, channel_type, url, enabled, created_at, updated_at
		FROM notification_channels
		WHERE project_id = ? AND enabled = 1
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.NotificationChannel
	for rows.Next() {
		ch := &models.NotificationChannel{}
		var enabled int
		if err := rows.Scan(&ch.ID, &ch.ProjectID, &ch.Name, &ch.ChannelType, &ch.URL, &enabled,
			&ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		ch.Enabled = enabled != 0
		list = append(list, ch)
	}
	return list, rows.Err()
}

package store

import (
	"database/sql"
	"fmt"
	"small-rpg-adhd-monolith/internal/core"
	"time"
)

// GetNotificationSettings retrieves notification settings for a user
// Returns default settings if none exist for the user
func (s *Store) GetNotificationSettings(userID int64) (*core.NotificationSettings, error) {
	settings := &core.NotificationSettings{}

	err := s.DB.QueryRow(
		"SELECT user_id, reminder_delta_minutes, snooze_default_minutes, created_at, updated_at FROM notification_settings WHERE user_id = ?",
		userID,
	).Scan(&settings.UserID, &settings.ReminderDeltaMinutes, &settings.SnoozeDefaultMinutes, &settings.CreatedAt, &settings.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default settings if none exist
			return &core.NotificationSettings{
				UserID:               userID,
				ReminderDeltaMinutes: 60, // 1 hour default
				SnoozeDefaultMinutes: 15, // 15 minutes default
				CreatedAt:            time.Now(),
				UpdatedAt:            time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get notification settings: %w", err)
	}

	return settings, nil
}

// UpdateNotificationSettings updates or creates notification settings for a user
// TODO: Add UI for users to configure their notification preferences
func (s *Store) UpdateNotificationSettings(settings *core.NotificationSettings) error {
	query := `
		INSERT INTO notification_settings (user_id, reminder_delta_minutes, snooze_default_minutes, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			reminder_delta_minutes = excluded.reminder_delta_minutes,
			snooze_default_minutes = excluded.snooze_default_minutes,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.DB.Exec(query, settings.UserID, settings.ReminderDeltaMinutes, settings.SnoozeDefaultMinutes)
	if err != nil {
		return fmt.Errorf("failed to update notification settings: %w", err)
	}

	return nil
}

// CreateNotification schedules a new task notification
func (s *Store) CreateNotification(notification *core.TaskNotification) error {
	result, err := s.DB.Exec(
		"INSERT INTO task_notifications (task_id, user_id, notification_type, scheduled_at) VALUES (?, ?, ?, ?)",
		notification.TaskID, notification.UserID, notification.NotificationType, notification.ScheduledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	notification.ID = id
	return nil
}

// GetPendingNotifications retrieves all notifications that should be sent
// (scheduled_at <= now and sent_at is NULL)
func (s *Store) GetPendingNotifications(now time.Time) ([]*core.TaskNotification, error) {
	rows, err := s.DB.Query(
		`SELECT id, task_id, user_id, notification_type, scheduled_at, sent_at, created_at 
		FROM task_notifications 
		WHERE sent_at IS NULL AND scheduled_at <= ?
		ORDER BY scheduled_at ASC`,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*core.TaskNotification
	for rows.Next() {
		notification := &core.TaskNotification{}
		var sentAt sql.NullTime

		if err := rows.Scan(
			&notification.ID,
			&notification.TaskID,
			&notification.UserID,
			&notification.NotificationType,
			&notification.ScheduledAt,
			&sentAt,
			&notification.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// MarkNotificationSent marks a notification as sent with the current timestamp
func (s *Store) MarkNotificationSent(notificationID int64) error {
	query := `UPDATE task_notifications SET sent_at = CURRENT_TIMESTAMP WHERE id = ?`

	result, err := s.DB.Exec(query, notificationID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// DeleteNotificationsByTask deletes all pending notifications for a task
// This should be called when a task is completed or deleted
func (s *Store) DeleteNotificationsByTask(taskID int64) error {
	query := `DELETE FROM task_notifications WHERE task_id = ? AND sent_at IS NULL`

	_, err := s.DB.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete notifications for task: %w", err)
	}

	return nil
}

// GetNotificationByID retrieves a notification by its ID
func (s *Store) GetNotificationByID(id int64) (*core.TaskNotification, error) {
	notification := &core.TaskNotification{}
	var sentAt sql.NullTime

	err := s.DB.QueryRow(
		"SELECT id, task_id, user_id, notification_type, scheduled_at, sent_at, created_at FROM task_notifications WHERE id = ?",
		id,
	).Scan(
		&notification.ID,
		&notification.TaskID,
		&notification.UserID,
		&notification.NotificationType,
		&notification.ScheduledAt,
		&sentAt,
		&notification.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	if sentAt.Valid {
		notification.SentAt = &sentAt.Time
	}

	return notification, nil
}
